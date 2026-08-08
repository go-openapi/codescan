// Runs one scan and reports what it cost.
//
// In a worker for two reasons: a scan is seconds of solid CPU, and the shim's poll_oneoff busy-waits
// rather than yielding, so on the main thread it would stall the page outright.

import {
  WASI, File, Directory, OpenFile, ConsoleStdout, PreopenDirectory,
} from "./node_modules/@bjorn3/browser_wasi_shim/dist/index.js";
import { goMod, petSrc } from "./fixture.js";

const enc = new TextEncoder();
const file = (s) => new File(enc.encode(s));

// Compiling a 14 MB module is the expensive part and the result is reusable: proc_exit ends an
// instance for good, so every run needs a fresh instance but not a fresh compile.
let compiled = null;
let compileMs = 0;

async function moduleOnce(url) {
  if (compiled) return compiled;
  const t0 = performance.now();
  compiled = await WebAssembly.compileStreaming(fetch(url));
  compileMs = performance.now() - t0;

  return compiled;
}

function guestTree() {
  return new PreopenDirectory("/src", new Map([
    ["go.mod", file(goMod)],
    ["models", new Directory(new Map([["pet.go", file(petSrc)]]))],
  ]));
}

async function run(url) {
  const module = await moduleOnce(url);

  let out = "", err = "";
  const fds = [
    new OpenFile(new File([])),
    ConsoleStdout.lineBuffered((l) => { out += l + "\n"; }),
    ConsoleStdout.lineBuffered((l) => { err += l + "\n"; }),
    guestTree(),
  ];

  const args = [
    "genspec-wasi.wasm", "-loader=own", "-stub-stdlib",
    "-goos", "linux", "-goarch", "amd64",
    "-workdir", "/src", "./...",
  ];
  const wasi = new WASI(args, [], fds);

  // The shim spins inside poll_oneoff for the whole requested delay. Measure it: if Go asks for real
  // sleeps this is dead wall-clock, and it is the one thing that would rule this shim out.
  let pollCalls = 0, pollMs = 0;
  const imports = wasi.wasiImport;
  const rawPoll = imports.poll_oneoff.bind(imports);
  imports.poll_oneoff = (...a) => {
    const t = performance.now();
    const r = rawPoll(...a);
    pollMs += performance.now() - t;
    pollCalls++;

    return r;
  };

  const t1 = performance.now();
  const instance = await WebAssembly.instantiate(module, { wasi_snapshot_preview1: imports });
  const instantiateMs = performance.now() - t1;

  const t2 = performance.now();
  const code = wasi.start(instance);
  const runMs = performance.now() - t2;

  return { out, err, code, compileMs, instantiateMs, runMs, pollCalls, pollMs };
}

self.onmessage = async (e) => {
  try {
    self.postMessage({ ok: true, ...(await run(e.data.url)) });
  } catch (ex) {
    self.postMessage({ ok: false, error: String(ex && ex.stack || ex) });
  }
};
