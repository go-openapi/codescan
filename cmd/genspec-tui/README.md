<!--
SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
SPDX-License-Identifier: Apache-2.0
-->

# genspec-tui

An interactive terminal front-end for [codescan][codescan]:

you may browse a Go source tree on the left, watch the Swagger spec it produces on the right,
and see the scanner's diagnostics underneath — all regenerated on every save.

In either panel you may activate the "track mode": spec items point back to the source that created them,
source line point to the spec item it generate and a diagnostic points to the source line that emitted it.

> Its reason to exist is the loop: change an annotation, hit save, see the spec change.
> Beyond that it links the two sides together, so you can ask "which Go code produced this node?"
> and "what did this field turn into?" and get an answer by position rather than by guessing at names.

Intended audience: codescan/go-swagger maintainers and contributors - but any experienced spec author could benefit from it.

## Install and run

`genspec-tui` is a **separate Go module** inside the codescan repo, so
bubbletea and its dependency tree never reach the lean library.

```sh
go install github.com/go-openapi/codescan/cmd/genspec-tui@latest

# scan the module in the current directory
genspec-tui

# or point it somewhere, and narrow the scope
genspec-tui -workdir ../my-api -packages ./internal/models/...,./internal/api/...
```

When working from a checkout, the repo's `go.work` wires the module to the local library:

```sh
go run ./cmd/genspec-tui -workdir ./fixtures -packages ./goparsing/petstore/...
```

| Flag | Default | Meaning |
|------|---------|---------|
| `-workdir` | `.` | module directory the scan runs in (codescan `WorkDir`) |
| `-packages` | `./...` | comma-separated package patterns, relative to `-workdir` |
| `-scan-models` | `true` | also emit definitions for `swagger:model` types |
| `-build-tags` | — | comma-separated go build tags to apply while loading |
| `-include` / `-exclude` | — | comma-separated patterns selecting which packages are scanned |
| `-include-tags` / `-exclude-tags` | — | comma-separated swagger tags selecting which operations are emitted |
| `-name-from-tags` | `json` | ordered struct tags a field's name derives from, e.g. `form,json` for gin. Pass `-name-from-tags=` (empty) to use the Go field name instead |
| `-name-concat-budget` | `0.65` | readability cutoff when deconflicting colliding definition names |

A second group decides what gets built and how it is loaded — the go environment
the scan runs under, plus codescan's own loader:

| Flag | Default | Meaning |
|------|---------|---------|
| `-goos` / `-goarch` | this machine's | the platform the scanned code is built for, so `//go:build` lines and `_linux.go` suffixes resolve the way that platform resolves them |
| `-goflags` | — | default go command flags, as `GOFLAGS` (e.g. `-tags=integration`); `-build-tags` wins |
| `-gowork` | search upwards | workspace selection, as `GOWORK`: `off` to disable, or the path to a `go.work` |
| `-goexperiment` | — | toolchain experiments, as `GOEXPERIMENT` (e.g. `jsonv2`) |
| `-toolchain-free-loader` | `false` | load packages with codescan's own loader instead of the go command (experimental) |
| `-stub-stdlib` | `false` | synthesize the standard library instead of reading GOROOT (needs `-toolchain-free-loader`) |

Every boolean scanner option can be toggled live with `o`; the spec re-renders
on close, which makes the popup the fastest way to see what a flag such as
`EmitRefSiblings` actually changes. The rows are grouped (discovery & scope ·
`$ref` & composition · naming · docs & comments · types & extensions · package
loading), and a knob that only bites in combination says so — `PruneUnusedModels`
shows `(needs ScanModels)` until that one is on, and `EmitXGoType` shows
`(moot: SkipExtensions)` while extensions are suppressed.

The value-typed options are flags rather than popup rows, since a checkbox list
cannot express them — see the tables above. The three booleans that appear in
both places (`-scan-models` and the two loader flags) let a session start one way
and change without a restart. The one option with no route in at all is
`InputSpec` (overlay mode).

## Layout

```
┌───────────────────────┬──────────────────────────────────────┐
│ source tree           │ spec · JSON                          │
│  or the file viewer   │  the generated document              │
├───────────────────────┴──────────────────────────────────────┤
│ diagnostics                                                  │
├──────────────────────────────────────────────────────────────┤
│ status / help                                                │
└──────────────────────────────────────────────────────────────┘
```

