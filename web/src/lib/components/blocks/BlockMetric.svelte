<script lang="ts">
  import type { MetricData } from "$lib/common/types";

  interface Props {
    data: MetricData;
  }

  const { data }: Props = $props();
</script>

<div class="block-metric">
  <div class="value">{data.value}{#if data.unit}<span class="unit">{data.unit}</span>{/if}</div>
  <div class="label">{data.label}</div>
</div>

<style>
  .block-metric {
    display: inline-flex;
    flex-direction: column;
    gap: 0.15rem;
    padding: 0.75rem 1rem;
    border: 1px solid var(--color-border);
    border-radius: var(--radius);
    background: var(--color-surface);
    min-width: 8rem;
  }

  .value {
    font-size: 1.75rem;
    font-weight: 700;
    line-height: 1;
  }

  .unit {
    margin-left: 0.25rem;
    font-size: 1rem;
    font-weight: 400;
    color: var(--color-text-muted);
  }

  .label {
    font-size: 0.85rem;
    color: var(--color-text-muted);
  }
</style>
