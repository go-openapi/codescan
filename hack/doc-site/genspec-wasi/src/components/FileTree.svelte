<script lang="ts">
  import { usePlayground } from '../lib/store.svelte';
  import { build, flatten, pathsToOpen } from '../lib/filetree';
  import type { Severity } from '../lib/types';

  let { onPick }: { onPick: () => void } = $props();

  const playground = usePlayground();

  const tree = $derived(build(playground.files.map((f) => f.path)));

  // vendor/ starts closed. It is scanned either way, and in a vendored module it is 95% of the rows.
  let open = $state(new Set<string>());
  let touched = $state(false);

  // Until the reader opens something themselves, show the module's own tree expanded and leave
  // vendor folded - which is the shape they came to look at.
  const opened = $derived.by(() => {
    if (touched) {
      return open;
    }
    const initial = new Set<string>();
    for (const node of tree) {
      if (node.dir && !node.path.startsWith('vendor')) {
        initial.add(node.path);
      }
    }
    for (const path of pathsToOpen(tree, playground.selected)) {
      initial.add(path);
    }

    return initial;
  });

  const rows = $derived(flatten(tree, (path) => opened.has(path)));

  // Worst severity per file, so a package with nothing reported against it looks that way.
  const worst = $derived.by(() => {
    const rank: Record<Severity, number> = { error: 0, warning: 1, hint: 2 };
    const map = new Map<string, Severity>();
    for (const d of playground.visibleDiagnostics) {
      if (!d.file) {
        continue;
      }
      const held = map.get(d.file);
      if (!held || rank[d.severity] < rank[held]) {
        map.set(d.file, d.severity);
      }
    }

    return map;
  });

  let at = $state(0);
  let list: HTMLUListElement | null = $state(null);

  function toggle(path: string) {
    touched = true;
    const next = new Set(opened);
    if (next.has(path)) {
      next.delete(path);
    } else {
      next.add(path);
    }
    open = next;
  }

  function activate(index: number) {
    const row = rows[index];
    if (!row) {
      return;
    }
    if (row.node.dir) {
      toggle(row.node.path);

      return;
    }
    playground.selected = row.node.path;
    onPick();
  }

  function move(next: number) {
    at = Math.max(0, Math.min(next, rows.length - 1));
    queueMicrotask(() => {
      list?.querySelector<HTMLElement>('[data-at="true"]')?.scrollIntoView({ block: 'nearest' });
    });
  }

  function onKeyDown(event: KeyboardEvent) {
    const row = rows[at];
    switch (event.key) {
      case 'ArrowDown':
        move(at + 1);
        break;
      case 'ArrowUp':
        move(at - 1);
        break;
      case 'Home':
        move(0);
        break;
      case 'End':
        move(rows.length - 1);
        break;
      case 'PageDown':
        move(at + 12);
        break;
      case 'PageUp':
        move(at - 12);
        break;
      // Right opens a directory, left closes it - the tree convention, and what makes a deep tree
      // navigable without reaching for the pointer.
      case 'ArrowRight':
        if (row?.node.dir && !opened.has(row.node.path)) {
          toggle(row.node.path);
        } else {
          move(at + 1);
        }
        break;
      case 'ArrowLeft':
        if (row?.node.dir && opened.has(row.node.path)) {
          toggle(row.node.path);
        } else {
          move(at - 1);
        }
        break;
      case 'Enter':
      case ' ':
        activate(at);
        break;
      default:
        return;
    }
    event.preventDefault();
  }
</script>

<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<ul
  bind:this={list}
  class="cs-scroll tree"
  role="tree"
  tabindex="0"
  aria-label="Files in the module"
  aria-activedescendant={rows[at] ? `tree-${at}` : undefined}
  onkeydown={onKeyDown}
>
  {#each rows as row, i (row.node.path)}
    <!-- The keys live on the tree, not on each row: with aria-activedescendant no row is ever
         focused, which is what keeps the browser from scrolling under us on every step. -->
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <li
      id="tree-{i}"
      role="treeitem"
      aria-selected={row.node.path === playground.selected}
      aria-expanded={row.node.dir ? opened.has(row.node.path) : undefined}
      class="row"
      class:at={i === at}
      class:current={row.node.path === playground.selected}
      class:dir={row.node.dir}
      data-at={i === at}
      style="--depth: {row.depth}"
      onclick={() => { at = i; activate(i); }}
      title={row.node.path}
    >
      <span class="twist" aria-hidden="true">{row.node.dir ? (opened.has(row.node.path) ? '▾' : '▸') : ''}</span>
      <span class="name">{row.node.name}</span>
      {#if !row.node.dir && worst.get(row.node.path)}
        <span class="dot dot-{worst.get(row.node.path)}" aria-hidden="true"></span>
      {/if}
    </li>
  {/each}
</ul>

<style>
  .tree {
    margin: 0;
    padding: var(--cs-s1) 0;
    list-style: none;
    background: var(--cs-bg);
  }

  .tree:focus-visible {
    outline: none;
  }

  .row {
    display: flex;
    align-items: center;
    gap: var(--cs-s1);
    padding: 1px var(--cs-s2) 1px calc(var(--cs-s2) + var(--depth) * 0.85rem);
    font-family: var(--cs-font-mono);
    font-size: var(--cs-fs-sm);
    white-space: nowrap;
    cursor: pointer;
    border-left: 3px solid transparent;
  }

  .row:hover {
    background: var(--cs-bg-hover);
  }

  .row.dir .name {
    color: var(--cs-fg-muted);
  }

  .row.current .name {
    font-weight: 600;
  }

  /* Same vocabulary as everywhere else: the accent wash and bar mean "this is the one". */
  .row.at {
    background: var(--cs-accent-soft);
    border-left-color: var(--cs-accent);
  }

  .twist {
    width: 0.75em;
    font-size: 10px;
    color: var(--cs-fg-faint);
  }

  .name {
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    margin-left: auto;
    flex: none;
  }

  .dot-error { background: var(--cs-error); }
  .dot-warning { background: var(--cs-warning); }
  .dot-hint { background: var(--cs-hint); }
</style>
