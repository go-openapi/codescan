<script lang="ts">
  import { usePlayground } from '../lib/store.svelte';
  import type { Severity } from '../lib/types';

  const playground = usePlayground();

  let open = $state(false);
  let list: HTMLUListElement | null = $state(null);

  const counts = $derived(playground.counts);
  const total = $derived(playground.diagnostics.length);

  // Errors open the drawer by themselves. A hint does not: the scan still produced a document, and
  // stealing the reader's space for something they did not ask about is how a panel gets collapsed
  // once and never opened again.
  $effect(() => {
    if (counts.errors > 0) {
      open = true;
    }
  });

  const order: Record<Severity, number> = { error: 0, warning: 1, hint: 2 };
  const sorted = $derived(
    [...playground.visibleDiagnostics].sort((a, b) => order[a.severity] - order[b.severity]),
  );

  const severities: Severity[] = ['error', 'warning', 'hint'];
  const tally = $derived({
    error: counts.errors,
    warning: counts.warnings,
    hint: counts.hints,
  } as Record<Severity, number>);

  function toggleSeverity(severity: Severity) {
    playground.shown = { ...playground.shown, [severity]: !playground.shown[severity] };
    at = -1;
  }

  // ---- height ---------------------------------------------------------------

  // The two panes above can be rebalanced but this one could not, which made it the only thing on
  // screen whose size was somebody else's decision. Held in pixels rather than a percentage: a
  // reader sizes it to hold a number of rows, and rows are a fixed height.
  const minHeight = 80;
  const defaultHeight = 200;
  let height = $state(defaultHeight);
  let dragging = $state(false);

  function maxHeight(): number {
    const shell = drawer?.parentElement?.getBoundingClientRect().height ?? 0;

    return Math.max(minHeight, shell * 0.75);
  }

  function resize(to: number) {
    height = Math.max(minHeight, Math.min(to, maxHeight()));
  }

  let drawer: HTMLElement | null = $state(null);

  function onGrabDown(event: PointerEvent) {
    (event.currentTarget as HTMLElement).setPointerCapture(event.pointerId);
    dragging = true;
  }

  function onGrabMove(event: PointerEvent) {
    if (!dragging || !drawer) {
      return;
    }
    // From the drawer's own bottom edge, so the height follows the pointer exactly rather than
    // accumulating the drift a delta-based drag picks up when it outruns the frames.
    resize(drawer.getBoundingClientRect().bottom - event.clientY);
  }

  function onGrabUp(event: PointerEvent) {
    (event.currentTarget as HTMLElement).releasePointerCapture(event.pointerId);
    dragging = false;
  }

  function onGrabKey(event: KeyboardEvent) {
    const step = event.shiftKey ? 96 : 24;
    switch (event.key) {
      case 'ArrowUp':
        resize(height + step);
        break;
      case 'ArrowDown':
        resize(height - step);
        break;
      case 'Home':
        resize(maxHeight());
        break;
      case 'End':
        resize(minHeight);
        break;
      case 'Enter':
      case ' ':
        resize(defaultHeight);
        break;
      default:
        return;
    }
    event.preventDefault();
  }

  // The cursor into the list. Third pane, same idea as the other two: something is current, it is
  // visible, and the arrow keys move it.
  let at = $state(-1);

  // A rescan replaces the list, and an index into the old one means nothing. Reset rather than
  // leaving a highlight on whatever now happens to sit at that position.
  $effect(() => {
    void playground.diagnostics;
    at = -1;
  });

  function go(next: number) {
    if (!sorted.length) {
      return;
    }
    at = Math.max(0, Math.min(next, sorted.length - 1));
    open = true;

    // The row has to be visible to be the current one, and the drawer is short.
    queueMicrotask(() => {
      list?.querySelector<HTMLElement>('[data-at="true"]')?.scrollIntoView({ block: 'nearest' });
    });

    // Following on move rather than only on Enter is what makes arrowing through diagnostics useful:
    // each one takes you to the line that raised it. Gated on the track mode, like the other two
    // directions, so a reader who turned it off is not dragged around.
    if (playground.tracking) {
      reveal(at);
    }
  }

  function reveal(index: number) {
    const d = sorted[index];
    if (d?.file && d.line) {
      playground.reveal(d.file, d.line);
    }
  }

  function onKeyDown(event: KeyboardEvent) {
    switch (event.key) {
      case 'ArrowDown':
        go(at + 1);
        break;
      case 'ArrowUp':
        go(at - 1);
        break;
      case 'Home':
        go(0);
        break;
      case 'End':
        go(sorted.length - 1);
        break;
      case 'PageDown':
        go(at + 10);
        break;
      case 'PageUp':
        go(at - 10);
        break;
      case 'Enter':
      case ' ':
        // Always jumps, even with tracking off: pressing Enter on a row is asking, not browsing.
        if (at >= 0) {
          reveal(at);
        }
        break;
      default:
        return;
    }
    event.preventDefault();
  }

  function where(file?: string, line?: number): string {
    if (!file) {
      return '';
    }

    return line ? `${file}:${line}` : file;
  }
