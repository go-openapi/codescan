<script lang="ts">
  import { go } from '@codemirror/lang-go';
  import { usePlayground } from '../lib/store.svelte';
  import { spanFor } from '../lib/track';
  import Editor from './Editor.svelte';
  import FileTree from './FileTree.svelte';
  import type { GutterMark } from '../lib/editor';

  const playground = usePlayground();
  const current = $derived(playground.current);

  // The pane shows the tree or the file, never both - the terminal UI's left pane does the same, and
  // for the same reason: at a vendored module's scale a tree beside an editor leaves neither enough
  // width to read.
  let showing = $state<'file' | 'tree'>('file');

  const marks: GutterMark[] = $derived(
    playground.visibleDiagnostics
      .filter((d) => d.file === playground.selected && d.line)
      .map((d) => ({ line: d.line!, severity: d.severity, title: `${d.severity}: ${d.message}` })),
  );

  // The source side is highlighted when the spec side asked a question of it.
  const highlight = $derived(
    playground.trackedSource?.file === playground.selected
      ? { from: playground.trackedSource.line, to: playground.trackedSource.line }
      : null,
  );

  // What the cursor is currently pointing at, named so the mode is legible rather than mysterious.
  const pointing = $derived.by(() => {
    if (!playground.trackedPointer) {
      return '';
    }

    return spanFor(playground.rendered.spans, playground.trackedPointer)
      ? playground.trackedPointer
      : `${playground.trackedPointer} (not in the document)`;
  });
</script>

<section class="pane">
  <header class="head">
    <span class="cs-label">Source</span>

    <button
      class="cs-btn-quiet crumb"
      aria-pressed={showing === 'tree'}
      onclick={() => (showing = showing === 'tree' ? 'file' : 'tree')}
      title="Browse the {playground.files.length} files in this module"
    >
      <span aria-hidden="true">{showing === 'tree' ? '▾' : '▸'}</span>
      <span class="path">{playground.selected || 'no file'}</span>
    </button>
  </header>

  {#if showing === 'tree'}
    <FileTree onPick={() => (showing = 'file')} />
  {:else if current}
    <Editor
      text={current.text}
      language={go()}
      label="Source of {current.path}"
      {marks}
      {highlight}
      revealLine={playground.sourceReveal}
      onChange={(text) => playground.edit(current.path, text)}
      onCursorLine={(line) => playground.fromSource(line)}
    />
  {:else}
    <p class="empty cs-muted">No file selected.</p>
  {/if}

  {#if pointing}
    <p class="trail cs-mono cs-faint" title="The spec node this line produced">→ {pointing}</p>
  {/if}
</section>

<style>
  .pane {
    display: flex;
    flex-direction: column;
    min-width: 0;
    min-height: 0;
    flex: 1;
  }

  .head {
    display: flex;
    align-items: center;
    gap: var(--cs-s2);
    flex: none;
    height: var(--cs-tab-h);
    padding: 0 var(--cs-s2) 0 var(--cs-s3);
    background: var(--cs-bg-sunken);
    border-bottom: 1px solid var(--cs-line);
  }

  .crumb {
    min-width: 0;
    flex: 1;
    justify-content: flex-start;
    font-family: var(--cs-font-mono);
    font-size: var(--cs-fs-sm);
  }

  .path {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .empty {
    padding: var(--cs-s4);
    font-size: var(--cs-fs-sm);
  }

  /* Says where the cursor points without moving anything, so the join is legible even when the
     other pane is scrolled away or collapsed. */
  .trail {
    flex: none;
    padding: var(--cs-s1) var(--cs-s3);
    font-size: var(--cs-fs-xs);
    border-top: 1px solid var(--cs-line);
    background: var(--cs-bg-sunken);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
</style>
