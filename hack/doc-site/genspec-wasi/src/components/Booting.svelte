<script lang="ts">
  import { usePlayground } from '../lib/store.svelte';
  import { humanBytes } from '../lib/format';

  // What the first visit looks like while the scanner arrives.
  //
  // The wait was tolerable when a reader had pressed Scan, because they had chosen it. It is not now
  // that the playground scans on arrival: they land, see nothing, and cannot tell a slow network from
  // a broken page. So the wait says what it is waiting for - and says it is a download rather than a
  // scan, because eight megabytes of scanner is not the scanner being slow.

  const playground = usePlayground();

  const p = $derived(playground.progress);

  const fraction = $derived.by(() => {
    if (p?.phase !== 'fetching' || !p.total || !p.received) {
      return null;
    }

    return Math.min(1, p.received / p.total);
  });

  const line = $derived.by(() => {
    switch (p?.phase) {
      case 'fetching':
        return p.received
          ? `Fetching the scanner — ${humanBytes(p.received)}${p.total ? ` of ${humanBytes(p.total)}` : ''}`
          : 'Fetching the scanner';
      case 'compiling':
        return 'Compiling the scanner';
      default:
        return 'Scanning';
    }
  });
</script>

<div class="boot" role="status" aria-live="polite">
  <p class="line">{line}…</p>

  <!-- Determinate where the response said how much to expect, and a moving band where it did not:
       a bar that pretends to know how far along it is, is worse than one that admits it does not. -->
  <div class="track" class:indeterminate={fraction === null}>
    <div class="fill" style={fraction === null ? undefined : `width: ${fraction * 100}%`}></div>
  </div>

  {#if p?.phase === 'fetching' || p?.phase === 'compiling'}
    <p class="why cs-faint">
      codescan itself, compiled to WebAssembly. It is fetched once and then cached by your browser —
      later scans start straight away.
    </p>
  {/if}
</div>

<style>
  .boot {
    display: flex;
    flex-direction: column;
    gap: var(--cs-s2);
    padding: var(--cs-s5) var(--cs-s4);
    max-width: 32rem;
  }

  .line {
    font-size: var(--cs-fs-sm);
    color: var(--cs-fg-muted);
    font-variant-numeric: tabular-nums;
  }

  .track {
    height: 3px;
    background: var(--cs-bg-inset);
    border-radius: 999px;
    overflow: hidden;
  }

  .fill {
    height: 100%;
    background: var(--cs-accent);
    transition: width 120ms linear;
  }

  .track.indeterminate .fill {
    width: 35%;
    animation: sweep 1.1s ease-in-out infinite;
  }

  @keyframes sweep {
    from { margin-left: -35%; }
    to { margin-left: 100%; }
  }

  .why {
    font-size: var(--cs-fs-xs);
    line-height: 1.5;
  }

  @media (prefers-reduced-motion: reduce) {
    .track.indeterminate .fill {
      width: 100%;
      animation: none;
      opacity: 0.4;
    }
  }
</style>
