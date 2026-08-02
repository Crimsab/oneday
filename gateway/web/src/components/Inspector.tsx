import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type FormEvent,
  type ReactNode,
} from "react";
import { useTranslation } from "react-i18next";
import i18n from "../i18n";
import {
  Activity,
  ChevronDown,
  Clock3,
  Hash,
  MapPin,
  Maximize2,
  RefreshCw,
  Search,
  SendHorizontal,
  Sun,
  Users,
} from "lucide-react";
import { moduleSpecs } from "../commands";
import { getAgencyEvents, sendCraftMessage } from "../api";
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
import type {
  AgencyEventView,
  CraftConversationMessage,
  CraftingResponseView,
  JsonObject,
  JsonValue,
  MessageView,
  ModuleTab,
  RecordView,
  StorySnapshot,
  TimelineResponse,
  VisualAsset,
} from "../types";
import type { VisualCatalog } from "../visualAssets";
import type { SpatialEdge } from "../spatialMap";
import { characterAsset, normalizeKey, readyAssetUrl } from "../visualAssets";
import { HistoryReader, type HistoryReaderActions } from "./HistoryReader";
import { CanonicalMap } from "./CanonicalMap";

interface InspectorProps {
  snapshot: StorySnapshot | null;
  selectedTab: ModuleTab;
  visuals?: VisualCatalog;
  onRefresh: () => void;
  onOpenModule: () => void;
  onOpenNpcCodex: (npcId: string) => void;
  onOpenVisualAsset?: (assetId: string) => void;
  onMapTravel?: (locationName: string, route: SpatialEdge | null) => void;
  timeline?: TimelineResponse | null;
  onHistoryFork?: (sourceCommitId: string, turn: number) => void;
  onOpenHistoryModule?: (tab: "map" | "codex") => void;
}

