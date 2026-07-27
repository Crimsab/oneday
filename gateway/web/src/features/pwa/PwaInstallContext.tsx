import { createContext, type ReactNode, useContext, useEffect, useMemo, useState } from "react";

interface BeforeInstallPromptEvent extends Event {
  prompt(): Promise<void>;
  userChoice: Promise<{ outcome: "accepted" | "dismissed"; platform: string }>;
}

interface PwaInstallContextValue {
  available: boolean;
  hidden: boolean;
  installed: boolean;
  hide: () => void;
  show: () => void;
  install: () => Promise<void>;
}

const INSTALL_DISMISSED_KEY = "oneday-pwa-install-dismissed-v1";
const PwaInstallContext = createContext<PwaInstallContextValue | null>(null);

export function PwaInstallProvider({ children }: { children: ReactNode }) {
  const [prompt, setPrompt] = useState<BeforeInstallPromptEvent | null>(null);
  const [hidden, setHidden] = useState(() => localStorage.getItem(INSTALL_DISMISSED_KEY) === "true");
  const [installed, setInstalled] = useState(() => window.matchMedia("(display-mode: standalone)").matches);

  useEffect(() => {
    const onInstallPrompt = (event: Event) => {
      event.preventDefault();
      setPrompt(event as BeforeInstallPromptEvent);
    };
    const onInstalled = () => {
      setPrompt(null);
      setInstalled(true);
    };
    window.addEventListener("beforeinstallprompt", onInstallPrompt);
    window.addEventListener("appinstalled", onInstalled);
    return () => {
      window.removeEventListener("beforeinstallprompt", onInstallPrompt);
      window.removeEventListener("appinstalled", onInstalled);
    };
  }, []);

  const value = useMemo<PwaInstallContextValue>(() => ({
    available: Boolean(prompt),
    hidden,
    installed,
    hide: () => {
      localStorage.setItem(INSTALL_DISMISSED_KEY, "true");
      setHidden(true);
    },
    show: () => {
      localStorage.removeItem(INSTALL_DISMISSED_KEY);
      setHidden(false);
    },
    install: async () => {
      if (!prompt) return;
      await prompt.prompt();
      const choice = await prompt.userChoice;
      if (choice.outcome === "accepted") setPrompt(null);
    },
  }), [hidden, installed, prompt]);

  return <PwaInstallContext.Provider value={value}>{children}</PwaInstallContext.Provider>;
}

export function usePwaInstall(): PwaInstallContextValue {
  const value = useContext(PwaInstallContext);
  if (!value) throw new Error("usePwaInstall must be used inside PwaInstallProvider");
  return value;
}
