import { mount } from 'svelte';
import './styles/tokens.css';
import './styles/base.css';
import App from './App.svelte';

// Mounts into every element that asks for a playground, rather than into one fixed id.
//
// That is what lets a documentation page place one wherever it likes: the page owns the container,
// its size and its position, and says where the artifact lives. A fixed id would have made the app a
// page rather than a component, and a second one on the same page impossible.
//
//   <div data-codescan-playground data-wasm="/codescan/playground/genspec-wasi.wasm"></div>
//
// data-wasm is optional. Left out, the artifact is looked for beside the script that loaded the app,
// which is how the standalone page finds it.
function playgroundsOn(root: ParentNode): HTMLElement[] {
  return [...root.querySelectorAll<HTMLElement>('[data-codescan-playground]')];
}

// artifactURL is what the host element asked for, or nothing.
//
// Nothing means the store's own default, which is relative to the site base - correct for the
// standalone page, where the artifact sits beside index.html. Resolving it against the bundle's own
// address instead would look more robust and be wrong there, the bundle living in assets/.
function artifactURL(host: HTMLElement): string | undefined {
  const named = host.dataset.wasm;

  return named ? new URL(named, location.href).href : undefined;
}

for (const host of playgroundsOn(document)) {
  mount(App, { target: host, props: { wasmUrl: artifactURL(host) } });
}
