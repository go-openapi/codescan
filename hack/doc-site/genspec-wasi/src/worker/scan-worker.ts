// Owns the WebAssembly side of the playground.
//
// It runs in a worker for two reasons. A scan is seconds of solid CPU on a large tree, and the WASI
// shim implements poll_oneoff by spinning rather than yielding - rare, but on the main thread either
// would stall the page.

import {
  WASI, File, Directory, OpenFile, ConsoleStdout, PreopenDirectory,
} from '@bjorn3/browser_wasi_shim';
import type { Inode } from '@bjorn3/browser_wasi_shim';
import type { Envelope, Progress, ScanRequest, WorkerMessage, WorkerReply } from '../lib/types';
import { argvFor } from '../lib/flags';
import { explain } from '../lib/errors';
import { expectedBytes } from '../lib/format';

const encoder = new TextEncoder();

// The artifact is a WASI command: it exports _start and nothing else, runs to completion, and ends
// at proc_exit. So every scan needs its own instance - but not its own compile, which is the
// expensive half and is reusable.
let compiled: WebAssembly.Module | null = null;

// moduleOnce reports how long compiling took, and zero when it did not have to.
//
// That zero is the interesting number: it is the whole reason the worker is kept alive between
// scans, and the difference between the first scan of a session and every one after it.
async function moduleOnce(
  url: string,
  report: (p: Progress) => void,
): Promise<{ module: WebAssembly.Module; compileMs: number }> {
  if (compiled) {
    return { module: compiled, compileMs: 0 };
  }

  const started = performance.now();
  const response = await fetch(url);
  if (!response.ok || !response.body) {
    throw new Error(`the scanner could not be fetched (${response.status} ${response.statusText})`);
  }

  const total = expectedBytes(response.headers);

  // Counted through a transform rather than by reading the body to completion first: the module
  // still compiles as it arrives, which is most of why compileStreaming is worth using.
  let received = 0;
  let announced = 0;
  const counting = new TransformStream<Uint8Array, Uint8Array>({
    transform(chunk, controller) {
      received += chunk.byteLength;
      // A message per chunk would be thousands of them. Every quarter megabyte is smooth enough for
      // a bar and cheap enough to ignore.
      if (received - announced >= 256 * 1024) {
        announced = received;
        report({ phase: 'fetching', received, total });
      }
      controller.enqueue(chunk);
    },
  });

  report({ phase: 'fetching', received: 0, total });
  const counted = new Response(response.body.pipeThrough(counting), {
    headers: { 'content-type': 'application/wasm' },
  });

  compiled = await WebAssembly.compileStreaming(counted);
  report({ phase: 'compiling' });

  return { module: compiled, compileMs: performance.now() - started };
}

// The encoded tree, kept between scans.
//
// Encoding is the expensive half of preparing a scan - a vendored module is tens of megabytes of
// UTF-8 - and almost none of it changes between one scan and the next. Bytes are cached per path;
// the directory structure is rebuilt each run from them, which is pointer work.
//
// Rebuilt rather than reused because a PreopenDirectory is handed to a WASI instance, and each scan
// gets a fresh instance. Sharing the inode graph across instances would probably work and is not
// worth finding out about at the cost of a few object allocations.
const encoded = new Map<string, Uint8Array>();

function applyPatch(request: ScanRequest) {
  if (request.files) {
    encoded.clear();
    for (const file of request.files) {
      encoded.set(file.path, encoder.encode(file.text));
    }

    return;
  }

  for (const path of request.drop ?? []) {
    encoded.delete(path);
  }
  for (const file of request.patch ?? []) {
    encoded.set(file.path, encoder.encode(file.text));
  }
}

// treeFor turns the flat map of paths into the nested directories the guest filesystem expects.
function treeFor(): PreopenDirectory {
  const root = new Map<string, Inode>();

  for (const [path, bytes] of encoded) {
    const parts = path.split('/').filter(Boolean);
    const name = parts.pop();
    if (!name) {
      continue;
    }

    let dir: Map<string, Inode> = root;
    for (const part of parts) {
      const existing = dir.get(part);
      const child = existing instanceof Directory ? existing : new Directory(new Map());
      if (child !== existing) {
        dir.set(part, child);
      }
      dir = child.contents;
    }
    dir.set(name, new File(bytes));
  }

  return new PreopenDirectory('/src', root);
}

