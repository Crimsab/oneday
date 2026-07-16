export type SyncState =
  "Idle" | "Loading" | "Live" | "Sending" | "Saving" | "Paused" | "Reconnecting" | "Error";

export type ModuleTab =
  | "history"
  | "map"
  | "inventory"
  | "craft"
  | "stats"
  | "codex"
  | "fronts"
  | "investigations"
  | "projects"
  | "achievements"
  | "saves";

export type OverlayKind =
  "help" | "options" | "saves" | "new-story" | "meta" | "module" | null;

export type DensityPreference = "compact" | "balanced" | "comfortable";
export type DesktopRailMode = "expanded" | "collapsed";
export type FontSourcePreference = "bundled" | "system" | "imported" | "online";
export type FontStylePreference = "normal" | "italic";

export interface AppPreferences {
  locale: "en" | "it";
  density: DensityPreference;
  accent: string;
  accentHistory: string[];
  interfaceFontId: string;
  interfaceFontFamily: string;
  interfaceFontSource: FontSourcePreference;
  interfaceFontScale: number;
  readingFontId: string;
  readingFontFamily: string;
  readingFontSource: FontSourcePreference;
  readingFontSize: number;
  readingFontWeight: number;
  readingFontStyle: FontStylePreference;
  readingTextColor: string;
  desktopRailMode: DesktopRailMode;
  /** Runtime visibility only. savePreferences deliberately omits this field. */
  showLeftRail: boolean;
  showInspector: boolean;
  wrapTranscript: boolean;
  showChoiceDetails: boolean;
  automaticChallenges: boolean;
  timingFreeChallenges: boolean;
  challengeCooldown: boolean;
  disabledMiniGames: MiniGameKind[];
  showGenerationDiagnostics: boolean;
}

export interface CraftConversationMessage {
  role: "user" | "assistant";
  content: string;
}

export interface CraftedItemView {
  name: string;
  description: string;
  effect: string;
  materials: string[];
  crafted_at?: string;
}

export interface CraftingResponseView {
  feasible: boolean;
  narrative: string;
  item?: CraftedItemView | null;
  missing?: string[];
  alternatives?: string[];
  choices?: Array<{ id: number; text: string }>;
  resolved_outcome?: JsonValue;
}

export interface CraftResponseEnvelope {
  crafting: CraftingResponseView;
  snapshot: StorySnapshot;
}

export interface ModelProviderSetting {
  id: string;
  label: string;
  enabled: boolean;
  model?: string;
  reasoning?: string;
  supports_model: boolean;
  supports_reasoning: boolean;
}

export interface ModelRoutingActive {
  provider: string;
  narrative_model: string;
  utility_model: string;
  repair_model: string;
  repair_fallback_models: string[];
  image_model: string;
  ascii_model: string;
  embedding_provider: string;
  embedding_model: string;
  codex_reasoning: string;
}

export interface ModelSettings {
  config_path: string;
  config_revision: string;
  provider_priority: string[];
  providers: ModelProviderSetting[];
  narrative_models: string[];
  utility_models: string[];
  repair_models: string[];
  image_models: string[];
  ascii_models: string[];
  embedding_providers: string[];
  image_generation: ImageGenerationSetting;
  image_providers: ImageProviderCatalogEntry[];
  active: ModelRoutingActive;
  tts_status: string;
}

export interface ImageGenerationSetting {
  provider: string;
  map_icon_provider: string;
  base_url: string;
  api_key_configured: boolean;
  model: string;
  map_icon_model: string;
  openclaw_bridge_url: string;
  imagegen_bridge_url: string;
  imagegen_bridge_token_configured: boolean;
  imagegen_bridge_provider: string;
  imagegen_bridge_map_icon_provider: string;
  imagegen_bridge_fallbacks: string[];
  imagegen_bridge_fallback_policy: string;
  imagegen_bridge_compatibility: string;
  default_size: string;
  location_size: string;
  character_size: string;
  default_resolution: string;
  location_resolution: string;
  character_resolution: string;
  default_aspect_ratio: string;
  location_aspect_ratio: string;
  character_aspect_ratio: string;
  quality: string;
  output_format: string;
  background: string;
  timeout_seconds: number;
  auto_generate: boolean;
  append_negative_prompt: boolean;
  available: boolean;
  status: string;
}

