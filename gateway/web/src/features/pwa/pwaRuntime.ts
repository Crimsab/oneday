import { registerSW } from "virtual:pwa-register";

type RefreshListener = () => void;

const refreshListeners = new Set<RefreshListener>();
let started = false;
let applyWaitingUpdate: ((reloadPage?: boolean) => Promise<void>) | null = null;

export function subscribeToPwaUpdates(listener: RefreshListener): () => void {
  refreshListeners.add(listener);
  startRegistration();
  return () => refreshListeners.delete(listener);
}

export async function activateWaitingUpdate(): Promise<void> {
  await applyWaitingUpdate?.(true);
}

function startRegistration(): void {
  if (started || !import.meta.env.PROD || !("serviceWorker" in navigator)) return;
  started = true;
  applyWaitingUpdate = registerSW({
    immediate: true,
    onNeedRefresh() {
      refreshListeners.forEach((listener) => listener());
    },
    onRegisterError(error) {
      console.warn("OneDay PWA registration failed", error);
    },
  });
}

