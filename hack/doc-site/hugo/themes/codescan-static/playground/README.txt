The playground pack (assets/ and genspec-wasi.wasm) is NOT committed.

Like the Relearn theme and Mermaid, it is built per-environment:
  - CI: a step in .github/workflows/update-doc.yml
  - local: cd ../../../genspec-wasi && npm run dist

What it is: hack/doc-site/genspec-wasi, a Svelte front-end around codescan
compiled to wasip1/wasm. The `playground` shortcode mounts it into a page. The
scanner runs in the reader's browser and their source is never uploaded.

Why it is not committed: the artifact alone is 19.5 MB, it is reproducible from
the tree beside it, and it is tied to the toolchain that built it - the standard
library's export data is compiled into it, and export data is valid only for the
Go release that produced it.

The two entry files keep fixed names (assets/playground.js and
assets/playground.css) so the shortcode can reference them without reading a
build manifest. Everything else is content-hashed and fetched by the entry.