export interface ImageProviderCapabilities {
  generate: boolean;
  edit: boolean;
  sizes: string[];
  aspect_ratios: string[];
  qualities: string[];
  output_formats: string[];
  supports_transparency: boolean;
  /** Structured operation support. Optional while older gateways still return booleans. */
  operations?: ImageOperationCapability[];
}

export type ImageOperation =
  | "generate"
  | "edit"
  | "inpaint"
  | "image_transform"
  | "variation"
  | "reference_generate"
  | "outpaint";

export type ImageOperationAvailability =
  | "available"
  | "unavailable"
  | "deprecated"
  | "requires_configuration"
  | "unknown";

export interface ImageOperationCapability {
  operation: ImageOperation;
  supported: boolean;
  availability: ImageOperationAvailability;
  source_images?: {
    min: number;
    max: number;
    roles?: Array<"source" | "style" | "subject" | "composition">;
  };
  mask?: {
    required: boolean;
    kind: "raster";
    accepted_formats?: string[];
    soft_values?: "supported" | "thresholded" | "unsupported";
    provider_semantics?: "transparent_is_edit" | "white_is_edit" | "model_specific";
    adherence?: "best_effort" | "region_constrained";
  };
  controls?: {
    negative_prompt?: boolean;
    strength?: boolean;
    seed?: boolean;
    quality_values?: string[];
    output_formats?: string[];
  };
  provenance?: {
    kind: "static_verified" | "provider_schema" | "runtime_probe";
    verified_at: string;
    schema_hash?: string;
  };
}

export interface ImageProviderCatalogEntry {
  id: string;
  display_name: string;
  auth_type: string;
  default: boolean;
  configured: boolean;
  api_key_configured: boolean;
  status: string;
  base_url: string;
  api_version?: string;
  models: string[];
  model_validation: string;
  capabilities: ImageProviderCapabilities;
}

export interface ModelProviderUpdate {
  id: string;
  enabled?: boolean;
  model?: string;
  reasoning?: string;
}

export interface ModelSettingsUpdate {
  base_revision?: string;
  provider_priority?: string[];
  providers?: ModelProviderUpdate[];
  utility_model?: string;
  repair_model?: string;
  repair_fallback_models?: string[];
  image_model?: string;
  image_generation?: ImageGenerationUpdate;
  ascii_model?: string;
  embedding_model?: string;
  embedding_provider?: string;
}

export interface ImageGenerationUpdate {
  provider?: string;
  map_icon_provider?: string;
  base_url?: string;
  model?: string;
  map_icon_model?: string;
  openclaw_bridge_url?: string;
  imagegen_bridge_url?: string;
  imagegen_bridge_token?: string;
  clear_imagegen_bridge_token?: boolean;
  imagegen_bridge_provider?: string;
  imagegen_bridge_map_icon_provider?: string;
  imagegen_bridge_fallbacks?: string[];
  imagegen_bridge_fallback_policy?: string;
  imagegen_bridge_compatibility?: string;
  default_size?: string;
  location_size?: string;
  character_size?: string;
  default_resolution?: string;
  location_resolution?: string;
  character_resolution?: string;
  default_aspect_ratio?: string;
  location_aspect_ratio?: string;
  character_aspect_ratio?: string;
  quality?: string;
  output_format?: string;
  background?: string;
  timeout_seconds?: number;
  auto_generate?: boolean;
  append_negative_prompt?: boolean;
  provider_configs?: ImageProviderConfigUpdate[];
}

export interface ImageProviderConfigUpdate {
  id: string;
  base_url?: string;
  api_key?: string;
  clear_api_key?: boolean;
  api_version?: string;
  models?: string[];
}

export interface Health {
  status: string;
  stories: number;
  db_path: string;
  config_path: string;
  oneday_bin: string;
  static_dir: string;
}

export interface StorySummary {
  id: string;
  name: string;
  description: string;
  genre: string;
  tone: string;
  language: string;
  is_archived: boolean;
  updated_at: string;
}

export interface StoryUpdatePayload {
  name?: string;
  description?: string;
  genre?: string;
  tone?: string;
  language?: string;
  is_archived?: boolean;
}

export interface StoryDeleteResponse {
  story_id: string;
  cancelled_visual_jobs?: number;
}

