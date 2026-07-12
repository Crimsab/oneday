import { useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { Search } from "lucide-react";
import { searchSettings, settingsCategories, type SettingsSearchEntry, type SettingsSectionId } from "./settingsRegistry";

export interface SettingsSection {
  id: SettingsSectionId;
  content: ReactNode;
}

export function SettingsWorkspace({ sections, initialSection = "general" }: { sections: SettingsSection[]; initialSection?: SettingsSectionId }) {
  const [active, setActive] = useState<SettingsSectionId>(initialSection);
  const [query, setQuery] = useState("");
  const [pendingFocus, setPendingFocus] = useState<SettingsSearchEntry | null>(null);
  const contentRef = useRef<HTMLDivElement>(null);
  const results = useMemo(() => searchSettings(query), [query]);
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
      <aside className="settings-sidebar" aria-label="Options categories">
        <div className="settings-sidebar-title">
          <strong>Options</strong>
          <span>Choose an area</span>
        </div>
        <nav>
          {settingsCategories.map((item) => (
            <button key={item.id} type="button" className={active === item.id && !query ? "active" : ""} aria-current={active === item.id && !query ? "page" : undefined} onClick={() => { setActive(item.id); setQuery(""); }}>
              <strong>{item.title}</strong>
              <span>{item.description}</span>
            </button>
          ))}
        </nav>
      </aside>
      <div className="settings-main">
        <div className="settings-toolbar">
          <label>
            <Search size={16} aria-hidden="true" />
            <span className="sr-only">Search options</span>
            <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search options" type="search" />
          </label>
          {query && <button type="button" onClick={() => setQuery("")}>Clear</button>}
        </div>
        <div className="settings-content" ref={contentRef}>
          {query ? (
            <SettingsSearchResults query={query} results={results} onOpen={openResult} />
          ) : (
            <section className="settings-section" aria-labelledby={`settings-${active}-title`}>
              <header>
                <h3 id={`settings-${active}-title`} tabIndex={-1}>{category.title}</h3>
                <p>{category.description}</p>
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
  return (
    <section className="settings-search-results" aria-live="polite">
      <header><h3>Search results</h3><p>{results.length ? `${results.length} options match “${query}”.` : `No options match “${query}”.`}</p></header>
      {results.length > 0 && <div className="settings-result-list">{results.map((result) => {
        const category = settingsCategories.find((item) => item.id === result.section);
        return <button type="button" key={result.id} onClick={() => onOpen(result)}><span>{category?.title}</span><strong>{result.label}</strong><small>{result.description}</small></button>;
      })}</div>}
    </section>
  );
}