</script>

<section
  class="drawer"
  class:open
  class:dragging
  bind:this={drawer}
  style={open ? `height: ${height}px` : undefined}
>
  {#if open}
    <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
    <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
    <div
      class="grab"
      role="separator"
      tabindex="0"
      aria-orientation="horizontal"
      aria-label="Resize the diagnostics pane"
      aria-valuenow={Math.round(height)}
      aria-valuemin={minHeight}
      onpointerdown={onGrabDown}
      onpointermove={onGrabMove}
      onpointerup={onGrabUp}
      onpointercancel={onGrabUp}
      ondblclick={() => resize(defaultHeight)}
      onkeydown={onGrabKey}
    ></div>
  {/if}

  <h2 class="bar">
    <button
      class="cs-btn-quiet toggle"
      aria-expanded={open}
      aria-controls="diagnostics-body"
      onclick={() => (open = !open)}
    >
      <span class="chevron" aria-hidden="true">{open ? '▾' : '▸'}</span>
      <span class="cs-label">Diagnostics</span>
    </button>

    {#if total === 0}
      <span class="cs-pill">{playground.scanned ? 'none' : '—'}</span>
    {:else}
      <!-- Each severity is a switch as well as a count. At dockerctl's 193 the list is unreadable
           without one, and the counts were already sitting here doing half the job. -->
      {#each severities as severity (severity)}
        {#if tally[severity] > 0}
          <button
            class="cs-pill cs-pill-{severity} chip"
            class:off={!playground.shown[severity]}
            aria-pressed={playground.shown[severity]}
            onclick={() => toggleSeverity(severity)}
            title="{playground.shown[severity] ? 'Hide' : 'Show'} {severity}s"
          >{tally[severity]} {severity}{tally[severity] === 1 ? '' : 's'}</button>
        {/if}
      {/each}

      {#if open && at >= 0}
        <span class="cs-faint pos">{at + 1}/{sorted.length}</span>
      {/if}
    {/if}
  </h2>

  {#if open}
    <div class="cs-scroll body" id="diagnostics-body">
      {#if sorted.length === 0}
        <p class="empty cs-muted">
          {#if total > 0}
            All {total} are hidden by the filter above.
          {:else if playground.scanned}
            The scan had nothing to report.
          {:else}
            Nothing scanned yet.
          {/if}
        </p>
      {:else}
        <!-- A listbox rather than a list of buttons: there is a current item, the arrow keys move
             it, and focus stays on the list so the browser does not scroll under us on every step. -->
        <ul
          bind:this={list}
          role="listbox"
          tabindex="0"
          aria-label="Scan diagnostics"
          aria-activedescendant={at >= 0 ? `diag-${at}` : undefined}
          onkeydown={onKeyDown}
          onfocus={() => { if (at < 0) go(0); }}
        >
          {#each sorted as d, i (d.code + i)}
            <!--
              The keyboard handler belongs to the listbox, not to each option: with
              aria-activedescendant the options are never focused, which is the whole point of the
              pattern. The rule cannot see that the container above already handles the keys.
            -->
            <!-- svelte-ignore a11y_click_events_have_key_events -->
            <li
              id="diag-{i}"
              role="option"
              aria-selected={i === at}
              class="row"
              class:at={i === at}
              data-at={i === at}
              onclick={() => go(i)}
              title={d.file ? `Go to ${where(d.file, d.line)}` : 'This one names no position'}
            >
              <span class="sev sev-{d.severity}">{d.severity}</span>
              <span class="msg">{d.message}</span>
              <span class="meta cs-mono cs-faint">
                {#if d.file}<span>{where(d.file, d.line)}</span>{/if}
                <span>{d.code}</span>
              </span>
            </li>
          {/each}
        </ul>
      {/if}
    </div>
  {/if}
</section>

<style>
  .drawer {
    position: relative;
    display: flex;
    flex-direction: column;
    flex: none;
    border-top: 1px solid var(--cs-line);
    background: var(--cs-bg);
  }

  .drawer.open {
    min-height: 0;
  }

  /* The hit area is wider than the border it sits on, which is one pixel and impossible to grab. */
  .grab {
    position: absolute;
    top: -3px;
    left: 0;
    right: 0;
    height: 7px;
    cursor: row-resize;
    touch-action: none;
    z-index: 1;
  }

  .grab:hover,
  .drawer.dragging .grab {
    background: var(--cs-accent);
  }

  .drawer.dragging .body {
    pointer-events: none;
    user-select: none;
  }

  .bar {
    display: flex;
    align-items: center;
    gap: var(--cs-s2);
    height: var(--cs-tab-h);
    padding: 0 var(--cs-s2);
    background: var(--cs-bg-sunken);
    flex: none;
    font-size: inherit;
    font-weight: inherit;
  }

  .toggle {
    gap: var(--cs-s2);
  }

  .chip {
    border: 0;
    height: 1.25rem;
    cursor: pointer;
  }

  /* Off is stated by draining the colour, not by hiding the chip: the count still matters, and a
     control that vanishes when you switch it off cannot be switched back on. */
  .chip.off {
    color: var(--cs-fg-faint);
    background: transparent;
    box-shadow: inset 0 0 0 1px var(--cs-line-strong);
  }

  .chevron {
    font-size: 10px;
    width: 0.75em;
  }

  .pos {
    margin-left: auto;
    font-size: var(--cs-fs-xs);
    font-variant-numeric: tabular-nums;
    padding-right: var(--cs-s2);
  }

  .body {
    padding: var(--cs-s1) 0;
  }

  .empty {
    padding: var(--cs-s3);
    font-size: var(--cs-fs-sm);
  }

  ul {
    margin: 0;
    padding: 0;
    list-style: none;
  }

  ul:focus-visible {
    outline: none;
  }

  .row {
    display: grid;
    grid-template-columns: 4.5rem 1fr auto;
    gap: var(--cs-s3);
    align-items: baseline;
    padding: var(--cs-s1) var(--cs-s3);
    font-size: var(--cs-fs-sm);
    cursor: pointer;
    border-left: 3px solid transparent;
  }

  .row:hover {
    background: var(--cs-bg-hover);
  }

  /* The same treatment the editors give their active line, for the same reason: three panes, one
     idea of what "current" looks like. */
  .row.at {
    background: var(--cs-accent-soft);
    border-left-color: var(--cs-accent);
  }

  .sev {
    font-size: var(--cs-fs-xs);
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .sev-error { color: var(--cs-error); }
  .sev-warning { color: var(--cs-warning); }
  .sev-hint { color: var(--cs-hint); }

  .msg {
    min-width: 0;
  }

  .meta {
    display: flex;
    gap: var(--cs-s3);
    font-size: var(--cs-fs-xs);
    white-space: nowrap;
  }
</style>
