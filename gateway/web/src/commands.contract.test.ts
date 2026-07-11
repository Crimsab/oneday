import { execFileSync } from "node:child_process";
import { describe, expect, it } from "vitest";
import { fallbackCommandDescriptors } from "./commands";
import type { CommandDescriptor } from "./types";

interface DescriptorResponse {
  commands?: CommandDescriptor[];
  error?: string;
}

describe("Go command contract parity", () => {
  it("keeps the browser fallback descriptors aligned with the terminal contract", () => {
    const output = execFileSync("go", ["run", "./cmd/oneday", "gateway-command-descriptors"], {
      cwd: new URL("../../../", import.meta.url),
      encoding: "utf8",
      stdio: ["ignore", "pipe", "pipe"],
    });
    const response = JSON.parse(output) as DescriptorResponse;
    expect(response.error).toBeUndefined();
    expect(response.commands?.length).toBeGreaterThan(0);

    const browser = descriptorIndex(fallbackCommandDescriptors);
    for (const descriptor of response.commands ?? []) {
      const fallback = browser.get(descriptor.id);
      expect(fallback, `missing browser fallback descriptor ${descriptor.id}`).toBeDefined();
      expect(fallback).toMatchObject({
        canonical: descriptor.canonical,
        behavior: descriptor.behavior,
        parity: descriptor.parity,
        group: descriptor.group,
      });
      expect(fallback?.aliases ?? []).toEqual(descriptor.aliases ?? []);
      expect(fallback?.enabled_when).toBe(descriptor.enabled_when);
    }
  }, 30000);
});

function descriptorIndex(descriptors: CommandDescriptor[]): Map<string, CommandDescriptor> {
  return new Map(descriptors.map((descriptor) => [descriptor.id, descriptor]));
}
