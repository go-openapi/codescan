// Searching the rendered document.
//
// Line-based and case-insensitive, matching genspec-tui: a match is a line containing the query, not
// a character range. That is not a simplification - the whole spec pane is addressed by line, so a
// match being a line is what lets n/N, the tracked-node highlight and the cross-references all speak
// about the same thing.

/** 1-based line numbers containing the query, in order. Empty for an empty query. */
export function findMatches(text: string, query: string): number[] {
  const needle = query.trim().toLowerCase();
  if (!needle) {
    return [];
  }

  const hits: number[] = [];
  const lines = text.split('\n');
  for (let i = 0; i < lines.length; i++) {
    if (lines[i].toLowerCase().includes(needle)) {
      hits.push(i + 1);
    }
  }

  return hits;
}

// stepIndex moves through the matches, wrapping in both directions.
//
// Wrapping rather than stopping: the document is long, most matches are off-screen, and a search
// that silently refuses to advance reads as broken rather than as finished.
export function stepIndex(current: number, count: number, direction: 1 | -1): number {
  if (count <= 0) {
    return 0;
  }

  return (current + direction + count) % count;
}

// nearestMatch is where a fresh search starts: the first match at or after the line being read,
// wrapping to the first overall.
//
// Starting from the top regardless would throw away the reader's position in a document that is
// mostly below the fold.
export function nearestMatch(matches: number[], from: number): number {
  if (!matches.length) {
    return 0;
  }
  const at = matches.findIndex((line) => line >= from);

  return at === -1 ? 0 : at;
}
