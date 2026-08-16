// Typed per-endpoint wrappers over lib/api.ts's apiFetch/apiFetchBlob — one
// function per operation in api/api.yml, so every call site gets a typed
// request/response instead of hand-building URLs and casting JSON inline.

import { apiFetch, apiFetchBlob, type BlobResult } from "$lib/api";
import {
  API_EXPORT,
  API_LLM_MODELS,
  API_LLM_SETTINGS,
  API_OWNERS,
  API_REPOS,
  API_RUN,
  API_SESSIONS,
  API_SYNC,
  API_SYNC_STATUS,
  API_TEMPLATES,
  API_TEMPLATE_GENERATE,
  API_TIMELINE,
  DEFAULT_PAGE,
  DEFAULT_PER_PAGE,
  QUERY_PARAM_FROM,
  QUERY_PARAM_GAP,
  QUERY_PARAM_OWNER,
  QUERY_PARAM_PAGE,
  QUERY_PARAM_PER_PAGE,
  QUERY_PARAM_REPO,
  QUERY_PARAM_TO,
  QUERY_PARAM_TYPE,
  apiTemplateByID,
} from "$lib/common/constants";
import type {
  Document,
  ExportRequest,
  Filter,
  LLMModel,
  LLMSettings,
  LLMSettingsInput,
  RunRequest,
  SessionsResponse,
  SyncJob,
  SyncStatus,
  Template,
  TemplateInput,
  TimelinePage,
} from "$lib/common/types";

export function listOwners(): Promise<string[]> {
  return apiFetch<string[]>(API_OWNERS);
}

export function listRepos(owner: string): Promise<string[]> {
  return apiFetch<string[]>(API_REPOS, {
    query: { [QUERY_PARAM_OWNER]: owner },
  });
}

export interface TimelineParams extends Filter {
  page?: number;
  perPage?: number;
}

export function listTimeline(params: TimelineParams): Promise<TimelinePage> {
  return apiFetch<TimelinePage>(API_TIMELINE, {
    query: {
      [QUERY_PARAM_OWNER]: params.owner,
      [QUERY_PARAM_REPO]: params.repo,
      [QUERY_PARAM_TYPE]: params.type,
      [QUERY_PARAM_FROM]: params.from,
      [QUERY_PARAM_TO]: params.to,
      [QUERY_PARAM_PAGE]: params.page ?? DEFAULT_PAGE,
      [QUERY_PARAM_PER_PAGE]: params.perPage ?? DEFAULT_PER_PAGE,
    },
  });
}

export interface SessionsParams {
  owner?: string;
  from?: number;
  to?: number;
  gap?: number;
}

export function listSessions(
  params: SessionsParams,
): Promise<SessionsResponse> {
  return apiFetch<SessionsResponse>(API_SESSIONS, {
    query: {
      [QUERY_PARAM_OWNER]: params.owner,
      [QUERY_PARAM_FROM]: params.from,
      [QUERY_PARAM_TO]: params.to,
      [QUERY_PARAM_GAP]: params.gap,
    },
  });
}

export function triggerSync(): Promise<SyncJob> {
  return apiFetch<SyncJob>(API_SYNC, { method: "POST" });
}

export function getSyncStatus(): Promise<SyncStatus> {
  return apiFetch<SyncStatus>(API_SYNC_STATUS);
}

export function listTemplates(): Promise<Template[]> {
  return apiFetch<Template[]>(API_TEMPLATES);
}

export function createTemplate(input: TemplateInput): Promise<Template> {
  return apiFetch<Template>(API_TEMPLATES, { method: "POST", body: input });
}

export function updateTemplate(
  id: string,
  input: TemplateInput,
): Promise<Template> {
  return apiFetch<Template>(apiTemplateByID(id), {
    method: "PUT",
    body: input,
  });
}

export function deleteTemplate(id: string): Promise<void> {
  return apiFetch<void>(apiTemplateByID(id), { method: "DELETE" });
}

export function generateTemplate(prompt: string): Promise<Template> {
  return apiFetch<Template>(API_TEMPLATE_GENERATE, {
    method: "POST",
    body: { prompt },
  });
}

export function runTemplate(req: RunRequest): Promise<Document> {
  return apiFetch<Document>(API_RUN, { method: "POST", body: req });
}

export function exportDocument(req: ExportRequest): Promise<BlobResult> {
  return apiFetchBlob(API_EXPORT, { method: "POST", body: req });
}

export function listLLMModels(): Promise<LLMModel[]> {
  return apiFetch<LLMModel[]>(API_LLM_MODELS);
}

export function getLLMSettings(): Promise<LLMSettings> {
  return apiFetch<LLMSettings>(API_LLM_SETTINGS);
}

export function updateLLMSettings(
  input: LLMSettingsInput,
): Promise<LLMSettings> {
  return apiFetch<LLMSettings>(API_LLM_SETTINGS, {
    method: "PUT",
    body: input,
  });
}
