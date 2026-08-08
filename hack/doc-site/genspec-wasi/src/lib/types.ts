// A file in the tree the user is scanning. Paths are relative to the module root and always use
// forward slashes, matching what the guest filesystem expects.
export type SourceFile = {
  path: string;
  text: string;
};

// The subset of codescan's options the playground exposes. Every one of these is a genspec-wasi flag;
// see cmd/genspec-wasi/README.md for the whole set.
export type ScanOptions = {
  scanModels: boolean;
  pruneUnusedModels: boolean;
  refAliases: boolean;
  transparentAliases: boolean;
  setXNullableForPointers: boolean;
  skipExtensions: boolean;
  buildTags: string;
};

export const defaultOptions: ScanOptions = {
  scanModels: true,
  pruneUnusedModels: false,
  refAliases: false,
  transparentAliases: false,
  setXNullableForPointers: false,
  skipExtensions: false,
  buildTags: '',
};

export type Severity = 'error' | 'warning' | 'hint';

// One observation the scanner made, positioned. file is relative to the module root for anything in
// the scanned tree, and absolute for what came from outside it - GOROOT, a vendored dependency's
// origin - which is how a consumer tells "a file I hold" from "a file I do not".
export type Diagnostic = {
  severity: Severity;
  code: string;
  message: string;
  file?: string;
  line?: number;
  col?: number;
};

// Where a node in the emitted document came from. pointer is RFC 6901.
//
// Only anchor nodes appear: type declarations, fields, values, route and meta blocks. A finer node
// resolves to its nearest anchored ancestor here, at the consumer.
export type Anchor = {
  pointer: string;
  file?: string;
  line?: number;
  col?: number;
};

// What the Go runtime spent, read inside the guest just after the scan.
export type RuntimeStats = {
  /** Memory obtained from the host. Under wasm it never shrinks, so this is the high-water mark. */
  sys: number;
  /** Still live once the scan is done: the spec, the index, the type graph. */
  heapAlloc: number;
  /** Everything ever allocated, garbage included. */
  totalAlloc: number;
  /** How many collections the scan took. */
  collections: number;
};

// What genspec-wasi -format=json writes.
export type Envelope = {
  spec: unknown;
  diagnostics: Diagnostic[];
  provenance: Anchor[];
  runtime?: RuntimeStats;
};

export type ScanResult = Envelope & {
  /** Compiling the artifact. Zero on every scan after the first - the module is kept. */
  compileMs: number;
  /** Building an instance, which every scan needs afresh. */
  instantiateMs: number;
  /** Turning the tree into a guest filesystem. Small on a rescan, since only a patch is encoded. */
  prepareMs: number;
  /** The scan itself. */
  runMs: number;
  /**
   * The guest's linear memory after the run - what the tab actually paid for the scan.
   *
   * The closest thing a browser has to the RSS a host runtime would report, and it is a true peak:
   * WebAssembly memory grows and never shrinks.
   */
  memoryBytes: number;
};

// What the worker is told. The tree arrives whole the first time and as a patch afterwards: a
// vendored module is tens of megabytes, and shipping all of it across on every keystroke-triggered
// rescan costs a structured clone and a UTF-8 encode of the entire thing to say that one file moved.
export type ScanRequest = {
  options: ScanOptions;
  wasmUrl: string;
  /** Present on a full send; the worker forgets whatever it held. */
  files?: SourceFile[];
  /** Files added or changed since the last send. */
  patch?: SourceFile[];
  /** Paths that are gone. */
  drop?: string[];
};

// What the worker says while it works.
//
// The first scan of a session spends most of its time fetching 8 MB of artifact, which is not
// scanning and should not claim to be - a reader who arrives to an unexplained wait has no way to
// tell a slow network from a broken page.
export type Progress = {
  phase: 'fetching' | 'compiling' | 'scanning';
  /** Bytes of the artifact received so far. */
  received?: number;
  /** Bytes expected, when the response says so plainly enough to be trusted. */
  total?: number;
};

export type WorkerMessage =
  | ({ kind: 'done'; ok: true } & ScanResult)
  | { kind: 'done'; ok: false; error: string }
  | ({ kind: 'progress' } & Progress);

export type WorkerReply =
  | ({ ok: true } & ScanResult)
  | { ok: false; error: string };
