import { describe, expect, it } from "vitest";
import { languageCatalog, languageFlagUrl } from "./languageCatalog";

describe("translation language catalog", () => {
  it("provides local SVG flags for every supported language", () => {
    const languages = languageCatalog("en");

    expect(languages).toHaveLength(26);
    for (const language of languages) {
      expect(languageFlagUrl(language.code)).toMatch(/^(?:data:image\/svg\+xml,|.+\.svg$)/);
    }
  });

  it("does not invent a country flag for unsupported language codes", () => {
    expect(languageFlagUrl("eo")).toBeUndefined();
  });
});
