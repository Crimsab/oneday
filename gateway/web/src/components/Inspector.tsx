import { ChevronDown, RefreshCw } from "lucide-react";
import { moduleSpecs } from "../commands";
import { asArray, asObject, compactText, deriveCondition, displayClock, entryLabel, fieldRows, findString, numericStat, titleCase, valueToText } from "../format";
import type { JsonObject, JsonValue, ModuleTab, StorySnapshot } from "../types";

interface InspectorProps {
  snapshot: StorySnapshot | null;
  selectedTab: ModuleTab;
  onRefresh: () => void;
}

interface CardView {
  title: string;
  rows: Array<[string, string]>;
}

export function Inspector({ snapshot, selectedTab, onRefresh }: InspectorProps) {
  const title = moduleTitle(selectedTab);

  return (
    <aside className="right-inspector">
      <div className="inspector-head">
        <h2>Canonical State Inspector</h2>
        <span className="inspector-mode">{title}</span>
        <button type="button" className="square-button" onClick={onRefresh} title="Refresh snapshot">
          <RefreshCw size={15} />
        </button>
      </div>
      {!snapshot ? (
        <div className="empty-copy inspector-empty">Select a story to inspect canonical state.</div>
      ) : (
        <div className="inspector-body">
          {renderModule(selectedTab, snapshot)}
        </div>
      )}
    </aside>
  );
}

function renderModule(tab: ModuleTab, snapshot: StorySnapshot) {
  if (tab === "inventory") return <InventoryModule snapshot={snapshot} />;
  if (tab === "stats") return <StatsModule snapshot={snapshot} />;
  if (tab === "codex") return <CodexModule snapshot={snapshot} />;
  if (tab === "fronts") return <FrontsModule snapshot={snapshot} />;
  if (tab === "investigations") return <InvestigationsModule snapshot={snapshot} />;
  if (tab === "projects") return <ProjectsModule snapshot={snapshot} />;
  if (tab === "saves") return <SavesModule snapshot={snapshot} />;
  return <HistoryModule snapshot={snapshot} />;
}

function HistoryModule({ snapshot }: { snapshot: StorySnapshot }) {
  return (
    <>
      <InspectorSection title="Turn & Time" rows={turnRows(snapshot)} />
      <InspectorSection title="Location" rows={locationRows(snapshot)} />
      <InspectorSection title="Condition" rows={conditionRows(snapshot)} />
      <CardsSection title="Recent Transcript" cards={messageCards(snapshot)} emptyLabel="No transcript messages." />
      <CardsSection title="Timeline" cards={cardsFromValue(snapshot.world.timeline, "Timeline")} emptyLabel="No timeline entries." />
      <InspectorSection title="Relationships" rows={relationshipRows(snapshot)} />
    </>
  );
}

function InventoryModule({ snapshot }: { snapshot: StorySnapshot }) {
  return (
    <>
      <CardsSection title="Inventory" cards={cardsFromValue(snapshot.character.fields.inventory, "Item")} emptyLabel="Inventory is empty." />
      <CardsSection title="Known Recipes" cards={cardsFromValue(snapshot.character.fields.known_recipes, "Recipe")} emptyLabel="No known recipes." />
      <CardsSection title="Equipment" cards={cardsFromValue(snapshot.character.fields.equipment, "Equipment")} emptyLabel="No dedicated equipment slot data." />
      <InspectorSection title="Useful Context" rows={inventoryContextRows(snapshot)} />
    </>
  );
}

function StatsModule({ snapshot }: { snapshot: StorySnapshot }) {
  const stats = asObject(snapshot.character.fields.stats);
  return (
    <>
      <StatsSection snapshot={snapshot} />
      <InspectorSection title="Attributes" rows={fieldRows(stats.attributes).slice(0, 12)} />
      <InspectorSection title="Secondary" rows={fieldRows(stats.secondary).slice(0, 12)} />
      <InspectorSection title="Counters" rows={counterRows(stats)} />
      <CardsSection title="Skills" cards={cardsFromValue(stats.skills ?? snapshot.character.fields.skills, "Skill")} emptyLabel="No skills recorded." />
      <InspectorSection title="Traits" rows={traitRows(snapshot)} />
      <CardsSection title="Character Profile" cards={cardsFromValue(snapshot.character.fields.background, "Background")} emptyLabel="No background profile." />
    </>
  );
}

