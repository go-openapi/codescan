// Joining the two panes.
//
// The scanner reports an anchor per spec node that came from a code detail: an RFC 6901 pointer and
// a source position. The renderer reports which lines each pointer occupies. Between them every
// question the tracking modes ask can be answered by position rather than by guessing at names,
// which is the property the terminal UI has and the reason to copy it rather than invent something.
//
// Two of the three questions are exact. The third is not, and says so.

import type { Anchor } from './types';
import { parentOf, type Span } from './render';

export class AnchorIndex {
  readonly byPointer = new Map<string, Anchor>();
  readonly byFile = new Map<string, Anchor[]>();

  constructor(anchors: Anchor[]) {
    for (const anchor of anchors) {
      // A pointer is anchored once. If the scanner reported it twice, the first wins - later ones
      // are the same node reached again, not a different node.
      if (!this.byPointer.has(anchor.pointer)) {
        this.byPointer.set(anchor.pointer, anchor);
      }

      if (!anchor.file || !anchor.line) {
        continue;
      }
      const list = this.byFile.get(anchor.file) ?? [];
      list.push(anchor);
      this.byFile.set(anchor.file, list);
    }

    for (const list of this.byFile.values()) {
      list.sort((a, b) => (a.line ?? 0) - (b.line ?? 0) || a.pointer.localeCompare(b.pointer));
    }
  }

  get size(): number {
    return this.byPointer.size;
  }

  files(): string[] {
    return [...this.byFile.keys()];
  }

  // forLine answers "what did this line produce", and is the one inexact join here.
  //
  // An anchor is a point, not a range: the scanner records where a declaration or a field STARTS,
  // and says nothing about where it ends. So a cursor resting inside a struct body, or on the third
  // line of a doc comment, matches nothing exactly and has to be attributed.
  //
  // Nearest by line, then two tie-breaks, both stated rather than left to sort order:
  //
  //   - Prefer the anchor BELOW the cursor. Go documentation sits above what it documents, so a
  //     cursor equidistant between two is far more often inside the comment belonging to the one
  //     below than trailing the one above. A dropped-description diagnostic lands a line above its
  //     field for exactly this reason.
  //   - Then prefer the SHORTER pointer. Two anchors on one line means one is a descendant of the
  //     other - a validation keyword and the field carrying it - and where the line cannot separate
  //     them, the more general node is the safer answer.
  forLine(file: string, line: number): Anchor | null {
    const list = this.byFile.get(file);
    if (!list?.length) {
      return null;
    }

    let best: Anchor | null = null;
    for (const anchor of list) {
      if (best === null || closer(anchor, best, line)) {
        best = anchor;
      }
    }

    return best;
  }

  // anchorFor walks up from a pointer until it finds one the scanner anchored.
  //
  // This is the documented contract rather than a fallback: only code-detail nodes carry provenance,
  // so a cursor on a `minimum` or on the word `object` resolves to the field or the declaration that
  // owns it. Returning nothing for those would make most of the spec pane inert.
  anchorFor(pointer: string): Anchor | null {
    let at: string | null = pointer;
    while (at !== null) {
      const found = this.byPointer.get(at);
      if (found) {
        return found;
      }
      at = parentOf(at);
    }

    return null;
  }
}

// closer implements forLine's ordering: distance, then below-the-cursor, then the more general node.
function closer(candidate: Anchor, incumbent: Anchor, line: number): boolean {
  const a = Math.abs((candidate.line ?? 0) - line);
  const b = Math.abs((incumbent.line ?? 0) - line);
  if (a !== b) {
    return a < b;
  }

  const candidateBelow = (candidate.line ?? 0) >= line;
  const incumbentBelow = (incumbent.line ?? 0) >= line;
  if (candidateBelow !== incumbentBelow) {
    return candidateBelow;
  }

  return candidate.pointer.length < incumbent.pointer.length;
}

// pointerAt answers "what node is written on this line" - exact, since spans are ranges.
//
// The innermost node wins: on the line holding `"minimum": 1` every ancestor's span contains that
// line too, and the one the reader means is the tightest.
export function pointerAt(spans: Map<string, Span>, line: number): string | null {
  let best: string | null = null;
  let bestWidth = Infinity;

  for (const [pointer, span] of spans) {
    if (line < span.from || line > span.to) {
      continue;
    }
    const width = span.to - span.from;
    // A tie is a scalar and its key sharing one line - the deeper pointer is the more specific.
    if (width < bestWidth || (width === bestWidth && best !== null && pointer.length > best.length)) {
      best = pointer;
      bestWidth = width;
    }
  }

  return best;
}

// spanFor finds where to highlight for a pointer, falling back to the nearest ancestor that is
// written down.
//
// The fallback matters when the spec is shown as YAML: a key whose value is empty renders inline, so
// a pointer can be real and yet own no line of its own.
export function spanFor(spans: Map<string, Span>, pointer: string): Span | null {
  let at: string | null = pointer;
  while (at !== null) {
    const span = spans.get(at);
    if (span) {
      return span;
    }
    at = parentOf(at);
  }

  return null;
}