The left pane shows either the **source tree** or, once you open a file, the
**file viewer**. The viewer is read-only and navigable by default; `i` turns it
into an editor and `Esc` steps back out. Saving writes to disk, the watcher
notices, and the spec re-renders.

Clicking a pane focuses it, and the mouse wheel scrolls whichever pane is under
the pointer — `Tab` is never required.

Both dividers move: `ctrl+←`/`ctrl+→` for the one between the left pane and the
spec, `ctrl+↑`/`ctrl+↓` for the one above the diagnostics. Each key travels in
its own arrow's direction, and they work from every mode including the editor —
wanting more room for what you are typing in is exactly when you reach for a
resize.

The dividers are held as **proportions**, not cell counts, so resizing the
terminal keeps the layout you chose rather than letting one pane absorb the
difference. They stop short of either edge: a pane driven to nothing could not
be dragged back, because the keys that would restore it are advertised in a
status line it no longer has room for.

The binding surface is context-dependent — `f` follows from three different
panes, `Enter` opens a file in the tree but follows a `$ref` in the spec — so
the header carries a standing `h: help` banner, and `h` (or `?`) opens the full
keymap grouped by pane. The table below mirrors that overlay.

A rescan keeps you where you were: the cursor is restored to the same **node**,
not the same line number, so a definition appearing above what you are reading
does not slide you somewhere else. If that node is gone — you deleted the type —
the cursor falls back to its nearest surviving ancestor.

## Keys

### Anywhere

| Key | Action |
|-----|--------|
| `h` / `?` | the key-bindings overlay (also advertised in the header) |
| `Tab` / `shift+Tab` | cycle focus forward / backward |
| `ctrl+←` / `ctrl+→` | move the divider between the left pane and the spec |
| `ctrl+↑` / `ctrl+↓` | move the divider above the diagnostics strip |
| click | focus the pane under the pointer — or, on a `swagger:` directive, show what it means |
| wheel | scroll the pane under the pointer |
| `c` | copy the focused pane's raw content to the clipboard |
| `r` | rescan now |
| `v` / `V` | validate the generated spec / switch the diagnostics pane between scan and validation |
| `F5` | reload the open file from disk (asks before discarding unsaved edits) |
| `o` | scanner options popup (`space` toggles, `Esc`/`o` applies and rescans) |
| `ctrl+q` / `ctrl+c` | quit |

### Spec pane

| Key | Action |
|-----|--------|
| `↑` `↓` / `j` `k` | move the cursor |
| `PgUp` / `PgDn` | move it a page (the view never leaves the cursor behind) |
| `Home` / `End` | first / last line |
| `ctrl+j` / `ctrl+y` | render as JSON / YAML — keeps you on the same **node**, not the same line |
| `/` | search; `n` / `N` step through matches |
| `f` | toggle follow mode (spec drives, the source pane mirrors) |
| `F3` / `shift+F3` | next / previous **reference** to the node under the cursor |
| `Enter` | follow the `$ref` under the cursor to its definition |
| `Esc` | clear the search and the reference cycle |

The spec pane has a line cursor, and everything above acts on **the node under
it**. Searching parks the cursor on the match, so `/` then `F3` or `Enter`
composes.

### Source tree

| Key | Action |
|-----|--------|
| `↑` `↓` / `j` `k` | move the selection |
| `PgUp` / `PgDn`, `Home` / `End` | move a page at a time, or jump to the ends |
| `←` / `→` | collapse / expand a directory |
| `Enter` | open a file (or expand/collapse a directory) |
| `g` | locate the selected file's first node in the spec |

### File viewer (read-only)

| Key | Action |
|-----|--------|
| `↑` `↓` / `j` `k` | move the navigation line |
| `PgUp` / `PgDn`, `Home` / `End` | move a page at a time, or jump to the ends |
| `f` | toggle follow mode (source drives, the spec mirrors) |
| `K` | what the `swagger:` annotation on this line means |
| `i` / `Enter` | start editing |
| `Esc` | back to the tree |

The viewer shadows only these keys; every other binding (`/`, `o`, `r`, `g`,
`ctrl+j` / `ctrl+y`, `Tab`, `c`) still works while a file is open.

