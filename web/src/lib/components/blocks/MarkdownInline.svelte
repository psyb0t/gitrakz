<script lang="ts">
  import type { InlineToken } from "$lib/markdown";

  interface Props {
    tokens: InlineToken[];
  }

  const { tokens }: Props = $props();
</script>

<!--
  Every token is rendered through Svelte text interpolation (auto-escaped)
  or a typed attribute (href) — never {@html} — so there is no raw-HTML
  injection surface here regardless of what the markdown source contains.
-->
{#each tokens as token, i (i)}
  {#if token.kind === "text"}{token.text}{:else if token.kind === "bold"}<strong>{token.text}</strong
    >{:else if token.kind === "italic"}<em>{token.text}</em
    >{:else if token.kind === "code"}<code>{token.text}</code
    >{:else if token.kind === "link"}<a href={token.href} target="_blank" rel="noopener noreferrer"
      >{token.text}</a
    >{/if}
{/each}
