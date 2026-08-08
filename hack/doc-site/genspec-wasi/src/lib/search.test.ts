import { describe, expect, it } from 'vitest';
import { findMatches, nearestMatch, stepIndex } from './search';

const text = [
  '{',
  '  "definitions": {',
  '    "pet": {',
  '      "type": "object",',
  '      "properties": {',
  '        "id": { "type": "integer" }',
  '      }',
  '    }',
  '  }',
  '}',
].join('\n');

describe('findMatches', () => {
  it('finds every line holding the query, in order', () => {
    expect(findMatches(text, 'type')).toEqual([4, 6]);
  });

  it('ignores case, as the terminal UI does', () => {
    expect(findMatches(text, 'DEFINITIONS')).toEqual([2]);
    expect(findMatches(text, 'Pet')).toEqual([3]);
  });

  it('counts a line once however many times it matches', () => {
    // A match is a line, which is what lets n/N and the cross-references speak about the same thing.
    expect(findMatches('a a a\nb', 'a')).toEqual([1]);
  });

  it('finds nothing for an empty or blank query', () => {
    expect(findMatches(text, '')).toEqual([]);
    expect(findMatches(text, '   ')).toEqual([]);
  });

  it('finds nothing rather than everything when there is no match', () => {
    expect(findMatches(text, 'zzz')).toEqual([]);
  });
});

describe('stepIndex', () => {
  it('wraps in both directions', () => {
    expect(stepIndex(2, 3, 1)).toBe(0);
    expect(stepIndex(0, 3, -1)).toBe(2);
  });

  it('advances normally in between', () => {
    expect(stepIndex(0, 3, 1)).toBe(1);
    expect(stepIndex(2, 3, -1)).toBe(1);
  });

  it('stays put with nothing to step through', () => {
    expect(stepIndex(0, 0, 1)).toBe(0);
  });
});

describe('nearestMatch', () => {
  it('starts at the first match at or below where you are reading', () => {
    expect(nearestMatch([2, 10, 40], 1)).toBe(0);
    expect(nearestMatch([2, 10, 40], 10)).toBe(1);
    expect(nearestMatch([2, 10, 40], 11)).toBe(2);
  });

  it('wraps to the top when everything is above you', () => {
    expect(nearestMatch([2, 10], 99)).toBe(0);
  });

  it('has nowhere to start with no matches', () => {
    expect(nearestMatch([], 5)).toBe(0);
  });
});