### File editor

| Key | Action |
|-----|--------|
| `ctrl+f` | jump from the cursor's line to the spec node it produced |
| `ctrl+s` | save (triggers a rescan) |
| `F5` | reload from disk, discarding the edits (asks first) |
| `Esc` | back to the read-only viewer |

`ctrl+f` rather than `f` because the editor owns plain `f` for typing.

## Annotation reference (`K`)

With the viewer's navigation line on a comment carrying a `swagger:` directive,
`K` shows what that annotation does: its syntax, a one-line summary, and what
may be written in its body.

`K` rather than a letter of its own because it is vim's "look up what is under
the cursor", and what LSP clients bind hover to. It reads the **buffer**, so it
works on an annotation you are still typing — which is when it is wanted.

Clicking the directive does the same. The pointer has to be on the `swagger:…`
token itself, not merely on its line: clicking a pane to focus it is the most
ordinary thing you do here, and it must not throw a popup up because the pointer
came to rest inside a comment. The click asks "what is that" and leaves the
navigation line where it was.

The entries cover the twenty `swagger:` annotations. Individual body keywords
(`required`, `minLength`, `enum`, …) are not documented one by one — there are
too many for a popup to be the right place — so each annotation's entry says
instead which family of keywords its body accepts.

## Reloading

`F5` re-reads the open file from disk. Disk is the source of truth here, so
this is how you pick up a change made outside the TUI — a `git checkout`, a
`gofmt`, another editor — and equally how you throw away edits you no longer
want.

It asks first, and only when there is something to lose: with a clean buffer it
just reloads. Answer `y` to discard, `n` or `Esc` to keep editing. `Enter`
declines too — the destructive answer is the one you have to type on purpose.

Reload keeps the line you were on and always lands in the read-only viewer,
even when it interrupted an edit. The line is restored **by number**, which is
the honest approximation: unlike a rescan, where the spec cursor is restored by
node, nothing anchors a line of Go source to anything stable across an edit
made behind the TUI's back.

### Diagnostics pane — scan tab

| Key | Action |
|-----|--------|
| `↑` `↓` / `j` `k` | select a diagnostic |
| `PgUp` / `PgDn`, `Home` / `End` | select a page at a time, or jump to the ends |
| `Enter` | go to this diagnostic's source line and focus it |
| `f` | toggle follow mode (the selection drives, the source **and** spec panes mirror) |

### Diagnostics pane — validation tab

| Key | Action |
|-----|--------|
| `↑` `↓` / `j` `k` | select a finding |
| `PgUp` / `PgDn`, `Home` / `End` | select a page at a time, or jump to the ends |
| `Enter` | go to this finding's node in the spec |
| `f` | toggle follow mode (the selection drives, the spec pane mirrors) |

## Validating the generated spec (`v`)

`v` runs the document through [go-openapi/validate][validate] and lists what it
finds in a **validation** tab of the diagnostics pane; `V` switches between that
and the scan's own findings.

The two tabs answer different questions. The scan tab says whether your
**annotations** were understood. The validation tab says whether the
**document** they produced is legal Swagger 2.0 — a scan can be perfectly clean
and still emit something a consumer rejects.

They also track different things, which is why they are tabs rather than one
list. A scan diagnostic knows a source position, so it drives the source pane
(and the spec). A validation finding knows only a JSON pointer, so `Enter` and
`f` there drive the **spec** pane and nothing else.

The tab exists only once you have pressed `v`, and a rescan retires it: the
findings judged the document that scan has just replaced, and a list of
complaints about a spec that no longer exists invites navigating to nodes that
may have moved or gone. Press `v` again.

Each finding carries the JSON pointer the validator recorded as it walked, taken
from the result rather than read back out of the message. Navigation is therefore
exact for an ordinary path and for an indexed one alike:
`/paths/~1pets/get/parameters/0/type` lands on that parameter rather than on the
list.

A finding about something the document does **not** have — no `info` block, no
`paths` — is reported against the whole document, which RFC 6901 spells as the
empty pointer. That is a location, not the absence of one, so `Enter` goes to the
top of the spec. Such a row is labelled `(the whole document)`.

