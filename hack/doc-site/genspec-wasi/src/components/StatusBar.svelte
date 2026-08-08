<script lang="ts">
  import { usePlayground } from '../lib/store.svelte';
  import { humanBytes, humanDuration } from '../lib/format';

  const playground = usePlayground();
  const counts = $derived(playground.counts);

  const bytes = $derived(playground.files.reduce((n, f) => n + f.text.length, 0));

  // What the reader is told: one number, the wall clock of the last scan. The breakdown is a
  // tooltip, because three numbers on a status line is a report rather than a status.
  const total = $derived(
    playground.compileMs + playground.prepareMs + playground.instantiateMs + playground.runMs,
  );

  const breakdown = $derived(
    [
      playground.compileMs > 0
        ? `compile ${humanDuration(playground.compileMs)} (once per session)`
        : 'compile 0 — the module was already compiled',
      `prepare ${humanDuration(playground.prepareMs)} (encoding what changed)`,
      `instantiate ${humanDuration(playground.instantiateMs)}`,
      `scan ${humanDuration(playground.runMs)} — the scanner reads the whole tree every time`,
    ].join('\n'),
  );

  // The linear-memory figure says a scan got heavier. These say why, and the host cannot see them:
  // a live heap near Sys means the scan is HOLDING memory, while TotalAlloc far above it with many
  // collections means it is churning through it. Different problems, different fixes.
  const memoryDetail = $derived.by(() => {
    const r = playground.runtime;
    const peak = `peak linear memory ${humanBytes(playground.memoryBytes)} — what this tab paid`;
    if (!r) {
      return peak;
    }

    return [
      peak,
      '',
      'inside the guest, when the scan finished:',
      `  obtained from the host  ${humanBytes(r.sys)}`,
      `  still live              ${humanBytes(r.heapAlloc)}`,
      `  allocated in total      ${humanBytes(r.totalAlloc)}`,
      `  collections             ${r.collections}`,
    ].join('\n');
  });

  function plural(n: number, one: string, many = one + 's'): string {
    return `${n} ${n === 1 ? one : many}`;
  }

  function size(n: number): string {
    return n < 1024
      ? `${n} B`
      : n < 1024 * 1024
        ? `${(n / 1024).toFixed(0)} KB`
        : `${(n / 1024 / 1024).toFixed(1)} MB`;
  }
</script>

<footer class="status">
  <span>{plural(playground.files.length, 'file')}</span>
  <span class="cs-faint">{size(bytes)}</span>

  <span class="sep" aria-hidden="true"></span>

  <span>{plural(counts.paths, 'path')} · {plural(counts.definitions, 'definition')}</span>

  <span class="sep" aria-hidden="true"></span>

  {#if playground.running}
    <!-- The status line says which of the three it is, since the first visit spends most of its
         time on the first and none of it scanning. -->
    <span class="live" aria-live="polite">
      {playground.progress?.phase === 'fetching'
        ? 'fetching the scanner…'
        : playground.progress?.phase === 'compiling'
          ? 'compiling…'
          : 'scanning…'}
    </span>
  {:else if playground.error}
    <span class="cs-error">scan failed</span>
  {:else if playground.scanned}
    <span aria-live="polite" title={breakdown}>ready ({humanDuration(total)})</span>
    {#if playground.memoryBytes > 0}
      <span class="cs-faint" title={memoryDetail}>· {humanBytes(playground.memoryBytes)}</span>
    {/if}
    {#if playground.compileMs > 0}
      <span class="cs-faint">· first scan, compiled the artifact</span>
    {/if}
  {:else}
    <span class="cs-faint">not scanned yet</span>
  {/if}

  <span class="grow"></span>

  <span class="cs-faint privacy">Runs in your browser · nothing is uploaded</span>
</footer>

<style>
  .status {
    display: flex;
    align-items: center;
    gap: var(--cs-s2);
    flex: none;
    height: 1.75rem;
    padding: 0 var(--cs-s3);
    font-size: var(--cs-fs-xs);
    color: var(--cs-fg-muted);
    background: var(--cs-bg-sunken);
    border-top: 1px solid var(--cs-line);
    font-variant-numeric: tabular-nums;
  }

  .sep {
    width: 1px;
    height: 0.9rem;
    background: var(--cs-line);
  }

  .grow {
    flex: 1;
  }

  /* The promise the whole thing rests on, so it stays on screen rather than living in a README -
     but quietly, since it is reassurance and not news. */
  .privacy {
    white-space: nowrap;
  }

  @media (width < 52rem) {
    .privacy {
      display: none;
    }
  }
</style>
