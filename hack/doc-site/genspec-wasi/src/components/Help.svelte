<script lang="ts">
  import { humanBytes } from '../lib/format';
  import { limits } from '../lib/tree';

  // How to scan your own code, and what the keys do.
  //
  // A native <dialog> opened with showModal(): the focus trap, Escape, the backdrop and the
  // inertness of everything behind it are the platform's, and they are the whole reason a headless
  // component library would have been worth its bytes here. It is not, now.

  let dialog: HTMLDialogElement | null = $state(null);

  export function open() {
    dialog?.showModal();
  }
</script>

<button
  class="cs-btn-quiet mark"
  onclick={() => dialog?.showModal()}
  aria-label="How to scan your own code"
  title="How to scan your own code"
>?</button>

<dialog bind:this={dialog} class="cs-root sheet">
  <header>
    <h2>Scanning your own code</h2>
    <button class="cs-btn-quiet" onclick={() => dialog?.close()} aria-label="Close">×</button>
  </header>

  <div class="body">
    <ol class="steps">
      <li>
        <p class="what">Make sure it is a module</p>
        <pre>go mod init example.com/api</pre>
        <p class="why cs-muted">
          Skip this if there is already a <code>go.mod</code>. The scanner resolves import paths
          against the module, so without one there is nothing to resolve them against.
        </p>
      </li>

      <li>
        <p class="what">Vendor the dependencies</p>
        <pre>go mod vendor</pre>
        <p class="why cs-muted">
          There is no module cache in your browser and nothing is downloaded, so a dependency's types
          can only arrive as source. It matters more than it looks: a library that declares things in
          its <em>comments</em> — <code>strfmt</code> and its <code>swagger:strfmt</code> marks —
          cannot be understood any other way.
        </p>
      </li>

      <li>
        <p class="what">Open the folder</p>
        <p class="why cs-muted">
          <strong>Open module…</strong> takes the whole directory in one gesture. It keeps
          <code>.go</code> files, <code>go.mod</code> and <code>vendor/modules.txt</code>, skips
          tests, and re-roots on the outermost <code>go.mod</code> — so it does not matter whether you
          pick the module, its parent or a subdirectory. Up to {humanBytes(limits.maxBytes)}.
        </p>
      </li>
    </ol>

    <p class="privacy">
      The files never leave your browser. There is no server: the scanner itself is a WebAssembly
      build of codescan, running in this tab.
    </p>

    <h3>Keys</h3>
    <table>
      <tbody>
        <tr><td><kbd>↑</kbd> <kbd>↓</kbd> <kbd>Home</kbd> <kbd>End</kbd> <kbd>PgUp</kbd> <kbd>PgDn</kbd></td>
          <td>move in any pane — with <strong>Track</strong> on, the others follow</td></tr>
        <tr><td><kbd>/</kbd></td><td>search the document</td></tr>
        <tr><td><kbd>n</kbd> <kbd>N</kbd></td><td>next / previous match, wrapping</td></tr>
        <tr><td><kbd>→</kbd> <kbd>←</kbd></td><td>open / close a directory in the file tree</td></tr>
        <tr><td><kbd>Enter</kbd></td><td>open a file, or go to a diagnostic's line</td></tr>
        <tr><td><kbd>Esc</kbd></td><td>close the search, or this</td></tr>
      </tbody>
    </table>

    <h3>What the two panes are for</h3>
    <p class="prose cs-muted">
      The left is your source and the right is what codescan makes of it. Editing rescans after a
      pause. With <strong>Track</strong> on, a line on either side lights up what it produced — or
      what produced it — and a diagnostic lights up both.
    </p>
  </div>
</dialog>

<style>
  .mark {
    width: 1.85rem;
    padding: 0;
    justify-content: center;
    font-size: var(--cs-fs-lg);
  }

  .sheet {
    width: min(42rem, 92vw);
    max-height: 85vh;
    padding: 0;
    color: var(--cs-fg);
    background: var(--cs-bg-raised);
    border: 1px solid var(--cs-line-strong);
    border-radius: var(--cs-r3);
    box-shadow: var(--cs-shadow-2);
  }

  .sheet::backdrop {
    background: #0008;
  }

  header {
    display: flex;
    align-items: center;
    padding: var(--cs-s3) var(--cs-s4);
    border-bottom: 1px solid var(--cs-line);
  }

  h2 {
    flex: 1;
    font-size: var(--cs-fs-lg);
  }

  h3 {
    margin: var(--cs-s5) 0 var(--cs-s2);
    font-size: var(--cs-fs-md);
  }

  .body {
    padding: var(--cs-s4);
    overflow: auto;
    max-height: calc(85vh - 3.5rem);
  }

  .steps {
    margin: 0;
    padding-left: var(--cs-s5);
    display: flex;
    flex-direction: column;
    gap: var(--cs-s4);
  }

  .what {
    font-weight: 600;
  }

  .why,
  .prose {
    font-size: var(--cs-fs-sm);
    line-height: 1.55;
  }

  pre {
    margin: var(--cs-s1) 0;
    padding: var(--cs-s2) var(--cs-s3);
    background: var(--cs-bg-inset);
    border-radius: var(--cs-r2);
    overflow-x: auto;
  }

  code {
    padding: 0 2px;
    background: var(--cs-bg-inset);
    border-radius: var(--cs-r1);
  }

  .privacy {
    margin-top: var(--cs-s4);
    padding: var(--cs-s3);
    font-size: var(--cs-fs-sm);
    background: var(--cs-accent-soft);
    border-radius: var(--cs-r2);
  }

  table {
    width: 100%;
    border-collapse: collapse;
    font-size: var(--cs-fs-sm);
  }

  td {
    padding: var(--cs-s1) var(--cs-s2) var(--cs-s1) 0;
    vertical-align: baseline;
  }

  td:first-child {
    white-space: nowrap;
    width: 1%;
  }

  kbd {
    display: inline-block;
    padding: 0 4px;
    font-family: var(--cs-font-mono);
    font-size: var(--cs-fs-xs);
    background: var(--cs-bg-inset);
    border: 1px solid var(--cs-line-strong);
    border-bottom-width: 2px;
    border-radius: var(--cs-r1);
  }
</style>