export interface StoryDeleteCount {
  table: string;
  rows: number;
}

export interface StoryDeletePlan {
  story_id: string;
  story_name: string;
  counts: StoryDeleteCount[];
  total_rows: number;
  retained_asset_files: string[];
}

export interface StoryOverview {
  story: StorySummary;
  active_branch_id: string;
  revision: number;
  current_turn: number;
  branch_count: number;
  chapter_count: number;
  save_count: number;
  message_count: number;
  asset_count: number;
}

export interface RecordView {
  id: string;
  name: string;
  fields: JsonObject;
}

export interface WorldView {
  id: string;
  current_location: string;
  current_chapter: number;
  current_turn: number;
	current_location_id: string;
	spatial_regions?: JsonValue;
	spatial_locations: JsonValue;
	spatial_edges: JsonValue;
	world_time: JsonValue;
	weather: JsonValue;
  known_locations: JsonValue;
  global_events: JsonValue;
  faction_standings: JsonValue;
  story_hooks: JsonValue;
  world_reactions: JsonValue;
  investigations: JsonValue;
  projects: JsonValue;
  guidance: JsonValue;
  fronts: JsonValue;
  timeline: JsonValue;
  scene_contract: JsonValue;
  updated_at: string;
}

export interface SessionView {
  id: string;
  story_id: string;
  started_at: string;
  ended_at?: string | null;
  summary: string;
}

export interface MessageView {
  id: number;
  session_id: string;
  story_id: string;
  turn: number;
  role: string;
  content: string;
  message_type: string;
  metadata: JsonValue;
  created_at: string;
	branch_id: string;
	source_commit_id: string;
}

export type TTSMode = "off" | "narrator" | "dialogue" | "all";
export type VoiceEnabledMode = "inherit" | "on" | "off";

export interface TTSProviderStatus {
  id: string;
  available: boolean;
  reason?: string;
}

export interface VoiceProfile {
  id: string;
  provider: string;
  model: string;
  provider_voice_id: string;
  display_name: string;
  language_tags: string[];
  version: string;
  style_family: string;
  enabled: boolean;
}

export interface StoryTTSSettings {
  story_id: string;
  mode: TTSMode;
  autoplay: boolean;
  default_language_tag: string;
  provider_policy: Record<string, unknown>;
}

export interface VoiceAssignment {
  id: string;
  assignment_key: string;
  story_id: string;
  entity_id?: string;
  identity_id?: string;
  form_id?: string;
  role: "narrator" | "protagonist" | "npc";
  voice_profile_id: string;
  enabled_mode: VoiceEnabledMode;
  language_tag: string;
  locked: boolean;
  importance: "major" | "supporting" | "minor";
  allow_duplicate: boolean;
}

export interface AudioAsset {
  id: string;
  story_id: string;
  source_message_id: number;
  segment_index: number;
  segment_kind: string;
  status: "queued" | "running" | "ready" | "failed" | string;
  duration_ms: number;
  language_tag: string;
  error?: string;
}

export interface TTSJob {
  id: string;
  audio_asset_id: string;
  status: string;
  attempts: number;
  max_attempts: number;
  error?: string;
}

export interface PronunciationEntry {
  id: string;
  story_id: string;
  language_tag: string;
  source_text: string;
  pronunciation: string;
  alphabet: "ipa" | "x-sampa" | "provider";
  case_sensitive: boolean;
  revision: number;
}

export interface AudioCleanupResult {
  dry_run: boolean;
  files_scanned: number;
  orphan_files: number;
  files_removed: number;
  invalid_cache_rows: number;
  prunable_cache_rows: number;
  cache_rows_removed: number;
  errors?: string[];
}

export interface AudioManifest {
  format: "oneday-audio-manifest-v1";
  filename: string;
  generated_at: string;
  story_id: string;
  settings: StoryTTSSettings;
  providers: TTSProviderStatus[];
  voices: VoiceProfile[];
  assignments: VoiceAssignment[];
  pronunciations: PronunciationEntry[];
  assets: AudioAsset[];
  jobs: TTSJob[];
}

export interface TTSCatalogResponse {
  providers: TTSProviderStatus[];
  voices: VoiceProfile[];
}

