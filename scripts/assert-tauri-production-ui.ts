#!/usr/bin/env bun

import { existsSync, readFileSync } from "node:fs";
import { basename, resolve } from "node:path";

const [binaryArgument, distArgument] = Bun.argv.slice(2);
if (!binaryArgument || !distArgument) {
  console.error(
    "usage: bun scripts/assert-tauri-production-ui.ts <desktop-binary> <frontend-dist>",
  );
  process.exit(2);
}

const binaryPath = resolve(binaryArgument);
const distPath = resolve(distArgument);
const indexPath = resolve(distPath, "index.html");

for (const path of [binaryPath, indexPath]) {
  if (!existsSync(path)) {
    console.error(`tauri-ui-check: required file is missing: ${path}`);
    process.exit(1);
  }
}

const index = readFileSync(indexPath, "utf8");
const assets = [
  ...new Set(
    [...index.matchAll(/(?:src|href)=["']\.?\/?assets\/([^"'?#]+)["']/g)].map(
      (match) => basename(match[1]),
    ),
  ),
];

if (assets.length === 0) {
  console.error(`tauri-ui-check: ${indexPath} does not reference any built assets`);
  process.exit(1);
}

const binary = readFileSync(binaryPath);
const missing = assets.filter((asset) => !binary.includes(Buffer.from(asset)));
if (missing.length > 0) {
  console.error(
    `tauri-ui-check: ${binaryPath} does not embed the production frontend assets: ${missing.join(
      ", ",
    )}`,
  );
  console.error(
    "tauri-ui-check: build desktop releases with `tauri build`; a direct `cargo build` can retain devUrl and open a missing Vite server.",
  );
  process.exit(1);
}

console.log(
  `tauri-ui-check: ${assets.length} production frontend asset(s) are embedded in ${binaryPath}`,
);