function CodexModule({ snapshot }: { snapshot: StorySnapshot }) {
  return (
    <>
      <CardsSection title="Chapters" cards={chapterCards(snapshot)} emptyLabel="No chapters recorded." />
      <CardsSection title="Characters" cards={npcCards(snapshot)} emptyLabel="No characters recorded." />
      <CardsSection title="Known Locations" cards={cardsFromValue(snapshot.world.known_locations, "Location")} emptyLabel="No known locations." />
      <CardsSection title="Global Events" cards={cardsFromValue(snapshot.world.global_events, "Event")} emptyLabel="No global events." />
    </>
  );
}

function FrontsModule({ snapshot }: { snapshot: StorySnapshot }) {
  return (
    <>
      <CardsSection title="Active Fronts" cards={cardsFromValue(snapshot.world.fronts, "Front")} emptyLabel="No fronts recorded." />
      <CardsSection title="Story Hooks" cards={cardsFromValue(snapshot.world.story_hooks, "Hook")} emptyLabel="No story hooks recorded." />
      <CardsSection title="World Reactions" cards={cardsFromValue(snapshot.world.world_reactions, "Reaction")} emptyLabel="No world reactions." />
      <CardsSection title="Scene Contract" cards={cardsFromValue(snapshot.world.scene_contract, "Scene")} emptyLabel="No scene contract details." />
    </>
  );
}

function InvestigationsModule({ snapshot }: { snapshot: StorySnapshot }) {
  return (
    <>
      <CardsSection title="Investigations" cards={cardsFromValue(snapshot.world.investigations, "Investigation")} emptyLabel="No investigations are active." />
      <InspectorSection title="Flags" rows={flagRows(snapshot)} />
      <CardsSection title="Recent Clues" cards={messageCards(snapshot, ["clue", "investigation", "examine", "search"])} emptyLabel="No recent clue-like transcript entries." />
    </>
  );
}

function ProjectsModule({ snapshot }: { snapshot: StorySnapshot }) {
  return (
    <>
      <CardsSection title="Projects" cards={cardsFromValue(snapshot.world.projects, "Project")} emptyLabel="No projects are active." />
      <CardsSection title="Guidance" cards={cardsFromValue(snapshot.world.guidance, "Guidance")} emptyLabel="No guidance recorded." />
      <CardsSection title="Faction Standings" cards={cardsFromValue(snapshot.world.faction_standings, "Faction")} emptyLabel="No faction standings." />
    </>
  );
}

function SavesModule({ snapshot }: { snapshot: StorySnapshot }) {
  return (
    <>
      <CardsSection title="Saves" cards={saveCards(snapshot)} emptyLabel="No saved snapshots yet." />
      <CardsSection title="Sessions" cards={sessionCards(snapshot)} emptyLabel="No sessions recorded." />
      <CardsSection title="Achievements" cards={achievementCards(snapshot)} emptyLabel="No achievements recorded." />
    </>
  );
}

function InspectorSection({ title, rows }: { title: string; rows: Array<[string, string]> }) {
  return (
    <details className="inspector-section" open>
      <summary>
        <span>{title}</span>
        <ChevronDown size={14} />
      </summary>
      <div className="kv-list">
        {rows.length === 0 ? (
          <div className="empty-copy small">No data.</div>
        ) : (
          rows.map(([label, value]) => (
            <div className="kv-row" key={`${title}-${label}`}>
              <span>{label}</span>
              <strong>{value}</strong>
            </div>
          ))
        )}
      </div>
    </details>
  );
}

