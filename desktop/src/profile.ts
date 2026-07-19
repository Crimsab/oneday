import type { DesktopLifecycle, DesktopProfile } from "./bridge";

export function profileLabel(profile: DesktopProfile | null): string {
  if (!profile) return "Choose a profile";
  return profile.mode === "remote" ? "Remote server" : "Standalone on this device";
}

export function lifecycleLabel(lifecycle: DesktopLifecycle): string {
  switch (lifecycle.state) {
    case "ready": return "Ready";
    case "failed": return "Failed";
    case "starting": return "Starting";
    case "draining": return "Stopping";
    case "stopped": return "Stopped";
  }
}

export function isServerReady(lifecycle: DesktopLifecycle): boolean {
  return lifecycle.state === "ready";
}
