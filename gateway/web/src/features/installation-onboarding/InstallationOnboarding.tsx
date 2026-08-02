import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { ChevronDown } from "lucide-react";
import type {
  AppPreferences,
  SetupReadinessProbe,
  SetupReadinessReport,
} from "../../types";
import { CustomSelect } from "../../components/CustomSelect";
import { languageCatalog } from "../translation/languageCatalog";

type SetupState = "ready" | "warning" | "failed" | "skipped" | "unknown";
type SetupProbeName =
  | "narrative"
  | "embeddings"
  | "image"
  | "tts"
  | "gateway"
  | "storage"
  | "backup";

interface SetupItem {
  name: SetupProbeName;
  state: SetupState;
  required: boolean;
  code: string;
  summary: string;
  action: string;
}

const canonicalProbeOrder: SetupProbeName[] = [
  "narrative",
  "embeddings",
  "image",
  "tts",
  "gateway",
  "storage",
  "backup",
];
const essentialProbeNames = new Set<SetupProbeName>(["narrative", "storage"]);

export function installationSetupItems(
  readiness: SetupReadinessReport,
): SetupItem[] {
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
      action: probe?.action ?? "",
    };
  });
}

function isSetupState(value: string | undefined): value is SetupState {
  return (
    value === "ready" ||
    value === "warning" ||
    value === "failed" ||
    value === "skipped"
  );
}

export function InstallationOnboarding({
  readiness,
  preferences,
  onPreferencesChange,
  onConfigure,
  onStartStory,
  onRetry,
  reopened = false,
}: {
  readiness: SetupReadinessReport;
  preferences: AppPreferences;
  onPreferencesChange: (preferences: AppPreferences) => void;
  onConfigure: () => void;
  onStartStory: () => void;
  onRetry: () => void;
  reopened?: boolean;
}) {
  const { t, i18n } = useTranslation("installation");
  const items = useMemo(() => installationSetupItems(readiness), [readiness]);
  const essentials = items.filter((item) => essentialProbeNames.has(item.name));
  const optional = items.filter((item) => !essentialProbeNames.has(item.name));
  const readyEssentials = essentials.filter(
    (item) => item.state === "ready",
  ).length;
  const hasEssentialFailure = essentials.some(
    (item) => item.state === "failed",
  );
  const storyLanguages = useMemo(
    () =>
      languageCatalog(i18n.language).map((language) => ({
        value: language.code,
        label: language.name,
      })),
    [i18n.language],
  );

  return (
    <section
      className="installation-onboarding setup-console"
      aria-labelledby="installation-onboarding-title"
    >
      <header className="setup-console-header">
        <div>
          <p className="installation-onboarding-label">
            {t(reopened ? "reopened.label" : "label")}
          </p>
          <h1 id="installation-onboarding-title">
            {t(reopened ? "reopened.title" : "title")}
          </h1>
          <p>{t(reopened ? "reopened.description" : "description")}</p>
        </div>
        <span
          className={`setup-console-status ${hasEssentialFailure ? "blocked" : readyEssentials === essentials.length ? "ready" : "attention"}`}
          role="status"
        >
          {t("progress.value", {
            ready: readyEssentials,
            total: essentials.length,
          })}
        </span>
      </header>

      <SetupGroup
        title={t("groups.essential.title")}
        description={t("groups.essential.description")}
        items={essentials}
        essential
        t={t}
      />
      <section
        className="setup-console-group setup-story-language"
        aria-labelledby="setup-story-language-title"
      >
        <header>
          <h2 id="setup-story-language-title">{t("storyLanguage.title")}</h2>
          <p>{t("storyLanguage.description")}</p>
        </header>
        <label>
          <span>{t("storyLanguage.label")}</span>
          <CustomSelect
            value={preferences.defaultStoryLanguage}
            ariaLabel={t("storyLanguage.label")}
            onChange={(defaultStoryLanguage) =>
              onPreferencesChange({ ...preferences, defaultStoryLanguage })
            }
            options={storyLanguages}
          />
        </label>
      </section>
      <SetupGroup
        title={t("groups.optional.title")}
        description={t("groups.optional.description")}
        items={optional}
        collapsible
        t={t}
      />

      <details className="setup-console-guidance">
        <summary>
          <span>{t("guidance.title")}</span>
          <ChevronDown size={16} aria-hidden="true" />
        </summary>
        <div>
          <p>{t("guidance.web")}</p>
          <p>
            {t("guidance.cliBefore")} <code>oneday setup --reconfigure</code>{" "}
            {t("guidance.cliBetween")} <code>oneday doctor</code>
            {t("guidance.cliAfter")}
          </p>
        </div>
      </details>

      {hasEssentialFailure && (
        <p
          className="installation-onboarding-blocked"
          role="status"
          aria-live="polite"
        >
          {t("requiredBlocked")}
        </p>
      )}
      <footer className="installation-onboarding-actions">
        <button type="button" onClick={onRetry}>
          {t("recovery.retry")}
        </button>
        <button
          type="button"
          className={hasEssentialFailure ? "primary-action" : undefined}
          onClick={onConfigure}
        >
          {t("configure")}
        </button>
        <button
          type="button"
          className="primary-action"
          onClick={onStartStory}
          disabled={!reopened && hasEssentialFailure}
        >
          {t(reopened ? "reopened.complete" : "startStory")}
        </button>
      </footer>
      <p className="installation-onboarding-preserve">
        {t(reopened ? "reopened.preserve" : "preserve")}
      </p>
    </section>
  );
}

