import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import type { SetupReadinessProbe, SetupReadinessReport } from "../../types";

type SetupState = "ready" | "warning" | "failed" | "skipped" | "unknown";
type SetupProbeName = "narrative" | "embeddings" | "image" | "tts" | "gateway" | "storage" | "backup";

interface SetupItem { name: SetupProbeName; state: SetupState; required: boolean; code: string; summary: string; action: string; }

const canonicalProbeOrder: SetupProbeName[] = ["narrative", "embeddings", "image", "tts", "gateway", "storage", "backup"];
const essentialProbeNames = new Set<SetupProbeName>(["narrative", "storage", "gateway"]);

export function installationSetupItems(readiness: SetupReadinessReport): SetupItem[] {
  const probeFor = (name: SetupProbeName): SetupReadinessProbe | undefined => readiness.probes.find((probe) => probe.name === name);
  return canonicalProbeOrder.map((name) => {
    const probe = probeFor(name);
    return { name, state: isSetupState(probe?.status) ? probe.status : "unknown", required: probe?.required ?? (name === "narrative" || name === "storage"), code: probe?.code ?? "", summary: probe?.summary ?? "", action: probe?.action ?? "" };
  });
}

function isSetupState(value: string | undefined): value is SetupState {
  return value === "ready" || value === "warning" || value === "failed" || value === "skipped";
}

export function InstallationOnboarding({
  readiness, onConfigure, onStartStory, onRetry, reopened = false,
}: {
  readiness: SetupReadinessReport;
  onConfigure: () => void;
  onStartStory: () => void;
  onRetry: () => void;
  reopened?: boolean;
}) {
  const { t } = useTranslation("installation");
  const items = useMemo(() => installationSetupItems(readiness), [readiness]);
  const essentials = items.filter((item) => essentialProbeNames.has(item.name));
  const optional = items.filter((item) => !essentialProbeNames.has(item.name));
  const readyEssentials = essentials.filter((item) => item.state === "ready").length;
  const hasEssentialFailure = essentials.some((item) => item.state === "failed");

  return (
    <section className="installation-onboarding setup-console" aria-labelledby="installation-onboarding-title">
      <header className="setup-console-header">
        <div>
          <p className="installation-onboarding-label">{t(reopened ? "reopened.label" : "label")}</p>
          <h1 id="installation-onboarding-title">{t(reopened ? "reopened.title" : "title")}</h1>
          <p>{t(reopened ? "reopened.description" : "description")}</p>
        </div>
        <span className={`setup-console-status ${hasEssentialFailure ? "blocked" : readyEssentials === essentials.length ? "ready" : "attention"}`} role="status">
          {t("progress.value", { ready: readyEssentials, total: essentials.length })}
        </span>
      </header>

      <SetupGroup title={t("groups.essential.title")} description={t("groups.essential.description")} items={essentials} essential t={t} />
      <SetupGroup title={t("groups.optional.title")} description={t("groups.optional.description")} items={optional} t={t} />

      <section className="setup-console-guidance" aria-labelledby="setup-console-guidance-title">
        <h2 id="setup-console-guidance-title">{t("guidance.title")}</h2>
        <p>{t("guidance.web")}</p>
        <p>{t("guidance.cliBefore")} <code>oneday setup --reconfigure</code> {t("guidance.cliBetween")} <code>oneday doctor</code>{t("guidance.cliAfter")}</p>
      </section>

      {hasEssentialFailure && <p className="installation-onboarding-blocked" role="status" aria-live="polite">{t("requiredBlocked")}</p>}
      <footer className="installation-onboarding-actions">
        <button type="button" onClick={onRetry}>{t("recovery.retry")}</button>
        <button type="button" onClick={onConfigure}>{t("configure")}</button>
        <button type="button" className="primary-action" onClick={onStartStory} disabled={!reopened && hasEssentialFailure}>
          {t(reopened ? "reopened.complete" : "startStory")}
        </button>
      </footer>
      <p className="installation-onboarding-preserve">{t(reopened ? "reopened.preserve" : "preserve")}</p>
    </section>
  );
}

function SetupGroup({ title, description, items, essential = false, t }: { title: string; description: string; items: SetupItem[]; essential?: boolean; t: (key: string, options?: Record<string, unknown>) => string }) {
  return <section className="setup-console-group" aria-label={title}>
    <header><h2>{title}</h2><p>{description}</p></header>
    <ul>
      {items.map((item) => <li key={item.name} className={`setup-console-row ${item.state}`}>
        <div className="setup-console-row-copy"><strong>{t(`items.${item.name}.title`)}</strong><span>{item.code ? t(`codes.${item.code}`, { defaultValue: item.summary || t("summaryUnavailable") }) : item.summary || t("summaryUnavailable")}</span>{item.action && <details><summary>{t("recovery.details")}</summary><p>{t(`recovery.actions.${item.action}`, { defaultValue: t("recovery.unknown") })}</p></details>}</div>
        <span className="setup-console-row-state" aria-label={`${t(`items.${item.name}.title`)}: ${t(`states.${essential ? "essential" : "optional"}.${item.state}`)}`}>{t(`states.${essential ? "essential" : "optional"}.${item.state}`)}</span>
      </li>)}
    </ul>
  </section>;
}

export function InstallationReadinessPending() {
  const { t } = useTranslation("installation");
  return <section className="installation-onboarding installation-readiness-pending" aria-labelledby="installation-readiness-loading-title" aria-busy="true"><p className="installation-onboarding-label">{t("label")}</p><h1 id="installation-readiness-loading-title">{t("loading.title")}</h1><p role="status" aria-live="polite">{t("loading.description")}</p><div className="installation-readiness-skeleton" aria-hidden="true"><span /><span /><span /></div></section>;
}

export function InstallationReadinessError({ onRetry }: { onRetry: () => void }) {
  const { t } = useTranslation("installation");
  return <section className="installation-onboarding installation-readiness-error" aria-labelledby="installation-readiness-error-title"><p className="installation-onboarding-label">{t("label")}</p><h1 id="installation-readiness-error-title">{t("loading.errorTitle")}</h1><p role="alert">{t("loading.errorDescription")}</p><div className="installation-onboarding-actions"><button type="button" className="primary-action" onClick={onRetry}>{t("loading.retry")}</button></div></section>;
}
