<script lang="ts">
  // Dispatcher: switches on block.type, one renderer component per display
  // building block (per api/api.yml's Block/Document schema + the fixed
  // display-block library in .git-trakz.md). Block.data comes off the wire
  // as an untyped object; asData narrows it to the shape the matched type
  // guarantees, mirroring the Go side's Block.AsHeading()/AsText()/etc.
  // per-type accessors — the cast is the one, documented boundary where an
  // untyped payload becomes a typed one.
  import {
    BLOCK_TYPE_CHART,
    BLOCK_TYPE_CODE,
    BLOCK_TYPE_HEADING,
    BLOCK_TYPE_KEYVALUE,
    BLOCK_TYPE_LIST,
    BLOCK_TYPE_METRIC,
    BLOCK_TYPE_TABLE,
    BLOCK_TYPE_TEXT,
  } from "$lib/common/constants";
  import type {
    Block,
    ChartData,
    CodeData,
    Document,
    HeadingData,
    KeyValueData,
    ListData,
    MetricData,
    TableData,
    TextData,
  } from "$lib/common/types";
  import BlockChart from "./BlockChart.svelte";
  import BlockCode from "./BlockCode.svelte";
  import BlockHeading from "./BlockHeading.svelte";
  import BlockKeyValue from "./BlockKeyValue.svelte";
  import BlockList from "./BlockList.svelte";
  import BlockMetric from "./BlockMetric.svelte";
  import BlockTable from "./BlockTable.svelte";
  import BlockText from "./BlockText.svelte";

  interface Props {
    document: Document;
  }

  const { document: doc }: Props = $props();

  function asData<T>(block: Block): T {
    return block.data as unknown as T;
  }

  const KNOWN_BLOCK_TYPES = new Set<string>([
    BLOCK_TYPE_HEADING,
    BLOCK_TYPE_TEXT,
    BLOCK_TYPE_LIST,
    BLOCK_TYPE_TABLE,
    BLOCK_TYPE_KEYVALUE,
    BLOCK_TYPE_METRIC,
    BLOCK_TYPE_CODE,
    BLOCK_TYPE_CHART,
  ]);

  // Unknown block types degrade visibly in the markup below AND get logged
  // here — a shape a future block-type addition on the server produces
  // before the frontend knows about it should never disappear silently.
  $effect(() => {
    for (const block of doc) {
      if (!KNOWN_BLOCK_TYPES.has(block.type)) {
        console.warn("unrecognized block type", { type: block.type });
      }
    }
  });
</script>

<div class="block-renderer">
  {#each doc as block, i (i)}
    {#if block.type === BLOCK_TYPE_HEADING}
      <BlockHeading data={asData<HeadingData>(block)} />
    {:else if block.type === BLOCK_TYPE_TEXT}
      <BlockText data={asData<TextData>(block)} />
    {:else if block.type === BLOCK_TYPE_LIST}
      <BlockList data={asData<ListData>(block)} />
    {:else if block.type === BLOCK_TYPE_TABLE}
      <BlockTable data={asData<TableData>(block)} />
    {:else if block.type === BLOCK_TYPE_KEYVALUE}
      <BlockKeyValue data={asData<KeyValueData>(block)} />
    {:else if block.type === BLOCK_TYPE_METRIC}
      <BlockMetric data={asData<MetricData>(block)} />
    {:else if block.type === BLOCK_TYPE_CODE}
      <BlockCode data={asData<CodeData>(block)} />
    {:else if block.type === BLOCK_TYPE_CHART}
      <BlockChart data={asData<ChartData>(block)} />
    {:else}
      <!-- Unknown block type — degrade visibly instead of dropping it
           silently, and log so an unexpected server payload shape shows up
           in the console. -->
      <div class="unknown-block">
        <p>Unrecognized block type: <code>{block.type}</code></p>
        <pre>{JSON.stringify(block.data, null, 2)}</pre>
      </div>
    {/if}
  {/each}
</div>

<style>
  .block-renderer {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  .unknown-block {
    padding: 0.5rem 0.75rem;
    border: 1px dashed var(--color-danger);
    border-radius: var(--radius);
    color: var(--color-danger);
    font-size: 0.85rem;
  }

  .unknown-block pre {
    white-space: pre-wrap;
    word-break: break-word;
  }
</style>
