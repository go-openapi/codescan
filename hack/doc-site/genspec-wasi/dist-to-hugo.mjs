// Assembles the doc-site pack: the built app, and the artifact it drives.
//
// Both land under the Hugo site's static tree, where they are served beside each other. The
// shortcode writes a script and a link tag for the two fixed names, and the app finds the artifact
// relative to its own module URL - so nothing here has to know the site's base path, and the same
// pack works under a project sub-path or at a domain root.
//
// Run after `npm run wasm` and `npm run build`; `npm run dist` does all three.

import { cp, mkdir, rm, stat, readdir } from 'node:fs/promises';
import { join } from 'node:path';

const app = 'dist';
const artifact = 'public/genspec-wasi.wasm';
const target = '../hugo/themes/codescan-static/playground';

async function sizeOf(path) {
  const info = await stat(path);

  return `${(info.size / 1024 / 1024).toFixed(1)} MB`;
}

// Refuse rather than ship half a pack. A missing artifact is the likely mistake - `npm run build`
// alone leaves the app pointing at a scanner that is not there, and the failure surfaces in a
// browser rather than here.
for (const path of [app, artifact]) {
  try {
    await stat(path);
  } catch {
    console.error(`missing ${path}: run \`npm run dist\`, which builds both`);
    process.exit(1);
  }
}

// The generated parts are replaced wholesale, and only those. Emptying the directory instead takes
// the committed README.txt with it, which is the file that explains why the rest is not committed -
// and its absence is invisible until somebody goes looking for the explanation.
//
// Wholesale rather than merged: a stale hashed chunk from an earlier build is dead weight nothing
// will ever ask for, and it is not obvious by looking.
await rm(join(target, 'assets'), { recursive: true, force: true });
await rm(join(target, 'genspec-wasi.wasm'), { force: true });
await mkdir(target, { recursive: true });

await cp(join(app, 'assets'), join(target, 'assets'), { recursive: true });
await cp(artifact, join(target, 'genspec-wasi.wasm'));

const packed = await readdir(join(target, 'assets'));
console.log(`playground pack -> ${target}`);
for (const name of packed.sort()) {
  console.log(`  assets/${name}  ${await sizeOf(join(target, 'assets', name))}`);
}
console.log(`  genspec-wasi.wasm  ${await sizeOf(join(target, 'genspec-wasi.wasm'))}`);
