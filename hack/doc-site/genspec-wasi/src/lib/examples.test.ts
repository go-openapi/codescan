import { describe, expect, it } from 'vitest';
import { exampleById, examples, firstExample, sampleFiles } from './examples';

describe('examples', () => {
  it('offers each one exactly once, and can find it again', () => {
    const ids = examples.map((e) => e.id);
    expect(new Set(ids).size).toBe(ids.length);

    for (const example of examples) {
      expect(exampleById(example.id)).toBe(example);
    }
  });

  // The scanner takes a module, not a pile of files: without go.mod there is no module root, no
  // import paths, and nothing to scan.
  it('gives every example a go.mod and some Go', () => {
    for (const example of examples) {
      const files = example.files();
      expect(files.some((f) => f.path === 'go.mod'), `${example.id} has no go.mod`).toBe(true);
      expect(files.some((f) => f.path.endsWith('.go')), `${example.id} has no Go`).toBe(true);
    }
  });

  it('annotates every example, or it is teaching nothing', () => {
    for (const example of examples) {
      const go = example.files().filter((f) => f.path.endsWith('.go')).map((f) => f.text).join('\n');
      expect(go, `${example.id} carries no annotation`).toMatch(/swagger:(model|route|operation|parameters|response|enum|allOf)/);
    }
  });

  // Fresh objects each call: the store edits these in place, and a shared array would carry one
  // visitor's typing into the next example they open.
  it('builds fresh files on every call', () => {
    const first = examples[0].files();
    const second = examples[0].files();

    expect(first).not.toBe(second);
    expect(first[0]).not.toBe(second[0]);
    expect(first).toEqual(second);
  });

  it('opens on an example that exists', () => {
    expect(examples.map((e) => e.id)).toContain(firstExample);
    expect(sampleFiles().length).toBeGreaterThan(1);
  });

  it('falls back rather than failing on an unknown id', () => {
    expect(exampleById('nope')).toBe(examples[0]);
  });
});
