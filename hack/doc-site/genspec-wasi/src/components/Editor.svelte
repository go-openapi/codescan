<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { EditorView, keymap } from '@codemirror/view';
  import { EditorState, Compartment, type Extension } from '@codemirror/state';
  import { defaultKeymap, history, historyKeymap } from '@codemirror/commands';
  import {
    baseExtensions, lineOfSelection, scrollTo, setHighlight, setMarks, setSearch,
    type GutterMark, type Highlight, type Matches,
  } from '../lib/editor';

  // One editor for both panes: the source is editable Go, the result a read-only rendered document.
  // Making them the same component is what lets tracking be symmetric - highlighting a range and
  // scrolling it into view is one implementation rather than two that drift.

  let {
    text,
    language,
    readonly = false,
    marks = [],
    matches = { lines: [], current: null },
    highlight = null,
    revealLine = null,
    cursorTo = null,
    onChange,
    onCursorLine,
    label,
  }: {
    text: string;
    language: Extension;
    readonly?: boolean;
    marks?: GutterMark[];
    matches?: Matches;
    highlight?: Highlight;
    /** Bumped by the caller to ask for a line to be brought into view. */
    revealLine?: { line: number; nonce: number } | null;
    /**
     * Bumped to put the CURSOR on a line, not merely scroll to it.
     *
     * The distinction is the whole reason search is useful here: with the cursor on the match, every
     * cursor-driven thing - the tracked node, the source pane following - keeps working, and a
     * search becomes a way of navigating rather than only of finding.
     */
    cursorTo?: { line: number; nonce: number } | null;
    onChange?: (text: string) => void;
    onCursorLine?: (line: number) => void;
    label: string;
  } = $props();

  let host: HTMLDivElement;
  let view: EditorView | null = null;
  let lastLine = 0;
  const languageOf = new Compartment();

  onMount(() => {
    view = new EditorView({
      parent: host,
      state: EditorState.create({
        doc: text,
        extensions: [
          keymap.of([...defaultKeymap, ...historyKeymap]),
          history(),
          languageOf.of(language),
          // readOnly blocks edits; editable stays true so the view keeps a caret. Turning editable
          // off as well is the obvious reading of "read-only" and is wrong here: it drops
          // contenteditable, and with it the cursor, the arrow keys, Home/End and PgUp/PgDn - so the
          // result pane becomes a wall of text you can only look at, which is half of tracking gone.
          EditorState.readOnly.of(readonly),
          EditorView.contentAttributes.of({ 'aria-label': label }),
          ...baseExtensions,
          EditorView.updateListener.of((update) => {
            if (update.docChanged && !readonly) {
              onChange?.(update.state.doc.toString());
            }
            // Only a cursor the READER moved, and only when it changes line.
            //
            // Reporting on docChanged as well looks more thorough and is a bug: replacing the
            // document after a rescan sets a selection, which would be read as the reader pointing
            // at line 1 and would drag the other pane there on every scan.
            if (update.selectionSet && !update.docChanged) {
              const line = lineOfSelection(update.state);
              if (line !== lastLine) {
                lastLine = line;
                onCursorLine?.(line);
              }
            }
          }),
        ],
      }),
    });
  });

  onDestroy(() => view?.destroy());

  // Writing the document back only when it actually differs is what stops an edit from being undone
  // by its own round trip through the store: the keystroke updates the store, the store updates
  // `text`, and re-setting it would move the cursor to the end.
  $effect(() => {
    const next = text;
    if (view && next !== view.state.doc.toString()) {
      view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: next } });
    }
  });

  $effect(() => {
    view?.dispatch({ effects: languageOf.reconfigure(language) });
  });

  $effect(() => {
    view?.dispatch({ effects: setMarks.of(marks) });
  });

  $effect(() => {
    view?.dispatch({ effects: setSearch.of(matches) });
  });

  $effect(() => {
    const ask = cursorTo;
    if (!view || !ask) {
      return;
    }
    const clamped = Math.max(1, Math.min(ask.line, view.state.doc.lines));
    const pos = view.state.doc.line(clamped).from;
    view.dispatch({ selection: { anchor: pos }, scrollIntoView: true });
  });

  $effect(() => {
    view?.dispatch({ effects: setHighlight.of(highlight) });
  });

  $effect(() => {
    const ask = revealLine;
    if (view && ask) {
      scrollTo(view, ask.line);
    }
  });

  export function focusLine(line: number) {
    if (!view) {
      return;
    }
    const clamped = Math.max(1, Math.min(line, view.state.doc.lines));
    const pos = view.state.doc.line(clamped).from;
    view.dispatch({ selection: { anchor: pos }, scrollIntoView: true });
    view.focus();
  }
</script>

<div class="editor" bind:this={host}></div>

<style>
  .editor {
    flex: 1;
    min-height: 0;
    min-width: 0;
    overflow: hidden;
  }

  .editor :global(.cm-editor) {
    height: 100%;
  }
</style>
