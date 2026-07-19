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
        <h1 id="installation-onboarding-title">{t("title")}</h1>
        <p>{t("description")}</p>
        <p className="installation-onboarding-preserve">{t("preserve")}</p>
      </div>

      <section className="installation-readiness" aria-labelledby="installation-readiness-title">
        <div>
          <h2 id="installation-readiness-title">{t("summary.title")}</h2>
          <p id="installation-readiness-description">{t("summary.description")}</p>
        </div>
        <ul aria-describedby="installation-readiness-description">
          {items.map((item) => (
            <li key={item.name} className={`installation-readiness-item ${item.state}`}>
              <div>
                <strong>{t(`items.${item.name}.title`)}</strong>
                <span>{t(`codes.${item.code}`, { defaultValue: item.summary || t("summaryUnavailable") })}</span>
                {item.code && <code>{item.code}</code>}
              </div>
              <span aria-label={`${t(`items.${item.name}.title`)}: ${item.required ? t(`states.required.${item.state}`) : t(`states.optional.${item.state}`)}`}>
                {item.required ? t(`states.required.${item.state}`) : t(`states.optional.${item.state}`)}
              </span>
            </li>
          ))}
        </ul>
      </section>

      <div className="installation-onboarding-choices">
        <article>
          <h2>{t("images.title")}</h2>
          <p>{t("images.description")}</p>
          <button type="button" onClick={onConfigure}>{t("images.action")}</button>
        </article>
        <article>
          <h2>{t("voice.title")}</h2>
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
      {hasRequiredFailure && (
        <p className="installation-onboarding-blocked" role="status" aria-live="polite">
          {t("requiredBlocked")}
        </p>
      )}
    </section>
  );
}

export function InstallationReadinessPending() {
  const { t } = useTranslation("installation");
  return (
    <section className="installation-onboarding installation-readiness-pending" aria-labelledby="installation-readiness-loading-title" aria-busy="true">
      <p className="installation-onboarding-label">{t("label")}</p>
      <h1 id="installation-readiness-loading-title">{t("loading.title")}</h1>
      <p role="status" aria-live="polite">{t("loading.description")}</p>
      <div className="installation-readiness-skeleton" aria-hidden="true"><span /><span /><span /></div>
    </section>
  );
}

export function InstallationReadinessError({ onRetry }: { onRetry: () => void }) {
  const { t } = useTranslation("installation");
  return (
    <section className="installation-onboarding installation-readiness-error" aria-labelledby="installation-readiness-error-title">
      <p className="installation-onboarding-label">{t("label")}</p>
      <h1 id="installation-readiness-error-title">{t("loading.errorTitle")}</h1>
      <p role="alert">{t("loading.errorDescription")}</p>
      <div className="installation-onboarding-actions"><button type="button" className="primary-action" onClick={onRetry}>{t("loading.retry")}</button></div>
    </section>
  );
}
