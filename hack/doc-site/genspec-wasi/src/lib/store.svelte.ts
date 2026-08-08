import { getContext, setContext } from 'svelte';
import {
  defaultOptions,
  type Anchor,
  type Diagnostic,
  type ScanOptions,
  type RuntimeStats,
  type Progress,
  type ScanResult,
  type Severity,
  type SourceFile,
  type WorkerMessage,
  type WorkerReply,
} from './types';
import { exampleById, firstExample, sampleFiles } from './examples';
import { needsVendoring, reroot } from './tree';
import { render, type Format, type Rendered } from './render';
import { AnchorIndex, pointerAt, spanFor } from './track';
import { findMatches, nearestMatch, stepIndex } from './search';
import type { GutterMark } from './editor';

// The playground's whole state.
//
// A factory rather than a module-level singleton. One instance per page is the shape the doc site
// wants today - the playground replaces a whole example box rather than sitting beside another one -
// but a singleton makes a second mount silently share the first one's tree, and that is a bug nobody
// would think to look for. Cheap here; invasive once every component imports the instance directly.
export class Playground {
  files = $state<SourceFile[]>(sampleFiles());
  selected = $state<string>(sampleFiles()[1]?.path ?? '');
  options = $state<ScanOptions>({ ...defaultOptions });

  spec = $state<unknown>(null);
  diagnostics = $state<Diagnostic[]>([]);
  provenance = $state<Anchor[]>([]);
  error = $state('');

  // notice is something worth saying that is not a failure: a tree big enough to be slow, say.
  // Separate from error because the two want opposite treatments - one interrupts, one informs.
  notice = $state('');

  running = $state(false);

  /** What the worker is doing, while it is doing it. Null between runs. */
  progress = $state<Progress | null>(null);
  compileMs = $state(0);
  prepareMs = $state(0);
  instantiateMs = $state(0);
  runMs = $state(0);

  /** The guest's linear memory after the run: what the tab paid, and a true peak. */
  memoryBytes = $state(0);
  /** What the Go runtime inside the guest says it spent. The host cannot see this. */
  runtime = $state<RuntimeStats | null>(null);

  // scanned is false until a run has produced something. Distinguishes "nothing to show yet" from
  // "the scan found nothing", which look identical and mean opposite things.
  scanned = $state(false);

  /** Where the artifact is served from. Set by whoever mounts the app; see src/main.ts. */
  wasmUrl = $state(new URL(`${import.meta.env.BASE_URL}genspec-wasi.wasm`, location.href).href);

  format = $state<Format>('json');

  /** Which bundled example is loaded, or '' once the reader opens something of their own. */
  example = $state<string>(firstExample);

  // Which severities are being looked at. In the store rather than in the drawer because the gutters
  // read it too: hiding hints from the list while leaving their dots in the margin is half a filter.
  shown = $state<Record<Severity, boolean>>({ error: true, warning: true, hint: true });

  // Tracking is a mode rather than always-on. Following every cursor movement across the pane is
  // startling when you are reading rather than asking, and the terminal UI makes the same choice.
  tracking = $state(true);

  /** The spec node the source cursor produced. */
  trackedPointer = $state<string | null>(null);
  /** The source position the spec cursor came from. */
  trackedSource = $state<{ file: string; line: number } | null>(null);

  // A reveal is a request, not a position: the same line asked for twice has to scroll twice, so it
  // carries a nonce the pane can react to.
  specReveal = $state<{ line: number; nonce: number } | null>(null);
  specCursor = $state<{ line: number; nonce: number } | null>(null);
  sourceReveal = $state<{ line: number; nonce: number } | null>(null);
  #nonce = 0;

  // Rendering walks the whole document, so it is computed once per (spec, format) rather than on
  // each read - a spec with 200 definitions is not something to re-render per keystroke.
  rendered: Rendered = $derived(render(this.spec ?? {}, this.format));
  anchors: AnchorIndex = $derived(new AnchorIndex(this.provenance));

  // Searching the rendered document. Line-based and case-insensitive, as in the terminal UI.
  // Declared after `rendered`, which it reads: a class field cannot derive from one below it.
  query = $state('');
  matchIndex = $state(0);
  /** The last line the reader was on in the spec, so a fresh search starts near it. */
  specLine = $state(1);

