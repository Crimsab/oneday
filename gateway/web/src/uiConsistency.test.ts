import { describe, expect, it } from "vitest";
import { readFileSync, readdirSync } from "node:fs";
import { join } from "node:path";

function sourceFiles(root: string): string[] {
  return readdirSync(root, { withFileTypes: true }).flatMap((entry) => {
    const path = join(root, entry.name);
    if (entry.isDirectory()) return sourceFiles(path);
    if (!entry.name.endsWith(".tsx") || /\.(test|spec)\.tsx$/.test(entry.name)) return [];
    return [path];
  });
}

describe("UI consistency guardrails", () => {
  it("uses the shared accessible select instead of browser-native selects", () => {
    const offenders = sourceFiles(join(import.meta.dirname)).filter((path) =>
      /<select(?:\s|>)/.test(readFileSync(path, "utf8")),
    );
    expect(offenders).toEqual([]);
  });

  it("never renders the parameterized semantic-search status without variables", () => {
    const panelDrawer = readFileSync(join(import.meta.dirname, "components", "PanelDrawer.tsx"), "utf8");
    expect(panelDrawer).not.toMatch(/t\(["']models\.embedding["']\)/);
  });
});
