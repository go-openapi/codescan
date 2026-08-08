// Failures that deserve a better sentence than the one they arrive with.
//
// Kept out of the worker so it can be tested without pulling in the WASI shim, the same reason
// flags.ts and tree.ts live here.

// explain names the one failure whose message is technically accurate and completely unhelpful.
//
// The page and the artifact ship separately - one is a bundle, the other a 20 MB asset a browser is
// entitled to keep - so the page can outrun the artifact it is talking to. What comes back is the
// flag package refusing an argument, followed by the whole usage text, and nothing in it says the
// two are different ages.
export function explain(stderr: string): string {
  if (/flag provided but not defined:\s*-format/.test(stderr)) {
    return 'The scanner artifact is older than this page: it does not understand -format=json, '
      + 'which is how positions and cross-references get here at all.\n\n'
      + 'Running locally? Rebuild it with `npm run wasm`. Otherwise your browser is holding a '
      + 'cached copy — a reload bypassing the cache should pick up the current one.';
  }

  return stderr;
}
