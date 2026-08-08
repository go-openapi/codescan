const worker = new Worker("./worker.js", { type: "module" });
const $ = (id) => document.getElementById(id);

worker.onmessage = (e) => {
  const r = e.data;
  if (!r.ok) {
    $("status").innerHTML = `<span class="bad">FAILED</span>`;
    $("err").textContent = r.error;

    return;
  }

  const defs = (() => {
    try { return Object.keys(JSON.parse(r.out).definitions || {}); } catch { return null; }
  })();

  $("status").innerHTML = defs
    ? `<span class="ok">ran, exit ${r.code}, definitions: ${defs.join(", ") || "(none)"}</span>`
    : `<span class="bad">ran, exit ${r.code}, but stdout was not JSON</span>`;

  $("timings").textContent = [
    `compile      ${r.compileMs.toFixed(0)} ms   (once; reusable across runs)`,
    `instantiate  ${r.instantiateMs.toFixed(1)} ms`,
    `run          ${r.runMs.toFixed(0)} ms`,
    `poll_oneoff  ${r.pollCalls} calls, ${r.pollMs.toFixed(1)} ms spinning`,
  ].join("\n");
  $("out").textContent = r.out || "(empty)";
  $("err").textContent = r.err || "(empty)";
};

worker.postMessage({ url: "./genspec-wasi.wasm" });
