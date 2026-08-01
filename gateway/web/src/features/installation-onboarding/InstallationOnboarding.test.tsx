import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { InstallationOnboarding, installationSetupItems } from "./InstallationOnboarding";
import type { SetupReadinessReport } from "../../types";
import { setInterfaceLocale } from "../../i18n";

function readiness(overrides: Partial<SetupReadinessReport> = {}): SetupReadinessReport {
  return {
    probes: [
      { name: "narrative", code: "NARRATIVE_READY", status: "ready", required: true, summary: "narrative provider is ready", action: "" },
      { name: "embeddings", code: "EMBEDDINGS_DISABLED", status: "skipped", required: false, summary: "RAG embeddings are disabled", action: "" },
      { name: "image", code: "IMAGE_DISABLED", status: "skipped", required: false, summary: "image generation is disabled", action: "" },
      { name: "tts", code: "TTS_DISABLED", status: "skipped", required: false, summary: "text-to-speech is disabled", action: "" },
      { name: "gateway", code: "GATEWAY_NOT_CONFIGURED", status: "skipped", required: false, summary: "gateway readiness is disabled", action: "configure" },
      { name: "storage", code: "STORAGE_READY", status: "ready", required: true, summary: "data directory is available", action: "" },
      { name: "backup", code: "BACKUP_NO_DATABASE", status: "skipped", required: false, summary: "no database exists yet to back up", action: "" },
    ],
    ...overrides,
  };
}

describe("InstallationOnboarding", () => {
  beforeEach(async () => {
    await setInterfaceLocale("en");
  });

  afterEach(async () => {
    await setInterfaceLocale("en");
  });

  it("presents all seven canonical probes in essential and optional groups", () => {
    expect(installationSetupItems(readiness())).toEqual([
      { name: "narrative", state: "ready", required: true, code: "NARRATIVE_READY", summary: "narrative provider is ready", action: "" },
      { name: "embeddings", state: "skipped", required: false, code: "EMBEDDINGS_DISABLED", summary: "RAG embeddings are disabled", action: "" },
      { name: "image", state: "skipped", required: false, code: "IMAGE_DISABLED", summary: "image generation is disabled", action: "" },
      { name: "tts", state: "skipped", required: false, code: "TTS_DISABLED", summary: "text-to-speech is disabled", action: "" },
      { name: "gateway", state: "skipped", required: false, code: "GATEWAY_NOT_CONFIGURED", summary: "gateway readiness is disabled", action: "configure" },
      { name: "storage", state: "ready", required: true, code: "STORAGE_READY", summary: "data directory is available", action: "" },
      { name: "backup", state: "skipped", required: false, code: "BACKUP_NO_DATABASE", summary: "no database exists yet to back up", action: "" },
    ]);
    const html = renderToStaticMarkup(<InstallationOnboarding readiness={readiness()} onConfigure={() => undefined} onStartStory={() => undefined} onRetry={() => undefined} />);
    expect(html).toContain("Ready to create stories");
    expect(html).toContain("Optional enhancements");
    expect(html).toContain("Story narrator");
    expect(html).toContain("Browser gateway");
    expect(html).toContain("Database backup");
    expect(html).toContain("Recovery details");
    expect(html).toContain("Retry readiness checks");
  });

  it("preserves an existing configuration and keeps story onboarding separate", () => {
    const html = renderToStaticMarkup(<InstallationOnboarding readiness={readiness()} onConfigure={() => undefined} onStartStory={() => undefined} onRetry={() => undefined} />);

    expect(html).toContain("Existing shared configuration is preserved");
    expect(html).toContain("Story setup begins only after this installation is ready");
    expect(html).toContain("can be configured later");
    expect(html).toContain("Advanced: command-line setup");
    expect(html).toContain("oneday setup --reconfigure");
    expect(html).toContain("oneday doctor");
    expect(html).not.toContain('disabled=""');
  });

  it("blocks story setup only when a required probe fails", () => {
    const html = renderToStaticMarkup(<InstallationOnboarding readiness={readiness({ probes: [{ name: "narrative", code: "NARRATIVE_NOT_CONFIGURED", status: "failed", required: true, summary: "no narrative provider is enabled", action: "configure" }, { name: "image", code: "IMAGE_UNREACHABLE", status: "warning", required: false, summary: "image bridge is unavailable", action: "check_connection" }, { name: "tts", code: "TTS_DISABLED", status: "skipped", required: false, summary: "text-to-speech is disabled", action: "" }, { name: "storage", code: "STORAGE_READY", status: "ready", required: true, summary: "data directory is available", action: "" }] })} onConfigure={() => undefined} onStartStory={() => undefined} onRetry={() => undefined} />);

    expect(html).toContain("Resolve the required readiness checks before creating a story.");
    expect(html).toContain('disabled=""');
  });

  it("renders the canonical readiness summary in Italian", async () => {
    await setInterfaceLocale("it");
    const html = renderToStaticMarkup(<InstallationOnboarding readiness={readiness({ probes: [{ name: "narrative", code: "NARRATIVE_NOT_CONFIGURED", status: "failed", required: true, summary: "no narrative provider is enabled", action: "configure" }, { name: "image", code: "IMAGE_UNREACHABLE", status: "warning", required: false, summary: "image bridge is unavailable", action: "check_connection" }, { name: "tts", code: "TTS_DISABLED", status: "skipped", required: false, summary: "text-to-speech is disabled", action: "" }, { name: "storage", code: "STORAGE_READY", status: "ready", required: true, summary: "data directory is available", action: "" }] })} onConfigure={() => undefined} onStartStory={() => undefined} onRetry={() => undefined} />);

    expect(html).toContain("Pronto per creare storie");
    expect(html).toContain("Scegli e attiva un provider narrativo.");
    expect(html).toContain("Il servizio immagini non è raggiungibile.");
    expect(html).toContain("Il text-to-speech è disattivato.");
    expect(html).not.toContain("no narrative provider is enabled");
    expect(html).toContain("Da controllare");
    expect(html).toContain("Disattivato");
  });

  it("uses reassurance and a return action when setup is reopened", () => {
    const html = renderToStaticMarkup(<InstallationOnboarding readiness={readiness()} reopened onConfigure={() => undefined} onStartStory={() => undefined} onRetry={() => undefined} />);
    expect(html).toContain("Review this OneDay installation");
    expect(html).toContain("does not change existing worlds");
    expect(html).toContain("Return to stories");
  });
});
