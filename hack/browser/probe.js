const worker = new Worker("./worker.js", { type: "module" });
const $ = (id) => document.getElementById(id);

// The status line is assembled, not written as markup. Everything in it comes back from the
// worker — an exit code, and definition names read out of whatever module was scanned — and
// none of that has any business being parsed as HTML.
const status = (cls, text) => {
  const span = document.createElement("span");
  span.className = cls;
  span.textContent = text;
  $("status").replaceChildren(span);
};

worker.onmessage = (e) => {
  const r = e.data;
  if (!r.ok) {
    status("bad", "FAILED");
    $("err").textContent = r.error;

    return;
  }

  const defs = (() => {
    try { return Object.keys(JSON.parse(r.out).definitions || {}); } catch { return null; }
  })();

  status(
    defs ? "ok" : "bad",
    defs
      ? `ran, exit ${r.code}, definitions: ${defs.join(", ") || "(none)"}`
      : `ran, exit ${r.code}, but stdout was not JSON`,
  );

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
