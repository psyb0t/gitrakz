// Shared domain types mirroring api/api.yml's components.schemas — field
// names match the generated JSON exactly (tagliatelle camelCase), read
// straight from internal/pkg/http/api/api.gen.go rather than guessed.

export const EVENT_TYPES = [
  "commit",
  "pr",
  "review",
  "issue",
  "release",
] as const;
export type EventType = (typeof EVENT_TYPES)[number];

export interface Event {
  id: string;
  ts: number;
  type: EventType;
  owner: string;
  repo: string;
  sha?: string | null;
  number?: number | null;
  title?: string | null;
  url?: string | null;
  additions?: number | null;
  deletions?: number | null;
  branch?: string | null;
}

export interface TimelinePage {
  items: Event[];
  hasMore: boolean;
}

export interface Session {
  owner: string;
  day?: string | null;
  startTs: number;
  endTs: number;
  durationHours: number;
  events: Event[];
}

export interface SessionsResponse {
  sessions: Session[];
}

export interface SyncJob {
  jobId: string;
}

export interface SyncStatus {
  lastSyncedTs: number;
  inProgress: boolean;
  perOwner: Record<string, unknown>;
}

/**
 * A single display building block. `data` is typed per `type` by the
 * BLOCK_TYPE_* payload interfaces below — see block-renderer components.
 */
export interface Block {
  type: string;
  data: Record<string, unknown>;
}

export type Document = Block[];

export const EXPORT_FORMATS = ["csv", "pdf", "json"] as const;
export type ExportFormat = (typeof EXPORT_FORMATS)[number];

export interface FormField {
  name: string;
  label?: string | null;
  type: string;
  required?: boolean;
  default?: unknown;
}

export interface TransformStep {
  primitive: string;
  params?: Record<string, unknown>;
}

export interface TemplateInput {
  name: string;
  description?: string | null;
  form: FormField[];
  transform: TransformStep[];
  layout: Block[];
  exports: ExportFormat[];
  model?: string | null;
}

export interface Template extends TemplateInput {
  id: string;
}

export interface Filter {
  owner?: string;
  repo?: string;
  type?: EventType;
  from?: number;
  to?: number;
}

export interface RunRequest {
  templateId: string;
  filter?: Filter;
  formValues?: Record<string, unknown>;
}

export interface ExportRequest {
  document?: Document;
  templateId?: string;
  filter?: Filter;
  formValues?: Record<string, unknown>;
  format: ExportFormat;
}

export interface ErrorResponse {
  code: string;
  message: string;
  details?: Record<string, unknown> | null;
}

// --- Display block payload shapes (Block.data), per block.type ---

export interface HeadingData {
  level: number;
  text: string;
}

export interface TextData {
  markdown: string;
}

export interface ListData {
  ordered: boolean;
  items: string[];
}

export interface TableData {
  columns: string[];
  rows: string[][];
  footer: string[];
}

export interface KeyValuePair {
  key: string;
  value: string;
}

export interface KeyValueData {
  pairs: KeyValuePair[];
}

export interface MetricData {
  label: string;
  value: string;
  unit: string;
}

export interface CodeData {
  lang: string;
  content: string;
}

export interface ChartData {
  kind: string;
  labels: string[];
  values: number[];
}

export interface LLMModel {
  id: string;
  supportsReasoningEffort: boolean;
  maxReasoningEffort: string;
  supportsSamplingParams: boolean;
  contextSize: number;
}

export interface LLMSettingsInput {
  model: string;
  reasoningEffort: string;
  temperature: number;
}

export type LLMSettings = LLMSettingsInput;
