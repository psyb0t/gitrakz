<script lang="ts">
  // Templates manager. The Template API response has no `builtin` flag (see
  // api/api.yml's Template schema — allOf TemplateInput + {id}, nothing
  // else), so this UI can't pre-detect a built-in client-side. Per the task
  // spec: attempt the edit/delete normally, and when the server rejects it
  // (PERMISSION_DENIED — internal/pkg/http/server/handlers_templates.go
  // returns that for a builtin id), surface the rejection and offer
  // clone-on-edit (re-submit as a new custom template via POST) as the
  // recovery path instead of guessing ahead of time.
  import {
    createTemplate,
    deleteTemplate,
    generateTemplate,
    listTemplates,
    updateTemplate,
  } from "$lib/resources";
  import { ApiError } from "$lib/common/errors";
  import type { Template, TemplateInput } from "$lib/common/types";
  import { pushToast, reportError } from "$lib/stores/toast.svelte";
  import LoadingInline from "$lib/components/common/LoadingInline.svelte";
  import TemplateEditor from "./TemplateEditor.svelte";

  type Mode = "list" | "create" | "edit" | "clone";

  let templates = $state<Template[]>([]);
  let loading = $state(false);
  let mode = $state<Mode>("list");
  let editingTemplate = $state<Template | undefined>(undefined);
  let editingID = $state<string | undefined>(undefined);
  let editorNote = $state("");
  let saving = $state(false);

  let generatePrompt = $state("");
  let generating = $state(false);

  async function load(): Promise<void> {
    loading = true;
    try {
      templates = await listTemplates();
    } catch (err) {
      reportError("Failed to load templates", err);
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    void load();
  });

  function startCreate(): void {
    editingTemplate = undefined;
    editingID = undefined;
    editorNote = "";
    mode = "create";
  }

  function startEdit(tmpl: Template): void {
    editingTemplate = tmpl;
    editingID = tmpl.id;
    editorNote = "";
    mode = "edit";
  }

  function cancelEdit(): void {
    mode = "list";
    editingTemplate = undefined;
    editingID = undefined;
    editorNote = "";
  }

  function isBuiltinRejection(err: unknown): boolean {
    return err instanceof ApiError && err.status === 403;
  }

  async function saveTemplate(input: TemplateInput): Promise<void> {
    saving = true;
    try {
      if (mode === "edit" && editingID) {
        const updated = await updateTemplate(editingID, input);
        pushToast("success", `Saved "${updated.name}"`);
        mode = "list";
        editingTemplate = undefined;
        editingID = undefined;
        await load();

        return;
      }

      const created = await createTemplate(input);
      pushToast("success", `Created "${created.name}"`);
      mode = "list";
      editingTemplate = undefined;
      editingID = undefined;
      await load();
    } catch (err) {
      if (mode === "edit" && isBuiltinRejection(err)) {
        // Clone-on-edit: the server refused the PUT because this id is a
        // builtin. Switch to create mode with the same in-progress fields
        // so the next Save persists a new custom copy instead.
        editingID = undefined;
        mode = "clone";
        editorNote =
          "This is a built-in template and can't be edited directly. " +
          'Saving now will create a new custom template — click "Save" again to confirm.';
        reportError("Edit rejected", err);

        return;
      }

      reportError(mode === "edit" ? "Failed to save template" : "Failed to create template", err);
    } finally {
      saving = false;
    }
  }

  async function removeTemplate(tmpl: Template): Promise<void> {
    if (!confirm(`Delete template "${tmpl.name}"?`)) {
      return;
    }

    try {
      await deleteTemplate(tmpl.id);
      pushToast("success", `Deleted "${tmpl.name}"`);
      await load();
    } catch (err) {
      if (isBuiltinRejection(err)) {
        reportError("Built-in templates can't be deleted", err);

        return;
      }

      reportError("Failed to delete template", err);
    }
  }

  async function runGenerate(): Promise<void> {
    if (generatePrompt.trim() === "") {
      return;
    }

    generating = true;
    try {
      const draft = await generateTemplate(generatePrompt.trim());
      editingTemplate = draft;
      editingID = undefined;
      editorNote =
        "AI-generated draft — review the form/transform/layout below, then Save to persist it as a new template.";
      mode = "create";
    } catch (err) {
      reportError("Failed to generate template", err);
    } finally {
      generating = false;
    }
  }
</script>

<div class="templates-manager">
  {#if mode === "list"}
    <div class="toolbar">
      <button type="button" class="btn btn-primary" onclick={startCreate}>New template</button>
    </div>

    <div class="generate card">
      <h3>Generate with AI</h3>
      <p class="muted">
        Describe the shape you want — the LLM composes a draft from the fixed transform +
        display building blocks (no custom code, no raw HTML).
      </p>
      <textarea
        rows="2"
        placeholder="e.g. a weekly digest grouped by repo with a highlights list"
        bind:value={generatePrompt}
      ></textarea>
      <button type="button" class="btn" onclick={runGenerate} disabled={generating}>
        {generating ? "Generating…" : "Generate"}
      </button>
    </div>

    {#if loading}
      <LoadingInline label="Loading templates…" />
    {:else if templates.length === 0}
      <p class="muted">No templates yet.</p>
    {:else}
      <ul class="template-list">
        {#each templates as tmpl (tmpl.id)}
          <li class="card">
            <div class="info">
              <span class="name">{tmpl.name}</span>
              {#if tmpl.description}<span class="muted">{tmpl.description}</span>{/if}
              <span class="muted small">exports: {tmpl.exports.join(", ") || "none"}</span>
            </div>
            <div class="row-actions">
              <button type="button" class="btn" onclick={() => startEdit(tmpl)}>Edit</button>
              <button type="button" class="btn btn-danger" onclick={() => removeTemplate(tmpl)}>
                Delete
              </button>
            </div>
          </li>
        {/each}
      </ul>
    {/if}
  {:else}
    <TemplateEditor
      initial={editingTemplate}
      note={editorNote}
      saving={saving}
      saveLabel={mode === "edit" ? "Save" : "Save as new template"}
      onSave={saveTemplate}
      onCancel={cancelEdit}
    />
  {/if}
</div>

<style>
  .templates-manager {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .toolbar {
    display: flex;
    justify-content: flex-end;
  }

  .generate {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .generate textarea {
    padding: 0.5rem;
    border-radius: var(--radius);
    border: 1px solid var(--color-border);
    background: var(--color-surface-alt);
  }

  .template-list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .template-list li {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 1rem;
  }

  .info {
    display: flex;
    flex-direction: column;
    gap: 0.15rem;
  }

  .name {
    font-weight: 600;
  }

  .small {
    font-size: 0.75rem;
  }

  .row-actions {
    display: flex;
    gap: 0.5rem;
    flex-shrink: 0;
  }
</style>