export interface TTSSettingsResponse { settings: StoryTTSSettings; }
export interface VoiceAssignmentsResponse { assignments: VoiceAssignment[]; assignment?: VoiceAssignment; }
export interface MessageAudioResponse { assets: AudioAsset[]; jobs: TTSJob[]; }
export interface PronunciationsResponse { pronunciations: PronunciationEntry[]; pronunciation?: PronunciationEntry; }
export interface AudioCleanupResponse { cleanup: AudioCleanupResult; }
export interface AudioExportResponse { export: AudioManifest; }

export interface PendingTurnView {
  id: string;
  turn: number;
  source: string;
  detail: string;
  streamingText?: string;
  streamingSuppressed?: boolean;
  kind: "choice" | "free_text" | "command" | "meta";
}

export interface TurnStreamEvent {
  story_id: string;
  status:
    | "submitted"
    | "event"
    | "completed"
    | "failed"
    | "snapshot_changed"
    | "lagged"
    | string;
  client_turn?: number | null;
  action_kind?: string | null;
  action_text?: string | null;
  event_type?: string | null;
  event?: JsonValue;
  message_key?: string;
  message_args?: JsonObject;
  error_code?: string | null;
  message: string;
  created_at: string;
}

export interface AgencyEventView {
  id: number;
  story_id: string;
  branch_id: string;
  commit_id: string;
  canonical_turn: number;
  entity_id: string;
  entity_name: string;
  action: string;
  summary: string;
  created_at: string;
}

export interface ChoiceView {
  id: number;
  text: string;
  intent?: string;
  risk?: string;
  scope?: string;
  certainty?: string;
  related_stats?: string[];
}

export interface ChapterView {
  id: number;
  chapter_number: number;
  title: string;
  summary: string;
  start_turn: number;
  end_turn?: number | null;
  created_at: string;
	branch_id: string;
	source_commit_id: string;
}

export interface TimelineBranchView { id:string; story_id:string; name:string; fork_commit_id?:string; head_commit_id:string; head_turn:number; created_at:string; updated_at:string }
export interface TimelineCommitView { id:string; branch_id:string; parent_commit_id?:string; canonical_turn:number; kind:string; message?:string; created_at:string }
export interface TimelineResponse { active_branch_id:string; revision:number; branches:TimelineBranchView[]; head?:TimelineCommitView; commits:TimelineCommitView[] }
export interface TimelineEnvelope { action:"fork"|"fork_checkout"|"rename"|"checkout"; client_revision:number; branch_id?:string; from_commit_id?:string; name?:string }
export interface TimelineMutationResponse { timeline:TimelineResponse; snapshot:StorySnapshot }
export interface HistoryPage { items:MessageView[]; next_cursor?:number|null }
export interface ChapterPage { items:ChapterView[]; next_cursor?:number|null }
export interface StoryExport { format:"markdown"|"json"|"epub"|"replay"; filename:string; content:string; encoding?:"utf-8"|"base64"; content_type?:string }

export interface TelemetryUsage {
  input_tokens: number;
  output_tokens: number;
  reasoning_tokens: number;
  cached_input_tokens: number;
  total_tokens: number;
  cost_usd: number;
}

export interface GenerationAttemptDiagnostics {
  sequence: number;
  provider: string;
  requested_model: string;
  resolved_model: string;
  requested_streaming: boolean;
  observed_streaming: boolean;
  status: string;
  ttft_ms: number;
  duration_ms: number;
  usage: TelemetryUsage;
  retry_reason: string;
  error_class: string;
}

export interface GenerationDiagnostics {
  run_id: string;
  trace_id: string;
  parent_run_id: string;
  story_id: string;
  branch_id: string;
  source_commit_id: string;
  message_id?: number | null;
  stage: string;
  status: string;
  prompt_profile: string;
  prompt_revision: number;
  prompt_hash: string;
  requested_streaming: boolean;
  observed_streaming: boolean;
  ttft_ms: number;
  duration_ms: number;
  usage: TelemetryUsage;
  error_class: string;
  created_at: string;
  finished_at: string;
  attempts: GenerationAttemptDiagnostics[];
}

export interface TelemetryExport {
  format: "jsonl";
  filename: string;
  content: string;
  count: number;
  truncated: boolean;
}

export interface AchievementView {
  id: number;
  name: string;
  description: string;
  category: string;
  rarity: string;
  context: string;
  earned_at: string;
}