function CardsSection({ title, cards, emptyLabel }: { title: string; cards: CardView[]; emptyLabel: string }) {
  return (
    <details className="inspector-section card-section" open>
      <summary>
        <span>{title}</span>
        <ChevronDown size={14} />
      </summary>
      <div className="inspector-cards">
        {cards.length === 0 ? (
          <div className="empty-copy small">{emptyLabel}</div>
        ) : (
          cards.map((card, index) => (
            <article className="inspector-card" key={`${title}-${card.title}-${index}`}>
              <h3>{card.title}</h3>
              {card.rows.length > 0 && (
                <div className="kv-list compact">
                  {card.rows.map(([label, value]) => (
                    <div className="kv-row" key={`${card.title}-${label}`}>
                      <span>{label}</span>
                      <strong>{value}</strong>
                    </div>
                  ))}
                </div>
              )}
            </article>
          ))
        )}
      </div>
    </details>
  );
}

function StatsSection({ snapshot }: { snapshot: StorySnapshot }) {
  const stats = meterRows(snapshot);
  return (
    <details className="inspector-section" open>
      <summary>
        <span>Stats</span>
        <ChevronDown size={14} />
      </summary>
      <div className="stat-list">
        {stats.length === 0 ? (
          <div className="empty-copy small">No stats.</div>
        ) : (
          stats.map(({ label, value, text }) => (
            <div className="stat-row" key={label}>
              <span>{label}</span>
              <div className="stat-meter">
                <i style={{ width: `${value}%` }} />
              </div>
              <strong>{text}</strong>
            </div>
          ))
        )}
      </div>
    </details>
  );
}

export function moduleTitle(tab: ModuleTab): string {
  return moduleSpecs.find((item) => item.tab === tab)?.label ?? titleCase(tab);
}

function turnRows(snapshot: StorySnapshot): Array<[string, string]> {
  const clock = displayClock(snapshot.world.current_turn);
  return [
    ["Turn", String(snapshot.world.current_turn)],
    ["Time", clock.time],
    ["Cycle", clock.cycle],
  ];
}

function locationRows(snapshot: StorySnapshot): Array<[string, string]> {
  const known = snapshot.world.known_locations;
  return [
    ["Current", snapshot.world.current_location || "-"],
    ["Type", findString(known, ["type", "kind", "category"]) || "-"],
    ["Region", findString(known, ["region", "district", "area"]) || "-"],
    ["Details", compactText(findString(known, ["details", "description", "notes"]) || valueToText(known), 72)],
  ];
}

export function meterRows(snapshot: StorySnapshot): Array<{ label: string; value: number; text: string }> {
  const stats = asObject(snapshot.character.fields.stats);
  const preferred = ["health", "focus", "resolve", "stamina", "insight"];
  const nonMeterKeys = new Set(["currency", "deaths", "gold", "coins", "xp", "level"]);
  const rows: Array<{ label: string; value: number; text: string }> = [];

  const vitals = asObject(stats.vitals);
  for (const [key, value] of Object.entries(vitals)) {
    const object = asObject(value);
    const current = numericStat(object.current);
    const max = numericStat(object.max);
    if (current !== null && max !== null && max > 0) {
      rows.push({ label: titleCase(key), value: Math.min(100, Math.round((current / max) * 100)), text: `${current}/${max}` });
    }
  }

  for (const key of preferred) {
    const value = numericStat(stats[key] ?? stats[titleCase(key)]);
    if (value !== null) rows.push({ label: titleCase(key), value, text: `${value}/100` });
  }
  for (const [key, value] of Object.entries(stats)) {
    if (nonMeterKeys.has(key.toLowerCase())) continue;
    if (rows.some((row) => row.label.toLowerCase() === key.toLowerCase())) continue;
    const stat = numericStat(value);
    if (stat !== null) rows.push({ label: titleCase(key), value: stat, text: `${stat}/100` });
  }
  return rows.slice(0, 7);
}

function conditionRows(snapshot: StorySnapshot): Array<[string, string]> {
  return [
    [deriveCondition(snapshot), conditionDetail(snapshot)],
    ["Chapter", String(snapshot.world.current_chapter)],
  ];
}

