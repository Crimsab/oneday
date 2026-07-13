import { describe, expect, it } from "vitest";
import {
  emptyProfile,
  visualProfileForStyle,
  visualStylePreset,
} from "./visualStylePresets";

describe("visual style presets", () => {
  it("adapts photorealism into world and character-specific direction", () => {
    const profile = visualProfileForStyle("photorealistic", emptyProfile());
    expect(profile.world_style_prompt).toContain("physically real");
    expect(profile.character_style_prompt).toContain("Photorealistic real-human");
    expect(profile.character_style_prompt).toContain("Preserve every canonical identity cue");
    expect(profile.negative_prompt).toContain("cosplay");
  });

  it("leaves auto empty so the server can derive direction from the finished world", () => {
    expect(visualProfileForStyle("auto", emptyProfile())).toEqual(emptyProfile());
  });

  it("trims and preserves a custom visual profile", () => {
    expect(
      visualProfileForStyle("custom", {
        world_style_prompt: "  charcoal diorama  ",
        character_style_prompt: " paper-cut portraits ",
        negative_prompt: " glossy plastic ",
        palette: " ash and copper ",
      }),
    ).toEqual({
      world_style_prompt: "charcoal diorama",
      character_style_prompt: "paper-cut portraits",
      negative_prompt: "glossy plastic",
      palette: "ash and copper",
    });
    expect(visualStylePreset("custom").label).toBe("Custom prompt");
  });
});