export interface SaveView {
  id: string;
  name: string;
  turn: number;
  chapter: number;
  location: string;
  session_id?: string | null;
  metadata: JsonValue;
  created_at: string;
}

export interface PanelsView {
  chapters: ChapterView[];
  achievements: AchievementView[];
  npcs: RecordView[];
  sessions: SessionView[];
  saves: SaveView[];
}

export interface StoryVersion {
  turn: number;
  revision: number;
  story_updated_at: string;
  active_session_id: string;
  last_message_id: number;
  world_updated_at: string;
  character_updated_at: string;
  npc_count: number;
  npc_updated_at: string;
  chapter_count: number;
  achievement_count: number;
  latest_achievement_at: string;
  save_count: number;
  latest_save_at: string;
  visual_asset_updated_at: string;
  visual_job_updated_at: string;
  active_visual_job_count: number;
}

export interface StorySnapshot {
  server_time: string;
  version: StoryVersion;
  story: StorySummary;
  character: RecordView;
  world: WorldView;
  active_session: SessionView;
  choices: ChoiceView[];
  messages: MessageView[];
  panels: PanelsView;
}

export type VisualAssetStatus =
  "pending" | "queued" | "running" | "ready" | "failed";
export type VisualAssetKind = "location" | "character" | "world" | string;

export interface VisualProfile {
  id: string;
  story_id: string;
  revision: number;
  fingerprint: string;
  branch_id: string;
  source_commit_id: string;
  world_style_prompt: string;
  character_style_prompt: string;
  negative_prompt: string;
  palette: string;
  updated_at: string;
}

export interface VisualAsset {
  id: string;
  story_id: string;
  kind: VisualAssetKind;
  subject: string;
  entity_id: string;
  canonical_entity_id: string;
  canonical_location_id: string;
  map_scope_kind?: string;
  map_scope_id?: string;
  form_id: string;
  lineage_key: string;
  appearance_fingerprint: string;
  profile_revision_id: string;
  canon_status: string;
  gate_state: string;
  gate_reason: string;
  gate_reason_code?: string;
  generation_eligible: boolean;
  prompt: string;
  negative_prompt: string;
  status: VisualAssetStatus | string;
  url: string;
  provider: string;
  source: string;
  error: string;
  turn: number;
  branch_id: string;
  source_commit_id: string;
  selected_version_id?: number | null;
  can_undo_selection: boolean;
  can_redo_selection: boolean;
  inherited: boolean;
  updated_at: string;
  /** Asset/model-specific descriptors take precedence over provider defaults. */
  operation_capabilities?: ImageOperationCapability[];
}

export interface VisualAssetVersion {
  id: number;
  asset_id: string;
  story_id: string;
  kind: VisualAssetKind;
  subject: string;
  canonical_entity_id: string;
  canonical_location_id: string;
  form_id: string;
  appearance_fingerprint: string;
  profile_revision_id: string;
  canon_status: string;
  url: string;
  prompt: string;
  revised_prompt: string;
  negative_prompt: string;
  provider: string;
  turn: number;
  branch_id: string;
  source_commit_id: string;
  source_kind: "generated" | "upload" | "imported" | string;
  created_at: string;
}

export interface VisualAssetUploadResponse {
  asset_id: string;
  version_id: number;
  selected: boolean;
  visual_assets: VisualAssetsResponse;
}

export interface VisualGenerationJobView {
  id: number;
  asset_id: string;
  story_id: string;
  canonical_entity_id: string;
  canonical_location_id: string;
  form_id: string;
  appearance_fingerprint: string;
  profile_revision_id: string;
  status: string;
  attempts: number;
  max_attempts: number;
  locked_until: string;
  error: string;
  provider: string;
  started_at: string;
  finished_at: string;
  created_at: string;
  updated_at: string;
  branch_id: string;
  source_commit_id: string;
}

export interface VisualAssetsResponse {
  profile: VisualProfile;
  assets: VisualAsset[];
  jobs: VisualGenerationJobView[];
  /** Recent asynchronous edits; omitted by gateways predating native image editing. */
  operations?: ImageOperationView[];
  /** Effective descriptors for the currently configured image route. */
  operation_capabilities?: ImageOperationCapability[];
}

