// The CodeMirror pieces both panes are built from.
//
// Kept out of the component so the extensions can be read as a list of decisions rather than found
// among lifecycle code, and so the theme is written once for an editor that is sometimes a Go buffer
// and sometimes a rendered document.

import {
  EditorView, Decoration, type DecorationSet, gutter, GutterMarker, lineNumbers,
  highlightActiveLine, highlightActiveLineGutter, drawSelection, rectangularSelection,
} from '@codemirror/view';
import { EditorState, StateEffect, StateField, RangeSet } from '@codemirror/state';
import { HighlightStyle, syntaxHighlighting } from '@codemirror/language';
import { tags } from '@lezer/highlight';
import type { Severity } from './types';

// ---- highlighting the joined node ---------------------------------------

export type Highlight = { from: number; to: number } | null;

export const setHighlight = StateEffect.define<Highlight>();

const highlightLine = Decoration.line({ class: 'cs-cm-tracked' });

// The tracked range is a StateField rather than a prop the component re-applies, because it has to
// survive every other transaction - a scroll, a selection, a rescan - without being recomputed.
const highlightField = StateField.define<DecorationSet>({
  create: () => Decoration.none,
  update(current, tr) {
    for (const effect of tr.effects) {
      if (!effect.is(setHighlight)) {
        continue;
      }
      const range = effect.value;
      if (!range) {
        return Decoration.none;
      }

      const marks = [];
      const last = Math.min(range.to, tr.state.doc.lines);
      for (let line = Math.max(1, range.from); line <= last; line++) {
        marks.push(highlightLine.range(tr.state.doc.line(line).from));
      }

      return RangeSet.of(marks);
    }

    return current.map(tr.changes);
  },
  provide: (f) => EditorView.decorations.from(f),
});

// ---- search matches ------------------------------------------------------

export type Matches = { lines: number[]; current: number | null };

export const setSearch = StateEffect.define<Matches>();

const matchLine = Decoration.line({ class: 'cs-cm-match' });
const currentMatchLine = Decoration.line({ class: 'cs-cm-match cs-cm-match-current' });

const searchField = StateField.define<DecorationSet>({
  create: () => Decoration.none,
  update(current, tr) {
    for (const effect of tr.effects) {
      if (!effect.is(setSearch)) {
        continue;
      }
      const { lines, current: at } = effect.value;
      const marks = lines
        .filter((line) => line >= 1 && line <= tr.state.doc.lines)
        .map((line) => (line === at ? currentMatchLine : matchLine).range(tr.state.doc.line(line).from));

      return RangeSet.of(marks, true);
    }

    return current.map(tr.changes);
  },
  provide: (f) => EditorView.decorations.from(f),
});

// ---- the diagnostics gutter ---------------------------------------------

export type GutterMark = { line: number; severity: Severity; title: string };

export const setMarks = StateEffect.define<GutterMark[]>();

class SeverityMarker extends GutterMarker {
  constructor(private readonly severity: Severity, private readonly title: string) {
    super();
  }

  override eq(other: SeverityMarker): boolean {
    return other.severity === this.severity && other.title === this.title;
  }

  override toDOM(): Node {
    const dot = document.createElement('span');
    dot.className = `cs-cm-mark cs-cm-mark-${this.severity}`;
    dot.title = this.title;
    // Decoration only: the drawer below carries the same diagnostics as text, so nothing here is
    // the sole way to learn about one.
    dot.setAttribute('aria-hidden', 'true');

    return dot;
  }
}

const marksField = StateField.define<GutterMark[]>({
  create: () => [],
  update(current, tr) {
    for (const effect of tr.effects) {
      if (effect.is(setMarks)) {
        return effect.value;
      }
    }

    return current;
  },
});

const diagnosticsGutter = gutter({
  class: 'cs-cm-gutter',
  lineMarker(view, line) {
    const at = view.state.doc.lineAt(line.from).number;
    // Worst first: a line carrying both an error and a hint is an error line.
    const order: Severity[] = ['error', 'warning', 'hint'];
    const here = view.state.field(marksField).filter((m) => m.line === at);
    if (!here.length) {
      return null;
    }
    here.sort((a, b) => order.indexOf(a.severity) - order.indexOf(b.severity));

    return new SeverityMarker(here[0].severity, here.map((m) => m.title).join('\n'));
  },
  initialSpacer: () => new SeverityMarker('hint', ''),
});

// ---- theme ---------------------------------------------------------------

