import { useEffect, useRef, useState } from "react";
import { RefreshCw, Volume2 } from "lucide-react";
import { cancelAudioJob, createMessageAudio, getMessageAudio, getTTSSettings, retryAudioJob } from "../api";
import type { AudioAsset, MessageAudioResponse } from "../types";

const settingsRequests = new Map<string, Promise<boolean>>();

function autoplayForStory(storyId: string): Promise<boolean> {
  const existing = settingsRequests.get(storyId);
  if (existing) return existing;
  const request = getTTSSettings(storyId)
    .then((response) => response.settings.autoplay)
    .catch((error) => { settingsRequests.delete(storyId); throw error; });
  settingsRequests.set(storyId, request);
  return request;
}

export function AudioControls({ storyId, messageId }: { storyId: string; messageId: number }) {
  const [response, setResponse] = useState<MessageAudioResponse>({ assets: [], jobs: [] });
  const [autoplay, setAutoplay] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const played = useRef(new Set<string>());

  const load = async () => {
    try {
      const [audio, settings] = await Promise.all([
        getMessageAudio(storyId, messageId),
        autoplayForStory(storyId),
      ]);
      setResponse(audio);
      setAutoplay(settings);
      setError("");
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Audio status unavailable");
    }
  };

  useEffect(() => { void load(); }, [storyId, messageId]);

  useEffect(() => {
    const update = (event: Event) => {
      const detail = (event as CustomEvent<{ storyId: string; autoplay: boolean }>).detail;
      if (detail?.storyId === storyId) {
        settingsRequests.set(storyId, Promise.resolve(detail.autoplay));
        setAutoplay(detail.autoplay);
      }
    };
    window.addEventListener("oneday:tts-settings", update);
    return () => window.removeEventListener("oneday:tts-settings", update);
  }, [storyId]);

  useEffect(() => {
    if (!autoplay) return;
    const first = response.assets.find((asset) => asset.status === "ready" && !played.current.has(asset.id));
    if (!first) return;
    const audio = new Audio(assetUrl(first));
    played.current.add(first.id);
    void audio.play().catch(() => setError("Autoplay was blocked. Use the playback control below."));
  }, [autoplay, response.assets]);

  useEffect(() => {
    if (!response.jobs.some((job) => job.status === "queued" || job.status === "running")) return;
    const timer = window.setTimeout(() => { void load(); }, 1500);
    return () => window.clearTimeout(timer);
  }, [response.jobs, storyId, messageId]);

  const generate = async () => {
    setBusy(true);
    setError("");
    try {
      const retryable = response.jobs.find((job) => job.status === "failed" || job.status === "cancelled" || job.status === "canceled");
      setResponse(retryable ? await retryAudioJob(storyId, retryable.id) : await createMessageAudio(storyId, messageId));
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Audio generation failed");
    } finally {
      setBusy(false);
    }
  };

  const cancel = async () => {
    const job = response.jobs.find((item) => item.status === "queued" || item.status === "running");
    if (!job) return;
    setBusy(true); setError("");
    try { setResponse(await cancelAudioJob(storyId, job.id)); }
    catch (cause) { setError(cause instanceof Error ? cause.message : "Could not cancel audio generation"); }
    finally { setBusy(false); }
  };

  const ready = response.assets.filter((asset) => asset.status === "ready");
  const active = response.assets.some((asset) => asset.status === "queued" || asset.status === "running");
  const failed = response.assets.find((asset) => asset.status === "failed" || asset.status === "cancelled" || asset.status === "canceled");

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
