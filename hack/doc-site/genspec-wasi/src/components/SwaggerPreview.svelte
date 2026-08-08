<script lang="ts">
  import { usePlayground } from '../lib/store.svelte';

  // Swagger UI, rendering the document the scan just produced.
  //
  // Loaded on first activation and never before: it is 1.5 MB of JavaScript and 176 KB of CSS, which
  // is affordable against a 20 MB artifact and not affordable for a reader who only wanted the spec.

  const playground = usePlayground();

  // Whether to pull Swagger UI towards the app's theme, or show it as a consumer would see it.
  //
  // Only ever asked in a dark theme: in a light one Swagger UI already matches, and there is nothing
  // to reconcile.
  let adapt = $state(true);

  let host: HTMLDivElement | null = $state(null);
  let phase = $state<'idle' | 'loading' | 'ready' | 'failed'>('idle');
  let failure = $state('');

  type Render = (typeof import('swagger-ui-dist/swagger-ui-es-bundle.js'))['default'];
  let bundle: Render | null = null;

  // A document with no paths is not a broken document, and Swagger UI cannot tell the difference:
  // it renders an empty page that reads as a failure. dockerctl produces exactly this - a generated
  // client whose models are annotated and whose routes are not.
  const routes = $derived.by(() => {
    const doc = playground.spec as { paths?: Record<string, unknown> } | null;

    return Object.keys(doc?.paths ?? {}).length;
  });

  const renderable = $derived(playground.spec !== null && routes > 0);

  async function load() {
    if (bundle || phase === 'loading') {
      return;
    }
    phase = 'loading';
    try {
      const [mod] = await Promise.all([
        import('swagger-ui-dist/swagger-ui-es-bundle.js'),
        import('swagger-ui-dist/swagger-ui.css'),
      ]);
      bundle = mod.default;
      phase = 'ready';
    } catch (err) {
      phase = 'failed';
      failure = err instanceof Error ? err.message : String(err);
    }
  }

  $effect(() => {
    if (renderable) {
      void load();
    }
  });

  // Re-rendered rather than updated in place: Swagger UI has no "here is a new document" call, and
  // the spec changes only on a rescan, which is not often enough for the rebuild to be felt.
  $effect(() => {
    const spec = playground.spec;
    if (phase !== 'ready' || !host || !renderable) {
      return;
    }

    host.replaceChildren();
    bundle?.({
      domNode: host,
      spec,

      // Nothing here may reach the network, and two of Swagger UI's defaults would.
      //
      // validatorUrl defaults to validator.swagger.io, which it POSTs the whole document to in order
      // to draw a badge. That would send the user's source-derived spec off their machine and make a
      // liar of the line in the status bar.
      validatorUrl: null,
      // "Try it out" fires real requests at whatever host the document names - which for a spec
      // written from source is a host the reader has never agreed to talk to.
      supportedSubmitMethods: [],
      tryItOutEnabled: false,

      deepLinking: false,
      docExpansion: 'list',
      defaultModelsExpandDepth: 1,
    });
  });
</script>

<div class="preview">
  {#if playground.error}
    <p class="note cs-muted">The scan did not finish, so there is nothing to render.</p>
  {:else if playground.spec === null}
    <p class="note cs-muted">Nothing scanned yet.</p>
  {:else if routes === 0}
    <div class="note cs-muted">
      <p class="title">This document describes no paths</p>
      <p>
        Swagger UI renders operations, and the scan found none — which is what a package of annotated
        models produces, and is not a failure. The <strong>Spec</strong> tab has the definitions.
      </p>
    </div>
  {:else if phase === 'failed'}
    <div class="note">
      <p class="title cs-error">Swagger UI did not load</p>
      <pre class="detail">{failure}</pre>
    </div>
  {:else if phase === 'loading'}
    <p class="note cs-muted">Loading Swagger UI…</p>
  {/if}

  {#if phase === 'ready' && renderable}
    <div class="strip">
      <span class="cs-faint">Rendered with Swagger UI</span>
      <button
        class="cs-btn-quiet"
        aria-pressed={!adapt}
        onclick={() => (adapt = !adapt)}
        title="Swagger UI has one appearance and it is light; this pulls it towards the app's theme"
      >{adapt ? 'show as published' : 'match the theme'}</button>
    </div>
  {/if}

  <!-- Always present so the effect above has somewhere to render into; empty until it does. -->
  <div class="ui" class:on={phase === 'ready' && renderable} class:adapt bind:this={host}></div>
</div>

<style>
  .preview {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-height: 0;
    overflow: auto;
    background: var(--cs-bg);
  }

  .note {
    padding: var(--cs-s4);
    font-size: var(--cs-fs-sm);
  }

  .note .title {
    font-weight: 600;
    margin-bottom: var(--cs-s1);
  }

  .detail {
    margin-top: var(--cs-s2);
    padding: var(--cs-s2);
    background: var(--cs-error-soft);
    border-radius: var(--cs-r2);
    white-space: pre-wrap;
  }

  .strip {
    display: flex;
    align-items: center;
    gap: var(--cs-s2);
    flex: none;
    padding: var(--cs-s1) var(--cs-s3);
    font-size: var(--cs-fs-xs);
    background: var(--cs-bg-sunken);
    border-bottom: 1px solid var(--cs-line);
  }

  .strip button {
    margin-left: auto;
    font-size: var(--cs-fs-xs);
    height: 1.4rem;
  }

  .ui {
    display: none;
  }

  /* Swagger UI is a light interface and has no dark mode. Its own surface, always, so "as published"
     means exactly that. */
  .ui.on {
    display: block;
    background: #fff;
    color-scheme: light;
  }

  /* Pulled towards a dark theme by inverting and rotating the hue back.
   *
   * A heuristic, and the reason the strip above offers a way out of it: the pair is chosen because
   * hue-rotate(180deg) undoes what invert() does to hue, so blue stays blue - which matters here,
   * where the method badges carry meaning in their colour. 0.92 rather than a full invert keeps the
   * whites off pure black, which is where an inverted document looks harshest.
   *
   * Light theme is untouched: there Swagger UI already matches, and there is nothing to reconcile. */
  :global(.cs-root[data-theme='dark']) .ui.on.adapt {
    /* The background stays WHITE and is inverted along with everything else. Setting a dark one here
       was the obvious move and the wrong one: the filter applies to this element's own background
       too, so a near-black surface came back out near-white, and Swagger UI's inverted (now light)
       text sat on it washed out. */
    filter: invert(0.92) hue-rotate(180deg);
  }

  /* Media carries no theme and must not be inverted with the page around it. */
  :global(.cs-root[data-theme='dark']) .ui.on.adapt :global(img),
  :global(.cs-root[data-theme='dark']) .ui.on.adapt :global(svg image) {
    filter: invert(1) hue-rotate(180deg);
  }
</style>