  matches: number[] = $derived(findMatches(this.rendered.text, this.query));

  #worker: Worker | null = null;
  #timer: ReturnType<typeof setTimeout> | null = null;
  #stale = false;

  // What the worker currently holds, so a rescan can ship a patch instead of the whole tree. Cleared
  // whenever the worker's copy is not to be trusted - a new tree, or a scan that failed.
  #sent = new Map<string, string>();

  get current(): SourceFile | undefined {
    return this.files.find((f) => f.path === this.selected);
  }

  // counts is what the status line reads. Derived rather than carried in the envelope: the document
  // already answers it, and a second copy is a second thing to keep true.
  get visibleDiagnostics(): Diagnostic[] {
    return this.diagnostics.filter((d) => this.shown[d.severity]);
  }

  get counts(): { definitions: number; paths: number; errors: number; warnings: number; hints: number } {
    const doc = this.spec as { definitions?: object; paths?: object } | null;
    const by = (s: Diagnostic['severity']) => this.diagnostics.filter((d) => d.severity === s).length;

    return {
      definitions: Object.keys(doc?.definitions ?? {}).length,
      paths: Object.keys(doc?.paths ?? {}).length,
      errors: by('error'),
      warnings: by('warning'),
      hints: by('hint'),
    };
  }

  edit(path: string, text: string) {
    const file = this.files.find((f) => f.path === path);
    if (!file || file.text === text) {
      return;
    }
    file.text = text;

    // Edit and see it: the loop the whole thing exists for. Debounced because a scan is real work,
    // and re-armed on every keystroke so a burst of typing costs one scan rather than one each.
    if (this.#timer !== null) {
      clearTimeout(this.#timer);
    }
    this.#timer = setTimeout(() => {
      this.#timer = null;
      this.run();
    }, 700);
  }

  // ---- tracking ----------------------------------------------------------

  // fromSource answers "what did this line produce", and is the inexact direction: an anchor is a
  // point, so a cursor that is not on one is attributed to the nearest. See AnchorIndex.forLine.
  fromSource(line: number) {
    if (!this.tracking || !this.selected) {
      return;
    }
    const anchor = this.anchors.forLine(this.selected, line);
    this.trackedPointer = anchor?.pointer ?? null;
    this.trackedSource = null;

    const span = anchor ? spanFor(this.rendered.spans, anchor.pointer) : null;
    if (span) {
      this.specReveal = { line: span.from, nonce: ++this.#nonce };
    }
  }

  // fromSpec answers "where did this come from". Exact on the spec side - spans are ranges - and
  // then climbs to the nearest anchored ancestor, since only code-detail nodes carry provenance.
  fromSpec(line: number) {
    this.specLine = line;
    if (!this.tracking) {
      return;
    }
    const pointer = pointerAt(this.rendered.spans, line);
    const anchor = pointer ? this.anchors.anchorFor(pointer) : null;

    // Mark the node that answered, not the line that was clicked. A cursor on `"type": "object"`
    // resolves to the whole definition, and showing that span is what explains the answer.
    this.trackedPointer = anchor?.pointer ?? null;

    if (!anchor?.file || !anchor.line) {
      this.trackedSource = null;

      return;
    }
    this.openSource(anchor.file, anchor.line);
  }

  // reveal is the diagnostic's jump, and points BOTH panes.
  //
  // The other two directions each start from a cursor, so the pane it sits in already shows where you
  // are and only the far side needs marking. A diagnostic sits in neither: it is a third place
  // talking about the other two, so both have to be told, and both have to be scrolled.
  reveal(file: string, line: number) {
    if (!this.openSource(file, line)) {
      return;
    }

    const anchor = this.anchors.forLine(file, line);
    const span = anchor ? spanFor(this.rendered.spans, anchor.pointer) : null;
    this.trackedPointer = anchor?.pointer ?? null;
    if (span) {
      this.specReveal = { line: span.from, nonce: ++this.#nonce };
    }
  }

  // openSource selects a file and points the source pane at a line, reporting whether it could.
  //
  // A position outside the scanned tree - a type in GOROOT reached through a $ref - is not a file we
  // hold, so it is reported rather than silently doing nothing.
  openSource(file: string, line: number): boolean {
    if (!this.files.some((f) => f.path === file)) {
      this.notice = `${file}:${line} is outside the module you opened, so there is nothing to show.`;

      return false;
    }

    this.selected = file;
    this.trackedSource = { file, line };
    this.sourceReveal = { line, nonce: ++this.#nonce };

    return true;
  }

  // specMarks puts the scanner's complaints on the SPEC side.
  //
  // A diagnostic names a source position, so reaching the document means going the inexact way - the
  // nearest anchor to that line, then where its pointer is written. Worth it: without this the spec
  // pane has a gutter and nothing to put in it, and a reader looking at a node has no way to see that
  // something was said about it.
  get specMarks(): GutterMark[] {
    const marks: GutterMark[] = [];
    for (const d of this.visibleDiagnostics) {
      if (!d.file || !d.line) {
        continue;
      }
      const anchor = this.anchors.forLine(d.file, d.line);
      const span = anchor ? spanFor(this.rendered.spans, anchor.pointer) : null;
      if (span) {
        marks.push({ line: span.from, severity: d.severity, title: `${d.severity}: ${d.message}` });
      }
    }

    return marks;
  }

  // Where the spec cursor points back to, for the pane to show. The mirror of the source pane's
  // trail: each side says what the other would light up.
  get specOrigin(): { pointer: string; where: string } | null {
    const pointer = pointerAt(this.rendered.spans, this.specLine);
    if (!pointer) {
      return null;
    }
    const anchor = this.anchors.anchorFor(pointer);

    return {
      pointer,
      where: anchor?.file && anchor.line ? `${anchor.file}:${anchor.line}` : 'no source anchored',
    };
  }

  // ---- search ------------------------------------------------------------

  get matchInfo(): { current: number; total: number } {
    return { current: this.matches.length ? this.matchIndex + 1 : 0, total: this.matches.length };
  }

  get currentMatch(): number | null {
    return this.matches[this.matchIndex] ?? null;
  }

  search(query: string) {
    this.query = query;
    this.matchIndex = nearestMatch(this.matches, this.specLine);
    this.#goToMatch();
  }

  stepMatch(direction: 1 | -1) {
    if (!this.matches.length) {
      return;
    }
    this.matchIndex = stepIndex(this.matchIndex, this.matches.length, direction);
    this.#goToMatch();
  }

  clearSearch() {
    this.query = '';
    this.matchIndex = 0;
  }

  // The cursor goes on the match, not just the viewport. That is what makes a search a way of
  // navigating: with the caret there, the tracked node and the source pane follow it, exactly as
  // they would had the reader clicked.
  #goToMatch() {
    const line = this.currentMatch;
    if (line) {
      this.specCursor = { line, nonce: ++this.#nonce };
    }
  }

  clearTracking() {
    this.trackedPointer = null;
    this.trackedSource = null;
  }

  // open REPLACES the tree rather than merging into it.
  //
  // Merging was wrong in a way that only shows up on the second scan: whatever was already loaded -
  // the sample, or a previous upload - stays behind, and the scan sees two modules at once, with two
  // go.mod files and packages that do not belong together.
  open(incoming: SourceFile[]) {
    const files = reroot(incoming);
    if (!files.length) {
      this.error = 'No Go files found in that selection.';

      return;
    }

    this.example = '';
    this.files = files;
    this.selected = (files.find((f) => f.path.endsWith('.go')) ?? files[0]).path;
    this.clear();
  }

  // openExample replaces the tree with one of the bundled modules and scans it.
  //
  // Scanned rather than merely loaded: an example that sits there until the reader finds a button
  // teaches nothing, and seeing the output beside the source is the entire point of arriving here.
  openExample(id: string) {
    const chosen = exampleById(id);
    this.example = chosen.id;
    this.files = chosen.files();
    this.selected = (this.files.find((f) => f.path.endsWith('.go')) ?? this.files[0]).path;
    this.clear();
    this.run();
  }

  reset() {
    this.openExample(this.example || firstExample);
  }

  // vendorAdvice names a mistake that is answerable before the scan runs.
  //
  // The alternative is a wall of synthesized-import warnings describing the consequence rather than
  // the cause, which is a poor way to learn that a browser has no module cache.
  get vendorAdvice(): string {
    if (!needsVendoring(this.files)) {
      return '';
    }

    return 'This module requires dependencies and carries no vendor directory. Run `go mod vendor` '
      + 'and open it again — there is no module cache here, so their types and their annotations can '
      + 'only arrive as source.';
  }

  // clear drops a result that describes a tree which is gone.
  clear() {
    this.#sent.clear();
    this.clearSearch();
    this.spec = null;
    this.diagnostics = [];
    this.provenance = [];
    this.error = '';
    this.notice = '';
    this.compileMs = 0;
    this.prepareMs = 0;
    this.instantiateMs = 0;
    this.runMs = 0;
    this.memoryBytes = 0;
    this.runtime = null;
    this.scanned = false;
    this.clearTracking();
  }

  run() {
    if (this.#timer !== null) {
      clearTimeout(this.#timer);
      this.#timer = null;
    }
    if (this.running) {
      this.#stale = true;

      return;
    }
    this.running = true;
    this.error = '';
    this.progress = { phase: 'scanning' };

    // The worker keeps the compiled module between runs; only the instance is new each time.
    this.#worker ??= new Worker(new URL('../worker/scan-worker.ts', import.meta.url), { type: 'module' });
    this.#worker.onmessage = (event: MessageEvent<WorkerMessage>) => {
      const message = event.data;
      if (message.kind === 'progress') {
        this.progress = message;

        return;
      }
      this.progress = null;
      this.#accept(message);
    };

    const files = $state.snapshot(this.files) as SourceFile[];
    this.#worker.postMessage({
      ...this.#delta(files),
      options: $state.snapshot(this.options),
      wasmUrl: this.wasmUrl,
    });
  }

  // delta works out what the worker still needs to be told.
  //
  // The first send is everything; after that, only what moved. A tree the size of a vendored module
  // costs a structured clone and a UTF-8 encode of all of it otherwise, on every rescan, to report
  // that one file changed.
  #delta(files: SourceFile[]): { files?: SourceFile[]; patch?: SourceFile[]; drop?: string[] } {
    if (this.#sent.size === 0) {
      this.#remember(files);

      return { files };
    }

    const patch = files.filter((f) => this.#sent.get(f.path) !== f.text);
    const present = new Set(files.map((f) => f.path));
    const drop = [...this.#sent.keys()].filter((p) => !present.has(p));

    this.#remember(files);

    return { patch, drop };
  }

  #remember(files: SourceFile[]) {
    this.#sent = new Map(files.map((f) => [f.path, f.text]));
  }

  #accept(reply: WorkerReply) {
    this.running = false;

    // An edit that landed mid-scan describes a tree this result does not. Run again rather than
    // showing output for source that has already moved.
    if (this.#stale) {
      this.#stale = false;
      queueMicrotask(() => this.run());
    }

    if (!reply.ok) {
      this.error = reply.error;
      this.scanned = true;
      // The worker may have died before applying the patch, so what it holds is no longer known.
      // Start the next scan from a full send rather than from a guess.
      this.#sent.clear();

      return;
    }

    const {
      spec, diagnostics, provenance, runtime,
      compileMs, instantiateMs, prepareMs, runMs, memoryBytes,
    }: ScanResult = reply;
    this.spec = spec;
    this.diagnostics = diagnostics;
    this.provenance = provenance;
    this.compileMs = compileMs;
    this.prepareMs = prepareMs;
    this.instantiateMs = instantiateMs;
    this.runMs = runMs;
    this.memoryBytes = memoryBytes;
    this.runtime = runtime ?? null;
    this.scanned = true;
  }
}

const key = Symbol('codescan-playground');

export function providePlayground(): Playground {
  const instance = new Playground();
  setContext(key, instance);

  return instance;
}

export function usePlayground(): Playground {
  const instance = getContext<Playground | undefined>(key);
  if (!instance) {
    throw new Error('usePlayground() outside a playground: call providePlayground() in the root component');
  }

  return instance;
}
