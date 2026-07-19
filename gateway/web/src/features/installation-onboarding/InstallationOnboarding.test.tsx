import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { InstallationOnboarding, installationSetupItems } from "./InstallationOnboarding";
import type { SetupReadinessReport } from "../../types";
import { setInterfaceLocale } from "../../i18n";

function readiness(overrides: Partial<SetupReadinessReport> = {}): SetupReadinessReport {
  return {
    probes: [
      { name: "narrative", code: "NARRATIVE_READY", status: "ready", required: true, summary: "narrative provider is ready" },
      { name: "embeddings", code: "EMBEDDINGS_DISABLED", status: "skipped", required: false, summary: "RAG embeddings are disabled" },
      { name: "image", code: "IMAGE_DISABLED", status: "skipped", required: false, summary: "image generation is disabled" },
      { name: "tts", code: "TTS_DISABLED", status: "skipped", required: false, summary: "text-to-speech is disabled" },
      { name: "gateway", code: "GATEWAY_NOT_CONFIGURED", status: "skipped", required: false, summary: "gateway readiness is disabled" },
      { name: "storage", code: "STORAGE_READY", status: "ready", required: true, summary: "data directory is available" },
      { name: "backup", code: "BACKUP_NO_DATABASE", status: "skipped", required: false, summary: "no database exists yet to back up" },
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

  it("presents all seven canonical probes with their server summaries and codes", () => {
    expect(installationSetupItems(readiness())).toEqual([
      { name: "narrative", state: "ready", required: true, code: "NARRATIVE_READY", summary: "narrative provider is ready" },
      { name: "embeddings", state: "skipped", required: false, code: "EMBEDDINGS_DISABLED", summary: "RAG embeddings are disabled" },
      { name: "image", state: "skipped", required: false, code: "IMAGE_DISABLED", summary: "image generation is disabled" },
      { name: "tts", state: "skipped", required: false, code: "TTS_DISABLED", summary: "text-to-speech is disabled" },
      { name: "gateway", state: "skipped", required: false, code: "GATEWAY_NOT_CONFIGURED", summary: "gateway readiness is disabled" },
      { name: "storage", state: "ready", required: true, code: "STORAGE_READY", summary: "data directory is available" },
      { name: "backup", state: "skipped", required: false, code: "BACKUP_NO_DATABASE", summary: "no database exists yet to back up" },
    ]);
    const html = renderToStaticMarkup(<InstallationOnboarding readiness={readiness()} onConfigure={() => undefined} onStartStory={() => undefined} />);
    for (const code of ["NARRATIVE_READY", "EMBEDDINGS_DISABLED", "IMAGE_DISABLED", "TTS_DISABLED", "GATEWAY_NOT_CONFIGURED", "STORAGE_READY", "BACKUP_NO_DATABASE"]) {
      expect(html).toContain(code);
    }
  });

  it("preserves an existing configuration and keeps story onboarding separate", () => {
    const html = renderToStaticMarkup(<InstallationOnboarding readiness={readiness()} onConfigure={() => undefined} onStartStory={() => undefined} />);

    expect(html).toContain("Existing shared configuration is preserved");
    expect(html).toContain("Story setup begins only after this installation is ready");
    expect(html).toContain("Images are optional");
    expect(html).toContain("Spoken audio is optional");
    expect(html).toContain("EMBEDDINGS_DISABLED");
    expect(html).toContain("STORAGE_READY");
    expect(html).not.toContain('disabled=""');
  });

  it("blocks story setup only when a required probe fails", () => {
    const html = renderToStaticMarkup(<InstallationOnboarding readiness={readiness({ probes: [{ name: "narrative", code: "NARRATIVE_NOT_CONFIGURED", status: "failed", required: true, summary: "no narrative provider is enabled" }, { name: "image", code: "IMAGE_UNAVAILABLE", status: "warning", required: false, summary: "image bridge is unavailable" }, { name: "tts", code: "TTS_DISABLED", status: "skipped", required: false, summary: "text-to-speech is disabled" }, { name: "storage", code: "STORAGE_READY", status: "ready", required: true, summary: "data directory is available" }] })} onConfigure={() => undefined} onStartStory={() => undefined} />);

    expect(html).toContain("Resolve the required readiness checks before creating a story.");
    expect(html).toContain('disabled=""');
  });

  it("renders the canonical readiness summary in Italian", async () => {
    await setInterfaceLocale("it");
    const html = renderToStaticMarkup(<InstallationOnboarding readiness={readiness({ probes: [{ name: "narrative", code: "NARRATIVE_NOT_CONFIGURED", status: "failed", required: true, summary: "no narrative provider is enabled" }, { name: "image", code: "IMAGE_UNAVAILABLE", status: "warning", required: false, summary: "image bridge is unavailable" }, { name: "tts", code: "TTS_DISABLED", status: "skipped", required: false, summary: "text-to-speech is disabled" }, { name: "storage", code: "STORAGE_READY", status: "ready", required: true, summary: "data directory is available" }] })} onConfigure={() => undefined} onStartStory={() => undefined} />);

    expect(html).toContain("Stato dell’installazione");
    expect(html).toContain("Scegli e attiva un provider narrativo.");
    expect(html).toContain("Il servizio immagini non ha superato la verifica di disponibilità.");
    expect(html).toContain("Il text-to-speech è disattivato.");
    expect(html).not.toContain("no narrative provider is enabled");
    expect(html).toContain("NARRATIVE_NOT_CONFIGURED");
    expect(html).toContain("Facoltativo · attenzione");
    expect(html).toContain("Facoltativo · disattivato");
  });
});
