<script lang="ts">
  import { untrack, type Snippet } from 'svelte';

  // A two-pane split with a draggable separator.
  //
  // Hand-rolled rather than pulled from a component library: the whole of it is a pointer capture and
  // an arrow-key handler, and a splitter is exactly the kind of layout primitive that would have to
  // be re-themed to sit inside a doc-site page anyway.
  //
  // The separator is focusable and moves with the arrow keys, because a split that can only be
  // adjusted by dragging cannot be adjusted at all by somebody using a keyboard - and the ratio
  // decides how much of either pane you can read.

  let {
    start,
    end,
    initial = 50,
    min = 20,
    max = 80,
    startLabel = 'left pane',
    stacked = false,
  }: {
    start: Snippet;
    end: Snippet;
    initial?: number;
    min?: number;
    max?: number;
    startLabel?: string;
    stacked?: boolean;
  } = $props();

  // Only the starting value: `initial` is where a double-click and Enter reset to, not something
  // that drags the split around when a parent re-renders.
  let ratio = $state(untrack(() => initial));
  let container: HTMLDivElement;
  let dragging = $state(false);

  const clamp = (v: number) => Math.min(max, Math.max(min, v));

  function onPointerDown(event: PointerEvent) {
    // Capture on the separator itself: without it a fast drag outruns the pointer and the split
    // stops following, since the events land on whatever is underneath.
    (event.currentTarget as HTMLElement).setPointerCapture(event.pointerId);
    dragging = true;
  }

  function onPointerMove(event: PointerEvent) {
    if (!dragging) {
      return;
    }
    const box = container.getBoundingClientRect();
    const along = stacked
      ? ((event.clientY - box.top) / box.height) * 100
      : ((event.clientX - box.left) / box.width) * 100;

    ratio = clamp(along);
  }

  function onPointerUp(event: PointerEvent) {
    (event.currentTarget as HTMLElement).releasePointerCapture(event.pointerId);
    dragging = false;
  }

  function onKeyDown(event: KeyboardEvent) {
    const step = event.shiftKey ? 10 : 2;
    const back = stacked ? 'ArrowUp' : 'ArrowLeft';
    const forward = stacked ? 'ArrowDown' : 'ArrowRight';

    switch (event.key) {
      case back:
        ratio = clamp(ratio - step);
        break;
      case forward:
        ratio = clamp(ratio + step);
        break;
      case 'Home':
        ratio = min;
        break;
      case 'End':
        ratio = max;
        break;
      case 'Enter':
      case ' ':
        ratio = initial;
        break;
      default:
        return;
    }
    event.preventDefault();
  }
</script>

<div
  class="split"
  class:stacked
  class:dragging
  bind:this={container}
  style="--ratio: {ratio}%"
>
  <div class="pane">{@render start()}</div>

  <!--
    A focusable separator with arrow keys is the ARIA window-splitter pattern itself, not an
    oversight: the rule below cannot tell that role="separator" becomes a widget the moment it is
    given a tabindex, which is exactly what the pattern asks for.
  -->
  <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
  <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
  <div
    class="sep"
    role="separator"
    tabindex="0"
    aria-orientation={stacked ? 'horizontal' : 'vertical'}
    aria-label="Resize {startLabel}"
    aria-valuenow={Math.round(ratio)}
    aria-valuemin={min}
    aria-valuemax={max}
    onpointerdown={onPointerDown}
    onpointermove={onPointerMove}
    onpointerup={onPointerUp}
    onpointercancel={onPointerUp}
    ondblclick={() => (ratio = initial)}
    onkeydown={onKeyDown}
  ></div>

  <div class="pane">{@render end()}</div>
</div>

<style>
  .split {
    display: grid;
    grid-template-columns: var(--ratio) auto 1fr;
    flex: 1;
    min-height: 0;
    min-width: 0;
  }

  .split.stacked {
    grid-template-columns: 1fr;
    grid-template-rows: var(--ratio) auto 1fr;
  }

  .pane {
    display: flex;
    flex-direction: column;
    min-width: 0;
    min-height: 0;
    background: var(--cs-bg);
  }

  /* The hit area is wider than the line, which is one pixel and impossible to grab. */
  .sep {
    position: relative;
    width: 1px;
    background: var(--cs-line);
    cursor: col-resize;
    touch-action: none;
  }

  .split.stacked .sep {
    width: auto;
    height: 1px;
    cursor: row-resize;
  }

  .sep::after {
    content: '';
    position: absolute;
    inset: 0 -3px;
  }

  .split.stacked .sep::after {
    inset: -3px 0;
  }

  .sep:hover,
  .split.dragging .sep {
    background: var(--cs-accent);
  }

  /* While a drag is in flight the panes must not swallow the pointer, or a fast move over the
     editor lands on a text selection instead of the separator. */
  .split.dragging .pane {
    pointer-events: none;
    user-select: none;
  }
</style>