export interface ImageOperationView {
  id: string;
  asset_id?: string;
  operation: ImageOperation;
  status: "queued" | "running" | "succeeded" | "failed" | "cancelled" | string;
  provider: string;
  model: string;
  endpoint_id: string;
  source_version_id?: number | null;
  mask_id: string;
  result_version_id?: number | null;
  branch_id: string;
  error_code: string;
  error_summary: string;
  created_at: string;
  updated_at: string;
}

export interface VisualAssetOperationRequest {
  operation: Extract<ImageOperation, "edit" | "inpaint" | "image_transform">;
  source_version_id: number;
  prompt: string;
  negative_prompt?: string;
  mask_png_base64?: string;
  fallback: {
    mode: "forbid";
  };
  idempotency_key: string;
}

export interface VisualAssetCleanupRequest {
  dry_run?: boolean;
}

export interface VisualAssetCleanupResponse {
  story_id: string;
  dry_run: boolean;
  deleted_files: string[];
  kept_files: string[];
}

export interface GenerateVisualAssetsRequest {
  asset_ids?: string[];
  force?: boolean;
  allow_silhouette?: boolean;
  limit?: number;
}

export interface VisualProfileUpdate {
  world_style_prompt: string;
  character_style_prompt: string;
  negative_prompt: string;
  palette: string;
}

export interface VisualAssetPromptUpdate {
  prompt: string;
  negative_prompt: string;
}

export interface StoryCreateEnvelope {
  brief: string;
  character_name: string;
  character_background: string;
  world_style_prompt?: string;
  character_style_prompt?: string;
  negative_prompt?: string;
  palette?: string;
  start: boolean;
}

export interface StoryCreateResult {
  story_id: string;
  character_id: string;
  session_id?: string;
  started?: boolean;
  start_error?: string;
}

export interface StoryCreateResponse {
  story: StoryCreateResult;
  snapshot: StorySnapshot;
}

export interface StoryWizardAction {
  key: string;
  label: string;
  seed?: string;
}

export interface StoryWizardEnvelope {
  state?: JsonValue;
  input?: string;
  action?: string;
  world_style_prompt?: string;
  character_style_prompt?: string;
  negative_prompt?: string;
  palette?: string;
  start: boolean;
}

export interface StoryWizardResult {
  state?: JsonValue;
  phase: string;
  stage: string;
  stage_label: string;
  placeholder: string;
  message: string;
  actions: StoryWizardAction[];
  definition?: JsonValue;
  last_model?: string;
  last_latency_ms?: number;
  story_id?: string;
  character_id?: string;
  session_id?: string;
  started?: boolean;
  start_error?: string;
}

export interface StoryWizardResponse {
  wizard: StoryWizardResult;
  snapshot?: StorySnapshot | null;
}

export interface StoryEnhanceEnvelope {
  state?: JsonValue;
  stage: string;
  text: string;
  context?: string;
}

export interface StoryEnhanceResponse {
  text: string;
  model?: string;
  provider?: string;
  latency_ms?: number;
}

export type CommandGroup =
  "play" | "talk" | "state" | "save" | "meta" | "system" | "debug";
export type CommandParity = "shared" | "terminal_only" | "browser_only";
export type CommandBehavior =
  | "submit_action"
  | "submit_meta"
  | "open_panel"
  | "save_create"
  | "save_load"
  | "save_delete"
  | "insert_template"
  | "local_only"
	| "timeline";

export interface CommandArgDescriptor {
  name: string;
  label?: string;
  required?: boolean;
  variadic?: boolean;
  placeholder?: string;
  description?: string;
}

export interface CommandDescriptor {
  id: string;
  canonical: string;
  aliases?: string[];
  title: string;
  description: string;
  group: CommandGroup;
  parity: CommandParity;
  behavior: CommandBehavior;
  args?: CommandArgDescriptor[];
  completion_provider?: string;
  trailing_space?: boolean;
  examples?: string[];
  enabled_when?: string;
}

export interface PlayerAction {
  kind: "choice" | "free_text" | "command";
  text?: string;
  choice_id?: number;
}

