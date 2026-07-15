#!/usr/bin/env bun

import { existsSync, readFileSync, statSync } from "node:fs";
import { dirname, extname, join, normalize, resolve } from "node:path";

const root = resolve(import.meta.dir, "..");
const listed = Bun.spawnSync([
  "git",
  "ls-files",
  "--cached",
  "--others",
  "--exclude-standard",
  "-z",
  "--",
  "*.md",
], { cwd: root });

if (listed.exitCode !== 0) {
  process.stderr.write(listed.stderr.toString());
  process.exit(listed.exitCode);
}

const files = listed.stdout
  .toString()
  .split("\0")
  .filter(Boolean)
  .sort();
const errors: string[] = [];
let checkedLinks = 0;

function slugHeading(value: string): string {
  return value
    .trim()
    .toLowerCase()
    .replace(/<[^>]*>/g, "")
    .replace(/[\u2000-\u206f\u2e00-\u2e7f'!"#$%&()*+,./:;<=>?@[\\\]^`{|}~]/g, "")
    .replace(/\s+/g, "-");
}

function anchorsFor(markdown: string): Set<string> {
  const anchors = new Set<string>();
  const seen = new Map<string, number>();
  for (const line of markdown.split(/\r?\n/)) {
    const match = line.match(/^#{1,6}\s+(.+?)\s*#*\s*$/);
    if (!match) continue;
    const base = slugHeading(match[1]);
    const count = seen.get(base) ?? 0;
    seen.set(base, count + 1);
    anchors.add(count === 0 ? base : `${base}-${count}`);
  }
  return anchors;
}

function linkTargets(markdown: string): string[] {
  const targets: string[] = [];
  for (const match of markdown.matchAll(/!?\[[^\]]*\]\(([^)]+)\)/g)) {
    targets.push(match[1].trim());
  }
  for (const match of markdown.matchAll(/^\s*\[[^\]]+\]:\s+(\S+)/gm)) {
    targets.push(match[1].trim());
  }
  return targets;
}

function cleanTarget(raw: string): string {
  if (raw.startsWith("<") && raw.endsWith(">")) return raw.slice(1, -1);
  return raw.replace(/\s+(?:"[^"]*"|'[^']*'|\([^)]*\))$/, "");
}

for (const file of files) {
  const absolute = join(root, file);
  const markdown = readFileSync(absolute, "utf8");
  for (const raw of linkTargets(markdown)) {
    const target = cleanTarget(raw);
    if (/^(?:https?:|mailto:|data:|javascript:)/i.test(target)) continue;

    checkedLinks++;
    const [rawPath, rawFragment = ""] = target.split("#", 2);
    const decodedPath = decodeURIComponent(rawPath.split("?", 1)[0]);
    let targetPath = decodedPath ? resolve(dirname(absolute), decodedPath) : absolute;

    if (existsSync(targetPath) && statSync(targetPath).isDirectory()) {
      targetPath = join(targetPath, "README.md");
    }
    if (!existsSync(targetPath)) {
      errors.push(`${file}: missing local link target ${target}`);
      continue;
    }

    if (rawFragment && extname(targetPath).toLowerCase() === ".md") {
      const fragment = slugHeading(decodeURIComponent(rawFragment));
      const anchors = anchorsFor(readFileSync(targetPath, "utf8"));
      if (!anchors.has(fragment)) {
        errors.push(`${file}: missing heading #${rawFragment} in ${normalize(targetPath).replace(root + "/", "")}`);
      }
    }
  }
}

const requiredGuides = [
  "docs/first-story.md",
  "docs/features.md",
  "docs/story-systems.md",
  "docs/media.md",
  "docs/extensions.md",
];
for (const guide of requiredGuides) {
  for (const index of ["README.md", "docs/README.md"]) {
    const relative = index === "README.md" ? guide : guide.replace(/^docs\//, "");
    if (!readFileSync(join(root, index), "utf8").includes(`](${relative})`)) {
      errors.push(`${index}: public guide is not indexed: ${relative}`);
    }
  }
}

if (errors.length > 0) {
  for (const error of errors) console.error(`docs-check: ${error}`);
  console.error(`docs-check: failed with ${errors.length} error(s)`);
  process.exit(1);
}

console.log(`docs-check: ${files.length} Markdown files and ${checkedLinks} local links passed`);
