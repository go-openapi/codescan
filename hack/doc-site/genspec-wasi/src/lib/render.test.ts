import { describe, expect, it } from 'vitest';
import { load as parseYAML } from 'js-yaml';
import { parentOf, render, escapeToken, unescapeToken, type Format } from './render';

// A document with the shapes a Swagger spec actually contains: nested maps, an array of scalars, an
// array of maps, an empty container, a pointer segment needing escaping, and strings that YAML would
// misread if written bare.
const doc = {
  swagger: '2.0',
  info: { title: 'Pets', version: '1.0' },
  paths: {
    '/pets': {
      get: {
        tags: ['pets'],
        responses: { '200': { description: 'ok' } },
        parameters: [{ name: 'limit', in: 'query', required: false }],
      },
    },
  },
  definitions: {
    pet: {
      type: 'object',
      required: ['id'],
      properties: {
        id: { type: 'integer', minimum: 1 },
        name: { type: 'string', description: 'a name: with a colon' },
        kind: { type: 'string', enum: ['cat', 'dog'] },
      },
      'x-empty': {},
    },
  },
};

const formats: Format[] = ['json', 'yaml'];

function reparse(text: string, format: Format): unknown {
  return format === 'yaml' ? parseYAML(text) : JSON.parse(text);
}

describe('render', () => {
  for (const format of formats) {
    describe(format, () => {
      // The point of writing our own renderer is the span map, and it is worthless if the text it
      // describes is not the document.
      it('round-trips the document', () => {
        expect(reparse(render(doc, format).text, format)).toEqual(doc);
      });

      // The property everything downstream rests on: a pointer's span has to be the lines that node
      // is actually written on, or a cross-reference lands somewhere arbitrary.
      it('gives every node a span holding its own text', () => {
        const { text, spans } = render(doc, format);
        const lines = text.split('\n');

        const check = (value: unknown, pointer: string) => {
          if (pointer) {
            const span = spans.get(pointer);
            expect(span, `no span for ${pointer}`).toBeDefined();
            expect(span!.from).toBeGreaterThanOrEqual(1);
            expect(span!.to).toBeLessThanOrEqual(lines.length);
            expect(span!.to).toBeGreaterThanOrEqual(span!.from);
          }

          if (Array.isArray(value)) {
            value.forEach((v, i) => check(v, `${pointer}/${i}`));
          } else if (value && typeof value === 'object') {
            for (const [k, v] of Object.entries(value)) {
              check(v, `${pointer}/${escapeToken(k)}`);
            }
          }
        };

        check(doc, '');
      });

      it('nests a child inside its parent', () => {
        const { spans } = render(doc, format);
        const parent = spans.get('/definitions/pet')!;
        const child = spans.get('/definitions/pet/properties/id')!;

        expect(child.from).toBeGreaterThanOrEqual(parent.from);
        expect(child.to).toBeLessThanOrEqual(parent.to);
      });

      it('escapes a pointer segment that needs it', () => {
        const { spans } = render(doc, format);

        // "/pets" as a key becomes "~1pets" as a segment, which is how the scanner writes it too.
        expect(spans.has('/paths/~1pets/get')).toBe(true);
      });

      it('puts each definition on its own lines', () => {
        const { spans } = render(doc, format);
        const id = spans.get('/definitions/pet/properties/id')!;
        const name = spans.get('/definitions/pet/properties/name')!;

        expect(id.to).toBeLessThan(name.from);
      });
    });
  }

  it('quotes a YAML string that would otherwise be read as something else', () => {
    const tricky = {
      version: '1.0',
      yes: 'true',
      empty: '',
      colon: 'a name: with a colon',
      hash: 'trailing # hash',
      lead: '- dashed',
      pad: '  padded  ',
      multi: 'two\nlines',
    };

    expect(parseYAML(render(tricky, 'yaml').text)).toEqual(tricky);
  });

  it('writes an empty container inline in both formats', () => {
    for (const format of formats) {
      const { text } = render({ a: {}, b: [] }, format);
      expect(reparse(text, format)).toEqual({ a: {}, b: [] });
    }
  });

  it('renders a document that is not a mapping', () => {
    for (const format of formats) {
      expect(reparse(render('bare', format).text, format)).toBe('bare');
      expect(reparse(render(null, format).text, format)).toBe(null);
    }
  });
});

describe('pointer helpers', () => {
  it('escapes and unescapes in the order RFC 6901 requires', () => {
    // ~1 must not be produced by escaping / and then be read back as ~ by an eager unescape.
    for (const raw of ['plain', 'a/b', 'a~b', 'a~1b', '~/~']) {
      expect(unescapeToken(escapeToken(raw))).toBe(raw);
    }
  });

  it('walks up to the root and stops', () => {
    expect(parentOf('/definitions/pet/properties/id')).toBe('/definitions/pet/properties');
    expect(parentOf('/definitions')).toBe('');
    expect(parentOf('')).toBe(null);
  });
});