function conditionDetail(snapshot: StorySnapshot): string {
  const condition = deriveCondition(snapshot);
  if (condition === "Focused") return "No active penalties.";
  if (condition === "Injured") return "Health needs attention.";
  if (condition === "Exhausted") return "Stamina is under pressure.";
  return "Stable operating state.";
}

function flagRows(snapshot: StorySnapshot): Array<[string, string]> {
  const rows: Array<[string, string]> = [];
  collectFlags(rows, snapshot.world.scene_contract, "scene");
  collectFlags(rows, snapshot.world.story_hooks, "hook");
  collectFlags(rows, snapshot.world.investigations, "investigation");
  return rows.slice(0, 8);
}

function collectFlags(rows: Array<[string, string]>, value: JsonValue, prefix: string) {
  if (Array.isArray(value)) {
    value.slice(0, 5).forEach((item, index) => rows.push([`${prefix}_${index + 1}`, compactText(entryLabel(item, index), 34)]));
    return;
  }
  const object = asObject(value);
  for (const [key, child] of Object.entries(object)) {
    if (rows.length > 10) break;
    if (typeof child === "boolean") rows.push([key, child ? "true" : "false"]);
    else if (typeof child === "string" || typeof child === "number") rows.push([key, compactText(String(child), 34)]);
  }
}

function relationshipRows(snapshot: StorySnapshot): Array<[string, string]> {
  return snapshot.panels.npcs.slice(0, 6).map((npc) => {
    const disposition = numericStat(npc.fields.disposition);
    const relation = relationLabel(npc.fields.relationship);
    return [npc.name, `${relation} (${disposition ?? 0})`];
  });
}

function relationLabel(value: JsonValue | undefined): string {
  const object = asObject(value);
  const direct = object.label ?? object.status ?? object.kind;
  if (typeof direct === "string" && direct.trim()) return direct;
  const trust = numericStat(object.trust ?? object.Trust);
  if (trust !== null && trust > 60) return "Friendly";
  if (trust !== null && trust < 30) return "Cautious";
  return "Neutral";
}

function activeFrontRows(snapshot: StorySnapshot): Array<[string, string]> {
  const fronts = asArray(snapshot.world.fronts);
  const hooks = asArray(snapshot.world.story_hooks);
  const active = fronts[0] ?? hooks[0];
  if (!active) return [];
  if (active && typeof active === "object" && !Array.isArray(active)) {
    const object = active as JsonObject;
    return [
      ["Name", compactText(entryLabel(active, 0), 46)],
      ...fieldRows(object).filter(([key]) => !["Name", "Title", "Id"].includes(key)).slice(0, 4),
    ];
  }
  return [["Name", compactText(valueToText(active), 64)]];
}

function inventoryContextRows(snapshot: StorySnapshot): Array<[string, string]> {
  return [
    ["Character", snapshot.character.name || "-"],
    ["Location", snapshot.world.current_location || "-"],
    ["Active Front", activeFrontRows(snapshot)[0]?.[1] ?? "-"],
  ];
}

function traitRows(snapshot: StorySnapshot): Array<[string, string]> {
  const stats = asObject(snapshot.character.fields.stats);
  const traits = asArray(snapshot.character.fields.traits).length ? asArray(snapshot.character.fields.traits) : asArray(stats.traits);
  return traits.map((trait, index) => [`Trait ${index + 1}`, compactText(valueToText(trait), 80)] as [string, string]).slice(0, 12);
}

function counterRows(stats: JsonObject): Array<[string, string]> {
  const counters = ["currency", "gold", "coins", "deaths", "level", "xp"];
  const rows: Array<[string, string]> = [];
  for (const key of counters) {
    const value = stats[key] ?? stats[titleCase(key)];
    if (value !== undefined && value !== null) rows.push([titleCase(key), compactText(valueToText(value), 80)]);
  }
  return rows;
}

