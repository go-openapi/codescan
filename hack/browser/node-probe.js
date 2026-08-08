// The same run as the page, under Node, so the shim can be exercised without a browser.
//
// What this does NOT tell us: how long a browser engine takes to compile a 15 MB module. Everything
// else - the shim's filesystem, its poll_oneoff, and whether the artifact produces the right spec -
// is the same code.

import { readFile } from "node:fs/promises";
import {
  WASI, File, Directory, OpenFile, ConsoleStdout, PreopenDirectory,
} from "./node_modules/@bjorn3/browser_wasi_shim/dist/index.js";
import { goMod, petSrc } from "./fixture.js";

// The shim logs every path it touches unless told otherwise.
const { debug } = await import("./node_modules/@bjorn3/browser_wasi_shim/dist/debug.js");
debug.enable(false);

const enc = new TextEncoder();
const file = (s) => new File(enc.encode(s));

const bytes = await readFile("./genspec-wasi.wasm");
let t = performance.now();
const module = await WebAssembly.compile(bytes);
const compileMs = performance.now() - t;

// bigTree builds a module with many packages, so the scan runs long enough for Go's scheduler to
// idle - which is the only thing that makes it call poll_oneoff, and therefore the only thing that
// makes the shim's busy-wait matter.
function bigTree(pkgs, typesPer) {
  const entries = [["go.mod", file(goMod)]];
  for (let p = 0; p < pkgs; p++) {
    const files = new Map();
    for (let f = 0; f < 3; f++) {
      let src = `package p${p}\n\n`;
      for (let t = 0; t < typesPer; t++) {
        src += `// T${f}_${t} is a model.\n//\n// swagger:model t${p}_${f}_${t}\ntype T${f}_${t} struct {\n` +
          `\tID int64 \`json:"id"\`\n\tName string \`json:"name"\`\n\tTags []string \`json:"tags"\`\n}\n\n`;
      }
      files.set(`f${f}.go`, file(src));
    }
    entries.push([`p${p}`, new Directory(files)]);
  }

  return new PreopenDirectory("/src", new Map(entries));
}

async function run(label, extraArgs, tree) {
  let out = "", err = "";
  const fds = [
    new OpenFile(new File([])),
    ConsoleStdout.lineBuffered((l) => { out += l + "\n"; }),
    ConsoleStdout.lineBuffered((l) => { err += l + "\n"; }),
    tree || new PreopenDirectory("/src", new Map([
      ["go.mod", file(goMod)],
      ["models", new Directory(new Map([["pet.go", file(petSrc)]]))],
    ])),
  ];

  const args = ["genspec-wasi.wasm", "-loader=own", "-goos", "linux", "-goarch", "amd64",
    ...extraArgs, "-workdir", "/src", "./..."];
  const wasi = new WASI(args, [], fds);

  let pollCalls = 0, pollMs = 0, maxPollMs = 0;
  const imports = wasi.wasiImport;
  const rawPoll = imports.poll_oneoff.bind(imports);
  imports.poll_oneoff = (...a) => {
    const t0 = performance.now();
    const r = rawPoll(...a);
    const d = performance.now() - t0;
    pollMs += d; maxPollMs = Math.max(maxPollMs, d); pollCalls++;

    return r;
  };

  t = performance.now();
  const instance = await WebAssembly.instantiate(module, { wasi_snapshot_preview1: imports });
  const instantiateMs = performance.now() - t;

  t = performance.now();
  const code = wasi.start(instance);
  const runMs = performance.now() - t;

  let defs = null;
  try { defs = Object.keys(JSON.parse(out).definitions || {}); } catch { /* not JSON */ }

  console.log(`--- ${label} ---`);
  console.log(`  exit=${code}  instantiate=${instantiateMs.toFixed(1)}ms  run=${runMs.toFixed(0)}ms`);
  console.log(`  poll_oneoff: ${pollCalls} calls, ${pollMs.toFixed(1)}ms total, ${maxPollMs.toFixed(1)}ms worst`);
  console.log(`  definitions: ${defs ? defs.length : "STDOUT WAS NOT JSON"}`);
  if (err.trim()) console.log(`  stderr: ${err.trim().split("\n").slice(0, 3).join(" | ")}`);

  return { defs, out };
}

console.log(`compile: ${compileMs.toFixed(0)}ms for ${(bytes.length / 1048576).toFixed(1)} MB\n`);
const a = await run("small: 1 package", ["-stub-stdlib"]);
const b = await run("small again, same compiled module", ["-stub-stdlib"]);
await run("medium: 50 packages, 150 files, 750 models", ["-stub-stdlib"], bigTree(50, 5));
await run("large: 200 packages, 600 files, 6000 models", ["-stub-stdlib"], bigTree(200, 10));
console.log(`\nreuse of the compiled module across runs: ${JSON.stringify(a.defs) === JSON.stringify(b.defs) ? "OK" : "DIFFERENT"}`);
