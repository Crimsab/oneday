import { Download, Eye, EyeOff, MonitorCheck, MonitorDown } from "lucide-react";
import { useTranslation } from "react-i18next";
import { usePwaInstall } from "./PwaInstallContext";

export function PwaInstallSettings() {
  const { t } = useTranslation("settings_ui");
  const { available, hidden, installed, hide, show, install } = usePwaInstall();

  return (
    <section className="settings-group pwa-install-settings" aria-labelledby="pwa-install-title" data-setting-id="pwa-installation">
      <header>
        <div className="pwa-install-settings-copy">
          <h4 id="pwa-install-title"><MonitorDown size={17} aria-hidden="true" /> {t("advanced.pwaTitle")}</h4>
          <p>{t("advanced.pwaDesc")}</p>
        </div>
        <div className="pwa-install-settings-actions">
          {installed
            ? <p className="pwa-install-state"><MonitorCheck size={15} aria-hidden="true" /> {t("advanced.pwaInstalled")}</p>
            : hidden
              ? <><p className="pwa-install-state">{t("advanced.pwaHidden")}</p><button type="button" onClick={show}><Eye size={14} aria-hidden="true" /> {t("advanced.pwaShow")}</button></>
              : <>
                {available
                  ? <button type="button" className="primary-action" onClick={() => void install()}><Download size={14} aria-hidden="true" /> {t("advanced.pwaInstall")}</button>
                  : <p className="pwa-install-unavailable">{t("advanced.pwaUnavailable")}</p>}
                <button type="button" onClick={hide}><EyeOff size={14} aria-hidden="true" /> {t("advanced.pwaHide")}</button>
              </>}
        </div>
      </header>
    </section>
  );
}
