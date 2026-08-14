<script lang="ts">
  import type { CodeData } from "$lib/common/types";

  interface Props {
    data: CodeData;
  }

  const { data }: Props = $props();
</script>

<div class="block-code">
  {#if data.lang}<div class="lang">{data.lang}</div>{/if}
  <pre><code>{data.content}</code></pre>
</div>

<style>
  .block-code {
    margin: 0.5rem 0;
    border: 1px solid var(--color-border);
    border-radius: var(--radius);
    overflow: hidden;
  }

  .lang {
    padding: 0.25rem 0.6rem;
    font-size: 0.75rem;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--color-text-muted);
    background: var(--color-surface-alt);
    border-bottom: 1px solid var(--color-border);
  }

  pre {
    margin: 0;
    padding: 0.75rem;
    overflow-x: auto;
    background: var(--color-surface-alt);
    font-family:
      ui-monospace, SFMono-Regular, Menlo, Consolas, "Liberation Mono",
      monospace;
    font-size: 0.85rem;
  }
</style>
