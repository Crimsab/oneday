import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { InstallationOnboarding, installationSetupItems } from "./InstallationOnboarding";
import type { SetupReadinessReport } from "../../types";
import { setInterfaceLocale } from "../../i18n";

function readiness(overrides: Partial<SetupReadinessReport> = {}): SetupReadinessReport {
  return {
    probes: [
      { name: "narrative", code: "NARRATIVE_READY", status: "ready", required: true, summary: "narrative provider is ready" },
      { name: "image", code: "IMAGE_DISABLED", status: "skipped", required: false, summary: "image generation is disabled" },
      { name: "tts", code: "TTS_DISABLED", status: "skipped", required: false, summary: "text-to-speech is disabled" },
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

  it("distinguishes required narration from optional image and voice setup", () => {
    expect(installationSetupItems(readiness())).toEqual([
      { id: "narrative", state: "ready", required: true, summary: "narrative provider is ready" },
      { id: "images", state: "skipped", required: false, summary: "image generation is disabled" },
      { id: "voice", state: "skipped", required: false, summary: "text-to-speech is disabled" },
    ]);
  });

  it("preserves an existing configuration and keeps story onboarding separate", () => {
    const html = renderToStaticMarkup(<InstallationOnboarding readiness={readiness()} onConfigure={() => undefined} onStartStory={() => undefined} />);

    expect(html).toContain("Existing shared configuration is preserved");
    expect(html).toContain("Story setup begins only after this installation is ready");
    expect(html).toContain("Images are optional");
    expect(html).toContain("Spoken audio is optional");
  });

  it("blocks story setup until a narrator is configured", () => {
    const html = renderToStaticMarkup(<InstallationOnboarding readiness={readiness({ probes: [{ name: "narrative", code: "NARRATIVE_NOT_CONFIGURED", status: "failed", required: true, summary: "no narrative provider is enabled" }] })} onConfigure={() => undefined} onStartStory={() => undefined} />);

    expect(html).toContain("Configure a narrative provider before creating a story.");
    expect(html).toContain('disabled=""');
  });

  it("renders the canonical readiness summary in Italian", async () => {
    await setInterfaceLocale("it");
    const html = renderToStaticMarkup(<InstallationOnboarding readiness={readiness()} onConfigure={() => undefined} onStartStory={() => undefined} />);

    expect(html).toContain("Stato dell’installazione");
    expect(html).toContain("Obbligatorio · pronto");
    expect(html).toContain("Facoltativo · disattivato");
  });
});
