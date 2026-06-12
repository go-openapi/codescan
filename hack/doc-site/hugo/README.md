# Hugo documentation site

This directory holds the Hugo configuration that builds
<https://go-openapi.github.io/codescan/>.

## Layout

```text
hugo/
├── hugo.yaml                  # Static Hugo config
├── codescan.yaml.template     # Build-time config template (version info)
├── codescan.yaml              # Generated from the template (git-ignored output)
├── gendoc.go                  # Local development helper (`go run gendoc.go`)
├── themes/
│   ├── hugo-relearn/          # Relearn theme (downloaded by CI / dev script)
│   ├── codescan-assets/       # Custom logo / SCSS
│   └── codescan-static/       # Static branding (favicon, …)
└── layouts/
    ├── shortcodes/            # Custom Hugo shortcodes (e.g. `code`)
    └── partials/              # Custom partial templates
```

> **NOTE**: the branding images under `themes/codescan-assets/` and
> `themes/codescan-static/` are placeholders copied from the runtime doc-site.
> Replace `logo.png`, `colorized.png`, `images/favicon.png` and `github.png`
> with codescan-specific artwork.

## Content

Markdown content is mounted from `../../../docs/doc-site/` via the
`module.mounts` block in `hugo.yaml`. Editing those files (or adding new ones)
is enough — no codegen or generator is involved.

Runnable Go examples live under `../../../docs/examples/` (a separate Go
module) and are surfaced into pages with the `code` shortcode.

## The `code` shortcode

Includes a slice of an example source file with syntax highlighting and a
"Full source" link back to GitHub:

```text
{{< code file="basic/main.go" lang="go" region="runScan" >}}
```

- `file`   — path relative to `docs/examples`
- `lang`   — Chroma lexer name (`go`, `yaml`, …)
- `region` — named region delimited by `// snippet:NAME` / `// endsnippet:NAME`
- `lines`  — `N-M` line range (mutually exclusive with `region`)

Keeping example code as compilable `.go` files (rather than fenced blocks in
markdown) means the examples are vetted and tested by CI like any other code.

## Local preview

```sh
go run gendoc.go
```

The script:

1. Extracts version info from git tags and the root `go.mod`
2. Renders `codescan.yaml` from `codescan.yaml.template`
3. Starts `hugo server` on <http://localhost:1313/codescan/> with live reload

Requires `hugo` (extended, ≥ v0.150) and `git` on `PATH`. The Relearn theme is
not committed — download it once into `themes/hugo-relearn` (see the CI
workflow for the exact release used).

## Configuration

Two-layer config, mirroring the pattern used by other go-openapi doc sites:

1. **`hugo.yaml`** — static configuration (theme, mounts, menu, params)
2. **`codescan.yaml`** — dynamic configuration (Go version, latest release tag,
   build timestamp), generated from `codescan.yaml.template`

Both files are passed together via `--config hugo.yaml,codescan.yaml`. The
dynamic values land under `params.codescan.*` and are referenced from the
markdown content.

## Deployment

GitHub Actions workflow `.github/workflows/update-doc.yml`:

- Builds on every push to `master` and on tags `v*` that touch `docs/**`,
  `hack/doc-site/**`, or the workflow itself
- Publishes the rendered site to GitHub Pages
  (<https://go-openapi.github.io/codescan/>)