One case lands on an enclosing node. A finding inside a response or parameter
written as a `$ref` is addressed through the site that refers to it, so the
document as authored has nothing below that site and you land on the `$ref`
itself. The same finding is usually reported a second time against the shared
definition, and that one resolves exactly.

Resolution walks up to the nearest node actually rendered, so an imprecise
landing is always an *ancestor* of what was reported — never a sibling, and
never somewhere untrue.

## Cross-reference navigation

Two indexes, rebuilt on every render, meet at a JSON pointer:

- the **spec index** maps each rendered line to the pointer of the node on it;
- the **source index** maps pointers to Go source positions, from codescan's
  `OnProvenance` callback.

### Follow mode (`f`)

`f` turns on a persistent link between panes. The pane you pressed it in is the
**driver** and keeps focus; the others **mirror** it on every cursor move,
centring and highlighting the linked line. The roles are styled differently so
it is always clear which pane leads. A `SPEC ▸ SOURCE` badge names the
direction and the resolved target.

Follow works in three directions: spec → source, source → spec, and
diagnostic → source **and** spec. `Esc`, a second `f`, changing focus, or
starting to edit all leave it.

The diagnostics pane drives two followers where the others drive one, because
it is not itself either end of the link — it is a third place naming a source
position, and what a finding *says* is usually about what that position
produced. The two halves resolve independently: a diagnostic on a line that
produced no spec node is the ordinary case (a parse error means nothing was
built from it), so the source half still resolves and the badge reports both.

### References (`F3`, `Enter`)

`F3` steps through the places the node under the cursor is referenced, wrapping;
`shift+F3` goes back. `Enter` follows a `$ref` to its definition.

A cycle stays anchored to one definition while you keep pressing `F3`. Scroll
away and the next `F3` re-anchors on wherever you now are.

### Syntax highlighting

Both panes are coloured, by the same renderer and the same palette.

The **spec pane** is coloured by key, string, number, keyword and punctuation.
The classification is free: the lexer that builds the line↔pointer index already
identifies every token, so highlighting is a third product of the same walk
rather than a second parse.

The **source viewer** is coloured by `go/scanner` — the standard library's own
tokenizer, so no highlighting library is involved on either side.

Comments there get three classes rather than one, because in a spec generator a
comment is not uniformly commentary:

| Looks like | Reads as | Why |
|------------|----------|-----|
| `// swagger:model order` | a spec key | the annotation declares the thing; it is the input that produced the pane next to it |
| `// required: true` | a keyword | grammar the parser acts on — same class Go's own `type`/`func` get |
| `// the id of the order` | dimmed prose | freeform description |

Only the keyword itself is lifted out, so `// required: true` reads as dim `//`,
coloured `required`, dim `: true` — the way `"required": true` reads on the spec
side. Recognition uses the parser's own keyword table, so aliases (`min` →
`minimum`, `min length` → `minLength`) and letter case come for free, and what
lights up is what the parser will act on.

### Diagnostics at the site

The scanner's own findings are drawn on the token they name, underlined in the
severity's colour — red for an error, amber for a warning, blue for a hint. The
diagnostics pane below tells you *what* and *where*; this tells you *which
token*, without leaving the line you are reading.

Marks come from the last scan and are re-derived on every rescan, so they never
outlive the finding that produced them. Where codescan reports a position is
where the mark goes: a keyword-level diagnostic lands on the keyword, while
`swagger:type: "array" is deprecated` lands on the **declaration**, because that
is where the builder reports it. The mark says "there is a finding about this",
not "this is deprecated".

### Keyword scope

Keyword highlighting is scoped to files that declare at least one annotation.
`name`, `in` and `example` are ordinary English words, and lighting them up in a
file the scanner never reads would claim something untrue. Within such a file
the scope is the whole file, not the comment block: a field's constraints live
in the field's doc comment while the `swagger:model` that gives them meaning
sits on the enclosing type.

Precedence on a line is **cursor, then search match, then syntax**. The first two
answer questions you asked, so they take the whole line instead of competing
with colour for it.

### The gutter

Both panes mark which lines actually lead somewhere, so you can see what is
navigable without probing for it:

| Marker | In the spec pane | In the source viewer |
|--------|------------------|----------------------|
| `•` | this node has a source position of its **own**, so following it lands exactly there | this line produced a spec node |
| `→` | a followable `$ref` — `Enter` goes to its definition | — |