function SetupRows({
  items,
  essential,
  t,
}: {
  items: SetupItem[];
  essential: boolean;
  t: (key: string, options?: Record<string, unknown>) => string;
}) {
  return (
    <ul>
      {items.map((item) => (
        <li key={item.name} className={`setup-console-row ${item.state}`}>
          <div className="setup-console-row-copy">
            <strong>{t(`items.${item.name}.title`)}</strong>
            <span>
              {item.code
                ? t(`codes.${item.code}`, {
                    defaultValue: item.summary || t("summaryUnavailable"),
                  })
                : item.summary || t("summaryUnavailable")}
            </span>
            {item.action && (
              <details>
                <summary>
                  <span>{t("recovery.details")}</span>
                  <ChevronDown size={14} aria-hidden="true" />
                </summary>
                <p>
                  {t(`recovery.actions.${item.action}`, {
                    defaultValue: t("recovery.unknown"),
                  })}
                </p>
              </details>
            )}
          </div>
          <span
            className="setup-console-row-state"
            aria-label={`${t(`items.${item.name}.title`)}: ${t(`states.${essential ? "essential" : "optional"}.${item.state}`)}`}
          >
            {t(`states.${essential ? "essential" : "optional"}.${item.state}`)}
          </span>
        </li>
      ))}
    </ul>
  );
}

function SetupGroup({
  title,
  description,
  items,
  essential = false,
  collapsible = false,
  t,
}: {
  title: string;
  description: string;
  items: SetupItem[];
  essential?: boolean;
  collapsible?: boolean;
  t: (key: string, options?: Record<string, unknown>) => string;
}) {
  if (collapsible)
    return (
      <details className="setup-console-group setup-console-optional">
        <summary>
          <span>
            <strong>{title}</strong>
            <small>{description}</small>
          </span>
          <span className="setup-console-disclosure-end">
            <span>{items.length}</span>
            <ChevronDown size={16} aria-hidden="true" />
          </span>
        </summary>
        <SetupRows items={items} essential={essential} t={t} />
      </details>
    );
  return (
    <section className="setup-console-group" aria-label={title}>
      <header>
        <h2>{title}</h2>
        <p>{description}</p>
      </header>
      <SetupRows items={items} essential={essential} t={t} />
    </section>
  );
}

export function InstallationReadinessPending() {
  const { t } = useTranslation("installation");
  return (
    <section
      className="installation-onboarding installation-readiness-pending"
      aria-labelledby="installation-readiness-loading-title"
      aria-busy="true"
    >
      <p className="installation-onboarding-label">{t("label")}</p>
      <h1 id="installation-readiness-loading-title">{t("loading.title")}</h1>
      <p role="status" aria-live="polite">
        {t("loading.description")}
      </p>
      <div className="installation-readiness-skeleton" aria-hidden="true">
        <span />
        <span />
        <span />
      </div>
    </section>
  );
}

export function InstallationReadinessError({
  onRetry,
}: {
  onRetry: () => void;
}) {
  const { t } = useTranslation("installation");
  return (
    <section
      className="installation-onboarding installation-readiness-error"
      aria-labelledby="installation-readiness-error-title"
    >
      <p className="installation-onboarding-label">{t("label")}</p>
      <h1 id="installation-readiness-error-title">{t("loading.errorTitle")}</h1>
      <p role="alert">{t("loading.errorDescription")}</p>
      <div className="installation-onboarding-actions">
        <button type="button" className="primary-action" onClick={onRetry}>
          {t("loading.retry")}
        </button>
      </div>
    </section>
  );
}
