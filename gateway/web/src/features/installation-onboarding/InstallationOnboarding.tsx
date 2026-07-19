import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import type { SetupReadinessProbe, SetupReadinessReport } from "../../types";

type SetupState = "ready" | "warning" | "failed" | "skipped" | "unknown";
type SetupProbeName = "narrative" | "embeddings" | "image" | "tts" | "gateway" | "storage" | "backup";

interface SetupItem {
  name: SetupProbeName;
  state: SetupState;
  required: boolean;
  code: string;
  summary: string;
}

const canonicalProbeOrder: SetupProbeName[] = ["narrative", "embeddings", "image", "tts", "gateway", "storage", "backup"];

export function installationSetupItems(readiness: SetupReadinessReport): SetupItem[] {
  const probeFor = (name: SetupProbeName): SetupReadinessProbe | undefined =>
    readiness.probes.find((probe) => probe.name === name);

  return canonicalProbeOrder.map((name) => {
    const probe = probeFor(name);
    return {
      name,
      state: isSetupState(probe?.status) ? probe.status : "unknown",
      required: probe?.required ?? (name === "narrative" || name === "storage"),
      code: probe?.code ?? "",
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
  const hasRequiredFailure = items.some((item) => item.required && item.state === "failed");

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
            <li key={item.name} className={`installation-readiness-item ${item.state}`}>
              <div>
                <strong>{t(`items.${item.name}.title`)}</strong>
                <span>{t(`codes.${item.code}`, { defaultValue: item.summary || t("summaryUnavailable") })}</span>
                {item.code && <code>{item.code}</code>}
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
        <button type="button" className="primary-action" onClick={onStartStory} disabled={hasRequiredFailure}>
          {t("startStory")}
        </button>
      </div>
      {hasRequiredFailure && <p className="installation-onboarding-blocked" role="status">{t("requiredBlocked")}</p>}
    </section>
  );
}
