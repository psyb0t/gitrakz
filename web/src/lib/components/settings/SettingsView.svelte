<script lang="ts">
  // LLM settings page — one global model/reasoning-effort/temperature
  // selection (see api/api.yml's /llm/models + /llm/settings). Reasoning
  // effort and sampling params are model-capability-gated: LLMModel carries
  // supportsReasoningEffort/maxReasoningEffort and supportsSamplingParams,
  // so the corresponding controls only render — and only get sent — when
  // the currently selected model actually accepts them.
  import { untrack } from "svelte";
  import { listLLMModels, getLLMSettings, updateLLMSettings } from "$lib/resources";
  import type { LLMModel, LLMSettings, LLMSettingsInput } from "$lib/common/types";
  import {
    DEFAULT_TEMPERATURE,
    REASONING_EFFORT_LEVELS,
    TEMPERATURE_MAX,
    TEMPERATURE_MIN,
    TEMPERATURE_STEP,
  } from "$lib/common/constants";
  import { pushToast, reportError } from "$lib/stores/toast.svelte";
  import LoadingInline from "$lib/components/common/LoadingInline.svelte";

  let models = $state<LLMModel[]>([]);
  let loading = $state(true);
  let saving = $state(false);

  // Local, user-editable form fields. Seeded once per load() call from the
  // freshly-fetched settings via the untrack one-shot pattern (see
  // TemplateEditor.svelte's `initial` seeding) so an unrelated reactive
  // rerun never clobbers in-progress edits.
  let selectedModelID = $state("");
  let reasoningEffort = $state("");
  let temperature = $state(DEFAULT_TEMPERATURE);

  function seedFormFields(loaded: LLMSettings): void {
    const s = untrack(() => loaded);
    selectedModelID = s.model;
    reasoningEffort = s.reasoningEffort;
    temperature = s.temperature;
  }

  async function load(): Promise<void> {
    loading = true;
    try {
      const [loadedModels, loadedSettings] = await Promise.all([
        listLLMModels(),
        getLLMSettings(),
      ]);
      models = loadedModels;
      seedFormFields(loadedSettings);
    } catch (err) {
      reportError("Failed to load LLM settings", err);
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    void load();
  });

  const selectedModel = $derived.by(() => models.find((m) => m.id === selectedModelID));

  const allowedReasoningLevels = $derived.by(() => {
    if (!selectedModel?.supportsReasoningEffort) {
      return [];
    }

    const ceilingIndex = REASONING_EFFORT_LEVELS.indexOf(
      selectedModel.maxReasoningEffort as (typeof REASONING_EFFORT_LEVELS)[number],
    );
    if (ceilingIndex === -1) {
      return [];
    }

    return REASONING_EFFORT_LEVELS.slice(0, ceilingIndex + 1);
  });

  async function save(): Promise<void> {
    saving = true;
    try {
      const input: LLMSettingsInput = {
        model: selectedModelID,
        reasoningEffort: selectedModel?.supportsReasoningEffort ? reasoningEffort : "",
        temperature: selectedModel?.supportsSamplingParams ? temperature : DEFAULT_TEMPERATURE,
      };
      const updated = await updateLLMSettings(input);
      pushToast("success", `Saved LLM settings for "${updated.model}"`);
      await load();
    } catch (err) {
      reportError("Failed to save LLM settings", err);
    } finally {
      saving = false;
    }
  }
</script>

<div class="settings-view">
  {#if loading}
    <LoadingInline label="Loading LLM settings…" />
  {:else}
    <div class="card">
      <div class="field">
        <label for="llm-model">Model</label>
        <select id="llm-model" bind:value={selectedModelID}>
          {#each models as m (m.id)}
            <option value={m.id}>{m.id}</option>
          {/each}
        </select>
      </div>

      {#if selectedModel?.supportsReasoningEffort}
        <div class="field">
          <label for="llm-reasoning-effort">Reasoning effort</label>
          <select id="llm-reasoning-effort" bind:value={reasoningEffort}>
            <option value="">Default</option>
            {#each allowedReasoningLevels as level (level)}
              <option value={level}>{level}</option>
            {/each}
          </select>
        </div>
      {/if}

      {#if selectedModel?.supportsSamplingParams}
        <div class="field">
          <label for="llm-temperature">Temperature ({temperature.toFixed(1)})</label>
          <input
            id="llm-temperature"
            type="range"
            min={TEMPERATURE_MIN}
            max={TEMPERATURE_MAX}
            step={TEMPERATURE_STEP}
            bind:value={temperature}
          />
        </div>
      {/if}

      {#if selectedModel}
        <p class="muted small">
          Context size: {selectedModel.contextSize.toLocaleString()} tokens
        </p>
      {/if}

      <div class="actions">
        <button
          type="button"
          class="btn btn-primary"
          onclick={save}
          disabled={saving || !selectedModelID}
        >
          {saving ? "Saving…" : "Save"}
        </button>
      </div>
    </div>
  {/if}
</div>

<style>
  .settings-view {
    max-width: 28rem;
  }

  .small {
    font-size: 0.75rem;
  }

  .actions {
    display: flex;
    gap: 0.5rem;
    margin-top: 0.75rem;
  }
</style>
