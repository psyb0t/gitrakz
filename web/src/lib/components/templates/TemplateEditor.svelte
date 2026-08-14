<script lang="ts">
  // Template editor — exposes the three parts of a template (form fields,
  // transform pipeline, display layout) per .git-trakz.md's "Templates
  // manager" spec. transform.params and layout.data are free-form objects
  // (TransformStep/Block schemas), so they're edited as raw JSON textareas
  // with inline parse-error surfacing rather than a bespoke form per
  // primitive/block type — this stays generic across the whole fixed
  // building-block library without hardcoding per-primitive UI.
  import { BLOCK_TYPES, FORM_FIELD_TYPES, TRANSFORM_PRIMITIVES } from "$lib/common/constants";
  import { EXPORT_FORMATS } from "$lib/common/types";
  import type {
    Block,
    ExportFormat,
    FormField,
    Template,
    TemplateInput,
    TransformStep,
  } from "$lib/common/types";
  import { untrack } from "svelte";
  import BlockRenderer from "$lib/components/blocks/BlockRenderer.svelte";

  interface Props {
    initial?: Template;
    saving?: boolean;
    saveLabel?: string;
    note?: string;
    onSave: (input: TemplateInput) => void;
    onCancel: () => void;
  }

  const {
    initial,
    saving = false,
    saveLabel = "Save",
    note = "",
    onSave,
    onCancel,
  }: Props = $props();

  interface FormFieldRow {
    name: string;
    label: string;
    type: string;
    required: boolean;
    defaultText: string;
  }

  interface TransformRow {
    primitive: string;
    paramsText: string;
    paramsError: string;
  }

  interface LayoutRow {
    type: string;
    dataText: string;
    dataError: string;
  }

  function defaultToText(value: unknown): string {
    if (value === undefined || value === null) {
      return "";
    }

    return typeof value === "string" ? value : JSON.stringify(value);
  }

  // `initial` only ever seeds the editor's own locally-owned state once.
  // TemplatesManager always routes through its "list" branch (which
  // unmounts this component) before pointing a new TemplateEditor at a
  // different template, so `initial` never changes under an
  // already-mounted instance. untrack makes that one-shot read explicit
  // instead of a rune dependency — re-deriving these rows from `initial`
  // on every change would fight the user's own in-progress edits.
  const seed = untrack(() => initial);

  let name = $state(seed?.name ?? "");
  let description = $state(seed?.description ?? "");
  let model = $state(seed?.model ?? "");
  let selectedExports = $state<ExportFormat[]>([...(seed?.exports ?? [])]);

  let formRows = $state<FormFieldRow[]>(
    (seed?.form ?? []).map((f) => ({
      name: f.name,
      label: f.label ?? "",
      type: f.type,
      required: f.required ?? false,
      defaultText: defaultToText(f.default),
    })),
  );

  let transformRows = $state<TransformRow[]>(
    (seed?.transform ?? []).map((t) => ({
      primitive: t.primitive,
      paramsText: t.params ? JSON.stringify(t.params, null, 2) : "",
      paramsError: "",
    })),
  );

  let layoutRows = $state<LayoutRow[]>(
    (seed?.layout ?? []).map((b) => ({
      type: b.type,
      dataText: JSON.stringify(b.data, null, 2),
      dataError: "",
    })),
  );

  let nameTouched = $state(false);
  const nameError = $derived(name.trim() === "" ? "Name is required." : "");

  function addFormRow(): void {
    formRows.push({
      name: "",
      label: "",
      type: FORM_FIELD_TYPES[0],
      required: false,
      defaultText: "",
    });
  }

  function removeFormRow(index: number): void {
    formRows.splice(index, 1);
  }

  function addTransformRow(): void {
    transformRows.push({
      primitive: TRANSFORM_PRIMITIVES[0],
      paramsText: "",
      paramsError: "",
    });
  }

  function removeTransformRow(index: number): void {
    transformRows.splice(index, 1);
  }

  function addLayoutRow(): void {
    layoutRows.push({ type: BLOCK_TYPES[0], dataText: "{}", dataError: "" });
  }

  function removeLayoutRow(index: number): void {
    layoutRows.splice(index, 1);
  }

  function toggleExportFormat(fmt: ExportFormat): void {
    selectedExports = selectedExports.includes(fmt)
      ? selectedExports.filter((f) => f !== fmt)
      : [...selectedExports, fmt];
  }

  /** Parses free-text template default values, falling back to the raw
   * string for non-JSON input ("2026-01-05" isn't valid JSON but is a
   * perfectly good default value) — never throws. */
  function looseParseDefault(text: string): unknown {
    if (text.trim() === "") {
      return undefined;
    }

    try {
      return JSON.parse(text);
    } catch (err) {
      console.debug("default value is not JSON, using raw string", {
        reason: "loose_default_parse_failed",
        err,
      });

      return text;
    }
  }

  function validate(): boolean {
    nameTouched = true;
    let ok = name.trim() !== "";

    for (const row of transformRows) {
      if (row.paramsText.trim() === "") {
        row.paramsError = "";
        continue;
      }

      try {
        JSON.parse(row.paramsText);
        row.paramsError = "";
      } catch (err) {
        row.paramsError = err instanceof Error ? err.message : "invalid JSON";
        ok = false;
      }
    }

    for (const row of layoutRows) {
      try {
        JSON.parse(row.dataText.trim() === "" ? "{}" : row.dataText);
        row.dataError = "";
      } catch (err) {
        row.dataError = err instanceof Error ? err.message : "invalid JSON";
        ok = false;
      }
    }

    return ok;
  }

  function submit(): void {
    if (!validate()) {
      return;
    }

    const form: FormField[] = formRows.map((r) => ({
      name: r.name,
      label: r.label || null,
      type: r.type,
      required: r.required,
      default: looseParseDefault(r.defaultText),
    }));

    const transform: TransformStep[] = transformRows.map((r) => ({
      primitive: r.primitive,
      params:
        r.paramsText.trim() === ""
          ? undefined
          : (JSON.parse(r.paramsText) as Record<string, unknown>),
    }));

    const layout: Block[] = layoutRows.map((r) => ({
      type: r.type,
      data: JSON.parse(
        r.dataText.trim() === "" ? "{}" : r.dataText,
      ) as Record<string, unknown>,
    }));

    onSave({
      name: name.trim(),
      description: description.trim() === "" ? null : description.trim(),
      form,
      transform,
      layout,
      exports: selectedExports,
      model: model.trim() === "" ? null : model.trim(),
    });
  }

  let showPreview = $state(false);

  const previewDocument = $derived.by(() => {
    try {
      return layoutRows.map((r) => ({
        type: r.type,
        data: JSON.parse(
          r.dataText.trim() === "" ? "{}" : r.dataText,
        ) as Record<string, unknown>,
      }));
    } catch (err) {
      console.debug("layout preview parse failed", {
        reason: "invalid_layout_json",
        err,
      });

      return [];
    }
  });