// Drawn entirely from the design tokens, so the editor follows the theme with everything else rather
// than shipping a light and a dark variant of its own.
const theme = EditorView.theme({
  '&': {
    height: '100%',
    minHeight: '0',
    fontSize: 'var(--cs-fs-md)',
    backgroundColor: 'var(--cs-bg)',
    color: 'var(--cs-fg)',
  },
  '.cm-scroller': {
    fontFamily: 'var(--cs-font-mono)',
    lineHeight: 'var(--cs-lh-code)',
    overscrollBehavior: 'contain',
  },
  '.cm-content': { padding: 'var(--cs-s2) 0' },
  '.cm-gutters': {
    backgroundColor: 'var(--cs-bg)',
    color: 'var(--cs-fg-faint)',
    border: 'none',
  },
  '.cm-lineNumbers .cm-gutterElement': { padding: '0 var(--cs-s2) 0 var(--cs-s3)' },
  // Strong enough to find without hunting, weak enough not to compete with the tracked node - which
  // is the other thing on screen wearing a background.
  '.cm-activeLine': { backgroundColor: 'var(--cs-bg-inset)' },
  '.cm-activeLineGutter': {
    backgroundColor: 'var(--cs-bg-inset)',
    color: 'var(--cs-fg)',
    fontWeight: '600',
  },
  '&.cm-focused': { outline: 'none' },
  '&.cm-focused .cm-selectionBackground, .cm-selectionBackground, ::selection': {
    backgroundColor: 'var(--cs-accent-soft)',
  },
  '.cm-cursor': { borderLeftColor: 'var(--cs-fg)' },

  // The tracked node. A left bar carries most of the signal: the two can land on the same line -
  // you point at something and the other pane points back - and a wash alone would just look like a
  // slightly different active line.
  '.cs-cm-tracked': {
    backgroundColor: 'var(--cs-accent-soft)',
    boxShadow: 'inset 3px 0 0 0 var(--cs-accent)',
  },
  '.cs-cm-tracked.cm-activeLine': {
    backgroundColor: 'var(--cs-accent-soft)',
  },

  // Every match, and the one you are on. The current match reuses the accent so it reads as "here",
  // consistent with the tracked node and the active row in the diagnostics list.
  '.cs-cm-match': { backgroundColor: 'var(--cs-match)' },
  '.cs-cm-match-current': {
    backgroundColor: 'var(--cs-match-current)',
    boxShadow: 'inset 3px 0 0 0 var(--cs-accent)',
  },

  '.cs-cm-gutter': { width: '0.9rem' },
  '.cs-cm-mark': {
    display: 'block',
    width: '6px',
    height: '6px',
    margin: '0.45em auto 0',
    borderRadius: '50%',
  },
  '.cs-cm-mark-error': { backgroundColor: 'var(--cs-error)' },
  '.cs-cm-mark-warning': { backgroundColor: 'var(--cs-warning)' },
  '.cs-cm-mark-hint': { backgroundColor: 'var(--cs-hint)' },
});

const syntax = HighlightStyle.define([
  { tag: [tags.keyword, tags.moduleKeyword, tags.controlKeyword], color: 'var(--cs-syn-keyword)' },
  { tag: [tags.typeName, tags.className, tags.namespace], color: 'var(--cs-syn-type)' },
  { tag: [tags.string, tags.special(tags.string)], color: 'var(--cs-syn-string)' },
  { tag: [tags.number, tags.bool, tags.null], color: 'var(--cs-syn-number)' },
  { tag: [tags.comment, tags.lineComment, tags.blockComment], color: 'var(--cs-syn-comment)', fontStyle: 'italic' },
  { tag: [tags.propertyName, tags.definition(tags.propertyName)], color: 'var(--cs-syn-property)' },
  { tag: [tags.function(tags.variableName), tags.definition(tags.function(tags.variableName))], color: 'var(--cs-syn-func)' },
  { tag: [tags.operator, tags.punctuation, tags.separator], color: 'var(--cs-fg-muted)' },
]);

export const baseExtensions = [
  lineNumbers(),
  // Where you are, in both panes. Without it the caret is the only clue, and the caret is one pixel
  // wide in a document you are reading rather than editing.
  highlightActiveLine(),
  highlightActiveLineGutter(),
  // The native selection is invisible in a view that is not editable, so draw it.
  drawSelection(),
  rectangularSelection(),
  diagnosticsGutter,
  marksField,
  searchField,
  highlightField,
  syntaxHighlighting(syntax),
  theme,
  EditorView.lineWrapping,
];

// scrollTo puts a line in view without yanking it to the top, which loses the context around it.
export function scrollTo(view: EditorView, line: number) {
  const clamped = Math.max(1, Math.min(line, view.state.doc.lines));
  const pos = view.state.doc.line(clamped).from;
  view.dispatch({ effects: EditorView.scrollIntoView(pos, { y: 'center' }) });
}

export function lineOfSelection(state: EditorState): number {
  return state.doc.lineAt(state.selection.main.head).number;
}
