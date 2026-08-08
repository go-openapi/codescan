<script lang="ts">
  import { json } from '@codemirror/lang-json';
  import { yaml } from '@codemirror/lang-yaml';
  import { usePlayground } from '../lib/store.svelte';
  import { spanFor } from '../lib/track';
  import Tabs, { type Tab } from './Tabs.svelte';
  import Editor from './Editor.svelte';
  import SwaggerPreview from './SwaggerPreview.svelte';
  import Booting from './Booting.svelte';

  const playground = usePlayground();

  let active = $state('spec');

  let searching = $state(false);
  let box: HTMLInputElement | null = $state(null);

  function openSearch() {
    searching = true;
    // Always from empty. A search is started to look for something, not to resume the last one -
    // the terminal UI makes the same choice for the same reason.
    playground.clearSearch();
    queueMicrotask(() => box?.select());
  }

  function closeSearch() {
    searching = false;
    playground.clearSearch();
  }

  // Keys the pane answers when the prompt is closed. `/` is free here because the document is
  // read-only: there is nothing it could be typing into.
  function onPaneKey(event: KeyboardEvent) {
    if (searching || event.metaKey || event.ctrlKey || event.altKey) {
      return;
    }
    switch (event.key) {
      case '/':
        openSearch();
        break;
      case 'n':
        playground.stepMatch(1);
        break;
      case 'N':
        playground.stepMatch(-1);
        break;
      default:
        return;
    }
    event.preventDefault();
  }

  function onBoxKey(event: KeyboardEvent) {
    switch (event.key) {
      case 'Enter':
        playground.stepMatch(event.shiftKey ? -1 : 1);
        break;
      case 'Escape':
        closeSearch();
        break;
      default:
        return;
    }
    event.preventDefault();
  }

  const tabs: Tab[] = [
    { id: 'spec', label: 'Spec' },
    { id: 'ui', label: 'Swagger UI' },
  ];

  const has = $derived(playground.spec !== null && playground.spec !== undefined);
  const language = $derived(playground.format === 'yaml' ? yaml() : json());

  // The spec side is highlighted when the source side asked a question of it. Falls back to the
  // nearest rendered ancestor, since a pointer can be real and own no line - an empty object in
  // YAML, say - and highlighting nothing would read as the join being broken.
  const highlight = $derived(
    playground.trackedPointer ? spanFor(playground.rendered.spans, playground.trackedPointer) : null,
  );

  async function copy() {
    await navigator.clipboard.writeText(playground.rendered.text);
  }

  function download() {
    const yamlish = playground.format === 'yaml';
    const blob = new Blob([playground.rendered.text], {
      type: yamlish ? 'application/yaml' : 'application/json',
    });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = yamlish ? 'swagger.yaml' : 'swagger.json';
    a.click();
    URL.revokeObjectURL(url);
  }
</script>