</script>

<div class="template-editor">
  {#if note}<p class="note">{note}</p>{/if}

  <div class="field">
    <label for="tmpl-name">Name</label>
    <input id="tmpl-name" type="text" bind:value={name} onblur={() => (nameTouched = true)} />
    {#if nameTouched && nameError}<span class="error">{nameError}</span>{/if}
  </div>

  <div class="field">
    <label for="tmpl-desc">Description</label>
    <textarea id="tmpl-desc" rows="2" bind:value={description}></textarea>
  </div>

  <div class="field">
    <label for="tmpl-model">Model (optional — only for an LLM-backed layout block)</label>
    <input id="tmpl-model" type="text" bind:value={model} />
  </div>

  <fieldset>
    <legend>Form fields</legend>
    {#each formRows as row, i (i)}
      <div class="row">
        <input type="text" placeholder="name" bind:value={row.name} />
        <input type="text" placeholder="label" bind:value={row.label} />
        <select bind:value={row.type}>
          {#each FORM_FIELD_TYPES as t (t)}
            <option value={t}>{t}</option>
          {/each}
        </select>
        <label class="checkbox">
          <input type="checkbox" bind:checked={row.required} /> required
        </label>
        <input type="text" placeholder="default" bind:value={row.defaultText} />
        <button type="button" class="btn btn-danger" onclick={() => removeFormRow(i)}>Remove</button>
      </div>
    {/each}
    <button type="button" class="btn" onclick={addFormRow}>Add form field</button>
  </fieldset>

  <fieldset>
    <legend>Transform pipeline</legend>
    {#each transformRows as row, i (i)}
      <div class="row row-wide">
        <select bind:value={row.primitive}>
          {#each TRANSFORM_PRIMITIVES as p (p)}
            <option value={p}>{p}</option>
          {/each}
        </select>
        <textarea rows="2" placeholder="params (JSON)" bind:value={row.paramsText}></textarea>
        <button type="button" class="btn btn-danger" onclick={() => removeTransformRow(i)}>Remove</button>
      </div>
      {#if row.paramsError}<span class="error">{row.paramsError}</span>{/if}
    {/each}
    <button type="button" class="btn" onclick={addTransformRow}>Add transform step</button>
  </fieldset>

  <fieldset>
    <legend>Display layout</legend>
    {#each layoutRows as row, i (i)}
      <div class="row row-wide">
        <select bind:value={row.type}>
          {#each BLOCK_TYPES as t (t)}
            <option value={t}>{t}</option>
          {/each}
        </select>
        <textarea rows="3" placeholder="data (JSON)" bind:value={row.dataText}></textarea>
        <button type="button" class="btn btn-danger" onclick={() => removeLayoutRow(i)}>Remove</button>
      </div>
      {#if row.dataError}<span class="error">{row.dataError}</span>{/if}
    {/each}
    <button type="button" class="btn" onclick={addLayoutRow}>Add layout block</button>

    <button type="button" class="btn" onclick={() => (showPreview = !showPreview)}>
      {showPreview ? "Hide preview" : "Preview layout"}
    </button>
    {#if showPreview}
      <div class="preview card">
        <BlockRenderer document={previewDocument} />
      </div>
    {/if}
  </fieldset>

  <fieldset>
    <legend>Exports</legend>
    {#each EXPORT_FORMATS as fmt (fmt)}
      <label class="checkbox">
        <input
          type="checkbox"
          checked={selectedExports.includes(fmt)}
          onchange={() => toggleExportFormat(fmt)}
        />
        {fmt}
      </label>
    {/each}
  </fieldset>

  <div class="actions">
    <button type="button" class="btn btn-primary" onclick={submit} disabled={saving}>
      {saving ? "Saving…" : saveLabel}
    </button>
    <button type="button" class="btn" onclick={onCancel} disabled={saving}>Cancel</button>
  </div>
</div>

<style>
  .template-editor {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    max-width: 48rem;
  }

  .note {
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--color-primary);
    border-radius: var(--radius);
    color: var(--color-primary);
    font-size: 0.85rem;
  }

  fieldset {
    border: 1px solid var(--color-border);
    border-radius: var(--radius);
    padding: 0.75rem;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  legend {
    padding: 0 0.4rem;
    font-size: 0.85rem;
    color: var(--color-text-muted);
  }

  .row {
    display: grid;
    grid-template-columns: 1fr 1fr 8rem auto 1fr auto;
    gap: 0.4rem;
    align-items: center;
  }

  .row-wide {
    grid-template-columns: 10rem 1fr auto;
    align-items: start;
  }

  .row input,
  .row select,
  .row textarea {
    padding: 0.35rem 0.5rem;
    border-radius: var(--radius);
    border: 1px solid var(--color-border);
    background: var(--color-surface);
    font-size: 0.85rem;
  }

  .checkbox {
    display: flex;
    align-items: center;
    gap: 0.3rem;
    font-size: 0.85rem;
  }

  .error {
    color: var(--color-danger);
    font-size: 0.75rem;
  }

  .preview {
    margin-top: 0.5rem;
  }

  .actions {
    display: flex;
    gap: 0.5rem;
    margin-top: 0.5rem;
  }
</style>
