import { useCallback, useEffect, useMemo, useState } from "react";
import { RefreshCw, ServerOff, WifiOff } from "lucide-react";
import { useTranslation } from "react-i18next";
import { activateWaitingUpdate, subscribeToPwaUpdates } from "./pwaRuntime";
import "./pwaStatus.css";

type Connectivity = "connected" | "offline" | "server-unreachable";

const copy = {
  en: {
    statusLabel: "OneDay app status",
    updateTitle: "Update available",
    updateBody: "Reload to apply the latest fixes. Your stories stay safely on the server.",
    update: "Update now",
    later: "Later",
    offlineTitle: "You are offline",
    offlineBody: "The app shell is available, but stories and changes still require the OneDay server.",
    serverTitle: "Server unavailable",
    serverBody: "Your stories remain on the server. Reconnect or retry when it is reachable.",
    retry: "Retry",
  },
  it: {
    statusLabel: "Stato dell'app OneDay",
    updateTitle: "Aggiornamento disponibile",
    updateBody: "Ricarica per applicare le ultime correzioni. Le tue storie restano al sicuro sul server.",
    update: "Aggiorna ora",
    later: "Più tardi",
    offlineTitle: "Sei offline",
    offlineBody: "L'interfaccia resta disponibile, ma storie e modifiche richiedono il server OneDay.",
    serverTitle: "Server non raggiungibile",
    serverBody: "Le storie restano sul server. Riconnettiti o riprova quando è disponibile.",
    retry: "Riprova",
  },
} as const;

export async function checkServerConnectivity(
  online = navigator.onLine,
  request: typeof fetch = fetch,
): Promise<Connectivity> {
  if (!online) return "offline";
  try {
    const response = await request("/api/health", {
      cache: "no-store",
      credentials: "same-origin",
      headers: { Accept: "application/json" },
      signal: AbortSignal.timeout(5_000),
    });
    return response.ok && response.headers.get("content-type")?.includes("application/json")
      ? "connected"
      : "server-unreachable";
  } catch {
    return online ? "server-unreachable" : "offline";
  }
}

export function PwaStatus() {
  const { i18n } = useTranslation();
  const [updateReady, setUpdateReady] = useState(false);
  const [connectivity, setConnectivity] = useState<Connectivity>("connected");
  const strings = copy[(i18n.resolvedLanguage?.startsWith("it") ? "it" : "en")];

  const refreshConnectivity = useCallback(async () => {
    setConnectivity(await checkServerConnectivity());
  }, []);

  useEffect(() => subscribeToPwaUpdates(() => setUpdateReady(true)), []);

  useEffect(() => {
    void refreshConnectivity();
    const onConnectivityChange = () => void refreshConnectivity();
    const onVisibilityChange = () => {
      if (document.visibilityState === "visible") void refreshConnectivity();
    };
    const interval = window.setInterval(onConnectivityChange, 45_000);
    window.addEventListener("online", onConnectivityChange);
    window.addEventListener("offline", onConnectivityChange);
    document.addEventListener("visibilitychange", onVisibilityChange);
    return () => {
      window.clearInterval(interval);
      window.removeEventListener("online", onConnectivityChange);
      window.removeEventListener("offline", onConnectivityChange);
      document.removeEventListener("visibilitychange", onVisibilityChange);
    };
  }, [refreshConnectivity]);

  const connectionMessage = useMemo(() => {
    if (connectivity === "offline") {
      return { icon: WifiOff, title: strings.offlineTitle, body: strings.offlineBody };
    }
    if (connectivity === "server-unreachable") {
      return { icon: ServerOff, title: strings.serverTitle, body: strings.serverBody };
    }
    return null;
  }, [connectivity, strings]);

  if (!connectionMessage && !updateReady) return null;

  return (
    <aside className="pwa-status-stack" aria-label={strings.statusLabel} aria-live="polite">
      {connectionMessage && (
        <section className="pwa-status-card" data-status={connectivity} role="status">
          <connectionMessage.icon size={18} aria-hidden="true" />
          <div>
            <strong>{connectionMessage.title}</strong>
            <p>{connectionMessage.body}</p>
            <button type="button" onClick={() => void refreshConnectivity()}>
              <RefreshCw size={14} aria-hidden="true" /> {strings.retry}
            </button>
          </div>
        </section>
      )}

      {updateReady && (
        <section className="pwa-status-card" data-status="update" role="status">
          <RefreshCw size={18} aria-hidden="true" />
          <div>
            <strong>{strings.updateTitle}</strong>
            <p>{strings.updateBody}</p>
            <div className="pwa-status-actions">
              <button type="button" className="primary-action" onClick={() => void activateWaitingUpdate()}>
                {strings.update}
              </button>
              <button type="button" onClick={() => setUpdateReady(false)}>{strings.later}</button>
            </div>
          </div>
        </section>
      )}
    </aside>
  );
}
