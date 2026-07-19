import { useCallback, useEffect, useMemo, useState } from "react";
import { Download, RefreshCw, ServerOff, WifiOff } from "lucide-react";
import { useTranslation } from "react-i18next";
import { activateWaitingUpdate, subscribeToPwaUpdates } from "./pwaRuntime";
import "./pwaStatus.css";

type Connectivity = "connected" | "offline" | "server-unreachable";

interface BeforeInstallPromptEvent extends Event {
  prompt(): Promise<void>;
  userChoice: Promise<{ outcome: "accepted" | "dismissed"; platform: string }>;
}

const INSTALL_DISMISSED_KEY = "oneday-pwa-install-dismissed-v1";

const copy = {
  en: {
    statusLabel: "OneDay app status",
    installTitle: "Install OneDay",
    installBody: "Open OneDay like a desktop app while keeping stories safely on your server.",
    install: "Install",
    updateTitle: "Update available",
    updateBody: "A new OneDay build is ready. Apply it when you are ready to reload.",
    update: "Update now",
    later: "Later",
    offlineTitle: "You are offline",
    offlineBody: "The app shell is available, but stories and changes still require the OneDay server.",
    serverTitle: "Server unavailable",
    serverBody: "Your stories remain on the server. Reconnect or retry when it is reachable.",
    retry: "Retry",
    dismiss: "Dismiss",
  },
  it: {
    statusLabel: "Stato dell'app OneDay",
    installTitle: "Installa OneDay",
    installBody: "Apri OneDay come app desktop mantenendo le storie al sicuro sul tuo server.",
    install: "Installa",
    updateTitle: "Aggiornamento disponibile",
    updateBody: "Una nuova versione di OneDay è pronta. Applicala quando puoi ricaricare l'app.",
    update: "Aggiorna ora",
    later: "Più tardi",
    offlineTitle: "Sei offline",
    offlineBody: "L'interfaccia resta disponibile, ma storie e modifiche richiedono il server OneDay.",
    serverTitle: "Server non raggiungibile",
    serverBody: "Le storie restano sul server. Riconnettiti o riprova quando è disponibile.",
    retry: "Riprova",
    dismiss: "Chiudi",
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
  const [installPrompt, setInstallPrompt] = useState<BeforeInstallPromptEvent | null>(null);
  const [installDismissed, setInstallDismissed] = useState(() => localStorage.getItem(INSTALL_DISMISSED_KEY) === "true");
  const [installOpen, setInstallOpen] = useState(false);
  const [updateReady, setUpdateReady] = useState(false);
  const [connectivity, setConnectivity] = useState<Connectivity>("connected");
  const strings = copy[(i18n.resolvedLanguage?.startsWith("it") ? "it" : "en")];

  const refreshConnectivity = useCallback(async () => {
    setConnectivity(await checkServerConnectivity());
  }, []);

  useEffect(() => subscribeToPwaUpdates(() => setUpdateReady(true)), []);

  useEffect(() => {
    const standalone = window.matchMedia("(display-mode: standalone)").matches;
    if (standalone) return;

    const onInstallPrompt = (event: Event) => {
      event.preventDefault();
      setInstallPrompt(event as BeforeInstallPromptEvent);
    };
    const onInstalled = () => setInstallPrompt(null);
    window.addEventListener("beforeinstallprompt", onInstallPrompt);
    window.addEventListener("appinstalled", onInstalled);
    return () => {
      window.removeEventListener("beforeinstallprompt", onInstallPrompt);
      window.removeEventListener("appinstalled", onInstalled);
    };
  }, []);

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

  const requestInstall = async () => {
    if (!installPrompt) return;
    await installPrompt.prompt();
    await installPrompt.userChoice;
    setInstallPrompt(null);
    setInstallOpen(false);
  };

  const dismissInstall = () => {
    localStorage.setItem(INSTALL_DISMISSED_KEY, "true");
    setInstallDismissed(true);
    setInstallOpen(false);
  };

  if (!connectionMessage && !updateReady && (!installPrompt || installDismissed)) return null;

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

      {installPrompt && !installDismissed && <div className="pwa-install-control">
        <button type="button" className="pwa-install-trigger" onClick={() => setInstallOpen((value) => !value)} aria-expanded={installOpen} aria-controls="pwa-install-popover" title={strings.installTitle}>
          <Download size={16} aria-hidden="true" /><span className="sr-only">{strings.installTitle}</span>
        </button>
        {installOpen && <section className="pwa-install-popover" id="pwa-install-popover" role="dialog" aria-label={strings.installTitle}>
          <strong>{strings.installTitle}</strong><p>{strings.installBody}</p>
          <div className="pwa-status-actions"><button type="button" className="primary-action" onClick={() => void requestInstall()}>{strings.install}</button><button type="button" onClick={dismissInstall}>{strings.dismiss}</button></div>
        </section>}
      </div>}
    </aside>
  );
}
