import { useEffect, useMemo, useState } from "react";
import { getTTSCatalog, getTTSSettings, getVoiceAssignments, updateTTSSettings, updateVoiceAssignment } from "../api";
import type { RecordView, StoryTTSSettings, VoiceAssignment, VoiceProfile } from "../types";
import { AudioLanguageTools } from "./AudioLanguageTools";

interface VoiceAssignmentEditorProps {
  storyId: string;
  language: string;
  revision: number;
  protagonist: RecordView;
  npcs: RecordView[];
}

export function VoiceAssignmentEditor({ storyId, language, revision, protagonist, npcs }: VoiceAssignmentEditorProps) {
  const [settings, setSettings] = useState<StoryTTSSettings | null>(null);
  const [voices, setVoices] = useState<VoiceProfile[]>([]);
  const [providers, setProviders] = useState<{ id: string; available: boolean; reason?: string }[]>([]);
  const [assignments, setAssignments] = useState<VoiceAssignment[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const targets = useMemo(() => [
    { key: "narrator", label: "Narrator", role: "narrator" as const, entityId: "", importance: "supporting" as const },
    { key: `protagonist:${protagonist.id}`, label: protagonist.name || "Protagonist", role: "protagonist" as const, entityId: protagonist.id, importance: "major" as const },
    ...npcs.map((npc) => ({ key: `npc:${npc.id}`, label: npc.name, role: "npc" as const, entityId: npc.id, importance: "supporting" as const })),
  ], [npcs, protagonist]);

  useEffect(() => {
    setError("");
    void Promise.all([getTTSSettings(storyId), getTTSCatalog(language), getVoiceAssignments(storyId)])
      .then(([settingsResponse, catalog, assignmentResponse]) => {
        setSettings(settingsResponse.settings);
        setVoices(catalog.voices || []);
        setProviders(catalog.providers || []);
        setAssignments(assignmentResponse.assignments || []);
      })
      .catch((cause) => setError(cause instanceof Error ? cause.message : "Voice settings unavailable"));
  }, [storyId, language]);

  const saveSettings = async () => {
    if (!settings) return;
    setBusy(true); setError("");
    try {
      const saved = (await updateTTSSettings(storyId, settings, revision)).settings;
      setSettings(saved);
      window.dispatchEvent(new CustomEvent("oneday:tts-settings", { detail: { storyId, autoplay: saved.autoplay } }));
    }
    catch (cause) { setError(cause instanceof Error ? cause.message : "Could not save speech settings"); }
    finally { setBusy(false); }
  };

  const saveTarget = async (target: typeof targets[number], voiceId: string, enabledMode: VoiceAssignment["enabled_mode"]) => {
    const existing = assignments.find((item) => item.assignment_key === assignmentKey(target.role, target.entityId));
    if (!voiceId && !existing) return;
    const assignment: VoiceAssignment = {
      id: existing?.id || crypto.randomUUID(), assignment_key: assignmentKey(target.role, target.entityId), story_id: storyId,
      entity_id: target.entityId || undefined, role: target.role, voice_profile_id: voiceId || existing!.voice_profile_id,
      enabled_mode: enabledMode, language_tag: language, locked: existing?.locked || false,
      importance: target.importance, allow_duplicate: existing?.allow_duplicate || false,
    };
    setBusy(true); setError("");
    try {
      const saved = (await updateVoiceAssignment(storyId, assignment, revision)).assignment;
      if (saved) setAssignments((items) => [...items.filter((item) => item.id !== saved.id), saved]);
    } catch (cause) { setError(cause instanceof Error ? cause.message : "Could not save voice assignment"); }
    finally { setBusy(false); }
  };

  if (!settings) return <section className="voice-settings"><h3>Spoken audio</h3><p className="empty-copy">{error || "Loading voice registry…"}</p></section>;

  return (
    <section className="voice-settings" aria-labelledby="voice-settings-title">
      <div className="settings-section-head"><div><h3 id="voice-settings-title">Spoken audio</h3><p>Committed narration and dialogue only. Autoplay is controlled separately.</p></div><button type="button" disabled={busy} onClick={saveSettings}>Save audio settings</button></div>
      <div className="settings-grid voice-global-settings">
        <label><span>Speech mode</span><select value={settings.mode} onChange={(event) => setSettings({ ...settings, mode: event.target.value as StoryTTSSettings["mode"] })}><option value="off">Off</option><option value="narrator">Narrator only</option><option value="dialogue">Dialogue only</option><option value="all">Narration and dialogue</option></select></label>
        <label><span>Default language</span><input value={settings.default_language_tag} onChange={(event) => setSettings({ ...settings, default_language_tag: event.target.value })} placeholder="en-US" /></label>
        <label className="toggle-row"><span>Autoplay new audio</span><input type="checkbox" checked={settings.autoplay} onChange={(event) => setSettings({ ...settings, autoplay: event.target.checked })} /></label>
      </div>
      <div className="provider-status" aria-label="TTS provider status">{providers.map((provider) => <span key={provider.id} className={provider.available ? "available" : "unavailable"}>{provider.id}: {provider.available ? "available" : provider.reason || "unavailable"}</span>)}</div>
      {voices.length === 0 ? <p className="empty-copy">No enabled voice is currently available. Story audio remains safely off.</p> : (
        <div className="voice-assignment-list">
          {targets.map((target) => {
            const current = assignments.find((item) => item.assignment_key === assignmentKey(target.role, target.entityId));
            return <div className="voice-assignment-row" key={target.key}><strong>{target.label}</strong><label><span>Voice</span><select disabled={busy || current?.locked} value={current?.voice_profile_id || ""} onChange={(event) => void saveTarget(target, event.target.value, current?.enabled_mode || "inherit")}><option value="">Select a voice</option>{voices.map((voice) => <option key={voice.id} value={voice.id}>{voice.display_name} · {voice.provider}</option>)}</select></label><label><span>Playback</span><select disabled={busy} value={current?.enabled_mode || "inherit"} onChange={(event) => void saveTarget(target, current?.voice_profile_id || voices[0]?.id || "", event.target.value as VoiceAssignment["enabled_mode"])}><option value="inherit">Inherit story</option><option value="on">On</option><option value="off">Off</option></select></label></div>;
          })}
        </div>
      )}
      <p className="settings-feedback" role="status" aria-live="polite">{busy ? "Saving…" : error}</p>
      <AudioLanguageTools storyId={storyId} language={settings.default_language_tag || language} revision={revision} />
    </section>
  );
}

function assignmentKey(role: VoiceAssignment["role"], entityId: string): string {
  return `${role}:${entityId}::`;
}
