<script lang="ts">
  import { parseMarkdown } from "$lib/markdown";
  import type { TextData } from "$lib/common/types";
  import MarkdownInline from "./MarkdownInline.svelte";

  interface Props {
    data: TextData;
  }

  const { data }: Props = $props();

  const blocks = $derived(parseMarkdown(data.markdown));
</script>

<div class="block-text">
  {#each blocks as block, i (i)}
    {#if block.kind === "heading"}
      <svelte:element this={`h${Math.min(6, Math.max(1, block.level))}`}>
        <MarkdownInline tokens={block.inline} />
      </svelte:element>
    {:else if block.kind === "paragraph"}
      <p><MarkdownInline tokens={block.inline} /></p>
    {:else if block.kind === "quote"}
      <blockquote><MarkdownInline tokens={block.inline} /></blockquote>
    {:else if block.kind === "code"}
      <pre><code>{block.content}</code></pre>
    {:else if block.kind === "list"}
      <svelte:element this={block.ordered ? "ol" : "ul"}>
        {#each block.items as item, j (j)}
          <li><MarkdownInline tokens={item} /></li>
        {/each}
      </svelte:element>
    {/if}
  {/each}
</div>

<style>
  .block-text {
    line-height: 1.5;
  }

  .block-text pre {
    overflow-x: auto;
    padding: 0.75rem;
    border-radius: 4px;
    background: var(--color-surface-alt);
  }

  .block-text blockquote {
    margin: 0.5rem 0;
    padding-left: 0.75rem;
    border-left: 3px solid var(--color-border);
    color: var(--color-text-muted);
  }
</style>
