<script lang="ts">
  import { usePlayground } from '../lib/store.svelte';
  import type { ThemeStore } from '../lib/theme.svelte';
  import Examples from './Examples.svelte';
  import FilePicker from './FilePicker.svelte';
  import Help from './Help.svelte';
  import ThemeToggle from './ThemeToggle.svelte';

  let {
    theme,
    onToggleOptions,
    optionsOpen,
  }: {
    theme: ThemeStore;
    onToggleOptions: () => void;
    optionsOpen: boolean;
  } = $props();

  const playground = usePlayground();
</script>

<header class="bar">
  <span class="brand">codescan</span>
  <span class="cs-faint tag">playground</span>

  <span class="grow"></span>

  <Examples />

  <FilePicker />

  <button
    onclick={() => playground.reset()}
    title="Go back to the bundled example"
  >Reset</button>

  <button
    aria-pressed={playground.tracking}
    class:on={playground.tracking}
    onclick={() => {
      playground.tracking = !playground.tracking;
      if (!playground.tracking) {
        playground.clearTracking();
      }
    }}
    title="Follow the cursor between the two panes: a source line to the spec node it produced, and back"
  >Track</button>

  <button
    aria-expanded={optionsOpen}
    aria-controls="scanner-options"
    onclick={onToggleOptions}
  >Options</button>

  <Help />

  <ThemeToggle {theme} />

  <button
    class="cs-btn-primary"
    onclick={() => playground.run()}
    disabled={playground.running}
  >{playground.running ? 'Scanning…' : 'Scan'}</button>
</header>

<style>
  .bar {
    display: flex;
    align-items: center;
    gap: var(--cs-s2);
    flex: none;
    height: var(--cs-bar-h);
    padding: 0 var(--cs-s3);
    background: var(--cs-bg-sunken);
    border-bottom: 1px solid var(--cs-line);
  }

  .brand {
    font-weight: 650;
    letter-spacing: -0.01em;
  }

  .tag {
    font-size: var(--cs-fs-sm);
  }

  .grow {
    flex: 1;
  }

  .bar button.on {
    color: var(--cs-accent);
    border-color: var(--cs-accent);
    background: var(--cs-accent-soft);
  }

  /* Narrow: the label is what goes, not the control. Reset is the one that can wait, since it is
     recoverable in a way that scanning and opening a module are not. */
  @media (width < 34rem) {
    .tag,
    .bar :global(button[title="Go back to the bundled example"]) {
      display: none;
    }
  }
</style>
