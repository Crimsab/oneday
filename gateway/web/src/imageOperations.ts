import type { ImageOperationCapability, VisualAsset } from "./types";

export type EditableImageOperation = "edit" | "inpaint" | "image_transform";

const EDITABLE_OPERATIONS = new Set<EditableImageOperation>(["edit", "inpaint", "image_transform"]);

export function effectiveRouteCapabilities(
  descriptors: ImageOperationCapability[],
  routeConfigured: boolean,
): ImageOperationCapability[] {
  return descriptors.map((capability) => {
    if (!capability.supported) return capability;
    if (!routeConfigured && capability.availability === "available") {
      return { ...capability, availability: "requires_configuration" };
    }
    if (routeConfigured && capability.availability === "requires_configuration") {
      return { ...capability, availability: "available" };
    }
    return capability;
  });
}

export function availableVisualOperations(
  asset: Pick<VisualAsset, "operation_capabilities"> | null,
  routeCapabilities: ImageOperationCapability[] = [],
): Array<ImageOperationCapability & { operation: EditableImageOperation }> {
  const descriptors = asset?.operation_capabilities ?? routeCapabilities;
  return descriptors.filter(
    (capability): capability is ImageOperationCapability & { operation: EditableImageOperation } =>
      EDITABLE_OPERATIONS.has(capability.operation as EditableImageOperation)
      && capability.supported
      && capability.availability === "available",
  );
}

export function operationAcceptsNegativePrompt(capability: ImageOperationCapability): boolean {
  return capability.controls?.negative_prompt === true;
}
