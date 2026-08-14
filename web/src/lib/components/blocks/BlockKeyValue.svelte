<script lang="ts">
  import type { KeyValueData } from "$lib/common/types";

  interface Props {
    data: KeyValueData;
  }

  const { data }: Props = $props();
</script>

<dl class="block-keyvalue">
  {#each data.pairs as pair, i (i)}
    <div class="row">
      <dt>{pair.key}</dt>
      <dd>{pair.value}</dd>
    </div>
  {/each}
</dl>

<style>
  .block-keyvalue {
    margin: 0.5rem 0;
    display: grid;
    row-gap: 0.35rem;
  }

  .row {
    display: grid;
    grid-template-columns: minmax(8rem, 14rem) 1fr;
    column-gap: 0.75rem;
  }

  dt {
    color: var(--color-text-muted);
    font-weight: 500;
  }

  dd {
    margin: 0;
  }
</style>
