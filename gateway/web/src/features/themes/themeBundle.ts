import { strFromU8, strToU8, unzipSync, zipSync, type Unzipped } from "fflate";
import type { StoredFontRecord } from "../../fontLibrary";
import type { AppPreferences } from "../../types";
import { exportTheme, previewThemeImport, type PortableThemeV1, type ThemePreview } from "./themePortability";

const MAX_ARCHIVE_BYTES = 25 * 1024 * 1024;
const MAX_EXPANDED_BYTES = 40 * 1024 * 1024;
const MAX_FONT_BYTES = 20 * 1024 * 1024;
const MAX_ENTRIES = 10;

interface ThemeBundleManifest {
  kind: "oneday-theme-bundle";
  version: 1;
  fonts: Array<{ id: string; path: string; label: string; fileName: string; mimeType: string; sha256: string }>;
}

export interface ThemeBundlePreview extends ThemePreview { stagedFonts: StoredFontRecord[] }

export async function exportThemeBundle(preferences: AppPreferences, fonts: StoredFontRecord[], includeFonts: boolean): Promise<Blob> {
  const theme = exportTheme(preferences);
  const selected = includeFonts ? fonts.filter((font) => theme.fontRefs.includes(font.id)) : [];
  const entries: Record<string, Uint8Array> = {};
  const manifest: ThemeBundleManifest = { kind: "oneday-theme-bundle", version: 1, fonts: [] };
  for (const font of selected.slice(0, 8)) {
    const bytes = new Uint8Array(await font.data.arrayBuffer());
    if (bytes.byteLength > MAX_FONT_BYTES || !isWoff2(bytes)) continue;
    const sha256 = await sha256Hex(bytes);
    const path = `fonts/${sha256}.woff2`;
    entries[path] = bytes;
    manifest.fonts.push({ id: font.id, path, label: font.label, fileName: font.fileName, mimeType: "font/woff2", sha256 });
  }
  entries["theme.json"] = strToU8(JSON.stringify(theme, null, 2));
  entries["manifest.json"] = strToU8(JSON.stringify(manifest, null, 2));
  return new Blob([zipSync(entries, { level: 6, mtime: new Date("1980-01-01T00:00:00Z") })], { type: "application/zip" });
}

export async function previewThemeBundle(file: File, current: AppPreferences, availableFontIds: ReadonlySet<string>): Promise<ThemeBundlePreview> {
  if (file.size > MAX_ARCHIVE_BYTES) throw new Error("theme_bundle_too_large");
  const compressed = new Uint8Array(await file.arrayBuffer());
  let entryCount = 0; let expandedBytes = 0;
  const files = unzipSync(compressed, { filter: (entry) => {
    entryCount += 1; expandedBytes += entry.originalSize;
    if (entryCount > MAX_ENTRIES || expandedBytes > MAX_EXPANDED_BYTES || !safeEntry(entry.name)) throw new Error("invalid_theme_bundle");
    if (entry.name.startsWith("fonts/") && entry.originalSize > MAX_FONT_BYTES) throw new Error("theme_font_too_large");
    return true;
  } });
  const manifest = decodeManifest(readJson(files, "manifest.json"));
  const theme = structuredClone(readJson(files, "theme.json")) as PortableThemeV1;
  const stagedFonts: StoredFontRecord[] = [];
  const idMap = new Map<string, { id: string; family: string }>();
  for (const item of manifest.fonts) {
    const bytes = files[item.path];
    if (!bytes || !isWoff2(bytes) || await sha256Hex(bytes) !== item.sha256) throw new Error("invalid_theme_font");
    const id = `imported:${crypto.randomUUID()}`;
    const family = `OneDay Imported ${id.slice(-12)}`;
    idMap.set(item.id, { id, family });
    stagedFonts.push({ id, family, label: item.label, source: "imported", detail: item.fileName, fileName: item.fileName, mimeType: "font/woff2", createdAt: new Date().toISOString(), data: new Blob([bytes], { type: "font/woff2" }) });
  }
  rewriteFont(theme, "interfaceFont", idMap);
  rewriteFont(theme, "readingFont", idMap);
  theme.fontRefs = theme.fontRefs.map((id) => idMap.get(id)?.id ?? id);
  const known = new Set([...availableFontIds, ...stagedFonts.map((font) => font.id)]);
  return { ...previewThemeImport(theme, current, known), stagedFonts };
}

function rewriteFont(theme: PortableThemeV1, key: "interfaceFont" | "readingFont", idMap: Map<string, { id: string; family: string }>) {
  const font = theme.tokens.typography[key]; const replacement = idMap.get(font.id);
  if (replacement) theme.tokens.typography[key] = { ...font, ...replacement, source: "imported" };
}
function readJson(files: Unzipped, name: string): unknown {
  const bytes = files[name]; if (!bytes || bytes.byteLength > 128 * 1024) throw new Error("invalid_theme_bundle");
  return JSON.parse(strFromU8(bytes));
}
function decodeManifest(value: unknown): ThemeBundleManifest {
  if (!value || typeof value !== "object") throw new Error("invalid_theme_bundle");
  const candidate = value as ThemeBundleManifest;
  if (candidate.kind !== "oneday-theme-bundle" || candidate.version !== 1 || !Array.isArray(candidate.fonts) || candidate.fonts.length > 8) throw new Error("invalid_theme_bundle");
  for (const font of candidate.fonts) {
    if (!font || typeof font.id !== "string" || typeof font.label !== "string" || typeof font.fileName !== "string" || typeof font.sha256 !== "string" || font.path !== `fonts/${font.sha256}.woff2` || !/^[a-f0-9]{64}$/.test(font.sha256)) throw new Error("invalid_theme_bundle");
  }
  return candidate;
}
function safeEntry(name: string): boolean { return name === "manifest.json" || name === "theme.json" || /^fonts\/[a-f0-9]{64}\.woff2$/.test(name); }
function isWoff2(bytes: Uint8Array): boolean { return bytes.length >= 4 && bytes[0] === 0x77 && bytes[1] === 0x4f && bytes[2] === 0x46 && bytes[3] === 0x32; }
async function sha256Hex(bytes: Uint8Array): Promise<string> { const owned = Uint8Array.from(bytes); return [...new Uint8Array(await crypto.subtle.digest("SHA-256", owned.buffer))].map((value) => value.toString(16).padStart(2, "0")).join(""); }
