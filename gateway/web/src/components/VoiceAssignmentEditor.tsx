import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getTTSCatalog, getTTSSettings, getVoiceAssignments, updateTTSSettings, updateVoiceAssignment } from "../api";
import type { RecordView, StoryTTSSettings, VoiceAssignment, VoiceProfile } from "../types";
import { AudioLanguageTools } from "./AudioLanguageTools";
import { CustomSelect } from "./CustomSelect";

interface VoiceAssignmentEditorProps {
  storyId: string;
  language: string;
  revision: number;
  protagonist: RecordView;
  npcs: RecordView[];
  heading?: string;
}

export function VoiceAssignmentEditor({ storyId, language, revision, protagonist, npcs, heading }: VoiceAssignmentEditorProps) {
  const { t } = useTranslation(["audio", "common", "surfaces"]);
  const [settings, setSettings] = useState<StoryTTSSettings | null>(null);
  const [voices, setVoices] = useState<VoiceProfile[]>([]);
  const [providers, setProviders] = useState<{ id: string; available: boolean; reason?: string }[]>([]);
  const [assignments, setAssignments] = useState<VoiceAssignment[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const targets = useMemo(() => [
    { key: "narrator", label: t("surfaces:voice.narrator"), role: "narrator" as const, entityId: "", importance: "supporting" as const },
    { key: `protagonist:${protagonist.id}`, label: protagonist.name || t("surfaces:voice.protagonist"), role: "protagonist" as const, entityId: protagonist.id, importance: "major" as const },
    ...npcs.map((npc) => ({ key: `npc:${npc.id}`, label: npc.name, role: "npc" as const, entityId: npc.id, importance: "supporting" as const })),
  ], [npcs, protagonist, t]);

  useEffect(() => {
    setError("");
    void Promise.all([getTTSSettings(storyId), getTTSCatalog(language), getVoiceAssignments(storyId)])
      .then(([settingsResponse, catalog, assignmentResponse]) => {
        setSettings(settingsResponse.settings);
        setVoices(catalog.voices || []);
        setProviders(catalog.providers || []);
        setAssignments(assignmentResponse.assignments || []);
      })
      .catch(() => setError(t("audio:unavailable")));
  }, [storyId, language]);

  const saveSettings = async () => {
    if (!settings) return;
    setBusy(true); setError("");
    try {
      const saved = (await updateTTSSettings(storyId, settings, revision)).settings;
      setSettings(saved);
      window.dispatchEvent(new CustomEvent("oneday:tts-settings", { detail: { storyId, settings: saved } }));
    }
    catch { setError(t("audio:saveFailed")); }
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
    } catch { setError(t("audio:assignmentFailed")); }
    finally { setBusy(false); }
  };

  const displayHeading = heading || t("surfaces:voice.spokenAudio");
  if (!settings) return <section className="voice-settings"><h3>{displayHeading}</h3><p className="empty-copy">{error || t("audio:load")}</p></section>;

  return (
    <section className="voice-settings" aria-labelledby="voice-settings-title">
      <div className="settings-section-head"><div><h3 id="voice-settings-title">{displayHeading}</h3><p>{t("audio:committed")}</p></div><button type="button" disabled={busy} onClick={saveSettings}>{t("audio:save")}</button></div>
      <div className="settings-grid voice-global-settings">
        <label><span>{t("surfaces:voice.mode")}</span><CustomSelect value={settings.mode} ariaLabel={t("surfaces:voice.mode")} onChange={(value) => setSettings({ ...settings, mode: value as StoryTTSSettings["mode"] })} options={[{ value: "off", label: t("surfaces:voice.off") }, { value: "narrator", label: t("surfaces:voice.narratorOnly") }, { value: "dialogue", label: t("surfaces:voice.dialogueOnly") }, { value: "all", label: t("surfaces:voice.all") }]} /></label>
        <label><span>{t("surfaces:voice.defaultLanguage")}</span><input value={settings.default_language_tag} onChange={(event) => setSettings({ ...settings, default_language_tag: event.target.value })} placeholder="en-US" /></label>
        <label className="toggle-row"><span>{t("surfaces:voice.autoplay")}</span><input type="checkbox" checked={settings.autoplay} onChange={(event) => setSettings({ ...settings, autoplay: event.target.checked })} /></label>
      </div>
      <div className="provider-status" aria-label={t("surfaces:voice.providerStatus")}>{providers.map((provider) => <span key={provider.id} className={provider.available ? "available" : "unavailable"}>{provider.id}: {provider.available ? t("surfaces:voice.available") : provider.reason || t("surfaces:voice.unavailable")}</span>)}</div>
      {voices.length === 0 ? <p className="empty-copy">{t("surfaces:voice.noVoices")}</p> : (
        <div className="voice-assignment-list">
          {targets.map((target) => {
            const current = assignments.find((item) => item.assignment_key === assignmentKey(target.role, target.entityId));
            return <div className="voice-assignment-row" key={target.key}><strong>{target.label}</strong><label><span>{t("surfaces:voice.voice")}</span><CustomSelect disabled={busy || current?.locked} value={current?.voice_profile_id || ""} ariaLabel={t("surfaces:voice.voiceFor", { name: target.label })} onChange={(value) => void saveTarget(target, value, current?.enabled_mode || "inherit")} options={[{ value: "", label: t("surfaces:voice.select") }, ...voices.map((voice) => ({ value: voice.id, label: `${voice.display_name} · ${voice.provider}` }))]} /></label><label><span>{t("surfaces:voice.playback")}</span><CustomSelect disabled={busy} value={current?.enabled_mode || "inherit"} ariaLabel={t("surfaces:voice.playbackFor", { name: target.label })} onChange={(value) => void saveTarget(target, current?.voice_profile_id || voices[0]?.id || "", value as VoiceAssignment["enabled_mode"])} options={[{ value: "inherit", label: t("surfaces:voice.inherit") }, { value: "on", label: t("surfaces:voice.on") }, { value: "off", label: t("surfaces:voice.off") }]} /></label></div>;
          })}
        </div>
      )}
      <p className="settings-feedback" role="status" aria-live="polite">{busy ? t("surfaces:voice.saving") : error}</p>
      <AudioLanguageTools storyId={storyId} language={settings.default_language_tag || language} revision={revision} />
    </section>
  );
}

function assignmentKey(role: VoiceAssignment["role"], entityId: string): string {
  return `${role}:${entityId}::`;
}
