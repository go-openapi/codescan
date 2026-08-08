import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createTheme } from './theme.svelte';

// The store reads matchMedia and localStorage at construction, so each test states the world it
// wants before building one.
function world({ dark = false, saved = null as string | null } = {}) {
  vi.stubGlobal('matchMedia', () => ({
    matches: dark,
    addEventListener: () => {},
  }));
  vi.stubGlobal('localStorage', {
    getItem: () => saved,
    setItem: () => {},
  });
}

describe('createTheme', () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
  });

  it('resolves auto against the system, never leaving "auto" on the element', () => {
    // data-theme carries the resolved value because the stylesheet has two themes and no third one.
    world({ dark: true });
    expect(createTheme().resolved).toBe('dark');

    world({ dark: false });
    expect(createTheme().resolved).toBe('light');
  });

  it('lets an explicit choice win over the system, in both directions', () => {
    world({ dark: true });
    const theme = createTheme();

    theme.set('light');
    expect(theme.resolved).toBe('light');

    theme.set('dark');
    expect(theme.resolved).toBe('dark');
  });

  it('restores a saved choice', () => {
    world({ dark: true, saved: 'light' });
    const theme = createTheme();

    expect(theme.choice).toBe('light');
    expect(theme.resolved).toBe('light');
  });

  it('ignores a saved value that is not a theme', () => {
    world({ saved: 'chartreuse' });

    expect(createTheme().choice).toBe('auto');
  });

  it('cycles auto → light → dark → auto', () => {
    world();
    const theme = createTheme();

    expect(theme.choice).toBe('auto');
    theme.cycle();
    expect(theme.choice).toBe('light');
    theme.cycle();
    expect(theme.choice).toBe('dark');
    theme.cycle();
    expect(theme.choice).toBe('auto');
  });

  it('still themes a page whose storage throws', () => {
    vi.stubGlobal('matchMedia', () => ({ matches: false, addEventListener: () => {} }));
    vi.stubGlobal('localStorage', {
      getItem: () => { throw new Error('denied'); },
      setItem: () => { throw new Error('denied'); },
    });

    const theme = createTheme();
    expect(theme.choice).toBe('auto');
    expect(() => theme.set('dark')).not.toThrow();
    expect(theme.resolved).toBe('dark');
  });
});
