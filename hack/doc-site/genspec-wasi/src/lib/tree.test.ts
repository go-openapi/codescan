import { describe, expect, it } from 'vitest';
import { limits, needsVendoring, reroot, wanted, weigh } from './tree';

const at = (...paths: string[]) => paths.map((path) => ({ path, text: '' }));
const paths = (files: { path: string }[]) => files.map((f) => f.path);

describe('reroot', () => {
  it('leaves a tree already rooted at go.mod alone', () => {
    expect(paths(reroot(at('go.mod', 'models/pet.go')))).toEqual(['go.mod', 'models/pet.go']);
  });

  it('strips the folder a directory pick prefixes', () => {
    expect(paths(reroot(at('myapi/go.mod', 'myapi/models/pet.go'))))
      .toEqual(['go.mod', 'models/pet.go']);
  });

  it('re-roots when the pick sat above the module, dropping what falls outside', () => {
    expect(paths(reroot(at('work/proj/myapi/go.mod', 'work/proj/myapi/m.go', 'work/other/x.go'))))
      .toEqual(['go.mod', 'm.go']);
  });

  it('roots at the outermost go.mod, so a submodule does not steal it', () => {
    expect(paths(reroot(at('myapi/go.mod', 'myapi/tools/go.mod', 'myapi/tools/t.go', 'myapi/m.go'))))
      .toEqual(['go.mod', 'tools/go.mod', 'tools/t.go', 'm.go']);
  });

  it('keeps vendor, which is how a third-party import resolves with no module cache', () => {
    expect(paths(reroot(at('myapi/go.mod', 'myapi/vendor/github.com/x/y/y.go'))))
      .toContain('vendor/github.com/x/y/y.go');
  });

  it('leaves a selection with no go.mod untouched, for the scan to complain about', () => {
    expect(paths(reroot(at('loose/a.go', 'loose/b.go')))).toEqual(['loose/a.go', 'loose/b.go']);
  });

  it('survives an empty selection', () => {
    expect(reroot([])).toEqual([]);
  });
});

describe('wanted', () => {
  it('keeps Go source and go.mod, and drops tests', () => {
    expect(wanted('models/pet.go')).toBe(true);
    expect(wanted('go.mod')).toBe(true);
    expect(wanted('proj/go.mod')).toBe(true);
    expect(wanted('models/pet_test.go')).toBe(false);
    expect(wanted('README.md')).toBe(false);
    expect(wanted('go.sum')).toBe(false);
  });

  it('keeps vendored source', () => {
    expect(wanted('proj/vendor/github.com/go-openapi/strfmt/time.go')).toBe(true);
  });

  // Without it the loader does not consider the vendor directory authoritative, and every dependency
  // it holds is synthesized instead of read.
  it('keeps vendor/modules.txt, at the root or below', () => {
    expect(wanted('vendor/modules.txt')).toBe(true);
    expect(wanted('proj/vendor/modules.txt')).toBe(true);
  });

  it('does not keep a modules.txt that is not the vendor directory\'s', () => {
    expect(wanted('modules.txt')).toBe(false);
    expect(wanted('docs/modules.txt')).toBe(false);
  });
});

describe('weigh', () => {
  const mb = (n: number) => n * 1024 * 1024;

  it('accepts a real vendored module', () => {
    // go-swagger's dockerctl, the shape a visitor most plausibly arrives with: 5 MB of generated
    // client over 20 MB of vendored dependencies. The old 8 MB ceiling refused it outright.
    const budget = weigh([
      { path: 'dockerctl/cli/x.go', size: mb(5) },
      { path: 'dockerctl/vendor/github.com/a/a.go', size: mb(20) },
    ]);

    expect(budget.refused).toBe(false);
    expect(budget.heavy).toBe(true);
    expect(budget.bytes).toBe(mb(25));
  });

  it('refuses past the ceiling', () => {
    expect(weigh([{ path: 'huge/a.go', size: limits.maxBytes + 1 }]).refused).toBe(true);
    expect(weigh([{ path: 'ok/a.go', size: limits.maxBytes }]).refused).toBe(false);
  });

  it('names the directory holding the weight, so a refusal can point somewhere', () => {
    // "Too big, pick something smaller" is not advice when what you picked is the module: the answer
    // is a directory inside it, and nothing on screen says which.
    const budget = weigh([
      { path: 'proj/cli/a.go', size: mb(1) },
      { path: 'proj/vendor/x/a.go', size: mb(9) },
      { path: 'proj/vendor/y/b.go', size: mb(9) },
    ]);

    expect(budget.heaviest?.path).toBe('proj/vendor');
    expect(budget.heaviest?.bytes).toBe(mb(18));
  });

  it('groups a flat selection by its own top level', () => {
    const budget = weigh([{ path: 'main.go', size: 10 }]);

    expect(budget.heaviest?.path).toBe('main.go');
    expect(budget.files).toBe(1);
  });

  it('says nothing about an empty selection', () => {
    const budget = weigh([]);

    expect(budget).toMatchObject({ files: 0, bytes: 0, refused: false, heavy: false, heaviest: null });
  });
});


describe('needsVendoring', () => {
  const mod = (text: string) => ({ path: 'go.mod', text });

  it('says nothing about a module with no dependencies', () => {
    expect(needsVendoring([mod('module x\n\ngo 1.25.0\n')])).toBe(false);
  });

  it('spots both shapes the go command writes', () => {
    expect(needsVendoring([mod('module x\n\nrequire github.com/a/b v1.0.0\n')])).toBe(true);
    expect(needsVendoring([mod('module x\n\nrequire (\n\tgithub.com/a/b v1.0.0\n)\n')])).toBe(true);
  });

  it('is quiet once the tree carries a vendor directory', () => {
    expect(needsVendoring([
      mod('module x\n\nrequire github.com/a/b v1.0.0\n'),
      { path: 'vendor/modules.txt', text: '# github.com/a/b v1.0.0\n' },
    ])).toBe(false);
  });

  it('is not fooled by the word in a comment', () => {
    expect(needsVendoring([mod('module x\n// require nothing of me\n')])).toBe(false);
  });

  it('says nothing without a go.mod, having nothing to read', () => {
    expect(needsVendoring([{ path: 'main.go', text: 'package main' }])).toBe(false);
  });
});
