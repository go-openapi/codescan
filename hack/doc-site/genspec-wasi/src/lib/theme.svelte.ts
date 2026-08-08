// Theme choice, resolved to a concrete light or dark.
//
// "auto" is resolved here rather than in CSS so the stylesheet states each theme once. A
// prefers-color-scheme block would have to repeat the dark values - once for "the system says dark"
// and once for "you asked for dark" - and the two would drift.

export type ThemeChoice = 'auto' | 'light' | 'dark';
export type Theme = 'light' | 'dark';

const storageKey = 'codescan-playground-theme';

function stored(): ThemeChoice {
  try {
    const saved = localStorage.getItem(storageKey);
    if (saved === 'light' || saved === 'dark' || saved === 'auto') {
      return saved;
    }
  } catch {
    // A page with storage disabled still gets a theme; it just does not remember one.
  }

  return 'auto';
}

function systemPrefersDark(): boolean {
  return typeof matchMedia === 'function' && matchMedia('(prefers-color-scheme: dark)').matches;
}

export function createTheme() {
  let choice = $state<ThemeChoice>(stored());
  let systemDark = $state(systemPrefersDark());

  // Only meaningful while the choice is "auto", but subscribing unconditionally keeps systemDark
  // honest for the moment somebody switches back to it.
  if (typeof matchMedia === 'function') {
    matchMedia('(prefers-color-scheme: dark)')
      .addEventListener('change', (e) => { systemDark = e.matches; });
  }

  return {
    get choice() {
      return choice;
    },

    // resolved is what goes on data-theme: never "auto", because the stylesheet has no such theme.
    get resolved(): Theme {
      if (choice === 'auto') {
        return systemDark ? 'dark' : 'light';
      }

      return choice;
    },

    set(next: ThemeChoice) {
      choice = next;
      try {
        localStorage.setItem(storageKey, next);
      } catch {
        // See stored().
      }
    },

    // cycle is what the toolbar button does: the three states in the order somebody reaching for it
    // expects - away from whatever they are looking at, then back to following the system.
    cycle() {
      this.set(choice === 'auto' ? 'light' : choice === 'light' ? 'dark' : 'auto');
    },
  };
}

export type ThemeStore = ReturnType<typeof createTheme>;
