<script lang="ts">
  import type { ThemeStore } from '../lib/theme.svelte';

  let { theme }: { theme: ThemeStore } = $props();

  const glyph = $derived(theme.choice === 'auto' ? '◐' : theme.choice === 'light' ? '☀' : '☾');
  const title = $derived(
    theme.choice === 'auto'
      ? `Theme: follows your system (${theme.resolved})`
      : `Theme: ${theme.choice}`,
  );
</script>

<button class="cs-btn-quiet" onclick={() => theme.cycle()} title={title} aria-label={title}>
  <span aria-hidden="true">{glyph}</span>
</button>

<style>
  button {
    /* The three glyphs have different widths; a fixed box stops the toolbar twitching as it cycles. */
    width: 1.85rem;
    padding: 0;
    justify-content: center;
    font-size: var(--cs-fs-lg);
  }
</style>
