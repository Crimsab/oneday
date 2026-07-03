import { ChevronDown, RefreshCw } from "lucide-react";
import { asArray, asObject, compactText, deriveCondition, displayClock, entryLabel, fieldRows, findString, numericStat, titleCase, valueToText } from "../format";
import type { JsonObject, JsonValue, StorySnapshot } from "../types";

interface InspectorProps {
  snapshot: StorySnapshot | null;
  onRefresh: () => void;
}

export function Inspector({ snapshot, onRefresh }: InspectorProps) {
  return (
    <aside className="right-inspector">
      <div className="inspector-head">
        <h2>Canonical State Inspector</h2>
        <button type="button" className="square-button" onClick={onRefresh} title="Refresh snapshot">
          <RefreshCw size={15} />
        </button>
      </div>
      {!snapshot ? (
        <div className="empty-copy inspector-empty">Select a story to inspect canonical state.</div>
      ) : (
        <div className="inspector-body">
          <InspectorSection title="Turn & Time" rows={turnRows(snapshot)} />
          <InspectorSection title="Location" rows={locationRows(snapshot)} />
          <StatsSection snapshot={snapshot} />
          <InspectorSection title="Condition" rows={conditionRows(snapshot)} />
          <InspectorSection title="Flags" rows={flagRows(snapshot)} />
          <InspectorSection title="Relationships" rows={relationshipRows(snapshot)} />
          <InspectorSection title="Active Front" rows={activeFrontRows(snapshot)} />
        </div>
      )}
    </aside>
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

function StatsSection({ snapshot }: { snapshot: StorySnapshot }) {
  const stats = statRows(snapshot);
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
          stats.map(([label, value]) => (
            <div className="stat-row" key={label}>
              <span>{label}</span>
              <div className="stat-meter">
                <i style={{ width: `${value}%` }} />
              </div>
              <strong>{value}/100</strong>
            </div>
          ))
        )}
      </div>
    </details>
  );
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

function statRows(snapshot: StorySnapshot): Array<[string, number]> {
  const stats = asObject(snapshot.character.fields.stats);
  const preferred = ["health", "focus", "resolve", "stamina", "insight"];
  const rows: Array<[string, number]> = [];
  for (const key of preferred) {
    const value = numericStat(stats[key] ?? stats[titleCase(key)]);
    if (value !== null) rows.push([titleCase(key), value]);
  }
  for (const [key, value] of Object.entries(stats)) {
    if (rows.some(([label]) => label.toLowerCase() === key.toLowerCase())) continue;
    const stat = numericStat(value);
    if (stat !== null) rows.push([titleCase(key), stat]);
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
