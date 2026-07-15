import { useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { Search } from "lucide-react";
import { useTranslation } from "react-i18next";
import { searchSettings, settingsCategories, settingsSearchEntries, type SettingsSearchEntry, type SettingsSectionId } from "./settingsRegistry";

export interface SettingsSection {
  id: SettingsSectionId;
  content: ReactNode;
}

export function SettingsWorkspace({ sections, initialSection = "general" }: { sections: SettingsSection[]; initialSection?: SettingsSectionId }) {
  const { t } = useTranslation(["options", "common", "settings_search"]);
  const [active, setActive] = useState<SettingsSectionId>(initialSection);
  const [query, setQuery] = useState("");
  const [pendingFocus, setPendingFocus] = useState<SettingsSearchEntry | null>(null);
  const contentRef = useRef<HTMLDivElement>(null);
  const results = useMemo(() => {
    if (!query.trim()) return [];
    const normalized = query.toLocaleLowerCase();
    const ids = new Set(searchSettings(query).map((entry) => entry.id));
    for (const entry of settingsSearchEntries) {
      const localized = `${t(`settings_search:${entry.id}.0`)} ${t(`settings_search:${entry.id}.1`)}`.toLocaleLowerCase();
      if (localized.includes(normalized)) ids.add(entry.id);
    }
    return settingsSearchEntries.filter((entry) => ids.has(entry.id));
  }, [query, t]);
  const category = settingsCategories.find((item) => item.id === active) ?? settingsCategories[0];
  const section = sections.find((item) => item.id === active);

  useEffect(() => {
    setActive(initialSection);
    setQuery("");
  }, [initialSection]);

  useEffect(() => {
    if (!pendingFocus || query) return;
    const target = contentRef.current?.querySelector<HTMLElement>(`[data-setting-id="${pendingFocus.id}"]`);
    const focusTarget = target?.matches("input,select,button,textarea") ? target : target?.querySelector<HTMLElement>("input,select,button,textarea");
    (focusTarget ?? contentRef.current?.querySelector<HTMLElement>("h3"))?.focus();
    target?.classList.add("settings-search-target");
    const timeout = window.setTimeout(() => target?.classList.remove("settings-search-target"), 1400);
    setPendingFocus(null);
    return () => window.clearTimeout(timeout);
  }, [pendingFocus, query]);

  const openResult = (result: SettingsSearchEntry) => {
    setActive(result.section);
    setPendingFocus(result);
    setQuery("");
  };

  return (
    <div className="settings-workspace">
      <aside className="settings-sidebar" aria-label={t("options:categories")}>
        <div className="settings-sidebar-title">
          <strong>{t("options:title")}</strong>
          <span>{t("options:choose")}</span>
        </div>
        <nav>
          {settingsCategories.map((item) => (
            <button key={item.id} type="button" className={active === item.id && !query ? "active" : ""} aria-current={active === item.id && !query ? "page" : undefined} onClick={() => { setActive(item.id); setQuery(""); }}>
              <strong>{t(`options:${item.id}`)}</strong>
              <span>{t(`options:${item.id}Desc`)}</span>
            </button>
          ))}
        </nav>
      </aside>
      <div className="settings-main">
        <div className="settings-toolbar">
          <label>
            <Search size={16} aria-hidden="true" />
            <span className="sr-only">{t("options:search")}</span>
            <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder={t("options:search")} type="search" />
          </label>
          {query && <button type="button" onClick={() => setQuery("")}>{t("common:clear")}</button>}
        </div>
        <div className="settings-content" ref={contentRef}>
          {query ? (
            <SettingsSearchResults query={query} results={results} onOpen={openResult} />
          ) : (
            <section className="settings-section" aria-labelledby={`settings-${active}-title`}>
              <header>
                <h3 id={`settings-${active}-title`} tabIndex={-1}>{t(`options:${category.id}`)}</h3>
                <p>{t(`options:${category.id}Desc`)}</p>
              </header>
              {section?.content}
            </section>
          )}
        </div>
      </div>
    </div>
  );
}

function SettingsSearchResults({ query, results, onOpen }: { query: string; results: SettingsSearchEntry[]; onOpen: (result: SettingsSearchEntry) => void }) {
  const { t } = useTranslation(["options", "common"]);
  return (
    <section className="settings-search-results" aria-live="polite">
      <header><h3>{t("options:results")}</h3><p>{results.length ? t("common:match", { count: results.length, query }) : t("options:noMatch", { query })}</p></header>
      {results.length > 0 && <div className="settings-result-list">{results.map((result) => {
        const category = settingsCategories.find((item) => item.id === result.section);
        return <button type="button" key={result.id} onClick={() => onOpen(result)}><span>{category ? t(`options:${category.id}`) : ""}</span><strong>{t(`settings_search:${result.id}.0`)}</strong><small>{t(`settings_search:${result.id}.1`)}</small></button>;
      })}</div>}
    </section>
  );
}
