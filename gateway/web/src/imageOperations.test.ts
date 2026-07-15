import { describe, expect, it } from "vitest";
import {
  availableVisualOperations,
  effectiveRouteCapabilities,
  operationAcceptsNegativePrompt,
} from "./imageOperations";
import type { ImageOperationCapability } from "./types";

const capability = (
  operation: ImageOperationCapability["operation"],
  availability: ImageOperationCapability["availability"] = "available",
): ImageOperationCapability => ({ operation, availability, supported: true });

describe("visual image operation capabilities", () => {
  it("shows only explicit, available source-edit operations", () => {
    expect(availableVisualOperations(null, [
      capability("generate"),
      capability("edit"),
      capability("inpaint", "requires_configuration"),
      { ...capability("image_transform"), supported: false },
    ]).map((item) => item.operation)).toEqual(["edit"]);
  });

  it("prefers asset/model descriptors over route defaults", () => {
    const asset = { operation_capabilities: [capability("inpaint")] };
    expect(availableVisualOperations(asset, [capability("edit")]).map((item) => item.operation)).toEqual(["inpaint"]);
  });

  it("does not infer negative prompt support from field presence", () => {
    expect(operationAcceptsNegativePrompt(capability("edit"))).toBe(false);
    expect(operationAcceptsNegativePrompt({ ...capability("inpaint"), controls: { negative_prompt: true } })).toBe(true);
  });

  it("gates static descriptors by the configured route", () => {
    const ready = capability("edit");
    const configuredLater = capability("inpaint", "requires_configuration");
    const unsupported = {
      ...capability("image_transform", "unavailable"),
      supported: false,
    };

    expect(effectiveRouteCapabilities([ready, configuredLater, unsupported], false)
      .map((item) => item.availability))
      .toEqual(["requires_configuration", "requires_configuration", "unavailable"]);
    expect(effectiveRouteCapabilities([ready, configuredLater, unsupported], true)
      .map((item) => item.availability))
      .toEqual(["available", "available", "unavailable"]);
  });
});
