<script lang="ts">
  import { usePlayground } from '../lib/store.svelte';
  import { examples } from '../lib/examples';

  // The gallery. A row of what codescan can be shown doing, each one a whole module that scans.
  //
  // Sitting in the chrome rather than behind a dialog: a visitor arriving from a tutorial has not
  // decided to explore yet, and a gallery they have to go looking for is one they will not find.

  const playground = usePlayground();

  let openList = $state(false);

</script>

<div class="wrap">
  <button
    class="cs-btn-quiet trigger"
    aria-expanded={openList}
    aria-controls="examples-list"
    onclick={() => (openList = !openList)}
    title="Load one of the bundled examples"
  >
    <span aria-hidden="true">{openList ? '▾' : '▸'}</span>
    Examples
  </button>

  {#if openList}
    <!-- A plain absolutely-positioned list rather than a dialog: nothing here needs a focus trap,
         and clicking anywhere in it does the one thing it is for. -->
    <ul id="examples-list" class="list">
      {#each examples as example (example.id)}
        <li>
          <button
            class="item"
            class:on={example.id === playground.example}
            aria-current={example.id === playground.example ? 'true' : undefined}
            onclick={() => { playground.openExample(example.id); openList = false; }}
          >
            <span class="title">
              <span class="tick" aria-hidden="true">{example.id === playground.example ? '✓' : ''}</span>
              {example.title}
            </span>
            <span class="blurb cs-faint">{example.blurb}</span>
          </button>
        </li>
      {/each}
    </ul>
  {/if}
</div>

<svelte:window onclick={(e) => {
  // Closes on any click that did not land inside. Cheap, and correct for a list with no state to
  // lose: reopening costs one click.
  if (openList && !(e.target as HTMLElement).closest('.wrap')) {
    openList = false;
  }
}} />

<style>
  .wrap {
    position: relative;
  }

  .trigger {
    gap: var(--cs-s1);
  }

  .list {
    position: absolute;
    top: calc(100% + var(--cs-s1));
    left: 0;
    z-index: 10;
    width: min(26rem, 80vw);
    margin: 0;
    padding: var(--cs-s1);
    list-style: none;
    background: var(--cs-bg-raised);
    border: 1px solid var(--cs-line-strong);
    border-radius: var(--cs-r3);
    box-shadow: var(--cs-shadow-2);
  }

  .item {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 1px;
    width: 100%;
    height: auto;
    padding: var(--cs-s2) var(--cs-s3);
    text-align: left;
    background: transparent;
    border: 0;
    border-radius: var(--cs-r2);
    white-space: normal;
  }

  .item:hover {
    background: var(--cs-bg-hover);
  }

  .item.on {
    background: var(--cs-accent-soft);
  }

  .title {
    display: flex;
    align-items: baseline;
    gap: var(--cs-s1);
    font-weight: 600;
  }

  /* Reserved whether or not it is filled, so the titles line up and the list does not shuffle
     sideways as the loaded example changes. */
  .tick {
    width: 0.8em;
    color: var(--cs-accent);
  }

  .blurb {
    padding-left: calc(0.8em + var(--cs-s1));
    font-size: var(--cs-fs-xs);
    line-height: 1.4;
  }
</style>
