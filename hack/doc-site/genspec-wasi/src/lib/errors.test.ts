import { describe, expect, it } from 'vitest';
import { explain } from './errors';

describe('explain', () => {
  // The page and the artifact are shipped separately, so this failure is reachable by doing nothing
  // wrong at all: edit the app, reload, and the cached 20 MB asset is a version behind.
  it('translates an artifact that predates the flag it was asked for', () => {
    const said = explain('flag provided but not defined: -format\nusage: genspec-wasi [flags]…');

    expect(said).toContain('older than this page');
    expect(said).toContain('npm run wasm');
    expect(said).not.toContain('usage:');
  });

  it('leaves every other failure exactly as the scanner put it', () => {
    for (const raw of [
      'genspec-wasi: no packages matched ./...',
      'flag provided but not defined: -nonsense',
      '',
    ]) {
      expect(explain(raw)).toBe(raw);
    }
  });
});
