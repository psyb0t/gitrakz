<script lang="ts">
  import type { HeadingData } from "$lib/common/types";

  interface Props {
    data: HeadingData;
  }

  const { data }: Props = $props();

  const MIN_LEVEL = 1;
  const MAX_LEVEL = 6;

  const tag = $derived(
    `h${Math.min(MAX_LEVEL, Math.max(MIN_LEVEL, Math.trunc(data.level) || MIN_LEVEL))}`,
  );
</script>

<svelte:element this={tag} class="block-heading">{data.text}</svelte:element>

<style>
  .block-heading {
    margin: 1.25rem 0 0.5rem;
    line-height: 1.25;
  }
</style>