async function scan(request: ScanRequest, report: (p: Progress) => void): Promise<WorkerReply> {
  const { module, compileMs } = await moduleOnce(request.wasmUrl, report);

  // Under -format=json the whole result arrives on stdout as one object, and stderr carries only a
  // failure. Both are still collected: a guest that dies mid-write leaves half a document, and the
  // line on stderr is the only thing that says why.
  const preparing = performance.now();
  applyPatch(request);
  const tree = treeFor();
  const prepareMs = performance.now() - preparing;

  let out = '';
  let errs = '';
  const fds = [
    new OpenFile(new File([])),
    ConsoleStdout.lineBuffered((line) => { out += line + '\n'; }),
    ConsoleStdout.lineBuffered((line) => { errs += line + '\n'; }),
    tree,
  ];

  const wasi = new WASI(argvFor(request.options, 'embedded'), [], fds);

  // Every scan needs its own instance - the artifact is a WASI command that ends at proc_exit - so
  // this cost is paid every time, unlike the compile above.
  const instantiating = performance.now();
  const instance = await WebAssembly.instantiate(module, {
    wasi_snapshot_preview1: wasi.wasiImport as unknown as WebAssembly.ModuleImports,
  });
  const instantiateMs = performance.now() - instantiating;

  report({ phase: 'scanning' });

  const started = performance.now();
  // The shim wants the memory export named in the type; the artifact does export it.
  const exitCode = wasi.start(
    instance as unknown as { exports: { memory: WebAssembly.Memory; _start: () => unknown } },
  );
  const runMs = performance.now() - started;

  // The guest's linear memory, which is what the tab paid. A true peak rather than a sample:
  // WebAssembly memory grows and never shrinks, so reading it after the run reads the high-water
  // mark. The instance is discarded straight after this, so it has to be read here or not at all.
  const memoryBytes =
    (instance.exports.memory as WebAssembly.Memory | undefined)?.buffer.byteLength ?? 0;

  if (exitCode !== 0) {
    // The command prints one line and exits: a pattern it could not match, a flag it does not know,
    // a module it could not read. That line is the whole message, so pass it through rather than
    // reporting the exit code, which tells a user nothing.
    return { ok: false, error: explain(errs.trim()) || `the scanner exited with code ${exitCode}` };
  }

  const envelope = parse(out);
  if (!envelope) {
    return { ok: false, error: errs.trim() || 'the scanner produced no result' };
  }

  return { ok: true, ...envelope, compileMs, instantiateMs, prepareMs, runMs, memoryBytes };
}

// parse reads the envelope, tolerating a document that never finished being written.
//
// A scan that runs out of memory takes the guest down mid-write, and the truncated JSON that reaches
// us is not a result - surfacing it as a parse error would blame the format for what was a resource
// failure, so the caller reports what is on stderr instead.
function parse(out: string): Envelope | null {
  if (!out.trim()) {
    return null;
  }

  try {
    const value = JSON.parse(out) as Partial<Envelope>;
    if (!value || typeof value !== 'object' || !('spec' in value)) {
      return null;
    }

    return {
      spec: value.spec,
      diagnostics: value.diagnostics ?? [],
      provenance: value.provenance ?? [],
      runtime: value.runtime,
    };
  } catch {
    return null;
  }
}

self.onmessage = async (event: MessageEvent<ScanRequest>) => {
  const report = (p: Progress) => self.postMessage({ kind: 'progress', ...p } as WorkerMessage);

  try {
    self.postMessage({ kind: 'done', ...(await scan(event.data, report)) } as WorkerMessage);
  } catch (err) {
    // A failure here is the scanner refusing the input, not a crash to hide: surface it where the
    // diagnostics go.
    self.postMessage({
      kind: 'done',
      ok: false,
      error: err instanceof Error ? err.message : String(err),
    } as WorkerMessage);
  }
};
