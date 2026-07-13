import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { RefreshCw, Volume2 } from "lucide-react";
import { cancelAudioJob, createMessageAudio, getMessageAudio, getTTSSettings, retryAudioJob } from "../api";
import type { AudioAsset, MessageAudioResponse, StoryTTSSettings } from "../types";

const settingsRequests = new Map<string, Promise<StoryTTSSettings>>();
const autoplayedMessages = new Set<string>();

function settingsForStory(storyId: string): Promise<StoryTTSSettings> {
  const existing = settingsRequests.get(storyId);
  if (existing) return existing;
  const request = getTTSSettings(storyId)
    .then((response) => response.settings)
    .catch((error) => { settingsRequests.delete(storyId); throw error; });
  settingsRequests.set(storyId, request);
  return request;
}

export function useStoryTTSSettings(storyId: string): StoryTTSSettings | null {
  const [settings, setSettings] = useState<StoryTTSSettings | null>(null);

  useEffect(() => {
    let active = true;
    setSettings(null);
    if (!storyId) return () => { active = false; };

    void settingsForStory(storyId).then((nextSettings) => {
      if (active) setSettings(nextSettings);
    }).catch(() => undefined);

    const update = (event: Event) => {
      const detail = (event as CustomEvent<{ storyId: string; settings: StoryTTSSettings }>).detail;
      if (detail?.storyId !== storyId) return;
      settingsRequests.set(storyId, Promise.resolve(detail.settings));
      setSettings(detail.settings);
    };
    window.addEventListener("oneday:tts-settings", update);
    return () => {
      active = false;
      window.removeEventListener("oneday:tts-settings", update);
    };
  }, [storyId]);

  return settings;
}

export function AudioControls({ storyId, messageId, settings, autoplay = false }: { storyId: string; messageId: number; settings: StoryTTSSettings; autoplay?: boolean }) {
  const [response, setResponse] = useState<MessageAudioResponse>({ assets: [], jobs: [] });
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const requestVersion = useRef(0);

  const load = useCallback(async () => {
    const version = ++requestVersion.current;
    try {
      const audio = await getMessageAudio(storyId, messageId);
      if (version !== requestVersion.current) return;
      setResponse(audio);
      setError("");
    } catch (cause) {
      if (version !== requestVersion.current) return;
      setError(cause instanceof Error ? cause.message : "Audio status unavailable");
    }
  }, [messageId, storyId]);

  useEffect(() => {
    void load();
    return () => { requestVersion.current += 1; };
  }, [load]);

  const autoplayAsset = useMemo(() => firstReadyAudioAsset(response.assets), [response.assets]);
  useEffect(() => {
    if (!autoplay || !settings?.autoplay || settings.mode === "off" || !autoplayAsset) return;
    const claim = `${storyId}:${messageId}`;
    if (autoplayedMessages.has(claim)) return;
    autoplayedMessages.add(claim);
    const audio = new Audio(assetUrl(autoplayAsset));
    void audio.play().catch(() => setError("Autoplay was blocked. Use the playback control below."));
    return () => {
      audio.pause();
      audio.removeAttribute("src");
      audio.load();
    };
  }, [autoplay, autoplayAsset?.id, messageId, settings?.autoplay, settings?.mode, storyId]);

  useEffect(() => {
    if (!response.jobs.some((job) => job.status === "queued" || job.status === "running")) return;
    const timer = window.setTimeout(() => { void load(); }, 1500);
    return () => window.clearTimeout(timer);
  }, [load, response.jobs]);

  const generate = async () => {
    const version = ++requestVersion.current;
    setBusy(true);
    setError("");
    try {
      const retryable = response.jobs.find((job) => job.status === "failed" || job.status === "cancelled" || job.status === "canceled");
      const nextResponse = retryable ? await retryAudioJob(storyId, retryable.id) : await createMessageAudio(storyId, messageId);
      if (version === requestVersion.current) setResponse(nextResponse);
    } catch (cause) {
      if (version !== requestVersion.current) return;
      setError(cause instanceof Error ? cause.message : "Audio generation failed");
    } finally {
      if (version === requestVersion.current) setBusy(false);
    }
  };

  const cancel = async () => {
    const job = response.jobs.find((item) => item.status === "queued" || item.status === "running");
    if (!job) return;
    const version = ++requestVersion.current;
    setBusy(true); setError("");
    try {
      const nextResponse = await cancelAudioJob(storyId, job.id);
      if (version === requestVersion.current) setResponse(nextResponse);
    }
    catch (cause) {
      if (version === requestVersion.current) setError(cause instanceof Error ? cause.message : "Could not cancel audio generation");
    }
    finally { if (version === requestVersion.current) setBusy(false); }
  };

  const ready = response.assets.filter((asset) => asset.status === "ready");
  const active = response.assets.some((asset) => asset.status === "queued" || asset.status === "running");
  const failed = response.assets.find((asset) => asset.status === "failed" || asset.status === "cancelled" || asset.status === "canceled");

  if (!settings || settings.mode === "off") return null;

  return (
    <section className="message-audio" aria-label="Spoken audio">
      <div className="message-audio-head">
        <span><Volume2 size={15} aria-hidden="true" /> Spoken audio</span>
        <button type="button" disabled={busy || active} onClick={generate}>
          {failed ? <RefreshCw size={14} aria-hidden="true" /> : <Volume2 size={14} aria-hidden="true" />}
          {busy ? "Generating…" : failed ? "Retry" : ready.length ? "Regenerate" : "Generate"}
        </button>
        {active && <button type="button" disabled={busy} onClick={cancel}>Cancel</button>}
      </div>
      {ready.map((asset) => (
        <audio key={asset.id} controls preload="none" src={assetUrl(asset)}>
          Your browser does not support audio playback.
        </audio>
      ))}
      <div className="message-audio-status" aria-live="polite">
        {active ? "Speech synthesis is in progress." : failed?.error || error}
      </div>
    </section>
  );
}

export function assetUrl(asset: AudioAsset): string {
  return `/api/audio/${encodeURIComponent(asset.id)}`;
}

export function firstReadyAudioAsset(assets: AudioAsset[]): AudioAsset | undefined {
  return assets.find((asset) => asset.status === "ready");
}
