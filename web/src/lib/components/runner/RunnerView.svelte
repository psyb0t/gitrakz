<script lang="ts">
  // Range-select + Template runner. "Range" is the sidebar's Filter
  // (owner/repo/type/from/to — the only shape RunRequest.filter accepts) —
  // row-selection on the Timeline view is a convenience that derives a
  // from/to spanning the checked rows and writes it into that same filter,
  // per the "shift-click-select rows -> a selection bar shows N events"
  // spec, implemented here as checkbox multi-select since drag-select
  // needs no additional API surface beyond the same Filter shape.
  import { tick } from "svelte";
  import type { Document, ExportFormat, Template } from "$lib/common/types";
  import { exportDocument, listTemplates, runTemplate } from "$lib/resources";
  import { downloadBlob } from "$lib/api";
  import { getFilterState, setRange, toAPIFilter } from "$lib/stores/filter.svelte";
  import { clearSelection, getSelected, selectionRange } from "$lib/stores/selection.svelte";
  import { pushToast, reportError } from "$lib/stores/toast.svelte";
  import BlockRenderer from "$lib/components/blocks/BlockRenderer.svelte";
  import LoadingInline from "$lib/components/common/LoadingInline.svelte";

  const filter = getFilterState();

  let templates = $state<Template[]>([]);
  let templatesLoading = $state(false);
  let selectedTemplateID = $state("");
  let formValues = $state<Record<string, unknown>>({});

  let running = $state(false);
  let resultDocument = $state<Document | undefined>(undefined);
  let exportingFormat = $state<ExportFormat | undefined>(undefined);

  async function loadTemplates(): Promise<void> {
    templatesLoading = true;
    try {
      templates = await listTemplates();
      await applyDeepLink();
    } catch (err) {
      reportError("Failed to load templates", err);
    } finally {
      templatesLoading = false;
    }
  }

  // Deep-link: /?tpl=<id> pre-selects a template by setting the Svelte state
  // directly (so it works without a DOM change event), and &run=1 also runs
  // it once the selection + form-default effects have settled. Makes a
  // template run shareable as a URL.
  async function applyDeepLink(): Promise<void> {
    const params = new URLSearchParams(window.location.search);
    const tpl = params.get("tpl");
    if (tpl === null || !templates.some((t) => t.id === tpl)) {
      return;
    }

    selectedTemplateID = tpl;

    if (params.get("run") === "1") {
      await tick();
      await run();
    }
  }

  $effect(() => {
    void loadTemplates();
  });

  const selected = $derived(getSelected());
  const selectedCount = $derived(selected.size);

  const selectedTemplate = $derived(
    templates.find((t) => t.id === selectedTemplateID),
  );

  // Reset form values to the template's declared defaults whenever the
  // selected template changes.
  $effect(() => {
    const tmpl = selectedTemplate;
    const next: Record<string, unknown> = {};
    if (tmpl) {
      for (const field of tmpl.form) {
        next[field.name] = field.default ?? defaultForType(field.type);
      }
    }
    formValues = next;
  });

  function defaultForType(type: string): unknown {
    switch (type) {
      case "boolean":
        return false;
      case "number":
        return 0;
      default:
        return "";
    }
  }

  function useSelectionAsRange(): void {
    const range = selectionRange();
    if (!range) {
      return;
    }
    setRange(range.from, range.to);
  }

  async function run(): Promise<void> {
    if (!selectedTemplate) {
      return;
    }

    running = true;
    resultDocument = undefined;
    try {
      resultDocument = await runTemplate({
        templateId: selectedTemplate.id,
        filter: toAPIFilter(filter),
        formValues,
      });
      pushToast("success", "Template run complete");
    } catch (err) {
      reportError("Failed to run template", err);
    } finally {
      running = false;
    }
  }

  async function doExport(format: ExportFormat): Promise<void> {
    if (!selectedTemplate) {
      return;
    }

    exportingFormat = format;
    try {
      const { blob } = await exportDocument(
        resultDocument
          ? { document: resultDocument, format }
          : {
              templateId: selectedTemplate.id,
              filter: toAPIFilter(filter),
              formValues,
              format,
            },
      );
      const stamp = new Date().toISOString().replace(/[:.]/g, "-");
      downloadBlob(blob, `${selectedTemplate.name}-${stamp}.${format}`);
    } catch (err) {
      reportError(`Failed to export as ${format}`, err);
    } finally {
      exportingFormat = undefined;
    }
  }

  function onFieldInput(name: string, type: string, raw: string): void {
    if (type === "number") {
      const n = Number(raw);
      formValues[name] = Number.isFinite(n) ? n : 0;

      return;
    }

    formValues[name] = raw;
  }
