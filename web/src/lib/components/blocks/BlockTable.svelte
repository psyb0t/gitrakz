<script lang="ts">
  import type { TableData } from "$lib/common/types";

  interface Props {
    data: TableData;
  }

  const { data }: Props = $props();
</script>

<div class="block-table-wrap">
  <table class="block-table">
    <thead>
      <tr>
        {#each data.columns as column, i (i)}
          <th>{column}</th>
        {/each}
      </tr>
    </thead>
    <tbody>
      {#each data.rows as row, i (i)}
        <tr>
          {#each row as cell, j (j)}
            <td>{cell}</td>
          {/each}
        </tr>
      {/each}
    </tbody>
    {#if data.footer && data.footer.length > 0}
      <tfoot>
        <tr>
          {#each data.footer as cell, i (i)}
            <td>{cell}</td>
          {/each}
        </tr>
      </tfoot>
    {/if}
  </table>
</div>

<style>
  .block-table-wrap {
    overflow-x: auto;
    margin: 0.5rem 0;
  }

  .block-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.9rem;
  }

  .block-table th,
  .block-table td {
    padding: 0.4rem 0.6rem;
    border-bottom: 1px solid var(--color-border);
    text-align: left;
    white-space: nowrap;
  }

  .block-table thead th {
    font-weight: 600;
    background: var(--color-surface-alt);
  }

  .block-table tfoot td {
    font-weight: 600;
    border-top: 2px solid var(--color-border);
  }
</style>
