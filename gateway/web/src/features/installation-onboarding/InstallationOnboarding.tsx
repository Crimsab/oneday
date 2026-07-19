import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import type { SetupReadinessProbe, SetupReadinessReport } from "../../types";

type SetupState = "ready" | "warning" | "failed" | "skipped" | "unknown";

interface SetupItem {
  id: "narrative" | "images" | "voice";
  state: SetupState;
  required: boolean;
  summary: string;
}

const probesForItems: Record<SetupItem["id"], string> = {
  narrative: "narrative",
  images: "image",
  voice: "tts",
};

export function installationSetupItems(readiness: SetupReadinessReport): SetupItem[] {
  const probeFor = (id: SetupItem["id"]): SetupReadinessProbe | undefined =>
    readiness.probes.find((probe) => probe.name === probesForItems[id]);

  return (Object.keys(probesForItems) as SetupItem["id"][]).map((id) => {
    const probe = probeFor(id);
    return {
      id,
      state: isSetupState(probe?.status) ? probe.status : "unknown",
      required: probe?.required ?? id === "narrative",
      summary: probe?.summary ?? "",
    };
  });
}

function isSetupState(value: string | undefined): value is SetupState {
  return value === "ready" || value === "warning" || value === "failed" || value === "skipped";
}

export function InstallationOnboarding({
  readiness,
  onConfigure,
  onStartStory,
}: {
  readiness: SetupReadinessReport;
  onConfigure: () => void;
  onStartStory: () => void;
}) {
  const { t } = useTranslation("installation");
  const items = useMemo(() => installationSetupItems(readiness), [readiness]);
  const narrativeReady = items.find((item) => item.id === "narrative")?.state === "ready";

  return (
    <section className="installation-onboarding" aria-labelledby="installation-onboarding-title">
      <div className="installation-onboarding-intro">
        <p className="installation-onboarding-label">{t("label")}</p>
        <h2 id="installation-onboarding-title">{t("title")}</h2>
        <p>{t("description")}</p>
        <p className="installation-onboarding-preserve">{t("preserve")}</p>
      </div>

      <section className="installation-readiness" aria-labelledby="installation-readiness-title">
        <div>
          <h3 id="installation-readiness-title">{t("summary.title")}</h3>
          <p>{t("summary.description")}</p>
        </div>
        <ul>
          {items.map((item) => (
            <li key={item.id} className={`installation-readiness-item ${item.state}`}>
              <div>
                <strong>{t(`items.${item.id}.title`)}</strong>
                <span>{item.summary || t("summaryUnavailable")}</span>
              </div>
              <span>{item.required ? t(`states.required.${item.state}`) : t(`states.optional.${item.state}`)}</span>
            </li>
          ))}
        </ul>
      </section>

      <div className="installation-onboarding-choices">
        <article>
          <h3>{t("images.title")}</h3>
          <p>{t("images.description")}</p>
          <button type="button" onClick={onConfigure}>{t("images.action")}</button>
        </article>
        <article>
          <h3>{t("voice.title")}</h3>
          <p>{t("voice.description")}</p>
          <button type="button" onClick={onConfigure}>{t("voice.action")}</button>
        </article>
      </div>

      <div className="installation-onboarding-actions">
        <button type="button" onClick={onConfigure}>{t("configure")}</button>
        <button type="button" className="primary-action" onClick={onStartStory} disabled={!narrativeReady}>
          {t("startStory")}
        </button>
      </div>
      {!narrativeReady && <p className="installation-onboarding-blocked" role="status">{t("narrativeRequired")}</p>}
    </section>
  );
}
