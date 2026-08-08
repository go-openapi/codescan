<script lang="ts">
  // A tablist, to the WAI-ARIA authoring pattern: roving tabindex, arrow keys move and activate,
  // Home/End jump to the ends.
  //
  // Hand-rolled because that pattern is small and completely specified, and a styled component
  // library would arrive with a look this has to override anyway. The line is drawn at dialogs and
  // menus - focus trapping and inert backgrounds are where a library earns its bytes, and neither
  // exists yet.

  export type Tab = { id: string; label: string; badge?: string };

  let {
    tabs,
    active = $bindable(),
    label,
  }: {
    tabs: Tab[];
    active: string;
    label: string;
  } = $props();

  let buttons: HTMLButtonElement[] = $state([]);

  function focusTab(index: number) {
    const wrapped = (index + tabs.length) % tabs.length;
    active = tabs[wrapped].id;
    buttons[wrapped]?.focus();
  }

  function onKeyDown(event: KeyboardEvent, index: number) {
    switch (event.key) {
      case 'ArrowRight':
        focusTab(index + 1);
        break;
      case 'ArrowLeft':
        focusTab(index - 1);
        break;
      case 'Home':
        focusTab(0);
        break;
      case 'End':
        focusTab(tabs.length - 1);
        break;
      default:
        return;
    }
    event.preventDefault();
  }
</script>

<div class="tabs" role="tablist" aria-label={label}>
  {#each tabs as tab, i (tab.id)}
    <button
      bind:this={buttons[i]}
      role="tab"
      id="tab-{tab.id}"
      class="tab"
      class:active={active === tab.id}
      aria-selected={active === tab.id}
      aria-controls="panel-{tab.id}"
      tabindex={active === tab.id ? 0 : -1}
      onclick={() => (active = tab.id)}
      onkeydown={(e) => onKeyDown(e, i)}
    >
      {tab.label}
      {#if tab.badge}
        <span class="badge">{tab.badge}</span>
      {/if}
    </button>
  {/each}
</div>

<style>
  .tabs {
    display: flex;
    align-items: stretch;
    gap: var(--cs-s1);
    height: var(--cs-tab-h);
    padding: 0 var(--cs-s2);
    background: var(--cs-bg-sunken);
    border-bottom: 1px solid var(--cs-line);
    flex: none;
  }

  .tab {
    height: auto;
    border: 0;
    border-radius: 0;
    background: transparent;
    color: var(--cs-fg-muted);
    font-size: var(--cs-fs-sm);
    padding: 0 var(--cs-s3);
    /* The indicator is a border on the element itself rather than a pseudo-element, so the label
       never shifts by a pixel as it becomes selected. */
    border-bottom: 2px solid transparent;
    margin-bottom: -1px;
  }

  .tab:hover {
    color: var(--cs-fg);
    background: transparent;
  }

  .tab.active {
    color: var(--cs-fg);
    border-bottom-color: var(--cs-accent);
    font-weight: 600;
  }

  .badge {
    font-size: var(--cs-fs-xs);
    font-variant-numeric: tabular-nums;
    color: var(--cs-fg-muted);
    background: var(--cs-bg-inset);
    border-radius: 999px;
    padding: 0 var(--cs-s2);
  }
</style>
