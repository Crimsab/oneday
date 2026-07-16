import { describe, expect, it } from "vitest";
import { decodeTemplateCode, encodeTemplateCode } from "./templateCode";

describe("OneDay world template share codes", () => {
  it("round-trips a compressed, checksummed template", async () => {
    const value = JSON.stringify({ kind: "oneday-world-template", version: 1, world: "Dock 7" });
    await expect(decodeTemplateCode(await encodeTemplateCode(value))).resolves.toBe(value);
  });

  it("rejects a modified checksum", async () => {
    const code = await encodeTemplateCode(JSON.stringify({ version: 1 }));
    await expect(decodeTemplateCode(`${code.slice(0, -1)}x`)).rejects.toThrow(/checksum/i);
  });
});