Only *exact* anchors are marked. Nearly every line resolves to **something**
through its nearest anchored ancestor, so marking those would dot the whole
document and tell you nothing. External `$ref`s are not marked either, because
`Enter` cannot follow them.

The gutter column only appears when there is something to mark.

## Limitations

These are known and deliberate; the TUI says so rather than guessing.

- **Not every node has source.** codescan anchors *code-detail* nodes — type
  declarations, fields, values, route and meta blocks — and finer nodes resolve
  to their nearest anchored ancestor. A node with no anchored ancestor at all
  was not produced from code (an `InputSpec` overlay node, for instance); the
  follower holds position and says so instead of jumping somewhere plausible.
- **Positions are as of the last scan.** With unsaved edits in the buffer, every
  anchor below the edit has shifted, so follow shows a `STALE` badge. Saving
  triggers a rescan and clears it.
- **`$ref` resolution is a site index, not a resolver.** References are found by
  scanning the rendered document. Local `#/…` refs are followable; a ref into
  another file or a URL is reported as external rather than chased. Ref-to-ref
  chains and `$ref` nested in `allOf` are not unwound.
- **Keyword highlighting cannot know which declarations are scanned.** It knows
  the file is annotated and the word is in the grammar's table; it does not know
  whether codescan visits that particular type. A keyword-shaped line in an
  unrelated comment of an annotated file still lights up. Knowing better needs
  the AST, and the AST needs a file that parses — which the buffer you are
  editing may not.
- **The editor normalises whitespace.** `bubbles/textarea` rewrites tabs as four
  spaces when a file is loaded into it, and treats a lone CR as a line break, so
  files are converted to LF on the way in. Neither has an exported knob. The
  viewer, the highlighter and the cross-ref line numbers all agree with each
  other because they all read the same normalised text — but `Ctrl-S` writes the
  buffer, so **saving re-indents a tab-indented file with spaces and rewrites
  CRLF endings as LF**. Edit and save here only when you are content with that;
  the VIM/VS-Code integration is the real answer.
- **Only the read-only source viewer is highlighted.** `bubbles/textarea` owns
  its own rendering and emits the buffer verbatim, so edit mode shows plain
  text; `Esc` returns to the coloured viewer, re-tokenizing what you typed.
- **`shift+F3` is terminal-dependent.** bubbletea v1's key type carries no Shift
  modifier, and the xterm family reports shift+F3 as F15. Terminals that send
  something else have no previous-reference key; `F3` still wraps around.
- **The split keys are terminal-dependent too.** `ctrl`+arrow reaches us as
  `CSI 1;5<A-D>`, which most modern emulators send and some (notably inside a
  default `tmux` or `screen`) do not. A terminal that sends a bare arrow instead
  simply has no resize keys — nothing misfires, because a bare arrow is already
  a navigation key.
- **Split sizes last for the session only.** They are model state, so they
  survive rescans and terminal resizes but not a restart; persisting them needs
  a config file, which the TUI does not have.

## Development

```sh
go test ./...                         # from cmd/genspec-tui
go test work ./...                    # from the repo root: every module at once
golangci-lint run --new-from-rev master
```

The TUI has no CI workflow of its own: `go.work` lists it, so the shared
monorepo workflow lints and tests it alongside the library, across the
`{ubuntu, macos, windows} × {stable, oldstable}` matrix.

The package layout under `internal/ux`:

| Package | Contents |
|---------|----------|
| `ux` | the root bubbletea `Model`: key dispatch, layout, scan wiring, cross-ref navigation |
| `ux/panels` | the four panes — `Tree`, `FileView`, `Spec`, `Diagnostics` |
| `ux/index` | `SpecIndex` (line ↔ pointer), `RefIndex` (`$ref` sites), `SourceIndex` (pointer ↔ source position) |
| `ux/key` | `tea.KeyMsg` → a small named-binding enum |
| `ux/theme` | the shared lipgloss styles |
| `ux/gadgets` | clipboard support |

The scanner writes nothing to stdout or stderr: diagnostics arrive through
codescan's `OnDiagnostic` callback, and `main` discards the standard logger, so
nothing paints over the alt-screen.

[codescan]: https://github.com/go-openapi/codescan
[validate]: https://github.com/go-openapi/validate