export type OutcomeDegree = "critical_success" | "full_success" | "success_with_cost" | "failure_with_progress" | "hard_failure" | "catastrophe";
export interface ChallengeModifier { source: string; value: number }
export interface OutcomeEnvelope {
  version: 1;
  degree: OutcomeDegree;
  difficulty: number;
  seed: number;
  roll: number;
  modifiers?: ChallengeModifier[];
  total: number;
  margin: number;
  costs?: JsonValue[];
  consequences?: string[];
  state_deltas?: JsonValue[];
  revealed_facts?: string[];
  follow_up_pressure?: number;
}
export interface ChallengeDefinition { id: string; kind: string; description?: string; difficulty: number }
export interface ChallengeInstance { protocol_version: 1; id: string; story_id?: string; branch_id?: string; turn: number; definition: ChallengeDefinition; seed: number; policy: JsonObject; timing?: JsonObject }
export interface ChallengeInput { actor_id?: string; intent: string; choice_id?: number; modifiers?: ChallengeModifier[]; payload?: JsonValue; elapsed_ms?: number }
export interface ChallengeResolution { protocol_version: 1; instance_id: string; input: ChallengeInput; outcome: OutcomeEnvelope }

export type MiniGameKind = "rps" | "memory" | "quicktime" | "riddle" | "deduction" | "negotiation" | "pattern" | "bidding" | "courtroom" | "comedy";
export type MiniGamePhase = "ready" | "active" | "paused" | "resolved";
export interface MiniGameDefinition {
  id: string;
  kind: MiniGameKind;
  prompt?: string;
  difficulty: number;
  options?: string[];
  sequence?: string[];
  time_limit_ms?: number;
  rules?: Record<string, string>;
}
export interface MiniGameInput { action: "pause" | "resume" | "submit"; value?: string; values?: string[]; elapsed_ms?: number }
export interface MiniGameResult { passed: boolean; total?: number; difficulty?: number; detail: string; outcome?: OutcomeEnvelope }
export interface MiniGameInstance {
  protocol_version: number;
  id: string;
  story_id: string;
  branch_id: string;
  turn: number;
  seed: number;
  definition: MiniGameDefinition;
  runtime: { phase: MiniGamePhase; revision: number; state?: JsonObject; history?: MiniGameInput[]; result?: MiniGameResult };
}
export interface MiniGameResponse { instance?: MiniGameInstance | null }

export interface ActionEnvelope {
  session_id: string;
  client_turn: number;
  client_revision: number;
  idempotency_key: string;
  action: PlayerAction;
  stream?: boolean;
  capabilities: {
    images: boolean;
    ascii: boolean;
    roll_log: boolean;
    automatic_challenges: boolean;
    timing_free_challenges: boolean;
    challenge_cooldown: boolean;
    excluded_minigames: MiniGameKind[];
  };
}

export interface ActionResponse {
  events: JsonValue[];
  snapshot: StorySnapshot;
}

export type BrowserMetaKind = "btw" | "guide" | "narrator";

export interface MetaCommand {
  kind: BrowserMetaKind;
  text: string;
}

export interface MetaEnvelope extends MetaCommand {
  session_id: string;
  client_turn: number;
  client_revision: number;
}

export interface MetaResult {
  kind: BrowserMetaKind;
  title: string;
  message: string;
}

export interface MetaResponse {
  meta?: MetaResult | null;
  snapshot: StorySnapshot;
}

export interface SaveEnvelope {
  session_id: string;
  client_turn: number;
  client_revision: number;
  name: string;
  kind: "manual" | "quicksave";
}

export interface SaveResponse {
  save?: SaveView | null;
  snapshot: StorySnapshot;
}

export interface LoadEnvelope {
  session_id: string;
  client_turn: number;
  client_revision: number;
  save_id: string;
}

export interface LoadResponse {
  save?: SaveView | null;
  legacy?: boolean;
  snapshot_state: "full" | "legacy_partial";
  snapshot_detail?: string;
  snapshot: StorySnapshot;
}

export interface DeleteSaveEnvelope {
  session_id: string;
  client_turn: number;
  client_revision: number;
  save_id: string;
}

export interface DeleteSaveResponse {
  save?: SaveView | null;
  snapshot: StorySnapshot;
}

export interface RecentCommand {
  id: string;
  text: string;
  turn: number;
  source: "browser" | "history";
}

export type JsonPrimitive = string | number | boolean | null;
export type JsonValue = JsonPrimitive | JsonObject | JsonValue[];
export interface JsonObject {
  [key: string]: JsonValue;
}
