import { useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { Activity, ChevronDown, Clock3, Hash, MapPin, Maximize2, RefreshCw, Search, Sun, Users } from "lucide-react";
import { moduleSpecs } from "../commands";
import {
  asArray,
  asObject,
  compactText,
  deriveCondition,
  displayClock,
  displayTimestamp,
  entryLabel,
  fieldRows,
  findString,
  numericStat,
  readableStructuredText,
  titleCase,
  valueToText,
} from "../format";
import type { JsonObject, JsonValue, ModuleTab, RecordView, StorySnapshot, VisualAsset } from "../types";
import type { VisualCatalog } from "../visualAssets";
import { characterAsset, readyAssetUrl } from "../visualAssets";

interface InspectorProps {
  snapshot: StorySnapshot | null;
  selectedTab: ModuleTab;
  visuals?: VisualCatalog;
  onRefresh: () => void;
  onOpenModule: () => void;
  onOpenNpcCodex: (npcId: string) => void;
  onOpenVisualAsset?: (assetId: string) => void;
}

interface CardView {
  id?: string;
  title: string;
  rows: Array<[string, string]>;
  imageUrl?: string;
  imageState?: string;
}

const stackedRowLabels = new Set([
  "appearance",
  "context",
  "description",
  "details",
  "notes",
  "summary",
  "text",
  "value",
]);

export function Inspector({ snapshot, selectedTab, visuals, onRefresh, onOpenModule, onOpenNpcCodex, onOpenVisualAsset }: InspectorProps) {
  const title = moduleTitle(selectedTab);

  return (
    <aside className="right-inspector">
      <div className="inspector-head">
        <h2>{selectedTab === "history" ? "World State" : title}</h2>
        <div className="inspector-head-actions">
          {selectedTab !== "history" && <span className="inspector-mode">{title}</span>}
          <button type="button" className="square-button" onClick={onOpenModule} title="Open module in modal">
            <Maximize2 size={15} />
          </button>
          <button type="button" className="square-button" onClick={onRefresh} title="Refresh snapshot">
            <RefreshCw size={15} />
          </button>
        </div>
      </div>
      {!snapshot ? (
        <div className="empty-copy inspector-empty">Select a story to inspect canonical state.</div>
      ) : (
        <div className="inspector-body">
          <ModuleContent tab={selectedTab} snapshot={snapshot} visuals={visuals} onOpenNpcCodex={onOpenNpcCodex} onOpenVisualAsset={onOpenVisualAsset} />
        </div>
      )}
    </aside>
  );
}

export function ModuleContent({
  tab,
  snapshot,
  visuals,
  expanded = false,
  focusCardId,
  onOpenNpcCodex,
  onOpenVisualAsset,
}: {
  tab: ModuleTab;
  snapshot: StorySnapshot;
  visuals?: VisualCatalog;
  expanded?: boolean;
  focusCardId?: string | null;
  onOpenNpcCodex?: (npcId: string) => void;
  onOpenVisualAsset?: (assetId: string) => void;
}) {
  return (
    <>
      {renderModule(tab, snapshot, visuals, focusCardId, onOpenNpcCodex, onOpenVisualAsset)}
      {expanded && <RawStateSection tab={tab} snapshot={snapshot} />}
    </>
  );
}

function renderModule(
  tab: ModuleTab,
  snapshot: StorySnapshot,
  visuals?: VisualCatalog,
  focusCardId?: string | null,
  onOpenNpcCodex?: (npcId: string) => void,
  onOpenVisualAsset?: (assetId: string) => void,
) {
  if (tab === "inventory") return <InventoryModule snapshot={snapshot} />;
  if (tab === "craft") return <CraftModule snapshot={snapshot} />;
  if (tab === "stats") return <StatsModule snapshot={snapshot} />;
  if (tab === "codex") return <CodexModule snapshot={snapshot} visuals={visuals} focusCardId={focusCardId} />;
  if (tab === "fronts") return <FrontsModule snapshot={snapshot} />;
  if (tab === "investigations") return <InvestigationsModule snapshot={snapshot} />;
  if (tab === "projects") return <ProjectsModule snapshot={snapshot} />;
  if (tab === "saves") return <SavesModule snapshot={snapshot} />;
  return <WorldStateModule snapshot={snapshot} visuals={visuals} onOpenNpcCodex={onOpenNpcCodex} onOpenVisualAsset={onOpenVisualAsset} />;
}

function RawStateSection({ tab, snapshot }: { tab: ModuleTab; snapshot: StorySnapshot }) {
  return (
    <details className="inspector-section raw-state-section">
      <summary>
        <span>Raw Structured State</span>
        <ChevronDown size={14} />
      </summary>
      <pre>{JSON.stringify(sanitizePlayerVisibleValue(rawStateForModule(tab, snapshot)), null, 2)}</pre>
    </details>
  );
}

function rawStateForModule(tab: ModuleTab, snapshot: StorySnapshot): Record<string, unknown> {
  if (tab === "inventory") {
    return {
      inventory: snapshot.character.fields.inventory,
      known_recipes: snapshot.character.fields.known_recipes,
      equipment: snapshot.character.fields.equipment,
      character: snapshot.character,
    };
  }
  if (tab === "craft") {
    return {
      inventory: snapshot.character.fields.inventory,
      known_recipes: snapshot.character.fields.known_recipes,
      crafting_context: {
        location: snapshot.world.current_location,
        scene_contract: snapshot.world.scene_contract,
        projects: snapshot.world.projects,
      },
      character: snapshot.character,
    };
  }
  if (tab === "stats") {
    return {
      stats: snapshot.character.fields.stats,
      traits: snapshot.character.fields.traits,
      character: snapshot.character,
    };
  }
  if (tab === "codex") {
    return {
      chapters: snapshot.panels.chapters,
      npcs: snapshot.panels.npcs,
      known_locations: snapshot.world.known_locations,
      global_events: snapshot.world.global_events,
    };
  }
  if (tab === "fronts") {
    return {
      fronts: snapshot.world.fronts,
      story_hooks: snapshot.world.story_hooks,
      world_reactions: snapshot.world.world_reactions,
      scene_contract: snapshot.world.scene_contract,
    };
  }
  if (tab === "investigations") {
    return {
      investigations: snapshot.world.investigations,
      scene_contract: snapshot.world.scene_contract,
      story_hooks: snapshot.world.story_hooks,
    };
  }
  if (tab === "projects") {
    return {
      projects: snapshot.world.projects,
      guidance: snapshot.world.guidance,
      faction_standings: snapshot.world.faction_standings,
    };
  }
  if (tab === "saves") {
    return {
      saves: snapshot.panels.saves,
      sessions: snapshot.panels.sessions,
      achievements: snapshot.panels.achievements,
    };
  }
  return {
    world: snapshot.world,
    recent_messages: snapshot.messages.slice(-8),
    npcs: snapshot.panels.npcs,
  };
}

function HistoryModule({ snapshot, visuals }: { snapshot: StorySnapshot; visuals?: VisualCatalog }) {
  return <WorldStateModule snapshot={snapshot} visuals={visuals} />;
}

function WorldStateModule({
  snapshot,
  visuals,
  onOpenNpcCodex,
  onOpenVisualAsset,
}: {
  snapshot: StorySnapshot;
  visuals?: VisualCatalog;
  onOpenNpcCodex?: (npcId: string) => void;
  onOpenVisualAsset?: (assetId: string) => void;
}) {
  const clock = displayClock(snapshot.world.current_turn);
  const condition = deriveCondition(snapshot);
  const conditionNote = conditionDetail(snapshot);
  const conditionTone = conditionToneFor(condition);
  const locationThumb = readyAssetUrl(visuals?.location ?? null);
  const locationState = visuals?.location?.status;
  const known = snapshot.world.known_locations;
  const locationType = findString(known, ["type", "kind", "category"]) || "-";
  const locationRegion = findString(known, ["region", "district", "area"]) || "";
  const npcs = snapshot.panels.npcs;
  const threads = threadItems(snapshot);
  const facts = quickFacts(snapshot);

  return (
    <div className="world-state">
      <div className="ws-metrics">
        <MetricTile icon={<Hash size={14} />} label="Turn" value={String(snapshot.world.current_turn)} />
        <MetricTile icon={<Clock3 size={14} />} label="Time" value={clock.time} />
        <MetricTile icon={<Sun size={14} />} label="Cycle" value={clock.cycle} />
      </div>

      <section className="ws-block">
        <header className="ws-block-head">
          <MapPin size={14} />
          <span>Location</span>
        </header>
        <div className="ws-location">
          <div className="ws-location-copy">
            <strong title={snapshot.world.current_location}>{snapshot.world.current_location || "Unknown location"}</strong>
            <div className="ws-location-rows">
              <span>Type</span>
              <small>{locationType}</small>
              <span>Region</span>
              <small>{locationRegion || "-"}</small>
            </div>
          </div>
          <div className={`ws-thumb ${locationThumb ? "ready" : "empty"}`}>
            {locationThumb && visuals?.location && onOpenVisualAsset ? (
              <button type="button" onClick={() => onOpenVisualAsset(visuals.location!.id)} title={`Edit image prompt for ${visuals.location.subject}`}>
                <img src={locationThumb} alt="" />
              </button>
            ) : locationThumb ? (
              <img src={locationThumb} alt="" />
            ) : (
              <span>{locationState ? `Image ${locationState}` : "No image"}</span>
            )}
          </div>
        </div>
      </section>

      <section className="ws-block">
        <header className="ws-block-head">
          <Activity size={14} />
          <span>Condition</span>
        </header>
        <div className="ws-condition" data-condition-tone={conditionTone}>
          <div className="ws-condition-top">
            <div className="ws-condition-copy">
              <strong>
                <i aria-hidden="true" />
                {condition}
              </strong>
              <span>{conditionNote}</span>
              <dl>
                <dt>Chapter</dt>
                <dd>{snapshot.world.current_chapter}</dd>
              </dl>
            </div>
            <HeartbeatLine condition={condition} />
          </div>
        </div>
      </section>

      {npcs.length > 0 && (
        <section className="ws-block">
          <header className="ws-block-head ws-block-head-split">
            <Users size={14} />
            <span>Factions & NPCs</span>
            <small>{npcs.length}</small>
          </header>
          <NpcList npcs={npcs} visuals={visuals} onOpenNpcCodex={onOpenNpcCodex} onOpenVisualAsset={onOpenVisualAsset} />
        </section>
      )}

      {threads.length > 0 && (
        <section className="ws-block">
          <header className="ws-block-head">
            <span>Current Threads</span>
          </header>
          <ul className="ws-threads">
            {threads.map((thread) => (
              <li key={thread.key} className={thread.tone}>
                <i aria-hidden="true" />
                <span>{thread.label}</span>
                <small>{thread.status}</small>
              </li>
            ))}
          </ul>
        </section>
      )}

      {facts.length > 0 && (
        <section className="ws-block">
          <header className="ws-block-head">
            <span>Quick Facts</span>
          </header>
          <ul className="ws-facts">
            {facts.map((fact) => (
              <li key={fact.label}>
                <small>{fact.label}</small>
                <strong>{fact.value}</strong>
              </li>
            ))}
          </ul>
        </section>
      )}
    </div>
  );
}

function MetricTile({ icon, label, value }: { icon?: ReactNode; label: string; value: string }) {
  return (
    <div className="ws-metric" title={`${label}: ${value}`}>
      <small>
        {icon}
        {label}
      </small>
      <strong>{value}</strong>
    </div>
  );
}

function HeartbeatLine({ condition }: { condition: string }) {
  const flat = condition === "Idle";
  const stable = condition === "Stable";
  const path = stable
    ? "M4 24 C22 24 31 24 42 24 L49 24 L54 15 L60 31 L66 21 L72 25 L78 10 L86 36 L92 19 L99 27 L108 22 L117 25 L125 16 L132 23 C144 22 152 22 164 22 L176 22"
    : flat
      ? "M4 24 H176"
      : "M4 24 C22 24 31 24 42 24 L49 24 L54 10 L60 36 L66 22 L78 22 L85 12 L92 32 L99 22 L130 22 L138 15 L145 31 L152 22 L176 22";
  return (
    <svg className={`ws-heartbeat ${flat ? "flat" : ""} ${stable ? "stable" : ""}`} viewBox="0 0 180 46" preserveAspectRatio="none" aria-hidden="true">
      <path d={path} fill="none" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" vectorEffect="non-scaling-stroke" />
      <circle cx="176" cy="22" r="2.3" fill="currentColor" />
    </svg>
  );
}

function NpcCard({
  npc,
  asset,
  onOpenNpcCodex,
  onOpenVisualAsset,
}: {
  npc: RecordView;
  asset: VisualAsset | null;
  onOpenNpcCodex?: (npcId: string) => void;
  onOpenVisualAsset?: (assetId: string) => void;
}) {
  const relation = npcRelationSummary(npc);
  const discovery = npcDiscoverySummary(npc);
  const role = npcRole(npc);
  const imageUrl = readyAssetUrl(asset);
  const content = (
    <>
      <div className={`ws-npc-img ${imageUrl ? "ready" : "empty"}`}>
        {imageUrl && asset && onOpenVisualAsset ? (
          <button
            type="button"
            onClick={(event) => {
              event.stopPropagation();
              onOpenVisualAsset(asset.id);
            }}
            title={`Edit image prompt for ${asset.subject}`}
          >
            <img src={imageUrl} alt="" />
          </button>
        ) : imageUrl ? (
          <img src={imageUrl} alt="" />
        ) : (
          <Users size={16} />
        )}
      </div>
      <div className="ws-npc-body">
        <div className="ws-npc-title">
          <strong>{discovery.publicLabel || npc.name}</strong>
          <span>{relation.label}</span>
        </div>
        <small>{role}</small>
        <div className="ws-npc-meta">
          <span data-stage={discovery.stage}>{discovery.label}</span>
          {discovery.visualLabel && <span>{discovery.visualLabel}</span>}
        </div>
        <div className="ws-relation">
          <div className="ws-relation-bar" role="meter" aria-label={`${npc.name} relationship`} aria-valuenow={relation.score} aria-valuemin={0} aria-valuemax={100}>
            {Array.from({ length: 4 }, (_, index) => (
              <i className={index < relation.filledBands ? "filled" : ""} key={index} />
            ))}
          </div>
          <em>{relation.score}/100</em>
        </div>
      </div>
    </>
  );
  const title = `${discovery.publicLabel || npc.name}: ${discovery.label}; ${relation.label} ${relation.score}/100`;
  if (!onOpenNpcCodex) {
    return (
      <article className="ws-npc" data-relation-tone={relation.tone} title={title}>
        {content}
      </article>
    );
  }
  return (
    <button
      type="button"
      className="ws-npc"
      data-relation-tone={relation.tone}
      title={`${title}. Open in Codex.`}
      aria-label={`Open ${npc.name} in Codex`}
      onClick={() => onOpenNpcCodex(npc.id)}
    >
      {content}
    </button>
  );
}

function NpcList({
  npcs,
  visuals,
  onOpenNpcCodex,
  onOpenVisualAsset,
}: {
  npcs: RecordView[];
  visuals?: VisualCatalog;
  onOpenNpcCodex?: (npcId: string) => void;
  onOpenVisualAsset?: (assetId: string) => void;
}) {
  const pageSize = 3;
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(0);
  const showPager = npcs.length > pageSize;
  const showSearch = npcs.length > 5;
  const normalizedQuery = query.trim().toLowerCase();
  const filtered = useMemo(() => {
    if (!normalizedQuery) return npcs;
    return npcs.filter((npc) => npcSearchText(npc).includes(normalizedQuery));
  }, [normalizedQuery, npcs]);
  const totalPages = Math.max(1, Math.ceil(filtered.length / pageSize));
  const safePage = Math.min(page, totalPages - 1);
  const visibleNpcs = showPager ? filtered.slice(safePage * pageSize, safePage * pageSize + pageSize) : filtered;
  const start = filtered.length === 0 ? 0 : safePage * pageSize + 1;
  const end = Math.min(filtered.length, safePage * pageSize + visibleNpcs.length);

  useEffect(() => {
    setPage(0);
  }, [normalizedQuery, npcs.length]);

  useEffect(() => {
    if (page > totalPages - 1) setPage(totalPages - 1);
  }, [page, totalPages]);

  return (
    <div className="ws-npc-module">
      {showSearch && (
        <label className="ws-npc-search">
          <Search size={13} />
          <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search characters..." />
        </label>
      )}
      <div className="ws-npcs">
        {visibleNpcs.length === 0 ? (
          <div className="empty-copy small">No matching characters.</div>
        ) : (
          visibleNpcs.map((npc) => (
            <NpcCard
              key={npc.id}
              npc={npc}
              asset={visuals ? characterAsset(visuals, npc) : null}
              onOpenNpcCodex={onOpenNpcCodex}
              onOpenVisualAsset={onOpenVisualAsset}
            />
          ))
        )}
      </div>
      {showPager && (
        <div className="ws-npc-pager">
          <span>
            {start}-{end} / {filtered.length}
          </span>
          <div>
            <button type="button" disabled={safePage === 0} onClick={() => setPage((value) => Math.max(0, value - 1))}>
              Prev
            </button>
            <button type="button" disabled={safePage >= totalPages - 1} onClick={() => setPage((value) => Math.min(totalPages - 1, value + 1))}>
              Next
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

type RelationTone = "ally" | "friendly" | "neutral" | "wary" | "hostile";

export interface NpcRelationSummary {
  label: string;
  score: number;
  tone: RelationTone;
  filledSegments: number;
  filledBands: number;
}

export interface NpcDiscoverySummary {
  stage: string;
  label: string;
  publicLabel: string;
  visualReadiness: string;
  visualLabel: string;
}

export function npcDiscoverySummary(npc: RecordView): NpcDiscoverySummary {
  const discovery = asObject(npc.fields.discovery);
  const stage = normalizeDiscoveryToken(
    findString(npc.fields, ["discovery_stage"]) ||
      findString(discovery, ["stage"]) ||
      "established",
  );
  const visualReadiness = normalizeDiscoveryToken(
    findString(npc.fields, ["visual_readiness"]) ||
      findString(discovery, ["visual_readiness"]) ||
      "none",
  );
  const publicLabel =
    findString(npc.fields, ["public_label"]) ||
    findString(discovery, ["public_label"]) ||
    npc.name;
  return {
    stage,
    label: discoveryStageLabel(stage),
    publicLabel,
    visualReadiness,
    visualLabel: visualReadiness === "none" ? "" : titleCase(visualReadiness),
  };
}

export function npcRelationSummary(npc: RecordView): NpcRelationSummary {
  const relationshipValue = npc.fields.relationship;
  const relationship = asObject(relationshipValue);
  const directLabel = relationshipLabelFromValue(relationshipValue);
  const score =
    numericStat(npc.fields.disposition) ??
    numericStat(relationship.disposition) ??
    numericStat(relationship.trust) ??
    numericStat(relationship.affinity) ??
    numericStat(relationship.score) ??
    numericStat(relationship.value) ??
    50;
  const label = directLabel || labelForRelationScore(score);
  const tone = toneForRelation(label, score);
  return {
    label,
    score,
    tone,
    filledSegments: Math.max(0, Math.min(10, Math.round(score / 10))),
    filledBands: Math.max(0, Math.min(4, Math.round(score / 25))),
  };
}

function npcRole(npc: RecordView): string {
  return findString(npc.fields, ["role", "occupation", "archetype", "type", "kind"]) || "Unknown";
}

function npcSearchText(npc: RecordView): string {
  const relation = npcRelationSummary(npc);
  const discovery = npcDiscoverySummary(npc);
  return `${npc.name} ${discovery.publicLabel} ${discovery.label} ${npcRole(npc)} ${relation.label}`.toLowerCase();
}

function normalizeDiscoveryToken(value: string): string {
  return value.trim().toLowerCase().replace(/[^a-z0-9]+/g, "_");
}

function discoveryStageLabel(stage: string): string {
  switch (stage) {
    case "rumor":
      return "Rumor";
    case "observed":
      return "Observed";
    case "identified":
      return "Identified";
    case "established":
      return "Established";
    case "dismissed":
      return "Dismissed";
    default:
      return titleCase(stage || "unknown");
  }
}

function relationshipLabelFromValue(value: JsonValue | undefined): string {
  if (typeof value === "string" && value.trim() && numericStat(value) === null) return titleCase(value);
  const object = asObject(value);
  for (const key of ["label", "status", "kind", "state", "relation"]) {
    const child = object[key];
    if (typeof child === "string" && child.trim() && numericStat(child) === null) return titleCase(child);
  }
  return "";
}

function labelForRelationScore(score: number): string {
  if (score <= 15) return "Enemy";
  if (score <= 30) return "Hostile";
  if (score <= 45) return "Wary";
  if (score <= 60) return "Neutral";
  if (score <= 80) return "Friendly";
  return "Ally";
}

function toneForRelation(label: string, score: number): RelationTone {
  const normalized = label.toLowerCase();
  if (/\b(ally|allied|loyal|devoted|alleat|fidat)/.test(normalized)) return "ally";
  if (/\b(friend|friendly|warm|trusted|amico|amica|fiducia)/.test(normalized)) return "friendly";
  if (/\b(enemy|hostile|nemic|ostil|rival|foe|threat)/.test(normalized)) return "hostile";
  if (/\b(wary|cautious|suspicious|tense|diffident|guarded|cauto|sospett)/.test(normalized)) return "wary";
  if (score <= 30) return "hostile";
  if (score <= 45) return "wary";
  if (score >= 81) return "ally";
  if (score >= 61) return "friendly";
  return "neutral";
}

function conditionToneFor(condition: string): string {
  const normalized = condition.toLowerCase();
  if (normalized.includes("injured")) return "danger";
  if (normalized.includes("exhausted")) return "warning";
  if (normalized.includes("focused")) return "focused";
  if (normalized.includes("idle")) return "idle";
  return "stable";
}

interface ThreadItem {
  key: string;
  label: string;
  status: string;
  tone: "lead" | "hook" | "clue" | "guide";
}

function threadItems(snapshot: StorySnapshot): ThreadItem[] {
  const items: ThreadItem[] = [];
  asArray(snapshot.world.fronts).slice(0, 2).forEach((entry, index) => {
    items.push({ key: `front-${index}`, label: compactText(entryLabel(entry, index), 60), status: "Active front", tone: "lead" });
  });
  asArray(snapshot.world.story_hooks).slice(0, 2).forEach((entry, index) => {
    items.push({ key: `hook-${index}`, label: compactText(entryLabel(entry, index), 60), status: "Hook", tone: "hook" });
  });
  asArray(snapshot.world.investigations).slice(0, 2).forEach((entry, index) => {
    items.push({ key: `inv-${index}`, label: compactText(entryLabel(entry, index), 60), status: "Lead", tone: "clue" });
  });
  const guidance = asArray(snapshot.world.guidance)[0];
  if (guidance) {
    items.push({ key: "guide", label: compactText(entryLabel(guidance, 0), 60), status: "Next lead", tone: "guide" });
  }
  return items.slice(0, 5);
}

function quickFacts(snapshot: StorySnapshot): Array<{ label: string; value: string }> {
  const facts: Array<{ label: string; value: string }> = [];
  const npc = snapshot.panels.npcs[0];
  if (npc) facts.push({ label: "Key Contact", value: npc.name });
  const front = activeFrontRows(snapshot)[0]?.[1];
  if (front && front !== "-") facts.push({ label: "Active Front", value: compactText(front, 60) });
  facts.push({ label: "Chapter", value: String(snapshot.world.current_chapter) });
  facts.push({ label: "Messages", value: String(snapshot.messages.length) });
  facts.push({ label: "Updated", value: compactTimestamp(snapshot.world.updated_at || snapshot.server_time) });
  return facts;
}

function compactTimestamp(value: string | undefined): string {
  const timestamp = displayTimestamp(value);
  const match = timestamp.match(/^(\d{4}-\d{2}-\d{2})[ T](\d{2}:\d{2})/);
  return match ? `${match[1]} ${match[2]}` : timestamp;
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

function CraftModule({ snapshot }: { snapshot: StorySnapshot }) {
  return (
    <>
      <InspectorSection title="Crafting Station" rows={craftingStationRows(snapshot)} />
      <CardsSection title="Known Recipes" cards={cardsFromValue(snapshot.character.fields.known_recipes, "Recipe")} emptyLabel="No known recipes." />
      <CardsSection title="Materials & Items" cards={cardsFromValue(snapshot.character.fields.inventory, "Material")} emptyLabel="No usable inventory items." />
      <CardsSection title="Crafting Projects" cards={cardsFromValue(snapshot.world.projects, "Project")} emptyLabel="No crafting projects are active." />
      <InspectorSection title="Scene Fit" rows={craftingSceneRows(snapshot)} />
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

function CodexModule({ snapshot, visuals, focusCardId }: { snapshot: StorySnapshot; visuals?: VisualCatalog; focusCardId?: string | null }) {
  return (
    <>
      <CardsSection title="Chapters" cards={chapterCards(snapshot)} emptyLabel="No chapters recorded." />
      <CardsSection title="Characters" cards={npcCards(snapshot, visuals)} emptyLabel="No characters recorded." focusCardId={focusCardId} />
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
            <div className={inspectorRowClass(label, value)} key={`${title}-${label}`}>
              <span>{label}</span>
              <strong title={value}>{value}</strong>
            </div>
          ))
        )}
      </div>
    </details>
  );
}

function CardsSection({ title, cards, emptyLabel, focusCardId }: { title: string; cards: CardView[]; emptyLabel: string; focusCardId?: string | null }) {
  const focusedCardRef = useRef<HTMLElement | null>(null);

  useEffect(() => {
    if (!focusCardId || !focusedCardRef.current) return;
    focusedCardRef.current.scrollIntoView({ block: "center", behavior: "smooth" });
    focusedCardRef.current.focus({ preventScroll: true });
  }, [focusCardId, cards.length]);

  return (
    <details className="inspector-section card-section" data-section={sectionSlug(title)} open>
      <summary>
        <span>{title}</span>
        <ChevronDown size={14} />
      </summary>
      <div className="inspector-cards">
        {cards.length === 0 ? (
          <div className="empty-copy small">{emptyLabel}</div>
        ) : (
          cards.map((card, index) => {
            const isFocused = Boolean(focusCardId && card.id === focusCardId);
            return (
              <article
                className="inspector-card"
                data-focused={isFocused ? "true" : undefined}
                key={`${title}-${card.title}-${index}`}
                ref={isFocused ? (node) => { focusedCardRef.current = node; } : undefined}
                tabIndex={isFocused ? -1 : undefined}
              >
                {card.imageUrl && (
                  <div className="inspector-card-image">
                    <img src={card.imageUrl} alt="" />
                  </div>
                )}
                {!card.imageUrl && card.imageState && <div className="inspector-card-image pending">{card.imageState}</div>}
                <h3 title={card.title}>{card.title}</h3>
                {card.rows.length > 0 && (
                  <div className="kv-list compact">
                    {card.rows.map(([label, value]) => (
                      <div className={inspectorRowClass(label, value)} key={`${card.title}-${label}`}>
                        <span>{label}</span>
                        <strong title={value}>{value}</strong>
                      </div>
                    ))}
                  </div>
                )}
              </article>
            );
          })
        )}
      </div>
    </details>
  );
}

function inspectorRowClass(label: string, value: string): string {
  return isLongInspectorRow(label, value) ? "kv-row kv-row-long" : "kv-row";
}

export function isLongInspectorRow(label: string, value: string): boolean {
  const normalized = normalizeFieldKey(label);
  if (stackedRowLabels.has(normalized)) return true;
  if (value.includes("\n")) return true;
  return value.length > 54 && /[\s,.;:]/.test(value);
}

function sectionSlug(title: string): string {
  return normalizeFieldKey(title).replaceAll("_", "-");
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

function craftingStationRows(snapshot: StorySnapshot): Array<[string, string]> {
  const inventory = asArray(snapshot.character.fields.inventory);
  const recipes = asArray(snapshot.character.fields.known_recipes);
  return [
    ["Character", snapshot.character.name || "-"],
    ["Location", snapshot.world.current_location || "-"],
    ["Inventory Items", String(inventory.length)],
    ["Known Recipes", String(recipes.length)],
    ["Active Front", activeFrontRows(snapshot)[0]?.[1] ?? "-"],
  ];
}

function craftingSceneRows(snapshot: StorySnapshot): Array<[string, string]> {
  const scene = asObject(snapshot.world.scene_contract);
  const rows = fieldRows(scene)
    .filter(([key]) => ["Tools", "Materials", "Weather", "Light", "Pressure", "Risk", "Opportunity", "Constraint", "Details"].includes(key))
    .slice(0, 8);
  if (rows.length > 0) return rows;
  return [
    ["Weather", findString(snapshot.world.scene_contract, ["weather", "forecast", "sky"]) || "Untracked"],
    ["Context", compactText(valueToText(snapshot.world.scene_contract), 90)],
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
        ["Text", readableStructuredText(message.content)],
      ],
    }));
}

function chapterCards(snapshot: StorySnapshot): CardView[] {
  return snapshot.panels.chapters.slice(-8).reverse().map((chapter) => ({
    title: chapter.title || `Chapter ${chapter.chapter_number}`,
    rows: [
      ["Chapter", String(chapter.chapter_number)],
      ["Turns", chapter.end_turn ? `${chapter.start_turn}-${chapter.end_turn}` : `${chapter.start_turn}+`],
      ["Summary", compactText(chapter.summary || "-", 260)],
    ],
  }));
}

function npcCards(snapshot: StorySnapshot, visuals?: VisualCatalog): CardView[] {
  return snapshot.panels.npcs.slice(0, 12).map((npc) => ({
    id: npc.id,
    title: npc.name,
    imageUrl: readyAssetUrl(visuals ? characterAsset(visuals, npc) : null),
    imageState: visuals ? characterAsset(visuals, npc)?.status : undefined,
    rows: [
      ...fieldRows(npc.fields)
        .filter(([key]) => !["Name", "Id"].includes(key) && !isPlayerHiddenField(key))
        .map(([key, value]) => [key, compactText(value, 220)] as [string, string])
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
      ["Created", displayTimestamp(save.created_at)],
    ],
  }));
}

function sessionCards(snapshot: StorySnapshot): CardView[] {
  return snapshot.panels.sessions.slice(0, 10).map((session) => ({
    title: displayTimestamp(session.started_at || session.id),
    rows: [
      ["Status", session.ended_at ? "Ended" : "Active"],
      ["Summary", compactText(session.summary || "-", 260)],
    ],
  }));
}

function achievementCards(snapshot: StorySnapshot): CardView[] {
  return snapshot.panels.achievements.slice(0, 12).map((achievement) => ({
    title: achievement.name,
    rows: [
      ["Category", achievement.category || "-"],
      ["Rarity", achievement.rarity || "-"],
      ["Description", compactText(achievement.description || "-", 220)],
      ["Earned", displayTimestamp(achievement.earned_at)],
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
    return [{ title: fallbackTitle, rows: [["Value", compactText(valueToText(value), 220)]] }];
  }

  const complexEntries = entries.filter(([, child]) => child && typeof child === "object");
  if (complexEntries.length > 0) {
    return complexEntries.slice(0, 16).map(([key, child], index) => cardFromEntry(child, titleCase(key), index));
  }

  return [
    {
      title: fallbackTitle,
      rows: entries
        .filter(([key]) => !isPlayerHiddenField(key))
        .slice(0, 12)
        .map(([key, child]) => [titleCase(key), compactText(valueToText(child), 220)]),
    },
  ];
}

function cardFromEntry(value: JsonValue, title: string, index: number): CardView {
  const rows = fieldRows(value)
    .filter(([key]) => !["Name", "Title", "Id"].includes(key) && !isPlayerHiddenField(key))
    .map(([key, child]) => [key, compactText(child, 220)] as [string, string])
    .slice(0, 8);
  return {
    title: compactText(title || `${index + 1}`, 120),
    rows,
  };
}

const playerHiddenFieldKeys = new Set([
  "private_thoughts",
  "notes_on_protagonist",
  "desires",
  "npc_desires",
  "nemesis",
  "nemesis_json",
  "hidden_truth",
  "hidden_truths",
  "gm_notes",
  "gm_only",
  "private_notes",
]);

export function isPlayerHiddenField(label: string): boolean {
  return playerHiddenFieldKeys.has(normalizeFieldKey(label));
}

export function sanitizePlayerVisibleValue(value: unknown): unknown {
  if (Array.isArray(value)) return value.map((item) => sanitizePlayerVisibleValue(item));
  if (!value || typeof value !== "object") return value;

  const entries = Object.entries(value as Record<string, unknown>)
    .filter(([key]) => !isPlayerHiddenField(key))
    .map(([key, child]) => [key, sanitizePlayerVisibleValue(child)]);
  return Object.fromEntries(entries);
}

function normalizeFieldKey(label: string): string {
  return label
    .trim()
    .replace(/([a-z0-9])([A-Z])/g, "$1_$2")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "_")
    .replace(/^_+|_+$/g, "");
}