function messageCards(snapshot: StorySnapshot, keywords: string[] = []): CardView[] {
  const wanted = keywords.map((item) => item.toLowerCase());
  return snapshot.messages
    .filter((message) => {
      if (wanted.length === 0) return true;
      const content = message.content.toLowerCase();
      return wanted.some((keyword) => content.includes(keyword));
    })
    .slice(-6)
    .reverse()
    .map((message) => ({
      title: `${message.role} - Turn ${message.turn}`,
      rows: [
        ["Type", message.message_type || message.role],
        ["Text", compactText(message.content, 160)],
      ],
    }));
}

function chapterCards(snapshot: StorySnapshot): CardView[] {
  return snapshot.panels.chapters.slice(-8).reverse().map((chapter) => ({
    title: chapter.title || `Chapter ${chapter.chapter_number}`,
    rows: [
      ["Chapter", String(chapter.chapter_number)],
      ["Turns", chapter.end_turn ? `${chapter.start_turn}-${chapter.end_turn}` : `${chapter.start_turn}+`],
      ["Summary", compactText(chapter.summary || "-", 140)],
    ],
  }));
}

function npcCards(snapshot: StorySnapshot): CardView[] {
  return snapshot.panels.npcs.slice(0, 12).map((npc) => ({
    title: npc.name,
    rows: [
      ...fieldRows(npc.fields)
        .filter(([key]) => !["Name", "Id"].includes(key))
        .map(([key, value]) => [key, compactText(value, 100)] as [string, string])
        .slice(0, 7),
    ],
  }));
}

function saveCards(snapshot: StorySnapshot): CardView[] {
  return snapshot.panels.saves.slice(0, 12).map((save) => ({
    title: save.name || `Save ${save.turn}`,
    rows: [
      ["Turn", String(save.turn)],
      ["Chapter", String(save.chapter)],
      ["Location", save.location || "-"],
      ["Created", save.created_at || "-"],
    ],
  }));
}

function sessionCards(snapshot: StorySnapshot): CardView[] {
  return snapshot.panels.sessions.slice(0, 10).map((session) => ({
    title: session.started_at || session.id,
    rows: [
      ["Status", session.ended_at ? "Ended" : "Active"],
      ["Summary", compactText(session.summary || "-", 140)],
    ],
  }));
}

function achievementCards(snapshot: StorySnapshot): CardView[] {
  return snapshot.panels.achievements.slice(0, 12).map((achievement) => ({
    title: achievement.name,
    rows: [
      ["Category", achievement.category || "-"],
      ["Rarity", achievement.rarity || "-"],
      ["Description", compactText(achievement.description || "-", 140)],
      ["Earned", achievement.earned_at || "-"],
    ],
  }));
}

export function cardsFromValue(value: JsonValue | undefined, fallbackTitle: string): CardView[] {
  if (Array.isArray(value)) {
    return value.slice(0, 16).map((item, index) => cardFromEntry(item, entryLabel(item, index), index));
  }

  const object = asObject(value);
  const entries = Object.entries(object);
  if (entries.length === 0) {
    if (value && typeof value === "object") return [];
    if (value === undefined || value === null) return [];
    return [{ title: fallbackTitle, rows: [["Value", compactText(valueToText(value), 120)]] }];
  }

  const complexEntries = entries.filter(([, child]) => child && typeof child === "object");
  if (complexEntries.length > 0) {
    return complexEntries.slice(0, 16).map(([key, child], index) => cardFromEntry(child, titleCase(key), index));
  }

  return [
    {
      title: fallbackTitle,
      rows: entries.slice(0, 12).map(([key, child]) => [titleCase(key), compactText(valueToText(child), 120)]),
    },
  ];
}

function cardFromEntry(value: JsonValue, title: string, index: number): CardView {
  const rows = fieldRows(value)
    .filter(([key]) => !["Name", "Title", "Id"].includes(key))
    .map(([key, child]) => [key, compactText(child, 120)] as [string, string])
    .slice(0, 8);
  return {
    title: compactText(title || `${index + 1}`, 70),
    rows,
  };
}