<section class="pane">
  <div class="head">
    <Tabs {tabs} bind:active label="Result view" />

    <div class="tools" class:hidden={active !== 'spec'}>
      <div class="formats" role="group" aria-label="Document format">
        {#each ['json', 'yaml'] as const as f (f)}
          <button
            class="cs-btn-quiet fmt"
            class:on={playground.format === f}
            aria-pressed={playground.format === f}
            onclick={() => (playground.format = f)}
          >{f.toUpperCase()}</button>
        {/each}
      </div>

      <button
        class="cs-btn-quiet"
        onclick={() => (searching ? closeSearch() : openSearch())}
        aria-pressed={searching}
        disabled={!has}
        title="Search the document (/)"
      >Find</button>

      <button class="cs-btn-quiet" onclick={copy} disabled={!has} title="Copy the document">Copy</button>
      <button class="cs-btn-quiet" onclick={download} disabled={!has} title="Download the document">Download</button>
    </div>
  </div>

  {#if active === 'ui'}
    <div class="body" role="tabpanel" id="panel-ui" aria-labelledby="tab-ui">
      <SwaggerPreview />
    </div>
  {:else}
  <!-- The keys are caught here rather than inside the editor because they belong to the pane, not to
       the document: `/` opens the pane's find bar, and n/N step the pane's matches. They reach here
       by bubbling out of the editor, which is read-only and so has no use for them itself.
       tabindex="-1" keeps the panel out of the tab order - the editor within it is the tab stop -
       while letting it be focused programmatically. -->
  <div
    class="body"
    role="tabpanel"
    tabindex="-1"
    id="panel-spec"
    aria-labelledby="tab-spec"
    onkeydown={onPaneKey}
  >
    {#if playground.error}
      <div class="notice">
        <p class="title cs-error">The scan did not finish</p>
        <pre class="detail">{playground.error}</pre>
      </div>
    {:else if has}
      <Editor
        text={playground.rendered.text}
        {language}
        readonly
        label="The generated specification"
        {highlight}
        marks={playground.specMarks}
        matches={{ lines: playground.matches, current: playground.currentMatch }}
        revealLine={playground.specReveal}
        cursorTo={playground.specCursor}
        onCursorLine={(line) => playground.fromSpec(line)}
      />
    {:else if playground.running}
      <Booting />
    {:else}
      <div class="notice cs-muted">
        <p class="title">Nothing scanned yet</p>
        <p>Edit the source on the left — it rescans as you type — or press <strong>Scan</strong>.</p>
      </div>
    {/if}
  </div>

  {/if}

  {#if active === 'spec' && has && playground.specOrigin}
    <p class="trail cs-mono cs-faint" title="Where this line came from">
      <span class="ptr">{playground.specOrigin.pointer || '/'}</span>
      <span class="arrow" aria-hidden="true">←</span>
      <span>{playground.specOrigin.where}</span>
    </p>
  {/if}

  {#if active === 'spec' && searching}
    <div class="find">
      <span class="slash cs-faint" aria-hidden="true">/</span>
      <input
        bind:this={box}
        type="text"
        value={playground.query}
        placeholder="search spec"
        aria-label="Search the document"
        oninput={(e) => playground.search((e.currentTarget as HTMLInputElement).value)}
        onkeydown={onBoxKey}
      />

      {#if playground.query}
        <span class="count" class:none={playground.matchInfo.total === 0}>
          {playground.matchInfo.total === 0
            ? 'no match'
            : `${playground.matchInfo.current}/${playground.matchInfo.total}`}
        </span>
      {/if}

      <button class="cs-btn-quiet" onclick={() => playground.stepMatch(-1)} title="Previous match (Shift+Enter, N)">↑</button>
      <button class="cs-btn-quiet" onclick={() => playground.stepMatch(1)} title="Next match (Enter, n)">↓</button>
      <button class="cs-btn-quiet" onclick={closeSearch} title="Close (Esc)">×</button>
    </div>
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
    align-items: stretch;
    flex: none;
    background: var(--cs-bg-sunken);
    border-bottom: 1px solid var(--cs-line);
  }

  /* The tablist carries its own bottom border for the selected-tab indicator; the header already
     draws one, so the inner one is cancelled rather than doubled. */
  .head :global(.tabs) {
    flex: 1;
    border-bottom: 0;
  }

  .tools.hidden {
    visibility: hidden;
  }

  .tools {
    display: flex;
    align-items: center;
    gap: var(--cs-s1);
    padding-right: var(--cs-s2);
  }

  .formats {
    display: flex;
    margin-right: var(--cs-s1);
  }

  .fmt {
    font-size: var(--cs-fs-xs);
    padding: 0 var(--cs-s2);
    height: 1.4rem;
  }

  .fmt.on {
    color: var(--cs-fg);
    background: var(--cs-bg-inset);
  }

  .body {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-height: 0;
    background: var(--cs-bg);
    overflow: hidden;
  }

  /* The mirror of the source pane's trail: each side says what the other would light up, so the join
     stays legible when one pane is scrolled away. */
  .trail {
    display: flex;
    gap: var(--cs-s2);
    flex: none;
    padding: var(--cs-s1) var(--cs-s3);
    font-size: var(--cs-fs-xs);
    border-top: 1px solid var(--cs-line);
    background: var(--cs-bg-sunken);
    white-space: nowrap;
    overflow: hidden;
  }

  .ptr {
    overflow: hidden;
    text-overflow: ellipsis;
    direction: rtl;
    text-align: left;
  }

  .arrow {
    flex: none;
  }

  /* Attached to what it searches rather than to the status line: in a browser a find bar belongs to
     its document, and the status line has status to show. */
  .find {
    display: flex;
    align-items: center;
    gap: var(--cs-s2);
    flex: none;
    padding: var(--cs-s1) var(--cs-s2);
    background: var(--cs-bg-sunken);
    border-top: 1px solid var(--cs-line);
  }

  .find input {
    flex: 1;
    min-width: 0;
    height: 1.6rem;
    font-family: var(--cs-font-mono);
    font-size: var(--cs-fs-sm);
  }

  .slash {
    font-family: var(--cs-font-mono);
  }

  .count {
    font-size: var(--cs-fs-xs);
    font-variant-numeric: tabular-nums;
    color: var(--cs-fg-muted);
    white-space: nowrap;
  }

  .count.none {
    color: var(--cs-error);
  }

  .find button {
    width: 1.6rem;
    padding: 0;
    justify-content: center;
  }

  .notice {
    padding: var(--cs-s4);
    font-size: var(--cs-fs-sm);
  }

  .notice .title {
    font-weight: 600;
    margin-bottom: var(--cs-s1);
  }

  .detail {
    margin-top: var(--cs-s2);
    padding: var(--cs-s2);
    background: var(--cs-error-soft);
    border-radius: var(--cs-r2);
    white-space: pre-wrap;
    color: var(--cs-fg);
  }
</style>
