<!--
SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
SPDX-License-Identifier: Apache-2.0
-->

# Browser probe

A throwaway check that the `wasip1` artifact runs in a browser, and what it costs. Not the
playground — the seed of one.

```sh
npm install
GOOS=wasip1 GOARCH=wasm go build -o genspec-wasi.wasm ../../cmd/genspec-wasi

node node-probe.js          # the whole thing without a browser
npm run serve               # then open http://localhost:8099/ for the real one
```

`genspec-wasi.wasm` and `node_modules/` are generated; neither is committed.

## What it establishes

Under [`@bjorn3/browser_wasi_shim`](https://github.com/bjorn3/browser_wasi_shim) (114 KB, no
dependencies), scanning a module held entirely in the shim's in-memory filesystem:

| workload | run | `poll_oneoff` | definitions |
|---|---|---|---|
| 1 package | 86 ms cold, 10 ms warm | 0 calls | 1 |
| 50 packages, 150 files | 492 ms | 0 calls | 750 |
| 200 packages, 600 files | 3548 ms | 0 calls | 6000 |

Compiling the 14.2 MB module takes 22 ms once; instantiating is ~4–5 ms and does not grow with the
workload.

**`poll_oneoff` never fires.** It was the risk worth checking, because the shim implements it by
spinning (`while (endTime > getNow()) {}`) rather than yielding. Go reaches it only through the
netpoller, and only when the scheduler runs out of work — which a single-goroutine scan never does.
It is not guaranteed to stay at zero: longer runs under wazero showed 9–14 calls. But the shim
supports exactly the shape Go emits (one clock subscription, which is all `netpollinit` ever
registers), so when it does fire it works, and the cost is bounded by the delay Go asked for.
That is a reason to keep the scan in a worker, not a reason to avoid this shim.

Measured under Node, which shares V8 with Chrome. What it does not measure is a browser engine's
own compile time for a 14 MB module, or anything about Firefox and Safari.

## One run per instance

The module exports `_start` and `memory`, and nothing else. It is a WASI **command**: run to
completion, terminated by `proc_exit`. Go's `wasip1` target emits no reactor form — there is no
`_initialize` and no callable export — so a fresh instance per scan is forced rather than chosen.

That costs nothing worth avoiding. Compile once, keep the `WebAssembly.Module`, and instantiate per
run; the measurements above are with the module reused across runs. It also means each scan starts
on a clean heap, so nothing leaks from one run into the next.
