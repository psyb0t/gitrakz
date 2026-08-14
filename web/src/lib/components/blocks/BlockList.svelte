<script lang="ts">
  import type { ListData } from "$lib/common/types";

  interface Props {
    data: ListData;
  }

  const { data }: Props = $props();
</script>

<svelte:element this={data.ordered ? "ol" : "ul"} class="block-list">
  {#each data.items as item, i (i)}
    <li>{item}</li>
  {/each}
</svelte:element>

<style>
  .block-list {
    margin: 0.5rem 0;
    padding-left: 1.5rem;
    line-height: 1.5;
  }
</style>
