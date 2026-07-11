export type SyncState =
  "Idle" | "Loading" | "Live" | "Sending" | "Saving" | "Paused" | "Reconnecting" | "Error";

export type ModuleTab =
  | "history"
  | "inventory"
  | "craft"
  | "stats"
  | "codex"
  | "fronts"
  | "investigations"
  | "projects"
  | "saves";

export type OverlayKind =
  "help" | "options" | "saves" | "new-story" | "meta" | "module" | null;

export type DensityPreference = "compact" | "balanced" | "comfortable";
export type FontSizePreference = "small" | "base" | "large";
export type AccentPreference = "amber" | "green" | "blue" | "rose";

export interface AppPreferences {
  density: DensityPreference;
  fontSize: FontSizePreference;
  accent: AccentPreference;
  showLeftRail: boolean;
  showInspector: boolean;
  wrapTranscript: boolean;
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
  active: ModelRoutingActive;
  tts_status: string;
}

export interface ImageGenerationSetting {
  provider: string;
  base_url: string;
  api_key_configured: boolean;
  model: string;
  openclaw_bridge_url: string;
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
  base_url?: string;
  model?: string;
  openclaw_bridge_url?: string;
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
}

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
  message: string;
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
  story_id: string;
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
  prompt: string;
  negative_prompt: string;
  status: VisualAssetStatus | string;
  url: string;
  provider: string;
  source: string;
  error: string;
  turn: number;
  updated_at: string;
}

export interface VisualAssetVersion {
  id: number;
  asset_id: string;
  story_id: string;
  kind: VisualAssetKind;
  subject: string;
  url: string;
  prompt: string;
  revised_prompt: string;
  negative_prompt: string;
  provider: string;
  turn: number;
  created_at: string;
}

export interface VisualGenerationJobView {
  id: number;
  asset_id: string;
  story_id: string;
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
}

export interface VisualAssetsResponse {
  profile: VisualProfile;
  assets: VisualAsset[];
  jobs: VisualGenerationJobView[];
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
}

export interface StoryWizardEnvelope {
  state?: JsonValue;
  input?: string;
  action?: string;
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
  | "local_only";

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