</script>

<div class="runner">
  <div class="selection-bar card">
    <span>
      {#if selectedCount > 0}
        <strong>{selectedCount}</strong> event(s) selected on the timeline.
        <button type="button" class="btn" onclick={useSelectionAsRange}>Use as range</button>
        <button type="button" class="btn" onclick={clearSelection}>Clear selection</button>
      {:else}
        No rows selected — using the sidebar filter as the range
        ({filter.owner || "all owners"}{filter.repo ? `/${filter.repo}` : ""}).
      {/if}
    </span>
  </div>

  <div class="field">
    <label for="runner-template">Template</label>
    <select id="runner-template" bind:value={selectedTemplateID}>
      <option value="">Select a template…</option>
      {#each templates as tmpl (tmpl.id)}
        <option value={tmpl.id}>{tmpl.name}</option>
      {/each}
    </select>
    {#if templatesLoading}<LoadingInline label="Loading templates…" />{/if}
  </div>

  {#if selectedTemplate}
    {#if selectedTemplate.form.length > 0}
      <fieldset class="form-fields">
        <legend>{selectedTemplate.name} — form</legend>
        {#each selectedTemplate.form as field (field.name)}
          <div class="field">
            <label for={`runner-field-${field.name}`}>
              {field.label || field.name}{field.required ? " *" : ""}
            </label>
            {#if field.type === "boolean"}
              <input
                id={`runner-field-${field.name}`}
                type="checkbox"
                checked={Boolean(formValues[field.name])}
                onchange={(e) => (formValues[field.name] = e.currentTarget.checked)}
              />
            {:else if field.type === "number"}
              <input
                id={`runner-field-${field.name}`}
                type="number"
                value={String(formValues[field.name] ?? "")}
                oninput={(e) => onFieldInput(field.name, field.type, e.currentTarget.value)}
              />
            {:else if field.type === "date"}
              <input
                id={`runner-field-${field.name}`}
                type="date"
                value={String(formValues[field.name] ?? "")}
                oninput={(e) => onFieldInput(field.name, field.type, e.currentTarget.value)}
              />
            {:else}
              <input
                id={`runner-field-${field.name}`}
                type="text"
                value={String(formValues[field.name] ?? "")}
                oninput={(e) => onFieldInput(field.name, field.type, e.currentTarget.value)}
              />
            {/if}
          </div>
        {/each}
      </fieldset>
    {/if}

    <div class="actions">
      <button type="button" class="btn btn-primary" onclick={run} disabled={running}>
        {running ? "Running…" : "Run"}
      </button>
      {#each selectedTemplate.exports as fmt (fmt)}
        <button
          type="button"
          class="btn"
          onclick={() => doExport(fmt)}
          disabled={exportingFormat !== undefined}
        >
          {exportingFormat === fmt ? "Exporting…" : `Export ${fmt.toUpperCase()}`}
        </button>
      {/each}
    </div>
  {/if}

  {#if running}
    <LoadingInline label="Running template…" />
  {:else if resultDocument}
    <div class="result card">
      <BlockRenderer document={resultDocument} />
    </div>
  {/if}
</div>

<style>
  .runner {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .selection-bar {
    font-size: 0.85rem;
  }

  .form-fields {
    border: 1px solid var(--color-border);
    border-radius: var(--radius);
    padding: 0.75rem;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  legend {
    font-size: 0.85rem;
    color: var(--color-text-muted);
  }

  .actions {
    display: flex;
    gap: 0.5rem;
    flex-wrap: wrap;
  }

  .result {
    margin-top: 0.5rem;
  }
</style>
