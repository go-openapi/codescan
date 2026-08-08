import { describe, expect, it } from 'vitest';
import { AnchorIndex, pointerAt, spanFor } from './track';
import { render } from './render';
import type { Anchor } from './types';

// The anchors codescan actually emits for the bundled sample, taken from a real run of the artifact
// rather than invented - including the validation-keyword anchors, which are finer than the plan
// assumed and are what makes clicking `minimum: 1` land on its own line.
const anchors: Anchor[] = [
  { pointer: '/definitions/pet', file: 'models/pet.go', line: 6, col: 6 },
  { pointer: '/definitions/pet/properties/id', file: 'models/pet.go', line: 11, col: 2 },
  { pointer: '/definitions/pet/properties/id/minimum', file: 'models/pet.go', line: 10, col: 5 },
  { pointer: '/definitions/pet/properties/name', file: 'models/pet.go', line: 17, col: 2 },
  { pointer: '/definitions/pet/properties/name/maxLength', file: 'models/pet.go', line: 16, col: 5 },
  { pointer: '/definitions/pet/properties/kind', file: 'models/pet.go', line: 22, col: 2 },
  { pointer: '/paths/~1pets/get', file: 'api/handlers.go', line: 5, col: 1 },
  { pointer: '/responses/petList', file: 'api/handlers.go', line: 15, col: 6 },
];

const index = new AnchorIndex(anchors);

describe('AnchorIndex.forLine', () => {
  it('matches a line that is itself an anchor', () => {
    expect(index.forLine('models/pet.go', 11)?.pointer).toBe('/definitions/pet/properties/id');
    expect(index.forLine('api/handlers.go', 15)?.pointer).toBe('/responses/petList');
  });

  // Anchors reach individual validation keywords, so the line carrying `// max length: 50` resolves
  // to maxLength rather than to the field. That is the point of them being that fine.
  it('resolves an annotation line to the keyword it wrote', () => {
    expect(index.forLine('models/pet.go', 16)?.pointer)
      .toBe('/definitions/pet/properties/name/maxLength');
  });

  it('takes the nearer anchor', () => {
    expect(index.forLine('models/pet.go', 12)?.pointer).toBe('/definitions/pet/properties/id');
  });

  // Go documentation sits above what it documents, so a cursor equidistant between two anchors is
  // far more often in the comment belonging to the one below.
  it('resolves a tie downwards, towards the thing being documented', () => {
    // A tie needs an even gap. Line 8 is two below the declaration at 6 and two above the `minimum`
    // annotation at 10, and resolves downwards - a cursor there is in the comment block being
    // written, not trailing the type it follows.
    expect(index.forLine('models/pet.go', 8)?.pointer)
      .toBe('/definitions/pet/properties/id/minimum');
  });

  // Two anchors on one line means one is a descendant of the other, and the line cannot separate
  // them; the more general node is the safer answer.
  it('prefers the more general node when two anchors share a line', () => {
    const shared = new AnchorIndex([
      { pointer: '/definitions/x/properties/f/minimum', file: 'a.go', line: 9 },
      { pointer: '/definitions/x/properties/f', file: 'a.go', line: 9 },
    ]);

    expect(shared.forLine('a.go', 9)?.pointer).toBe('/definitions/x/properties/f');
  });

  it('still answers past the last anchor rather than going blank', () => {
    expect(index.forLine('models/pet.go', 400)?.pointer).toBe('/definitions/pet/properties/kind');
  });

  it('knows nothing about a file it never saw', () => {
    expect(index.forLine('vendor/x/y.go', 3)).toBe(null);
  });

  it('keeps files apart', () => {
    // Line 6 exists in both; the answer must come from the file asked about.
    expect(index.forLine('api/handlers.go', 6)?.file).toBe('api/handlers.go');
  });
});

describe('AnchorIndex.anchorFor', () => {
  it('answers directly for an anchored pointer', () => {
    expect(index.anchorFor('/definitions/pet')?.line).toBe(6);
  });

  // Only code-detail nodes carry provenance. Returning nothing for the rest would leave most of the
  // spec pane inert, so a pointer resolves to the nearest ancestor that is anchored.
  it('climbs to the nearest anchored ancestor', () => {
    expect(index.anchorFor('/definitions/pet/properties/id/type')?.pointer)
      .toBe('/definitions/pet/properties/id');
    expect(index.anchorFor('/definitions/pet/required/0')?.pointer).toBe('/definitions/pet');
  });

  it('gives up at the root rather than looping', () => {
    expect(index.anchorFor('/info/title')).toBe(null);
    expect(index.anchorFor('')).toBe(null);
  });
});

describe('pointerAt', () => {
  const doc = {
    definitions: {
      pet: {
        type: 'object',
        properties: { id: { type: 'integer', minimum: 1 } },
      },
    },
  };

  for (const format of ['json', 'yaml'] as const) {
    it(`finds the innermost node on a line (${format})`, () => {
      const { text, spans } = render(doc, format);
      const lines = text.split('\n');

      const minimumLine = lines.findIndex((l) => l.includes('minimum')) + 1;
      expect(pointerAt(spans, minimumLine)).toBe('/definitions/pet/properties/id/minimum');

      // The line holding the definition's own key belongs to the definition, not to a child.
      const petLine = lines.findIndex((l) => /(^|\s)"?pet"?\s*:/.test(l)) + 1;
      expect(pointerAt(spans, petLine)).toBe('/definitions/pet');
    });
  }

  it('says nothing for a line outside the document', () => {
    const { spans } = render(doc, 'json');
    expect(pointerAt(spans, 9999)).toBe(null);
  });
});

describe('spanFor', () => {
  it('falls back to an ancestor for a pointer that owns no line', () => {
    const { spans } = render({ definitions: { pet: { type: 'object' } } }, 'json');

    // Nothing renders this node, so highlighting has to settle for what does.
    const span = spanFor(spans, '/definitions/pet/properties/id/minimum');
    expect(span).toEqual(spans.get('/definitions/pet'));
  });

  it('returns null when even the root is unknown', () => {
    expect(spanFor(new Map(), '/anything')).toBe(null);
  });
});
