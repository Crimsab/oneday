import { BookOpen, Globe2, ListChecks, ShieldCheck } from "lucide-react";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import type { JsonValue, StoryWizardResult } from "../../types";

type JsonRecord = Record<string, JsonValue>;

export function StoryWizardReview({ wizard }: { wizard: StoryWizardResult | null }) {
  const { t } = useTranslation("wizard");
  const definition = asRecord(wizard?.definition);
  const setting = asRecord(definition?.setting);
  const stats = asRecord(definition?.stats_schema);

  if (!wizard || wizard.stage === "brief") {
    return <p className="story-wizard-review-intro">{t("messages.brief")}</p>;
  }

  if (!definition) {
    return <div className="story-wizard-review-fallback">{paragraphs(wizard.message).map((value, index) => <p key={`${index}-${value}`}>{value}</p>)}</div>;
  }

  if (wizard.stage === "review_rules") {
    return <div className="story-wizard-review story-wizard-review-lists">
      <ReviewList title={t("review.worldRules")} values={strings(setting?.rules)} />
      <ReviewList title={t("review.factions")} values={strings(setting?.factions)} />
      <ReviewList title={t("review.cultures")} values={strings(setting?.cultures)} />
      <ReviewList title={t("review.dangers")} values={strings(setting?.dangers)} />
    </div>;
  }

  if (wizard.stage === "review_stats") {
    return <div className="story-wizard-review story-wizard-review-stats">
      <ReviewSection icon={<ShieldCheck size={17} />} title={t("review.rulesAndStats")}>
        <ReviewFacts values={[
          [t("review.combat"), booleanText(stats?.has_combat, t("review.yes"), t("review.no"))],
          [t("review.currency"), currencyText(stats?.currency) || t("review.none")],
        ]} />
      </ReviewSection>
      <ReviewList title={t("review.vitals")} values={statStrings(stats?.vitals)} />
      <ReviewList title={t("review.attributes")} values={statStrings(stats?.attributes)} />
      <ReviewList title={t("review.secondaryStats")} values={statStrings(stats?.secondary)} />
    </div>;
  }

  const isFinal = wizard.stage === "confirm";
  return <div className={`story-wizard-review ${isFinal ? "is-final" : ""}`}>
    <ReviewSection icon={<BookOpen size={17} />} title={t("review.storyIdentity")}>
      <ReviewFacts values={[
        [t("review.story"), string(definition.name)],
        [t("review.genre"), string(definition.genre)],
        [t("review.tone"), string(definition.tone)],
        [t("review.language"), string(definition.language)],
        [t("review.writingStyle"), string(definition.writing_style)],
      ]} />
      {string(definition.prompt_directives) && <div className="story-wizard-review-note"><strong>{t("review.directives")}</strong><p>{string(definition.prompt_directives)}</p></div>}
    </ReviewSection>

    <ReviewSection icon={<Globe2 size={17} />} title={t("review.world") }>
      <ReviewFacts values={[
        [t("review.worldName"), string(setting?.world_name)],
        [t("review.era"), string(setting?.era)],
        [t("review.geography"), string(setting?.geography)],
        [t("review.magicSystem"), string(setting?.magic_system)],
        [t("review.technology"), string(setting?.technology_level)],
        [t("review.society"), string(setting?.society)],
      ]} />
    </ReviewSection>

    {string(definition.description) && <ReviewSection icon={<ListChecks size={17} />} title={t("review.description")} wide>
      <p className="story-wizard-review-description">{string(definition.description)}</p>
    </ReviewSection>}

    {isFinal && <ReviewSection icon={<ShieldCheck size={17} />} title={t("review.finalChecks")} wide>
      <div className="story-wizard-review-counts">
        <Count value={strings(setting?.rules).length} label={t("review.worldRules")} />
        <Count value={strings(setting?.factions).length} label={t("review.factions")} />
        <Count value={strings(setting?.cultures).length} label={t("review.cultures")} />
        <Count value={strings(setting?.dangers).length} label={t("review.dangers")} />
      </div>
    </ReviewSection>}
  </div>;
}

function ReviewSection({ icon, title, wide = false, children }: { icon: ReactNode; title: string; wide?: boolean; children: ReactNode }) {
  return <section className={`story-wizard-review-section ${wide ? "is-wide" : ""}`}>
    <header><span aria-hidden="true">{icon}</span><h4>{title}</h4></header>
    {children}
  </section>;
}

function ReviewFacts({ values }: { values: Array<[string, string]> }) {
  const visible = values.filter(([, value]) => Boolean(value));
  if (!visible.length) return null;
  return <dl className="story-wizard-review-facts">{visible.map(([label, value]) => <div key={label}><dt>{label}</dt><dd>{value}</dd></div>)}</dl>;
}

function ReviewList({ title, values }: { title: string; values: string[] }) {
  const { t } = useTranslation("wizard");
  return <ReviewSection icon={<ListChecks size={17} />} title={title}>
    {values.length ? <ul>{values.map((value, index) => <li key={`${index}-${value}`}>{value}</li>)}</ul> : <p className="story-wizard-review-empty">{t("review.none")}</p>}
  </ReviewSection>;
}

function Count({ value, label }: { value: number; label: string }) {
  return <div><strong>{value}</strong><span>{label}</span></div>;
}

function asRecord(value: JsonValue | undefined): JsonRecord | null {
  return value && typeof value === "object" && !Array.isArray(value) ? value as JsonRecord : null;
}

function string(value: JsonValue | undefined): string {
  return typeof value === "string" ? value.trim() : "";
}

function strings(value: JsonValue | undefined): string[] {
  return Array.isArray(value) ? value.map((item) => typeof item === "string" ? item.trim() : "").filter(Boolean) : [];
}

function statStrings(value: JsonValue | undefined): string[] {
  if (!Array.isArray(value)) return [];
  return value.flatMap((item) => {
    const record = asRecord(item);
    if (!record) return [];
    const label = string(record.label) || string(record.key);
    if (!label) return [];
    const starting = typeof record.starting === "number" && record.starting !== 0 ? ` · ${record.starting}` : "";
    return [`${label}${starting}`];
  });
}

function currencyText(value: JsonValue | undefined): string {
  if (typeof value === "string") return value.trim();
  const record = asRecord(value);
  if (!record) return "";
  const name = string(record.name) || string(record.label) || string(record.key);
  const starting = typeof record.starting === "number" && record.starting !== 0 ? ` · ${record.starting}` : "";
  return `${name}${starting}`.trim();
}

function booleanText(value: JsonValue | undefined, yes: string, no: string): string {
  return value === true ? yes : no;
}

function paragraphs(value: string): string[] {
  return value.split(/\n{2,}/).map((part) => part.replace(/\n/g, " ").trim()).filter(Boolean);
}
