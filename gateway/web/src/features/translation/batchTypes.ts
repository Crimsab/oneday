export type TranslationEngine = "browser" | "ai";
export type TranslationStyle = "faithful" | "natural" | "literary";
export type TranslationJobStatus = "queued" | "running" | "paused" | "completed" | "partial" | "failed" | "cancelled";

export interface TranslationJobRequest {
  scope_kind: "chapter" | "story" | "selection";
  scope_id: string;
  message_ids?: number[];
  source_language: string;
  target_language: string;
  engine: TranslationEngine;
  provider: string;
  model: string;
  style: TranslationStyle;
}

export interface TranslationJob extends TranslationJobRequest {
  id: string;
  story_id: string;
  branch_id: string;
  status: TranslationJobStatus;
  total_items: number;
  completed_items: number;
  failed_items: number;
  cached_items: number;
  total_characters: number;
  processed_characters: number;
  error_code: string;
  error_summary: string;
  created_at: string;
  updated_at: string;
}

export interface TranslationEstimate { total_items: number; total_characters: number; cache_hits: number }
export interface BrowserTranslationItem { id: string; content_kind: string; content_id: string; source_text: string; source_language: string; target_language: string }
export interface TranslationGlossaryEntry { id: string; source_language: string; target_language: string; source_term: string; target_term: string; mode: "translate" | "preserve" }

