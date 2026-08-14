<script lang="ts">
  // A lightweight inline SVG bar/line chart — no charting library, per
  // "keep the bundle self-contained". `kind: "line"` renders a polyline,
  // anything else (including "bar") renders bars.
  import type { ChartData } from "$lib/common/types";

  interface Props {
    data: ChartData;
  }

  const { data }: Props = $props();

  const WIDTH = 480;
  const HEIGHT = 220;
  const PADDING = 28;

  const maxValue = $derived(Math.max(1, 0, ...data.values));
  const isLine = $derived(data.kind === "line");

  interface Point {
    x: number;
    y: number;
    value: number;
    label: string;
  }

  const points = $derived<Point[]>(
    data.values.map((value, i) => {
      const x =
        data.values.length <= 1
          ? WIDTH / 2
          : PADDING + (i * (WIDTH - PADDING * 2)) / (data.values.length - 1);
      const y = HEIGHT - PADDING - (value / maxValue) * (HEIGHT - PADDING * 2);

      return { x, y, value, label: data.labels[i] ?? "" };
    }),
  );

  const barWidth = $derived(
    data.values.length > 0 ? (WIDTH - PADDING * 2) / data.values.length : 0,
  );

  function barHeight(value: number): number {
    return (value / maxValue) * (HEIGHT - PADDING * 2);
  }
</script>

<figure class="block-chart">
  <svg viewBox={`0 0 ${WIDTH} ${HEIGHT}`} role="img" aria-label={`${data.kind} chart`}>
    <line
      x1={PADDING}
      y1={HEIGHT - PADDING}
      x2={WIDTH - PADDING}
      y2={HEIGHT - PADDING}
      class="axis"
    />
    {#if isLine}
      <polyline class="line" points={points.map((p) => `${p.x},${p.y}`).join(" ")} />
      {#each points as p, i (i)}
        <circle cx={p.x} cy={p.y} r="3" class="point" />
      {/each}
    {:else}
      {#each data.values as value, i (i)}
        <rect
          x={PADDING + i * barWidth + barWidth * 0.15}
          y={HEIGHT - PADDING - barHeight(value)}
          width={barWidth * 0.7}
          height={barHeight(value)}
          class="bar"
        />
      {/each}
    {/if}
  </svg>
  <ul class="legend">
    {#each points as p, i (i)}
      <li><span class="swatch"></span>{p.label}: {p.value}</li>
    {/each}
  </ul>
</figure>

<style>
  .block-chart {
    margin: 0.5rem 0;
  }

  svg {
    width: 100%;
    max-width: 32rem;
    height: auto;
  }

  .axis {
    stroke: var(--color-border);
    stroke-width: 1;
  }

  .bar {
    fill: var(--color-primary);
  }

  .line {
    fill: none;
    stroke: var(--color-primary);
    stroke-width: 2;
  }

  .point {
    fill: var(--color-primary);
  }

  .legend {
    list-style: none;
    margin: 0.5rem 0 0;
    padding: 0;
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem 1rem;
    font-size: 0.8rem;
    color: var(--color-text-muted);
  }

  .swatch {
    display: inline-block;
    width: 0.6rem;
    height: 0.6rem;
    border-radius: 2px;
    background: var(--color-primary);
    margin-right: 0.3rem;
    vertical-align: middle;
  }
</style>
