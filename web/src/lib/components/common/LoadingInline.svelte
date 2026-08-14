<script lang="ts">
  interface Props {
    label?: string;
  }

  const { label = "Loading…" }: Props = $props();
</script>

<div class="loading-inline muted">{label}</div>

<style>
  .loading-inline {
    padding: 0.5rem 0;
    font-size: 0.85rem;
  }
</style>
