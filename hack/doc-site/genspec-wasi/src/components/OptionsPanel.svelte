<script lang="ts">
  import { usePlayground } from '../lib/store.svelte';

  const playground = usePlayground();

  const knobs = [
    ['scanModels', 'Scan models', 'also emit definitions for swagger:model types'],
    ['pruneUnusedModels', 'Prune unused models', 'drop definitions nothing references'],
    ['refAliases', 'Ref aliases', 'aliases become $ref instead of being expanded'],
    ['transparentAliases', 'Transparent aliases', 'aliases never produce a definition'],
    ['setXNullableForPointers', 'x-nullable for pointers', 'mark pointer fields nullable'],
    ['skipExtensions', 'Skip extensions', 'suppress the x-go-* vendor extensions'],
  ] as const;
</script>

<aside id="scanner-options">
  <h3>Scanner options</h3>
  {#each knobs as [key, label, hint] (key)}
    <label>
      <input type="checkbox" bind:checked={playground.options[key]} />
      <span>{label}<br /><small class="muted">{hint}</small></span>
    </label>
  {/each}

  <label class="text">
    <span>Build tags</span>
    <input type="text" bind:value={playground.options.buildTags} placeholder="e.g. integration" />
  </label>

  <p class="muted">
    The standard library is synthesized rather than read, so structural detail is lost — see the
    <code>-stub-stdlib</code> advisory.
  </p>
</aside>

<style>
  aside {
    padding: .75rem;
    border-bottom: 1px solid var(--line);
    background: var(--panel);
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(15rem, 1fr));
    gap: .5rem 1rem;
  }
  h3 { grid-column: 1 / -1; margin: 0; font-size: 13px; }
  label { display: flex; gap: .5rem; align-items: start; }
  label.text { flex-direction: column; }
  small { font-size: 11px; }
  p { grid-column: 1 / -1; margin: .25rem 0 0; font-size: 12px; }
</style>
