<script lang="ts">
  import { usePlayground } from '../lib/store.svelte';
  import { limits, wanted, weigh, type Budget } from '../lib/tree';
  import { humanBytes } from '../lib/format';
  import type { SourceFile } from '../lib/types';

  const playground = usePlayground();

  let input: HTMLInputElement;
  let progress = $state<{ done: number; total: number } | null>(null);

  // Read in batches rather than one file at a time. A vendored module is well over a thousand files,
  // and awaiting each in turn serialises a thousand trips through the file API for no reason - the
  // browser is content to have several in flight.
  const batch = 64;

  // Directory picking is non-standard but is the only way to take a module in one gesture; the same
  // input still accepts a plain multi-file selection where it is unsupported.
  async function accept(event: Event) {
    const picked = Array.from((event.target as HTMLInputElement).files ?? []);
    input.value = '';

    // webkitRelativePath is set for a directory pick and empty for individual files. Keep it whole -
    // the store works out where the module root is.
    const keep = picked
      .map((file) => ({
        file,
        path: (file as File & { webkitRelativePath?: string }).webkitRelativePath || file.name,
      }))
      .filter((entry) => wanted(entry.path));

    // Weighed before a byte is read: the old check tallied as it went and refused halfway through,
    // having already spent the time to read everything up to that point.
    const budget = weigh(keep.map((e) => ({ path: e.path, size: e.file.size })));
    if (budget.refused) {
      playground.error = refusal(budget);

      return;
    }

    progress = { done: 0, total: keep.length };
    try {
      const files: SourceFile[] = [];
      for (let i = 0; i < keep.length; i += batch) {
        const slice = keep.slice(i, i + batch);
        const texts = await Promise.all(slice.map((e) => e.file.text()));
        slice.forEach((e, n) => files.push({ path: e.path, text: texts[n] }));
        progress = { done: files.length, total: keep.length };
      }

      playground.open(files);
      if (budget.heavy) {
        playground.notice = `${budget.files} files, ${humanBytes(budget.bytes)} — a tree this size `
          + 'takes a while to scan, and all of it is held in memory.';
      }
    } finally {
      progress = null;
    }
  }

  function refusal(budget: Budget): string {
    const where = budget.heaviest
      ? ` Most of it is ${budget.heaviest.path}, at ${humanBytes(budget.heaviest.bytes)}.`
      : '';

    return `That selection is ${humanBytes(budget.bytes)} of Go source across ${budget.files} files, `
      + `past the ${humanBytes(limits.maxBytes)} a browser can hold.${where} `
      + 'Pick the module you want to scan rather than everything above it.';
  }
</script>

<button onclick={() => input.click()} disabled={progress !== null}>
  {#if progress}
    Reading {progress.done}/{progress.total}…
  {:else}
    Open module…
  {/if}
</button>
<input
  bind:this={input}
  type="file"
  multiple
  webkitdirectory
  onchange={accept}
  hidden
/>
