import { invoke } from "@tauri-apps/api/core";
import { disable, enable, isEnabled } from "@tauri-apps/plugin-autostart";
import {
  isPermissionGranted,
  requestPermission,
  sendNotification,
} from "@tauri-apps/plugin-notification";

export interface DesktopState {
  serverUrl: string | null;
  startedMinimized: boolean;
  updater: { enabled: boolean; reason: string };
}

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

export const desktopBridge = {
  state: () => invoke<DesktopState>("desktop_state"),
  connect: (serverUrl: string) => invoke<void>("connect_server", { serverUrl }),
  showStoryWindow: () => invoke<void>("show_story_window"),
  stories: () => invoke<StorySummary[]>("list_remote_stories"),
  importPackage: () => invoke<TransferResult>("choose_and_import_story"),
  exportPackage: (storyId: string, kind: "archive" | "world") =>
    invoke<TransferResult>("choose_and_export_story", { storyId, kind }),
  updater: () => invoke<{ enabled: boolean; reason: string }>("updater_status"),
  checkAndInstallUpdate: () =>
    invoke<{ available: boolean; version: string | null; message: string }>(
      "check_and_install_update",
    ),
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
