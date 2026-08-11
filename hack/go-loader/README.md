# go-loader

## Objective

Keeping our pure go source loader toolchain faithful to the original.

The matcher decides what a `...` package pattern means. Reproducing those rules from their
prose description is a well-known way to get them subtly wrong, so they are used as written
rather than re-derived. Drift against upstream is checked by hack/go-loader.

`internal/packages` reimplements what `go list` does. Two things follow from that, and this tool is both of them:

- some of it is **copied** out of the Go tree, so it can drift when Go moves;
- all of it is **behaviour** the go command already has tests for.

Everything here needs a checkout of <https://github.com/golang/go>.
Nothing here runs in CI (yet), because CI has no such checkout — what CI sees is [`ledger.md`](ledger.md),
which is committed, so a run against a newer Go shows up as a diff.

```sh
GO=/path/to/golang/go

go run ./hack/go-loader -go $GO sync                     # has upstream moved?
go run ./hack/go-loader -go $GO sync -update             # take the new version
go run ./hack/go-loader -go $GO corpus                   # run their trees through both loaders
go run ./hack/go-loader -go $GO corpus -write-ledger     # …and record the result
go run ./hack/go-loader -go $GO corpus -only work_ -v    # one family, verbosely
```

## `sync` — the copied declarations

`internal/packages/list/pkgpattern.go` and its test are copied verbatim from
`cmd/internal/pkgpattern` (BSD-3-Clause, see `NOTICE` at the repository root). The
reasons are in the file header; the cost is that upstream can change underneath us without
anyone noticing.

`sync` compares each copied declaration byte for byte and reports `ok`, `DRIFTED`, `GONE`
(removed upstream) or `MISSING` (removed here). `-update` rewrites the local copy. Run it
when bumping the Go toolchain.

Add to the `copies` table in `sync.go` if anything else is ever taken from the Go tree.

## `corpus` — their test trees, our two loaders

`cmd/go/testdata/script/*.txt` is ~880 txtar scripts. Several hundred of them are package
trees built to be awkward: nested modules, workspaces, vendor directories, `...` in places
nobody would write by hand. That corpus is the expensive half of testing pattern resolution,
and it already exists.

What is **not** reused is the assertions. Running the scripts as written would mean
implementing their shell DSL, and there is no need — a real go command is available, so the
go/packages strategy *is* the oracle. The tool materialises each tree into a temporary
directory, loads `./...` through both strategies, and compares what each says the tree
contains: import path and file names per package, which is exactly what a spec is built
from.

The trees are read where they lie and never copied into this repository. They are
BSD-licensed test data, and vendoring several hundred of them to assert nothing about their
contents would be a poor trade.

### Reading the outcome

| status | meaning |
|---|---|
| `agree` | both strategies found the same packages, made of the same files |
| `differ` | a real disagreement — the reason to run this |
| `go-rejects` | the go command refuses the tree and we read it anyway |
| `skip` | the tree cannot stand alone here (see below) |
| `error` | the harness itself failed |

`go-rejects` is not a failure. This loader deliberately reads trees the go command will not:
one with no `go.mod` at all, one importing another module's `internal/` package, one whose
vendor directory the go command calls inconsistent. Documenting code that compiles for its
author matters more here than reproducing a refusal.

Skips are reported rather than hidden, because a harness that quietly drops most of its
corpus looks like a passing one. A tree is skipped when it needs the module proxy (`go mod
download`, `go get`, `rsc.io/...`), when it sets `GO111MODULE=off` (GOPATH mode, which this
loader does not implement), or when it has no `go.mod` or no Go source to speak of.

## When to run it, and what to do about the answer

**Cadence: every new Go minor.** That is when `cmd/go` moves, and it is the only
event that can change any of this underneath us. A patch release is not worth a
run; a toolchain bump in `go.mod` is not either, since neither changes `go list`.
Worth running once more before a release that touches the loader, on the grounds
that it is cheap and the alternative is finding out from a user.

Both subcommands exit non-zero when they find something, so a maintainer can wire
them into a release checklist without reading the output first.

### `sync` says DRIFTED

Upstream edited a declaration we copied. Read the upstream diff before taking it:
`-update` overwrites our copy, and the copy is deliberately verbatim, so whatever
changed is a change in *semantics we claim to reproduce*.

```sh
git -C $GO log -p --since='1 year' -- src/cmd/internal/pkgpattern/
```

Then `sync -update`, run the test suite, and run `corpus`. The table in
`pkgpattern_test.go` is copied too, so a rule change usually arrives as a new row
in it — and it will fail before anything else does.

### `sync` says GONE or MISSING

GONE means upstream deleted or renamed the declaration. Do not paper over it: find
what replaced it and decide whether to follow. MISSING means our copy lost it,
which is either an editing accident or someone reimplementing what was copied on
purpose.

### `corpus` shows a `differ`

A tree where the two strategies disagree about which packages exist or what files
they are made of. Read it as a bug in ours until proven otherwise — the go command
is the definition here, not a peer.

Work it as: read the script's txtar to see what the tree is built to exercise,
reproduce it as a focused test in `internal/packages/resolve_test.go`, fix, then
re-run `corpus -write-ledger` and check the ledger diff for anything you did not
mean to change.

If the disagreement turns out to be deliberate — the go command refusing a tree we
mean to read — it belongs in `go-rejects`, not `differ`. That means teaching
`runScript` to recognise it, not adding an exception to the fix.

### `corpus` shows more `skip` than last time

Usually harmless: new scripts arrive using the module proxy. Worth a glance
anyway, since a skip reason that starts matching trees it did not match before is
how a harness quietly stops testing anything.

### The ledger diff is the review artifact

`-write-ledger` is what makes a run reviewable. Commit it with whatever prompted
the run, so the diff shows what a new Go release changed: a script that moved
families, a skip reason that no longer applies, a tree that started disagreeing.

### What it caught

The first run found three defects and one blind spot:

- `./...` did not stop at a `go.mod` that fails to parse. The go command's boundary test is
  the *existence* of the file (`modload/search.go`, "Stop at module boundaries"); ours
  required a readable module path, so an empty `go.mod` let the walk through and the
  packages below it were labelled with the enclosing module's import path.
- one malformed directory aborted the entire load. `go list -e` reports the error on the
  package and carries on; we returned it as a load failure, so a single file mid-edit
  anywhere under `./...` produced no spec at all.
- the files `go/build` had already classified were thrown away with the error, where `go list -e` still reports them.
- our own wildcard matcher scored 34/37 against `pkgpattern`'s table, which is what led to copying it.
