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
  if (!applyWaitingUpdate) {
    window.location.reload();
    return;
  }

  const controllerChanged = new Promise<void>((resolve) => {
    navigator.serviceWorker.addEventListener("controllerchange", () => resolve(), { once: true });
  });
  await applyWaitingUpdate(false);
  await Promise.race([
    controllerChanged,
    new Promise<void>((resolve) => window.setTimeout(resolve, 4_000)),
  ]);
  window.location.reload();
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
