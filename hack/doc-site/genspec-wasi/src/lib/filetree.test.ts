import { describe, expect, it } from 'vitest';
import { build, flatten, pathsToOpen } from './filetree';

describe('build', () => {
  it('nests files under their directories', () => {
    const tree = build(['go.mod', 'models/pet.go', 'api/handlers.go']);

    // Directories first, then files, each alphabetical - a file manager's order.
    expect(tree.map((n) => n.name)).toEqual(['api', 'models', 'go.mod']);
    expect(tree[0].children.map((n) => n.path)).toEqual(['api/handlers.go']);
  });

  // Five levels holding nothing but each other is five rows of scrolling to reach a package name.
  it('collapses a chain of single-child directories into one row', () => {
    const tree = build(['vendor/github.com/go-openapi/swag/mangling/name.go']);

    expect(tree).toHaveLength(1);
    expect(tree[0].name).toBe('vendor/github.com/go-openapi/swag/mangling');
    expect(tree[0].path).toBe('vendor/github.com/go-openapi/swag/mangling/');
    expect(tree[0].children.map((n) => n.name)).toEqual(['name.go']);
  });

  it('stops collapsing where the tree actually branches', () => {
    const tree = build(['a/b/c/one.go', 'a/b/d/two.go']);

    expect(tree[0].name).toBe('a/b');
    expect(tree[0].children.map((n) => n.name)).toEqual(['c', 'd']);
  });

  it('stops collapsing at a directory that holds a file of its own', () => {
    const tree = build(['a/own.go', 'a/b/deep.go']);

    expect(tree[0].name).toBe('a');
    expect(tree[0].children.map((n) => n.path)).toEqual(['a/b/', 'a/own.go']);
  });

  it('builds nothing from nothing', () => {
    expect(build([])).toEqual([]);
  });
});

describe('flatten', () => {
  const tree = build(['a/one.go', 'a/two.go', 'b/three.go']);

  it('shows only what is open', () => {
    expect(flatten(tree, () => false).map((r) => r.node.name)).toEqual(['a', 'b']);
  });

  it('walks in display order, carrying depth', () => {
    const rows = flatten(tree, (p) => p === 'a/');

    expect(rows.map((r) => `${r.depth}:${r.node.name}`))
      .toEqual(['0:a', '1:one.go', '1:two.go', '0:b']);
  });
});

describe('pathsToOpen', () => {
  it('names every row that has to open, not every directory in the path', () => {
    const tree = build(['a/b/c/deep.go', 'a/other.go']);

    // `a` holds a file of its own so it does not collapse, while b and c do - two rows to open for
    // a path four levels deep.
    expect(pathsToOpen(tree, 'a/b/c/deep.go')).toEqual(['a/', 'a/b/c/']);
  });

  it('names nothing for a file that is not there', () => {
    expect(pathsToOpen(build(['a/x.go']), 'nope.go')).toEqual([]);
  });
});
