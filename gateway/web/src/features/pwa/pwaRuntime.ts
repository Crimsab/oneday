import { registerSW } from "virtual:pwa-register";
import { checkForServiceWorkerUpdate } from "./pwaUpdate";

type RefreshListener = () => void;

const refreshListeners = new Set<RefreshListener>();
let started = false;
let applyWaitingUpdate: ((reloadPage?: boolean) => Promise<void>) | null = null;
const UPDATE_INTERVAL_MS = 15 * 60 * 1000;

export function subscribeToPwaUpdates(listener: RefreshListener): () => void {
  refreshListeners.add(listener);
  startRegistration();
  return () => refreshListeners.delete(listener);
}

export async function activateWaitingUpdate(): Promise<void> {
  if (!applyWaitingUpdate) {
    window.location.reload();
    return;
  }
  await applyWaitingUpdate(true);
}

function startRegistration(): void {
  if (started || !import.meta.env.PROD || !("serviceWorker" in navigator)) return;
  started = true;
  applyWaitingUpdate = registerSW({
    immediate: true,
    onNeedRefresh() {
      refreshListeners.forEach((listener) => listener());
    },
    onRegisteredSW(swUrl, registration) {
      if (!registration) return;
      const check = () => void checkForServiceWorkerUpdate(swUrl, registration);
      window.setInterval(check, UPDATE_INTERVAL_MS);
      window.addEventListener("online", check);
      document.addEventListener("visibilitychange", () => {
        if (document.visibilityState === "visible") check();
      });
    },
    onRegisterError(error) {
      console.warn("OneDay PWA registration failed", error);
    },
  });
}
