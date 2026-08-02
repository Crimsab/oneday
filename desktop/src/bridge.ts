import { invoke } from "@tauri-apps/api/core";
import { disable, enable, isEnabled } from "@tauri-apps/plugin-autostart";
import {
  isPermissionGranted,
  requestPermission,
  sendNotification,
} from "@tauri-apps/plugin-notification";

export interface DesktopState {
  profile: DesktopProfile | null;
  serverUrl: string | null;
  lifecycle: DesktopLifecycle;
  startedMinimized: boolean;
  updater: UpdaterStatus;
  startupWarning: string | null;
}

export type DesktopProfile =
  | { mode: "remote"; serverUrl: string }
  | { mode: "standalone"; profileId: string };

export type DesktopLifecycle =
  | { state: "stopped" | "starting" | "draining" }
  | { state: "ready"; endpoint: string }
  | { state: "failed"; message: string };

export interface StorySummary {
  id: string;
  name: string;
}

export interface TransferResult {
  cancelled: boolean;
  message: string;
  storyId?: string;
  path?: string;
}

export interface CodexStatus {
  available: boolean;
  state: "missing" | "app_only" | "legacy_cli" | "unusable" | "signed_out" | "ready";
  source: "missing" | "global" | "managed" | "system";
  version: string | null;
  authenticated: boolean;
  desktopAppDetected: boolean;
  legacyCliDetected: boolean;
  managedVersion: string;
  message: string;
  launcher: string | null;
  diagnosticShell: "powershell" | "terminal";
  diagnosticCommand: string;
  installScope: "global" | "managed";
}

export interface ClaudeStatus {
  available: boolean;
  version: string | null;
  authenticated: boolean;
  installSupported: boolean;
  installMethod: string | null;
  message: string;
}

export interface UpdaterStatus {
  enabled: boolean;
  currentVersion: string;
  channel: string;
  reason: string;
}

export interface UpdateCheck {
  available: boolean;
  version: string | null;
  notes: string | null;
  publishedAt: string | null;
  message: string;
}

export const desktopBridge = {
  state: () => invoke<DesktopState>("desktop_state"),
  connect: (serverUrl: string) => invoke<void>("connect_server", { serverUrl }),
  startStandalone: () => invoke<void>("start_standalone"),
  restartStandalone: () => invoke<void>("restart_standalone"),
  stopStandalone: () => invoke<void>("stop_standalone"),
  showStoryWindow: () => invoke<void>("show_story_window"),
  showProviderSetup: () => invoke<void>("show_provider_setup"),
  stories: () => invoke<StorySummary[]>("list_remote_stories"),
  importPackage: () => invoke<TransferResult>("choose_and_import_story"),
  exportPackage: (storyId: string, kind: "archive" | "world") =>
    invoke<TransferResult>("choose_and_export_story", { storyId, kind }),
  updater: () => invoke<UpdaterStatus>("updater_status"),
  checkUpdate: () => invoke<UpdateCheck>("check_update"),
  installUpdate: () => invoke<void>("install_update"),
  codexStatus: () => invoke<CodexStatus>("codex_status"),
  installCodex: () => invoke<CodexStatus>("install_codex_component"),
  loginCodex: () => invoke<CodexStatus>("login_codex"),
  claudeStatus: () => invoke<ClaudeStatus>("claude_status"),
  installClaude: () => invoke<ClaudeStatus>("install_claude"),
  loginClaude: () => invoke<ClaudeStatus>("login_claude"),
  openClaudeInstallGuide: () => invoke<void>("open_claude_install_guide"),
  autostartEnabled: () => isEnabled(),
  setAutostart: async (enabled: boolean) => (enabled ? enable() : disable()),
  notificationsEnabled: () => isPermissionGranted(),
  requestNotifications: async () => (await requestPermission()) === "granted",
  testNotification: () =>
    sendNotification({ title: "OneDay", body: "Desktop notifications are ready." }),
};

export function friendlyError(error: unknown): string {
  if (typeof error === "string") return error;
  if (error instanceof Error) return error.message;
  return "The operation could not be completed.";
}