interface CardView {
  id?: string;
  title: string;
  rows: Array<[string, string]>;
  imageUrl?: string;
  imageState?: string;
  imageAssetId?: string;
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

function tr(key: string, values?: Record<string, string | number>): string {
  return i18n.t(key, { ns: "inspector_extra", ...values });
}

export function Inspector({
  snapshot,
  selectedTab,
  visuals,
  onRefresh,
  onOpenModule,
  onOpenNpcCodex,
  onOpenVisualAsset,
  onMapTravel,
  timeline,
  onHistoryFork,
  onOpenHistoryModule,
}: InspectorProps) {
  useTranslation("inspector_extra");
  const title = moduleTitle(selectedTab);

  return (
    <aside
      className="right-inspector"
      id="story-details"
      aria-label={tr("storyDetailsAria", { title })}
    >
      <div className="inspector-head">
        <div>
          <span>{tr("storyDetails")}</span>
          <h2>{title}</h2>
        </div>
        <div className="inspector-head-actions">
          <button
            type="button"
            className="square-button"
            onClick={onOpenModule}
            title={tr("openLarge", { title })}
            aria-label={tr("openLarge", { title })}
          >
            <Maximize2 size={15} />
          </button>
          <button
            type="button"
            className="square-button"
            onClick={onRefresh}
            title={tr("refresh")}
            aria-label={tr("refresh")}
          >
            <RefreshCw size={15} />
          </button>
        </div>
      </div>
      {!snapshot ? (
        <div className="empty-copy inspector-empty">{tr("selectStory")}</div>
      ) : (
        <div className="inspector-body">
          <ModuleContent
            tab={selectedTab}
            snapshot={snapshot}
            visuals={visuals}
            onOpenNpcCodex={onOpenNpcCodex}
            onOpenVisualAsset={onOpenVisualAsset}
            onExpandMap={onOpenModule}
            onMapTravel={onMapTravel}
            timeline={timeline}
            onHistoryFork={onHistoryFork}
            onOpenHistoryModule={onOpenHistoryModule}
          />
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
  onExpandMap,
  onMapTravel,
  timeline,
  onHistoryFork,
  onOpenHistoryModule,
}: {
  tab: ModuleTab;
  snapshot: StorySnapshot;
  visuals?: VisualCatalog;
  expanded?: boolean;
  focusCardId?: string | null;
  onOpenNpcCodex?: (npcId: string) => void;
  onOpenVisualAsset?: (assetId: string) => void;
  onExpandMap?: () => void;
  onMapTravel?: (locationName: string, route: SpatialEdge | null) => void;
  timeline?: TimelineResponse | null;
  onHistoryFork?: (sourceCommitId: string, turn: number) => void;
  onOpenHistoryModule?: (tab: "map" | "codex") => void;
}) {
  useTranslation("inspector_extra");
  return (
    <>
      {renderModule(
        tab,
        snapshot,
        visuals,
        focusCardId,
        onOpenNpcCodex,
        onOpenVisualAsset,
        expanded,
        onExpandMap,
        onMapTravel,
        timeline,
        onHistoryFork,
        onOpenHistoryModule,
      )}
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
  expanded = false,
  onExpandMap?: () => void,
  onMapTravel?: (locationName: string, route: SpatialEdge | null) => void,
  timeline?: TimelineResponse | null,
  onHistoryFork?: (sourceCommitId: string, turn: number) => void,
  onOpenHistoryModule?: (tab: "map" | "codex") => void,
) {
  if (tab === "inventory") return <InventoryModule snapshot={snapshot} />;
  if (tab === "craft") return <CraftModule snapshot={snapshot} />;
  if (tab === "stats") return <StatsModule snapshot={snapshot} />;
  if (tab === "codex")
    return (
      <CodexModule
        snapshot={snapshot}
        visuals={visuals}
        focusCardId={focusCardId}
        onOpenVisualAsset={onOpenVisualAsset}
      />
    );
  if (tab === "fronts") return <FrontsModule snapshot={snapshot} />;
  if (tab === "investigations")
    return <InvestigationsModule snapshot={snapshot} />;
  if (tab === "projects") return <ProjectsModule snapshot={snapshot} />;
  if (tab === "achievements") return <AchievementsModule snapshot={snapshot} />;
  if (tab === "saves") return <SavesModule snapshot={snapshot} />;
  if (tab === "history")
    return (
      <HistoryReader
        snapshot={snapshot}
        activeBranchId={timeline?.active_branch_id}
        branches={timeline?.branches}
        actions={historyReaderActions(
          onHistoryFork,
          onOpenHistoryModule,
          timeline,
        )}
      />
    );
  if (tab === "map")
    return (
      <WorldStateModule
        snapshot={snapshot}
        visuals={visuals}
        onOpenNpcCodex={onOpenNpcCodex}
        onOpenVisualAsset={onOpenVisualAsset}
        expanded={expanded}
        onExpandMap={onExpandMap}
        onMapTravel={onMapTravel}
      />
    );
  return (
    <WorldStateModule
      snapshot={snapshot}
      visuals={visuals}
      onOpenNpcCodex={onOpenNpcCodex}
      onOpenVisualAsset={onOpenVisualAsset}
    />
  );
}

export function historyReaderActions(
  onHistoryFork?: (sourceCommitId: string, turn: number) => void,
  onOpenHistoryModule?: (tab: "map" | "codex") => void,
  timeline?: TimelineResponse | null,
): HistoryReaderActions | undefined {
  const actions: HistoryReaderActions = {
    ...(onHistoryFork
      ? {
          onFork: (message: MessageView) => {
            if (!message.source_commit_id) return;
            const commit = timeline?.commits.find(
              (item) => item.id === message.source_commit_id,
            );
            if (commit?.parent_commit_id) {
              onHistoryFork(
                commit.parent_commit_id,
                Math.max(0, message.turn - 1),
              );
            } else {
              onHistoryFork(message.source_commit_id, message.turn);
            }
          },
        }
      : {}),
    ...(onOpenHistoryModule
      ? {
          onOpenMap: () => onOpenHistoryModule("map"),
          onOpenCodex: () => onOpenHistoryModule("codex"),
        }
      : {}),
  };
  return Object.keys(actions).length ? actions : undefined;
}

function RawStateSection({
  tab,
  snapshot,
}: {
  tab: ModuleTab;
  snapshot: StorySnapshot;
}) {
  return (
    <details className="inspector-section raw-state-section">
      <summary>
        <span>{tr("rawState")}</span>
        <ChevronDown size={14} />
      </summary>
      <pre>
        {JSON.stringify(
          sanitizePlayerVisibleValue(rawStateForModule(tab, snapshot)),
          null,
          2,
        )}
      </pre>
    </details>
  );
}

function rawStateForModule(
  tab: ModuleTab,
  snapshot: StorySnapshot,
): Record<string, unknown> {
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
  if (tab === "achievements") {
    return { achievements: snapshot.panels.achievements };
  }
  if (tab === "saves") {
    return {
      saves: snapshot.panels.saves,
      sessions: snapshot.panels.sessions,
    };
  }
  return {
    world: snapshot.world,
    recent_messages: snapshot.messages.slice(-8),
    npcs: snapshot.panels.npcs,
  };
}

function WorldStateModule({
  snapshot,
  visuals,
  onOpenNpcCodex,
  onOpenVisualAsset,
  expanded = false,
  onExpandMap,
  onMapTravel,
}: {
  snapshot: StorySnapshot;
  visuals?: VisualCatalog;
  onOpenNpcCodex?: (npcId: string) => void;
  onOpenVisualAsset?: (assetId: string) => void;
  expanded?: boolean;
  onExpandMap?: () => void;
  onMapTravel?: (locationName: string, route: SpatialEdge | null) => void;
}) {
  const clock = displayClock(snapshot);
  const condition = deriveCondition(snapshot);
  const conditionNote = conditionDetail(snapshot);
  const conditionTone = conditionToneFor(condition);
  const locationThumb = readyAssetUrl(visuals?.location ?? null);
  const locationState = visuals?.location?.status;
  const known = snapshot.world.known_locations;
  const locationType = findString(known, ["type", "kind", "category"]) || "-";
  const locationRegion =
    findString(known, ["region", "district", "area"]) || "";
  const npcs = snapshot.panels.npcs;
  const threads = threadItems(snapshot);
  const facts = quickFacts(snapshot);

  return (
    <div className={`world-state ${expanded ? "expanded" : ""}`}>
      <div className="ws-metrics">
        <MetricTile
          icon={<Hash size={14} />}
          label={tr("turn")}
          value={String(snapshot.world.current_turn)}
        />
        <MetricTile
          icon={<Clock3 size={14} />}
          label={tr("time")}
          value={clock.time}
        />
        <MetricTile
          icon={<Sun size={14} />}
          label={tr("cycle")}
          value={clock.cycle}
        />
      </div>

      <section className="ws-block">
        <header className="ws-block-head">
          <MapPin size={14} />
          <span>{tr("location")}</span>
        </header>
        <div className="ws-location">
          <div className="ws-location-copy">
            <strong title={snapshot.world.current_location}>
              {snapshot.world.current_location || tr("unknownLocation")}
            </strong>
            <div className="ws-location-rows">
              <span>{tr("type")}</span>
              <small>{locationType}</small>
              <span>{tr("region")}</span>
              <small>{locationRegion || "-"}</small>
            </div>
          </div>
          <div className={`ws-thumb ${locationThumb ? "ready" : "empty"}`}>
            {locationThumb && visuals?.location && onOpenVisualAsset ? (
              <button
                type="button"
                onClick={() => onOpenVisualAsset(visuals.location!.id)}
                title={tr("editImage", { subject: visuals.location.subject })}
                aria-label={tr("editImage", {
                  subject: visuals.location.subject,
                })}
              >
                <img src={locationThumb} alt="" />
              </button>
            ) : locationThumb ? (
              <img src={locationThumb} alt="" />
            ) : (
              <span>
                {locationState
                  ? tr("imageState", {
                      status: localizedImageStatus(locationState),
                    })
                  : tr("noImage")}
              </span>
            )}
          </div>
        </div>
      </section>

      <section className="ws-block">
        <header className="ws-block-head">
          <Activity size={14} />
          <span>{tr("condition")}</span>
        </header>
        <div className="ws-condition" data-condition-tone={conditionTone}>
          <div className="ws-condition-top">
            <div className="ws-condition-copy">
              <strong>
                <i aria-hidden="true" />
                {localizedCondition(condition)}
              </strong>
              <span>{conditionNote}</span>
              <dl>
                <dt>{tr("chapter")}</dt>
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
            <span>{tr("peoplePresent")}</span>
            <small>{npcs.length}</small>
          </header>
          <NpcList
            npcs={npcs}
            visuals={visuals}
            onOpenNpcCodex={onOpenNpcCodex}
            onOpenVisualAsset={onOpenVisualAsset}
          />
        </section>
      )}

      <AgencyFeed storyId={snapshot.story.id} />

      <section className="ws-block ws-map-block">
        <header className="ws-block-head ws-block-head-split">
          <MapPin size={14} />
          <span>{tr("map")}</span>
          {!expanded && onExpandMap && (
            <button
              type="button"
              className="map-expand-button"
              onClick={onExpandMap}
              title={tr("openMap")}
              aria-label={tr("openMap")}
            >
              <Maximize2 size={14} />
            </button>
          )}
        </header>
        <CanonicalMap
          regionsValue={snapshot.world.spatial_regions}
          locationsValue={snapshot.world.spatial_locations}
          edgesValue={snapshot.world.spatial_edges}
          currentLocationId={snapshot.world.current_location_id}
          visuals={visuals}
          expanded={expanded}
          onOpenVisualAsset={onOpenVisualAsset}
          onTravel={onMapTravel}
        />
      </section>

      {threads.length > 0 && (
        <section className="ws-block">
          <header className="ws-block-head">
            <span>{tr("currentThreads")}</span>
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
            <span>{tr("quickFacts")}</span>
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

function AgencyFeed({ storyId }: { storyId: string }) {
  const [events, setEvents] = useState<AgencyEventView[]>([]);
  const [error, setError] = useState("");
  useEffect(() => {
    let active = true;
    void getAgencyEvents(storyId, 8)
      .then((items) => {
        if (active) {
          setEvents(items);
          setError("");
        }
      })
      .catch((cause) => {
        if (active)
          setError(
            cause instanceof Error ? cause.message : tr("activityUnavailable"),
          );
      });
    return () => {
      active = false;
    };
  }, [storyId]);
  return (
    <section className="ws-block agency-feed">
      <header className="ws-block-head">
        <Activity size={14} />
        <span>{tr("activity")}</span>
      </header>
      {events.length > 0 ? (
        events.map((event) => (
          <div key={event.id}>
            <span>{event.summary}</span>
            <small>
              {tr("activityMeta", {
                turn: event.canonical_turn,
                action: event.action.replaceAll("_", " "),
              })}
            </small>
          </div>
        ))
      ) : (
        <p className="empty-copy">{error || tr("noActivity")}</p>
      )}
    </section>
  );
}

function MetricTile({
  icon,
  label,
  value,
}: {
  icon?: ReactNode;
  label: string;
  value: string;
}) {
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
    <svg
      className={`ws-heartbeat ${flat ? "flat" : ""} ${stable ? "stable" : ""}`}
      viewBox="0 0 180 46"
      preserveAspectRatio="none"
      aria-hidden="true"
    >
      <path
        d={path}
        fill="none"
        strokeWidth="2.2"
        strokeLinecap="round"
        strokeLinejoin="round"
        vectorEffect="non-scaling-stroke"
      />
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
            title={tr("editImage", { subject: asset.subject })}
            aria-label={tr("editImage", { subject: asset.subject })}
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
          <div
            className="ws-relation-bar"
            role="meter"
            aria-label={tr("relationship", { name: npc.name })}
            aria-valuenow={relation.score}
            aria-valuemin={0}
            aria-valuemax={100}
          >
            {Array.from({ length: 4 }, (_, index) => (
              <i
                className={index < relation.filledBands ? "filled" : ""}
                key={index}
              />
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
      <article
        className="ws-npc"
        data-relation-tone={relation.tone}
        title={title}
      >
        {content}
      </article>
    );
  }
  return (
    <button
      type="button"
      className="ws-npc"
      data-relation-tone={relation.tone}
      title={tr("openCodexTitle", { title })}
      aria-label={tr("openCodex", { name: npc.name })}
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
  const visibleNpcs = showPager
    ? filtered.slice(safePage * pageSize, safePage * pageSize + pageSize)
    : filtered;
  const start = filtered.length === 0 ? 0 : safePage * pageSize + 1;
  const end = Math.min(
    filtered.length,
    safePage * pageSize + visibleNpcs.length,
  );

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
          <input
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder={tr("searchCharacters")}
            aria-label={tr("searchCharacters")}
          />
        </label>
      )}
      <div className="ws-npcs">
        {visibleNpcs.length === 0 ? (
          <div className="empty-copy small">{tr("noMatchingCharacters")}</div>
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
            <button
              type="button"
              disabled={safePage === 0}
              onClick={() => setPage((value) => Math.max(0, value - 1))}
            >
              {tr("previous")}
            </button>
            <button
              type="button"
              disabled={safePage >= totalPages - 1}
              onClick={() =>
                setPage((value) => Math.min(totalPages - 1, value + 1))
              }
            >
              {tr("next")}
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
    visualLabel:
      visualReadiness === "none"
        ? ""
        : localizedVisualReadiness(visualReadiness),
  };
}

export function npcRelationSummary(npc: RecordView): NpcRelationSummary {
  const relationshipValue = npc.fields.relationship;
  const relationship = asObject(relationshipValue);
  const directLabel = relationshipLabelFromValue(relationshipValue);
  const score =
    relationshipScore(npc.fields.disposition) ??
    relationshipScore(relationship.disposition) ??
    relationshipScore(relationship.trust) ??
    relationshipScore(relationship.affinity) ??
    relationshipScore(relationship.score) ??
    relationshipScore(relationship.value) ??
    0;
  const label = directLabel || labelForRelationScore(score);
  const tone = toneForRelation(label, score);
  return {
    label,
    score,
    tone,
    filledSegments: Math.max(0, Math.min(10, Math.round((score + 100) / 20))),
    filledBands: Math.max(0, Math.min(4, Math.round((score + 100) / 50))),
  };
}

function relationshipScore(value: JsonValue | undefined): number | null {
  const parsed =
    typeof value === "number"
      ? value
      : typeof value === "string"
        ? Number.parseFloat(value)
        : Number.NaN;
  return Number.isFinite(parsed)
    ? Math.max(-100, Math.min(100, Math.round(parsed)))
    : null;
}

function npcRole(npc: RecordView): string {
  return (
    findString(npc.fields, [
      "role",
      "occupation",
      "archetype",
      "type",
      "kind",
    ]) || tr("unknown")
  );
}

function npcSearchText(npc: RecordView): string {
  const relation = npcRelationSummary(npc);
  const discovery = npcDiscoverySummary(npc);
  return `${npc.name} ${discovery.publicLabel} ${discovery.label} ${npcRole(npc)} ${relation.label}`.toLowerCase();
}

function normalizeDiscoveryToken(value: string): string {
  return value
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "_");
}

function discoveryStageLabel(stage: string): string {
  switch (stage) {
    case "rumor":
      return tr("discoveryRumor");
    case "observed":
      return tr("discoveryObserved");
    case "identified":
      return tr("discoveryIdentified");
    case "established":
      return tr("discoveryEstablished");
    case "dismissed":
      return tr("discoveryDismissed");
    default:
      return titleCase(stage || "unknown");
  }
}

function relationshipLabelFromValue(value: JsonValue | undefined): string {
  if (typeof value === "string" && value.trim() && numericStat(value) === null)
    return titleCase(value);
  const object = asObject(value);
  for (const key of ["label", "status", "kind", "state", "relation"]) {
    const child = object[key];
    if (
      typeof child === "string" &&
      child.trim() &&
      numericStat(child) === null
    )
      return titleCase(child);
  }
  return "";
}

function labelForRelationScore(score: number): string {
  if (score >= 50) return tr("relationAllied");
  if (score >= 15) return tr("relationFriendly");
  if (score >= -14) return tr("relationNeutral");
  if (score >= -49) return tr("relationUnfriendly");
  return tr("relationHostile");
}

function toneForRelation(label: string, score: number): RelationTone {
  const normalized = label.toLowerCase();
  if (/\b(ally|allied|loyal|devoted|alleat|fidat)/.test(normalized))
    return "ally";
  if (/\b(friend|friendly|warm|trusted|amico|amica|fiducia)/.test(normalized))
    return "friendly";
  if (/\b(enemy|hostile|nemic|ostil|rival|foe|threat)/.test(normalized))
    return "hostile";
  if (
    /\b(wary|cautious|suspicious|tense|diffident|guarded|cauto|sospett)/.test(
      normalized,
    )
  )
    return "wary";
  if (score <= -50) return "hostile";
  if (score <= -15) return "wary";
  if (score >= 50) return "ally";
  if (score >= 15) return "friendly";
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
  asArray(snapshot.world.fronts)
    .slice(0, 2)
    .forEach((entry, index) => {
      items.push({
        key: `front-${index}`,
        label: compactText(entryLabel(entry, index), 60),
        status: tr("activeFront"),
        tone: "lead",
      });
    });
  asArray(snapshot.world.story_hooks)
    .slice(0, 2)
    .forEach((entry, index) => {
      items.push({
        key: `hook-${index}`,
        label: compactText(entryLabel(entry, index), 60),
        status: tr("hook"),
        tone: "hook",
      });
    });
  asArray(snapshot.world.investigations)
    .slice(0, 2)
    .forEach((entry, index) => {
      items.push({
        key: `inv-${index}`,
        label: compactText(entryLabel(entry, index), 60),
        status: tr("lead"),
        tone: "clue",
      });
    });
  const guidance = asArray(snapshot.world.guidance)[0];
  if (guidance) {
    items.push({
      key: "guide",
      label: compactText(entryLabel(guidance, 0), 60),
      status: tr("nextLead"),
      tone: "guide",
    });
  }
  return items.slice(0, 5);
}

function quickFacts(
  snapshot: StorySnapshot,
): Array<{ label: string; value: string }> {
  const facts: Array<{ label: string; value: string }> = [];
  const npc = snapshot.panels.npcs[0];
  if (npc) facts.push({ label: tr("keyContact"), value: npc.name });
  const front = activeFrontRows(snapshot)[0]?.[1];
  if (front && front !== "-")
    facts.push({ label: tr("activeFront"), value: compactText(front, 60) });
  facts.push({
    label: tr("chapter"),
    value: String(snapshot.world.current_chapter),
  });
  facts.push({
    label: tr("messages"),
    value: String(snapshot.messages.length),
  });
  facts.push({
    label: tr("updated"),
    value: compactTimestamp(snapshot.world.updated_at || snapshot.server_time),
  });
  return facts;
}

function compactTimestamp(value: string | undefined): string {
  const timestamp = displayTimestamp(value);
  const match = timestamp.match(/^(\d{4}-\d{2}-\d{2})[ T](\d{2}:\d{2})/);
  return match ? `${match[1]} ${match[2]}` : timestamp;
}

function InventoryModule({ snapshot }: { snapshot: StorySnapshot }) {
  const { t } = useTranslation("inspector");
  return (
    <>
      <CardsSection
        title={t("inventory")}
        cards={cardsFromValue(snapshot.character.fields.inventory, tr("item"))}
        emptyLabel={t("inventoryEmpty")}
      />
      <CardsSection
        title={t("knownRecipes")}
        cards={cardsFromValue(
          snapshot.character.fields.known_recipes,
          tr("recipe"),
        )}
        emptyLabel={t("noRecipes")}
      />
      <CardsSection
        title={t("equipment")}
        cards={cardsFromValue(
          snapshot.character.fields.equipment,
          tr("equipment"),
        )}
        emptyLabel={t("noEquipment")}
      />
      <InspectorSection
        title={t("usefulContext")}
        rows={inventoryContextRows(snapshot)}
      />
    </>
  );
}

function CraftModule({ snapshot }: { snapshot: StorySnapshot }) {
  const { t } = useTranslation("inspector");
  const [workingSnapshot, setWorkingSnapshot] = useState<StorySnapshot | null>(
    null,
  );
  const [history, setHistory] = useState<CraftConversationMessage[]>([]);
  const [entries, setEntries] = useState<
    Array<{
      id: number;
      role: "user" | "assistant";
      text: string;
      response?: CraftingResponseView;
    }>
  >([]);
  const [draft, setDraft] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const current =
    workingSnapshot?.story.id === snapshot.story.id
      ? workingSnapshot
      : snapshot;

  useEffect(() => {
    setWorkingSnapshot(null);
    setHistory([]);
    setEntries([]);
    setDraft("");
    setError("");
  }, [snapshot.story.id]);

  const send = async (message: string) => {
    const text = message.trim();
    if (!text || busy) return;
    const id = Date.now();
    setBusy(true);
    setError("");
    setDraft("");
    setEntries((items) => [...items, { id, role: "user", text }]);
    try {
      const envelope = await sendCraftMessage(snapshot.story.id, text, history);
      const assistantContext = JSON.stringify(envelope.crafting);
      setHistory((items) => [
        ...items,
        { role: "user", content: text },
        { role: "assistant", content: assistantContext },
      ]);
      setEntries((items) => [
        ...items,
        {
          id: id + 1,
          role: "assistant",
          text: envelope.crafting.narrative,
          response: envelope.crafting,
        },
      ]);
      setWorkingSnapshot(envelope.snapshot);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : tr("craftingFailed"));
      setDraft(text);
    } finally {
      setBusy(false);
    }
  };
  const submit = (event: FormEvent) => {
    event.preventDefault();
    void send(draft);
  };

  return (
    <div className="craft-workspace">
      <ModuleOverview
        label={tr("craftLabel")}
        title={tr("craftTitle")}
        description={tr("craftDescription")}
        metrics={[
          [
            tr("items"),
            String(recordCount(current.character.fields.inventory)),
          ],
          [
            tr("recipes"),
            String(recordCount(current.character.fields.known_recipes)),
          ],
          [tr("turn"), String(current.world.current_turn)],
        ]}
      />
      <section className="craft-conversation" aria-label={t("crafting")}>
        <div className="craft-chat" aria-live="polite">
          {entries.length === 0 ? (
            <div className="craft-welcome">
              <strong>{tr("craftReady")}</strong>
              <p>{tr("craftReadyHint")}</p>
            </div>
          ) : (
            entries.map((entry) => (
              <article key={entry.id} className={entry.role}>
                <span>
                  {entry.role === "user" ? tr("you") : tr("workbench")}
                </span>
                <p>{entry.text}</p>
                {entry.response && (
                  <CraftResult
                    response={entry.response}
                    onTry={(value) => void send(value)}
                    busy={busy}
                  />
                )}
              </article>
            ))
          )}
          {busy && (
            <div className="craft-thinking">
              <i />
              <span>{tr("evaluating")}</span>
            </div>
          )}
        </div>
        <form className="craft-composer" onSubmit={submit}>
          <textarea
            value={draft}
            onChange={(event) => setDraft(event.target.value)}
            rows={3}
            placeholder={t("craftPlaceholder")}
            disabled={busy}
          />
          <button type="submit" disabled={busy || !draft.trim()}>
            <SendHorizontal size={15} />
            {tr("evaluate")}
          </button>
        </form>
        {error && <p className="craft-error">{error}</p>}
      </section>
      <InspectorSection
        title={tr("workbenchContext")}
        rows={craftingStationRows(current)}
      />
      <CardsSection
        title={tr("knownRecipes")}
        cards={cardsFromValue(
          current.character.fields.known_recipes,
          tr("recipe"),
        )}
        emptyLabel={tr("noRecipe")}
      />
      <CardsSection
        title={tr("materialsItems")}
        cards={cardsFromValue(
          current.character.fields.inventory,
          tr("material"),
        )}
        emptyLabel={tr("noUsableItem")}
      />
      <CardsSection
        title={tr("longCraftProjects")}
        cards={cardsFromValue(current.world.projects, tr("project"))}
        emptyLabel={tr("noCraftProject")}
      />
      <InspectorSection
        title={tr("sceneConstraints")}
        rows={craftingSceneRows(current)}
      />
    </div>
  );
}

function CraftResult({
  response,
  onTry,
  busy,
}: {
  response: CraftingResponseView;
  onTry: (value: string) => void;
  busy: boolean;
}) {
  const suggestions = [
    ...(response.alternatives ?? []),
    ...(response.choices ?? []).map((choice) => choice.text),
  ]
    .filter(
      (value, index, list) =>
        value.trim() &&
        !/^leave|exit|esci/i.test(value) &&
        list.indexOf(value) === index,
    )
    .slice(0, 4);
  return (
    <div
      className={`craft-result ${response.feasible ? "feasible" : "blocked"}`}
    >
      <strong>{response.feasible ? tr("feasible") : tr("notFeasible")}</strong>
      {response.item && (
        <dl>
          <div>
            <dt>{tr("created")}</dt>
            <dd>{response.item.name}</dd>
          </div>
          <div>
            <dt>{tr("effect")}</dt>
            <dd>{response.item.effect || response.item.description}</dd>
          </div>
          <div>
            <dt>{tr("used")}</dt>
            <dd>
              {response.item.materials?.join(", ") || tr("noCanonicalMaterial")}
            </dd>
          </div>
        </dl>
      )}
      {Boolean(response.missing?.length) && (
        <p>
          <b>{tr("missing")}</b> {response.missing!.join(", ")}
        </p>
      )}
      {suggestions.length > 0 && (
        <div className="craft-suggestions">
          {suggestions.map((suggestion) => (
            <button
              type="button"
              key={suggestion}
              disabled={busy}
              onClick={() => onTry(suggestion)}
            >
              {suggestion}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

function StatsModule({ snapshot }: { snapshot: StorySnapshot }) {
  const stats = asObject(snapshot.character.fields.stats);
  return (
    <>
      <StatsSection snapshot={snapshot} />
      <InspectorSection
        title={tr("attributes")}
        rows={fieldRows(stats.attributes).slice(0, 12)}
      />
      <InspectorSection
        title={tr("secondary")}
        rows={fieldRows(stats.secondary).slice(0, 12)}
      />
      <InspectorSection title={tr("counters")} rows={counterRows(stats)} />
      <CardsSection
        title={tr("skills")}
        cards={cardsFromValue(
          stats.skills ?? snapshot.character.fields.skills,
          tr("skill"),
        )}
        emptyLabel={tr("noSkills")}
      />
      <InspectorSection title={tr("traits")} rows={traitRows(snapshot)} />
      <CardsSection
        title={tr("characterProfile")}
        cards={cardsFromValue(
          snapshot.character.fields.background,
          tr("background"),
        )}
        emptyLabel={tr("noBackground")}
      />
    </>
  );
}

function CodexModule({
  snapshot,
  visuals,
  focusCardId,
  onOpenVisualAsset,
}: {
  snapshot: StorySnapshot;
  visuals?: VisualCatalog;
  focusCardId?: string | null;
  onOpenVisualAsset?: (assetId: string) => void;
}) {
  return (
    <>
      <ModuleOverview
        label={tr("codexLabel")}
        title={tr("codexTitle")}
        description={tr("codexDescription")}
        metrics={[
          [tr("characters"), String(snapshot.panels.npcs.length)],
          [tr("places"), String(recordCount(snapshot.world.known_locations))],
          [tr("chapters"), String(snapshot.panels.chapters.length)],
        ]}
      />
      <CardsSection
        title={tr("chapters")}
        cards={chapterCards(snapshot)}
        emptyLabel={tr("noChapters")}
      />
      <CardsSection
        title={tr("characters")}
        cards={npcCards(snapshot, visuals)}
        emptyLabel={tr("noCharacters")}
        focusCardId={focusCardId}
        onOpenVisualAsset={onOpenVisualAsset}
      />
      <CardsSection
        title={tr("knownLocations")}
        cards={locationCards(snapshot, visuals)}
        emptyLabel={tr("noLocations")}
        onOpenVisualAsset={onOpenVisualAsset}
      />
      <CardsSection
        title={tr("globalEvents")}
        cards={cardsFromValue(snapshot.world.global_events, tr("event"))}
        emptyLabel={tr("noEvents")}
      />
    </>
  );
}

function FrontsModule({ snapshot }: { snapshot: StorySnapshot }) {
  return (
    <>
      <ModuleOverview
        label={tr("frontsLabel")}
        title={tr("frontsTitle")}
        description={tr("frontsDescription")}
        metrics={[
          [tr("fronts"), String(recordCount(snapshot.world.fronts))],
          [tr("hooks"), String(recordCount(snapshot.world.story_hooks))],
          [
            tr("reactions"),
            String(recordCount(snapshot.world.world_reactions)),
          ],
        ]}
      />
      <CardsSection
        title={tr("activeThreats")}
        cards={cardsFromValue(snapshot.world.fronts, tr("front"))}
        emptyLabel={tr("noThreats")}
      />
      <CardsSection
        title={tr("openHooks")}
        cards={cardsFromValue(snapshot.world.story_hooks, tr("hook"))}
        emptyLabel={tr("noHooks")}
      />
      <CardsSection
        title={tr("worldFallout")}
        cards={cardsFromValue(snapshot.world.world_reactions, tr("reaction"))}
        emptyLabel={tr("noFallout")}
      />
      <CardsSection
        title={tr("scenePressure")}
        cards={cardsFromValue(snapshot.world.scene_contract, tr("scene"))}
        emptyLabel={tr("noScenePressure")}
      />
    </>
  );
}

function InvestigationsModule({ snapshot }: { snapshot: StorySnapshot }) {
  return (
    <>
      <ModuleOverview
        label={tr("investigationsLabel")}
        title={tr("investigationsTitle")}
        description={tr("investigationsDescription")}
        metrics={[
          [tr("cases"), String(recordCount(snapshot.world.investigations))],
          [tr("signals"), String(flagRows(snapshot).length)],
          [
            tr("evidence"),
            String(
              messageCards(snapshot, [
                "clue",
                "investigation",
                "examine",
                "search",
              ]).length,
            ),
          ],
        ]}
      />
      <CardsSection
        title={tr("openCases")}
        cards={cardsFromValue(
          snapshot.world.investigations,
          tr("investigation"),
        )}
        emptyLabel={tr("noInvestigation")}
      />
      <InspectorSection
        title={tr("structuredSignals")}
        rows={flagRows(snapshot)}
      />
      <CardsSection
        title={tr("recentEvidence")}
        cards={messageCards(snapshot, [
          "clue",
          "investigation",
          "examine",
          "search",
        ])}
        emptyLabel={tr("noEvidence")}
      />
    </>
  );
}

function ProjectsModule({ snapshot }: { snapshot: StorySnapshot }) {
  return (
    <>
      <ModuleOverview
        label={tr("projectsLabel")}
        title={tr("projectsTitle")}
        description={tr("projectsDescription")}
        metrics={[
          [tr("projects"), String(recordCount(snapshot.world.projects))],
          [tr("guidance"), String(recordCount(snapshot.world.guidance))],
          [
            tr("factions"),
            String(recordCount(snapshot.world.faction_standings)),
          ],
        ]}
      />
      <CardsSection
        title={tr("ongoingWork")}
        cards={cardsFromValue(snapshot.world.projects, tr("project"))}
        emptyLabel={tr("noProject")}
      />
      <CardsSection
        title={tr("playerGuidance")}
        cards={cardsFromValue(snapshot.world.guidance, tr("guidance"))}
        emptyLabel={tr("noGuidance")}
      />
      <CardsSection
        title={tr("factionContext")}
        cards={cardsFromValue(snapshot.world.faction_standings, tr("faction"))}
        emptyLabel={tr("noFaction")}
      />
    </>
  );
}

function AchievementsModule({ snapshot }: { snapshot: StorySnapshot }) {
  const achievements = snapshot.panels.achievements;
  const categories = new Set(
    achievements.map((item) => item.category).filter(Boolean),
  );
  const rare = achievements.filter((item) =>
    /rare|epic|legend/i.test(item.rarity || ""),
  ).length;
  return (
    <>
      <ModuleOverview
        label={tr("achievementsLabel")}
        title={tr("achievementsTitle")}
        description={tr("achievementsDescription")}
        metrics={[
          [tr("earned"), String(achievements.length)],
          [tr("categories"), String(categories.size)],
          [tr("rarePlus"), String(rare)],
        ]}
      />
      <CardsSection
        title={tr("earnedAchievements")}
        cards={achievementCards(snapshot)}
        emptyLabel={tr("noAchievements")}
      />
    </>
  );
}

function SavesModule({ snapshot }: { snapshot: StorySnapshot }) {
  return (
    <>
      <ModuleOverview
        label={tr("savesLabel")}
        title={tr("savesTitle")}
        description={tr("savesDescription")}
        metrics={[
          [tr("saves"), String(snapshot.panels.saves.length)],
          [tr("sessions"), String(snapshot.panels.sessions.length)],
          [tr("currentTurn"), String(snapshot.world.current_turn)],
        ]}
      />
      <CardsSection
        title={tr("savedSnapshots")}
        cards={saveCards(snapshot)}
        emptyLabel={tr("noSaves")}
      />
      <CardsSection
        title={tr("playSessions")}
        cards={sessionCards(snapshot)}
        emptyLabel={tr("noSessions")}
      />
    </>
  );
}

function ModuleOverview({
  label,
  title,
  description,
  metrics,
}: {
  label: string;
  title: string;
  description: string;
  metrics: Array<[string, string]>;
}) {
  return (
    <header className="module-overview">
      <span>{label}</span>
      <h3>{title}</h3>
      <p>{description}</p>
      <dl>
        {metrics.map(([metric, value]) => (
          <div key={metric}>
            <dt>{metric}</dt>
            <dd>{value}</dd>
          </div>
        ))}
      </dl>
    </header>
  );
}

function InspectorSection({
  title,
  rows,
}: {
  title: string;
  rows: Array<[string, string]>;
}) {
  return (
    <details className="inspector-section" open>
      <summary>
        <span>{title}</span>
        <ChevronDown size={14} />
      </summary>
      <div className="kv-list">
        {rows.length === 0 ? (
          <div className="empty-copy small">{tr("noData")}</div>
        ) : (
          rows.map(([label, value]) => (
            <div
              className={inspectorRowClass(label, value)}
              key={`${title}-${label}`}
            >
              <span>{localizedFieldLabel(label)}</span>
              <strong title={value}>{value}</strong>
            </div>
          ))
        )}
      </div>
    </details>
  );
}

function CardsSection({
  title,
  cards,
  emptyLabel,
  focusCardId,
  onOpenVisualAsset,
}: {
  title: string;
  cards: CardView[];
  emptyLabel: string;
  focusCardId?: string | null;
  onOpenVisualAsset?: (assetId: string) => void;
}) {
  const focusedCardRef = useRef<HTMLElement | null>(null);

  useEffect(() => {
    if (!focusCardId || !focusedCardRef.current) return;
    focusedCardRef.current.scrollIntoView({
      block: "center",
      behavior: "smooth",
    });
    focusedCardRef.current.focus({ preventScroll: true });
  }, [focusCardId, cards.length]);

  return (
    <details
      className="inspector-section card-section"
      data-section={sectionSlug(title)}
      open
    >
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
                ref={
                  isFocused
                    ? (node) => {
                        focusedCardRef.current = node;
                      }
                    : undefined
                }
                tabIndex={isFocused ? -1 : undefined}
              >
                {card.imageUrl && (
                  <div className="inspector-card-image">
                    {card.imageAssetId && onOpenVisualAsset ? (
                      <button
                        type="button"
                        onClick={() => onOpenVisualAsset(card.imageAssetId!)}
                        aria-label={tr("openImage", { title: card.title })}
                        title={tr("editCardImage", { title: card.title })}
                      >
                        <img src={card.imageUrl} alt="" />
                      </button>
                    ) : (
                      <img src={card.imageUrl} alt="" />
                    )}
                  </div>
                )}
                {!card.imageUrl && card.imageState && (
                  <div className="inspector-card-image pending">
                    {localizedImageStatus(card.imageState)}
                  </div>
                )}
                <h3 title={card.title}>{card.title}</h3>
                {card.rows.length > 0 && (
                  <div className="kv-list compact">
                    {card.rows.map(([label, value]) => (
                      <div
                        className={inspectorRowClass(label, value)}
                        key={`${card.title}-${label}`}
                      >
                        <span>{localizedFieldLabel(label)}</span>
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

const fieldLabelKeys: Record<string, string> = {
  active_front: "activeFront",
  alphabet: "alphabet",
  category: "category",
  chapter: "chapter",
  character: "character",
  condition: "condition",
  context: "context",
  created: "created",
  description: "description",
  effect: "effect",
  equipment: "equipment",
  location: "location",
  material: "material",
  materials: "materialsItems",
  name: "name",
  project: "project",
  rarity: "rarity",
  recipe: "recipe",
  region: "region",
  status: "status",
  summary: "summary",
  text: "text",
  time: "time",
  turn: "turn",
  turns: "turns",
  type: "type",
  used: "used",
  value: "value",
  weather: "weather",
};

function localizedFieldLabel(label: string): string {
  const numberedItem = label.match(/^Item (\d+)$/);
  if (numberedItem) return `${tr("item")} ${numberedItem[1]}`;
  const key = fieldLabelKeys[normalizeFieldKey(label)];
  return key ? tr(key) : label;
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
        <span>{tr("stats")}</span>
        <ChevronDown size={14} />
      </summary>
      <div className="stat-list">
        {stats.length === 0 ? (
          <div className="empty-copy small">{tr("noStats")}</div>
        ) : (
          stats.map(({ label, value, text }) => (
            <div className="stat-row" key={label}>
              <span>{localizedFieldLabel(label)}</span>
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
  const keys: Partial<Record<ModuleTab, string>> = {
    inventory: "moduleInventory",
    craft: "moduleCraft",
    stats: "moduleStats",
    codex: "moduleCodex",
    fronts: "moduleFronts",
    investigations: "moduleInvestigations",
    projects: "moduleProjects",
    achievements: "moduleAchievements",
    saves: "moduleSaves",
    history: "moduleHistory",
    map: "moduleMap",
  };
  const key = keys[tab];
  return key
    ? tr(key)
    : (moduleSpecs.find((item) => item.tab === tab)?.label ?? titleCase(tab));
}

function localizedCondition(condition: string): string {
  const keys: Record<string, string> = {
    idle: "conditionIdle",
    stable: "conditionStable",
    "not tracked": "conditionNotTracked",
    injured: "conditionInjured",
    exhausted: "conditionExhausted",
    focused: "conditionFocused",
  };
  const key = keys[condition.trim().toLowerCase()];
  return key ? tr(key) : condition;
}

function localizedImageStatus(status: string): string {
  const keys: Record<string, string> = {
    pending: "statusPending",
    queued: "statusQueued",
    running: "statusRunning",
    ready: "statusReady",
    failed: "statusFailed",
    cancelled: "statusCancelled",
  };
  const key = keys[status.trim().toLowerCase()];
  return key ? tr(key) : titleCase(status);
}

function localizedVisualReadiness(value: string): string {
  const keys: Record<string, string> = {
    canonical: "visualCanonical",
    provisional: "visualProvisional",
    observed: "visualObserved",
  };
  const key = keys[value];
  return key ? tr(key) : titleCase(value);
}

export function meterRows(
  snapshot: StorySnapshot,
): Array<{ label: string; value: number; text: string }> {
  const stats = asObject(snapshot.character.fields.stats);
  const preferred = ["health", "focus", "resolve", "stamina", "insight"];
  const nonMeterKeys = new Set([
    "currency",
    "deaths",
    "gold",
    "coins",
    "xp",
    "level",
  ]);
  const rows: Array<{ label: string; value: number; text: string }> = [];

  const vitals = asObject(stats.vitals);
  for (const [key, value] of Object.entries(vitals)) {
    const object = asObject(value);
    const current = numericStat(object.current);
    const max = numericStat(object.max);
    if (current !== null && max !== null && max > 0) {
      rows.push({
        label: titleCase(key),
        value: Math.min(100, Math.round((current / max) * 100)),
        text: `${current}/${max}`,
      });
    }
  }

  for (const key of preferred) {
    const value = numericStat(stats[key] ?? stats[titleCase(key)]);
    if (value !== null)
      rows.push({ label: titleCase(key), value, text: `${value}/100` });
  }
  for (const [key, value] of Object.entries(stats)) {
    if (nonMeterKeys.has(key.toLowerCase())) continue;
    if (rows.some((row) => row.label.toLowerCase() === key.toLowerCase()))
      continue;
    const stat = numericStat(value);
    if (stat !== null)
      rows.push({ label: titleCase(key), value: stat, text: `${stat}/100` });
  }
  return rows.slice(0, 7);
}

function conditionDetail(snapshot: StorySnapshot): string {
  const condition = deriveCondition(snapshot);
  return condition === "Not tracked"
    ? tr("conditionMissing")
    : tr("conditionCanonical");
}

function flagRows(snapshot: StorySnapshot): Array<[string, string]> {
  const rows: Array<[string, string]> = [];
  collectFlags(rows, snapshot.world.scene_contract, "scene");
  collectFlags(rows, snapshot.world.story_hooks, "hook");
  collectFlags(rows, snapshot.world.investigations, "investigation");
  return rows.slice(0, 8);
}

function collectFlags(
  rows: Array<[string, string]>,
  value: JsonValue,
  prefix: string,
) {
  if (Array.isArray(value)) {
    value
      .slice(0, 5)
      .forEach((item, index) =>
        rows.push([
          `${prefix}_${index + 1}`,
          compactText(entryLabel(item, index), 34),
        ]),
      );
    return;
  }
  const object = asObject(value);
  for (const [key, child] of Object.entries(object)) {
    if (rows.length > 10) break;
    if (typeof child === "boolean") rows.push([key, child ? "true" : "false"]);
    else if (typeof child === "string" || typeof child === "number")
      rows.push([key, compactText(String(child), 34)]);
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
      ...fieldRows(object)
        .filter(([key]) => !["Name", "Title", "Id"].includes(key))
        .slice(0, 4),
    ];
  }
  return [["Name", compactText(valueToText(active), 64)]];
}

function inventoryContextRows(
  snapshot: StorySnapshot,
): Array<[string, string]> {
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
    .filter(([key]) =>
      [
        "Tools",
        "Materials",
        "Weather",
        "Light",
        "Pressure",
        "Risk",
        "Opportunity",
        "Constraint",
        "Details",
      ].includes(key),
    )
    .slice(0, 8);
  if (rows.length > 0) return rows;
  return [
    [
      "Weather",
      findString(snapshot.world.scene_contract, [
        "weather",
        "forecast",
        "sky",
      ]) || "Untracked",
    ],
    ["Context", compactText(valueToText(snapshot.world.scene_contract), 90)],
  ];
}

function traitRows(snapshot: StorySnapshot): Array<[string, string]> {
  const stats = asObject(snapshot.character.fields.stats);
  const traits = asArray(snapshot.character.fields.traits).length
    ? asArray(snapshot.character.fields.traits)
    : asArray(stats.traits);
  return traits
    .map(
      (trait, index) =>
        [
          tr("trait", { number: index + 1 }),
          compactText(valueToText(trait), 80),
        ] as [string, string],
    )
    .slice(0, 12);
}

function counterRows(stats: JsonObject): Array<[string, string]> {
  const counters = ["currency", "gold", "coins", "deaths", "level", "xp"];
  const rows: Array<[string, string]> = [];
  for (const key of counters) {
    const value = stats[key] ?? stats[titleCase(key)];
    if (value !== undefined && value !== null)
      rows.push([titleCase(key), compactText(valueToText(value), 80)]);
  }
  return rows;
}

function messageCards(
  snapshot: StorySnapshot,
  keywords: string[] = [],
): CardView[] {
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
      title: `${message.role === "user" ? tr("roleUser") : message.role === "assistant" ? tr("roleAssistant") : message.role} - ${tr("turn")} ${message.turn}`,
      rows: [
        ["Type", message.message_type || message.role],
        ["Text", readableStructuredText(message.content)],
      ],
    }));
}

function chapterCards(snapshot: StorySnapshot): CardView[] {
  return snapshot.panels.chapters
    .slice(-8)
    .reverse()
    .map((chapter) => ({
      title: chapter.title || `${tr("chapter")} ${chapter.chapter_number}`,
      rows: [
        ["Chapter", String(chapter.chapter_number)],
        [
          "Turns",
          chapter.end_turn
            ? `${chapter.start_turn}-${chapter.end_turn}`
            : `${chapter.start_turn}+`,
        ],
        ["Summary", compactText(chapter.summary || "-", 260)],
      ],
    }));
}

function npcCards(
  snapshot: StorySnapshot,
  visuals?: VisualCatalog,
): CardView[] {
  return snapshot.panels.npcs.slice(0, 12).map((npc) => {
    const asset = visuals ? characterAsset(visuals, npc) : null;
    return {
      id: npc.id,
      title: npc.name,
      imageUrl: readyAssetUrl(asset),
      imageState: asset?.status,
      imageAssetId: asset?.id,
      rows: [
        ...fieldRows(npc.fields)
          .filter(
            ([key]) =>
              !["Name", "Id"].includes(key) && !isPlayerHiddenField(key),
          )
          .map(
            ([key, value]) =>
              [key, compactText(value, 220)] as [string, string],
          )
          .slice(0, 7),
      ],
    };
  });
}

function locationCards(
  snapshot: StorySnapshot,
  visuals?: VisualCatalog,
): CardView[] {
  const source = asArray(snapshot.world.spatial_locations).length
    ? asArray(snapshot.world.spatial_locations)
    : asArray(snapshot.world.known_locations);
  return source.slice(0, 16).map((location, index) => {
    const object = asObject(location);
    const id = typeof object.id === "string" ? object.id : "";
    const name =
      typeof object.name === "string"
        ? object.name
        : entryLabel(location, index);
    const asset = visuals?.mapIcons.get(normalizeKey(id || name));
    const card = cardFromEntry(location, name, index);
    return {
      ...card,
      id: id || undefined,
      imageUrl: readyAssetUrl(asset),
      imageState: asset?.status,
      imageAssetId: asset?.id,
    };
  });
}

function recordCount(value: JsonValue | undefined): number {
  if (Array.isArray(value)) return value.length;
  if (value && typeof value === "object") return Object.keys(value).length;
  return value === undefined || value === null || value === "" ? 0 : 1;
}

function saveCards(snapshot: StorySnapshot): CardView[] {
  return snapshot.panels.saves.slice(0, 12).map((save) => ({
    title: save.name || tr("save", { turn: save.turn }),
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
      ["Status", session.ended_at ? tr("ended") : tr("active")],
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

export function cardsFromValue(
  value: JsonValue | undefined,
  fallbackTitle: string,
): CardView[] {
  if (Array.isArray(value)) {
    return value.slice(0, 16).map((item, index) => {
      const label = entryLabel(item, index);
      return cardFromEntry(
        item,
        label === `Item ${index + 1}` ? `${fallbackTitle} ${index + 1}` : label,
        index,
      );
    });
  }

  const object = asObject(value);
  const entries = Object.entries(object);
  if (entries.length === 0) {
    if (value && typeof value === "object") return [];
    if (value === undefined || value === null) return [];
    return [
      {
        title: fallbackTitle,
        rows: [["Value", compactText(valueToText(value), 220)]],
      },
    ];
  }

  const complexEntries = entries.filter(
    ([, child]) => child && typeof child === "object",
  );
  if (complexEntries.length > 0) {
    return complexEntries
      .slice(0, 16)
      .map(([key, child], index) =>
        cardFromEntry(child, titleCase(key), index),
      );
  }

  return [
    {
      title: fallbackTitle,
      rows: entries
        .filter(([key]) => !isPlayerHiddenField(key))
        .slice(0, 12)
        .map(([key, child]) => [
          titleCase(key),
          compactText(valueToText(child), 220),
        ]),
    },
  ];
}

function cardFromEntry(
  value: JsonValue,
  title: string,
  index: number,
): CardView {
  const rows = fieldRows(value)
    .filter(
      ([key]) =>
        !["Name", "Title", "Id"].includes(key) && !isPlayerHiddenField(key),
    )
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
  if (Array.isArray(value))
    return value.map((item) => sanitizePlayerVisibleValue(item));
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
