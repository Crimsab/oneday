#!/usr/bin/env bun

import { copyFile, mkdir, rm } from "node:fs/promises";
import { dirname, join, resolve } from "node:path";

const root = resolve(import.meta.dir, "..");
const output = join(root, ".mkdocs-site-src");
const rootSources = [
  "CHANGELOG.md",
  "CODE_OF_CONDUCT.md",
  "CONTRIBUTING.md",
  "LICENSE",
  "SECURITY.md",
  "SUPPORT.md",
  "go.mod",
];

const listed = Bun.spawnSync([
  "git",
  "ls-files",
  "--cached",
  "--others",
  "--exclude-standard",
  "-z",
  "--",
  "docs",
], { cwd: root });

if (listed.exitCode !== 0) {
  process.stderr.write(listed.stderr.toString());
  process.exit(listed.exitCode);
}

const docs = listed.stdout
  .toString()
  .split("\0")
  .filter((path) => path.endsWith(".md"))
  .sort();
const sources = ["README.md", ...rootSources, ...docs];

await rm(output, { recursive: true, force: true });

for (const source of sources) {
  const destination = source === "README.md" ? "index.md" : source;
  const target = join(output, destination);
  await mkdir(dirname(target), { recursive: true });
  await copyFile(join(root, source), target);
}

console.log(`docs-stage: prepared ${sources.length} files in .mkdocs-site-src`);
