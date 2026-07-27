import { Download, EyeOff, MonitorDown } from "lucide-react";
import { useTranslation } from "react-i18next";
import { usePwaInstall } from "./PwaInstallContext";

export function PwaInstallSettings() {
  const { t } = useTranslation("settings_ui");
  const { available, hidden, installed, hide, install } = usePwaInstall();

  if (hidden || installed) return null;

  return (
    <section className="settings-group pwa-install-settings" aria-labelledby="pwa-install-title" data-setting-id="pwa-installation">
      <header>
        <div>
          <h4 id="pwa-install-title"><MonitorDown size={17} aria-hidden="true" /> {t("advanced.pwaTitle")}</h4>
          <p>{t("advanced.pwaDesc")}</p>
        </div>
      </header>
      <div className="pwa-install-settings-actions">
        {available
          ? <button type="button" className="primary-action" onClick={() => void install()}><Download size={14} aria-hidden="true" /> {t("advanced.pwaInstall")}</button>
          : <p className="pwa-install-unavailable">{t("advanced.pwaUnavailable")}</p>}
        <button type="button" onClick={hide}><EyeOff size={14} aria-hidden="true" /> {t("advanced.pwaHide")}</button>
      </div>
    </section>
  );
}
