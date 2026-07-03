export type SyncState = "Idle" | "Loading" | "Live" | "Sending" | "Paused" | "Reconnecting" | "Error";

export type ModuleTab =
  | "history"
  | "inventory"
  | "stats"
  | "codex"
  | "fronts"
  | "investigations"
  | "projects"
  | "saves";

export type OverlayKind = "help" | "options" | "saves" | "new-story" | null;

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
  last_message_id: number;
  world_updated_at: string;
  achievement_count: number;
  save_count: number;
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

export interface PlayerAction {
  kind: "choice" | "free_text" | "command";
  text?: string;
  choice_id?: number;
}

export interface ActionEnvelope {
  session_id: string;
  client_turn: number;
  idempotency_key: string;
  action: PlayerAction;
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
