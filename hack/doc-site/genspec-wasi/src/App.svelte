<script lang="ts">
  import { untrack } from 'svelte';
  import { providePlayground } from './lib/store.svelte';
  import { createTheme } from './lib/theme.svelte';
  import Toolbar from './components/Toolbar.svelte';
  import OptionsPanel from './components/OptionsPanel.svelte';
  import SourcePane from './components/SourcePane.svelte';
  import SpecPane from './components/SpecPane.svelte';
  import SplitPane from './components/SplitPane.svelte';
  import DiagnosticsDrawer from './components/DiagnosticsDrawer.svelte';
  import StatusBar from './components/StatusBar.svelte';

  // The shell owns both stores and hands them down through context, so mounting a second playground
  // on the same page gives it its own tree rather than sharing this one's.
  // Where the artifact lives is the page's business, not ours: the standalone page and a doc-site
  // page serve it from different places, and the app should not have to know which it is in.
  let { wasmUrl }: { wasmUrl?: string } = $props();

  const playground = providePlayground();
  const theme = createTheme();

  // Read once, deliberately: the mount point states where the artifact is and does not change its
  // mind, so untrack says that rather than leaving the linter to guess.
  const supplied = untrack(() => wasmUrl);
  if (supplied) {
    playground.wasmUrl = supplied;
  }

  let showOptions = $state(false);

  // Scan on arrival. The first frame a visitor sees should be a specification beside the source that
  // produced it - an idle app with a Scan button is a puzzle, and the artifact has to be fetched
  // either way, so waiting for permission buys nothing.
  $effect(() => {
    if (!playground.scanned && !playground.running) {
      playground.run();
    }
  });

  // Measured on the shell rather than the viewport: embedded in a doc-site article the app may be
  // half the width of the window, and it is our own box that decides whether two panes fit.
  let width = $state(0);
  const stacked = $derived(width > 0 && width < 720);
</script>

<div
  class="cs-root shell"
  data-theme={theme.resolved}
  bind:clientWidth={width}
>
  <Toolbar {theme} onToggleOptions={() => (showOptions = !showOptions)} optionsOpen={showOptions} />

  {#if showOptions}
    <OptionsPanel />
  {/if}

  {#if playground.vendorAdvice}
    <p class="notice advice">{playground.vendorAdvice}</p>
  {/if}

  {#if playground.notice}
    <p class="notice">
      {playground.notice}
      <button class="cs-btn-quiet dismiss" onclick={() => (playground.notice = '')} aria-label="Dismiss">×</button>
    </p>
  {/if}

  <SplitPane startLabel="the source pane" {stacked}>
    {#snippet start()}<SourcePane />{/snippet}
    {#snippet end()}<SpecPane />{/snippet}
  </SplitPane>

  <DiagnosticsDrawer />
  <StatusBar />
</div>

<style>
  /* Says something true about what you just opened, and gets out of the way. Not an error strip:
     it never carries a failure, so it does not take the eye the way one should. */
  .notice {
    display: flex;
    align-items: center;
    gap: var(--cs-s2);
    flex: none;
    padding: var(--cs-s2) var(--cs-s3);
    font-size: var(--cs-fs-sm);
    color: var(--cs-fg-muted);
    background: var(--cs-bg-inset);
    border-bottom: 1px solid var(--cs-line);
  }

  /* Advice, not a failure: it says what to do next, and stays until the tree it is about is gone. */
  .advice {
    color: var(--cs-fg);
    background: var(--cs-warning-soft);
  }

  .dismiss {
    margin-left: auto;
    width: 1.5rem;
    padding: 0;
    justify-content: center;
    font-size: var(--cs-fs-lg);
    line-height: 1;
  }

  .shell {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
    overflow: hidden;
  }
</style>
