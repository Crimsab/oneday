import type { FontSourcePreference } from "./types";

const DATABASE_NAME = "oneday-font-library";
const DATABASE_VERSION = 2;
const STORE_NAME = "fonts";
const JOURNAL_STORE_NAME = "theme_import_journal";
export const MAX_STORED_FONT_BYTES = 20 * 1024 * 1024;

export interface FontChoice {
  id: string;
  family: string;
  label: string;
  source: FontSourcePreference;
  detail?: string;
}

export interface StoredFontRecord extends FontChoice {
  source: "imported" | "online";
  fileName: string;
  mimeType: string;
  createdAt: string;
  updatedAt?: string;
  sourceUrl?: string;
  data: Blob;
}

export interface ImportedFontRecord extends StoredFontRecord {
  source: "imported";
}

export interface OnlineFontRecord extends StoredFontRecord {
  source: "online";
  sourceUrl: string;
}

export type FontFormat = "woff" | "woff2" | "ttf" | "otf";

interface LocalFontData {
  family: string;
  fullName: string;
  postscriptName: string;
  style: string;
}

type LocalFontWindow = Window & {
  queryLocalFonts?: () => Promise<LocalFontData[]>;
};

const registeredFaces = new Map<string, FontFace>();

export const bundledFontChoices: FontChoice[] = [
  { id: "bundled:ibm-plex-sans", family: "IBM Plex Sans Variable", label: "IBM Plex Sans", source: "bundled", detail: "OneDay" },
  { id: "system:ui-sans", family: "system-ui", label: "Sistema", source: "system", detail: "Interfaccia del dispositivo" },
  { id: "system:serif", family: "ui-serif", label: "Serif di sistema", source: "system" },
  { id: "system:monospace", family: "ui-monospace", label: "Monospace di sistema", source: "system" },
];

export function supportsLocalFontAccess(): boolean {
  return typeof window !== "undefined" && typeof (window as LocalFontWindow).queryLocalFonts === "function";
}

export async function querySystemFonts(): Promise<FontChoice[]> {
  const query = typeof window === "undefined" ? undefined : (window as LocalFontWindow).queryLocalFonts;
  if (!query) return [];
  const fonts = await query.call(window);
  return dedupeSystemFonts(fonts);
}

export function dedupeSystemFonts(fonts: LocalFontData[]): FontChoice[] {
  const families = new Map<string, FontChoice>();
  for (const font of fonts) {
    const family = font.family?.trim();
    if (!family) continue;
    const key = family.toLocaleLowerCase();
    if (!families.has(key)) {
      families.set(key, {
        id: `system:${font.postscriptName || family}`,
        family,
        label: family,
        source: "system",
        detail: font.style || font.fullName,
      });
    }
  }
  return [...families.values()].sort((left, right) => left.label.localeCompare(right.label));
}

export function isSupportedFontFile(file: Pick<File, "name" | "type">): boolean {
  return /\.(woff2?|ttf|otf)$/i.test(file.name) || /^(font\/(woff2?|ttf|otf|sfnt)|application\/(font-woff|x-font-ttf|x-font-opentype))$/i.test(file.type);
}

export function fontNameFromFile(fileName: string): string {
  const stem = fileName.replace(/\.(woff2?|ttf|otf)$/i, "").replace(/[_-]+/g, " ").replace(/\s+/g, " ").trim();
  return stem || "Font importato";
}

export function fontNameFromUrl(value: string): string {
  try {
    const url = new URL(value);
    return fontNameFromFile(decodeURIComponent(url.pathname.split("/").pop() || ""));
  } catch {
    return "Font online";
  }
}

export function normalizeOnlineFontUrl(value: string): string {
  const normalized = value.trim();
  if (!normalized || normalized.length > 2048) throw new Error("Inserisci un URL valido per il font");
  let url: URL;
  try {
    url = new URL(normalized);
  } catch {
    throw new Error("Inserisci un URL completo che inizi con http:// o https://");
  }
  if ((url.protocol !== "https:" && url.protocol !== "http:") || !url.hostname || url.username || url.password) {
    throw new Error("Il link del font deve usare HTTP o HTTPS e non può contenere credenziali");
  }
  return url.toString();
}

export function fontFormatFromBytes(bytes: Uint8Array): FontFormat | null {
  if (bytes.length < 4) return null;
  const signature = String.fromCharCode(bytes[0], bytes[1], bytes[2], bytes[3]);
  if (signature === "wOFF") return "woff";
  if (signature === "wOF2") return "woff2";
  if (signature === "OTTO") return "otf";
  if (signature === "true" || signature === "typ1" || (bytes[0] === 0 && bytes[1] === 1 && bytes[2] === 0 && bytes[3] === 0)) return "ttf";
  return null;
}

export function cssFontFamily(family: string): string {
  if (family === "system-ui" || family === "ui-serif" || family === "ui-monospace") return family;
  return `${JSON.stringify(family)}, system-ui, sans-serif`;
}

export async function importFontFile(file: File): Promise<ImportedFontRecord> {
  if (!isSupportedFontFile(file)) throw new Error("Formato font non supportato");
  if (file.size > MAX_STORED_FONT_BYTES) throw new Error("Il font supera il limite di 20 MB");
  const id = `imported:${typeof crypto !== "undefined" && crypto.randomUUID ? crypto.randomUUID() : `${Date.now()}-${Math.random().toString(16).slice(2)}`}`;
  const label = fontNameFromFile(file.name);
  const record: ImportedFontRecord = {
    id,
    family: `OneDay Imported ${id.slice(-12)}`,
    label,
    source: "imported",
    detail: file.name,
    fileName: file.name,
    mimeType: file.type || "application/octet-stream",
    createdAt: new Date().toISOString(),
    data: file.slice(),
  };
  await registerStoredFont(record);
  await withStore("readwrite", (store) => store.put(record));
  return record;
}

export async function saveOnlineFont({ id, label, sourceUrl }: { id?: string; label?: string; sourceUrl: string }): Promise<OnlineFontRecord> {
  const normalizedUrl = normalizeOnlineFontUrl(sourceUrl);
  const previous = id ? await getStoredFont(id) : undefined;
  const data = await downloadFontBlob(normalizedUrl);
  const fontId = id ?? `online:${typeof crypto !== "undefined" && crypto.randomUUID ? crypto.randomUUID() : `${Date.now()}-${Math.random().toString(16).slice(2)}`}`;
  const now = new Date().toISOString();
  const resolvedLabel = label?.trim() || fontNameFromUrl(normalizedUrl);
  const record: OnlineFontRecord = {
    id: fontId,
    family: previous?.family ?? `OneDay Online ${fontId.slice(-12)}`,
    label: resolvedLabel.slice(0, 120),
    source: "online",
    detail: new URL(normalizedUrl).hostname,
    fileName: fileNameFromOnlineFont(normalizedUrl, data.type),
    mimeType: data.type,
    createdAt: previous?.createdAt ?? now,
    updatedAt: now,
    sourceUrl: normalizedUrl,
    data,
  };
  unregisterStoredFont(fontId);
  await registerStoredFont(record);
  await withStore("readwrite", (store) => store.put(record));
  return record;
}

export async function listStoredFonts(): Promise<StoredFontRecord[]> {
  const records = await withStore<StoredFontRecord[]>("readonly", (store) => store.getAll());
  return records.sort((left, right) => right.createdAt.localeCompare(left.createdAt));
}

export async function loadStoredFonts(): Promise<StoredFontRecord[]> {
  await recoverInterruptedThemeFontImport();
  const records = await listStoredFonts();
  const registrations = await Promise.allSettled(records.map(registerStoredFont));
  return records.filter((_, index) => registrations[index].status === "fulfilled");
}

export async function stageThemeFonts(records: StoredFontRecord[]): Promise<void> {
  await withNamedStore("readwrite", JOURNAL_STORE_NAME, (store) => store.put({ id: "active", fontIds: records.map((record) => record.id), createdAt: new Date().toISOString() }));
  for (const record of records) {
    await registerStoredFont(record);
    await withStore("readwrite", (store) => store.put(record));
  }
}

export async function commitThemeFontImport(): Promise<void> {
  await withNamedStore("readwrite", JOURNAL_STORE_NAME, (store) => store.delete("active"));
}

export async function rollbackThemeFontImport(records: StoredFontRecord[]): Promise<void> {
  for (const record of records) await deleteStoredFont(record.id);
  await commitThemeFontImport();
}

async function recoverInterruptedThemeFontImport(): Promise<void> {
  const journal = await withNamedStore<{ id: string; fontIds: string[] } | undefined>("readonly", JOURNAL_STORE_NAME, (store) => store.get("active"));
  if (!journal) return;
  for (const id of journal.fontIds) {
    unregisterStoredFont(id);
    await withStore("readwrite", (store) => store.delete(id));
  }
  await commitThemeFontImport();
}

export async function deleteStoredFont(id: string): Promise<void> {
  unregisterStoredFont(id);
  await withStore("readwrite", (store) => store.delete(id));
}

export function unregisterStoredFont(id: string): void {
  const face = registeredFaces.get(id);
  if (face && typeof document !== "undefined") document.fonts.delete(face);
  registeredFaces.delete(id);
}

export async function registerStoredFont(record: StoredFontRecord): Promise<void> {
  if (registeredFaces.has(record.id) || typeof document === "undefined") return;
  const buffer = await record.data.arrayBuffer();
  const face = new FontFace(record.family, buffer, { display: "swap" });
  await face.load();
  document.fonts.add(face);
  registeredFaces.set(record.id, face);
}

async function getStoredFont(id: string): Promise<StoredFontRecord | undefined> {
  return withStore<StoredFontRecord | undefined>("readonly", (store) => store.get(id));
}

async function downloadFontBlob(url: string): Promise<Blob> {
  let response: Response;
  try {
    response = await fetch(url, { cache: "no-store", credentials: "omit", mode: "cors", redirect: "follow" });
  } catch {
    throw new Error("Download del font non riuscito. Controlla il link e che il server consenta richieste CORS");
  }
  if (!response.ok) throw new Error(`Download del font non riuscito (HTTP ${response.status})`);
  const declaredSize = Number(response.headers.get("content-length") || 0);
  if (declaredSize > MAX_STORED_FONT_BYTES) throw new Error("Il font online supera il limite di 20 MB");

  const parts: ArrayBuffer[] = [];
  let size = 0;
  const reader = response.body?.getReader();
  if (reader) {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      size += value.byteLength;
      if (size > MAX_STORED_FONT_BYTES) {
        await reader.cancel();
        throw new Error("Il font online supera il limite di 20 MB");
      }
      parts.push(value.buffer.slice(value.byteOffset, value.byteOffset + value.byteLength) as ArrayBuffer);
    }
  } else {
    const buffer = await response.arrayBuffer();
    if (buffer.byteLength > MAX_STORED_FONT_BYTES) throw new Error("Il font online supera il limite di 20 MB");
    parts.push(buffer);
  }

  const bytes = new Uint8Array(await new Blob(parts).arrayBuffer());
  const format = fontFormatFromBytes(bytes);
  if (!format) throw new Error("Il link non contiene un font WOFF, WOFF2, TTF o OTF valido");
  return new Blob(parts, { type: fontMimeType(format) });
}

function fontMimeType(format: FontFormat): string {
  return format === "ttf" ? "font/ttf" : format === "otf" ? "font/otf" : `font/${format}`;
}

function fileNameFromOnlineFont(url: string, mimeType: string): string {
  let name = "";
  try {
    name = decodeURIComponent(new URL(url).pathname.split("/").pop() || "");
  } catch {
    name = "";
  }
  if (/\.(woff2?|ttf|otf)$/i.test(name)) return name;
  const extension = mimeType.replace("font/", "") || "woff2";
  return `${fontNameFromUrl(url).replace(/\s+/g, "-")}.${extension}`;
}

function openDatabase(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    if (typeof indexedDB === "undefined") {
      reject(new Error("La libreria font non è disponibile in questo browser"));
      return;
    }
    const request = indexedDB.open(DATABASE_NAME, DATABASE_VERSION);
    request.onupgradeneeded = () => {
      if (!request.result.objectStoreNames.contains(STORE_NAME)) {
        request.result.createObjectStore(STORE_NAME, { keyPath: "id" });
      }
      if (!request.result.objectStoreNames.contains(JOURNAL_STORE_NAME)) {
        request.result.createObjectStore(JOURNAL_STORE_NAME, { keyPath: "id" });
      }
    };
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error ?? new Error("Impossibile aprire la libreria font"));
  });
}

async function withNamedStore<T = void>(mode: IDBTransactionMode, storeName: string, action: (store: IDBObjectStore) => IDBRequest): Promise<T> {
  const database = await openDatabase();
  return new Promise((resolve, reject) => {
    const transaction = database.transaction(storeName, mode);
    const request = action(transaction.objectStore(storeName));
    request.onsuccess = () => resolve(request.result as T);
    request.onerror = () => reject(request.error ?? new Error("Operazione font non riuscita"));
    transaction.oncomplete = () => database.close();
    transaction.onabort = () => { database.close(); reject(transaction.error ?? new Error("Operazione font interrotta")); };
  });
}

async function withStore<T = void>(mode: IDBTransactionMode, action: (store: IDBObjectStore) => IDBRequest): Promise<T> {
  const database = await openDatabase();
  return new Promise((resolve, reject) => {
    const transaction = database.transaction(STORE_NAME, mode);
    const request = action(transaction.objectStore(STORE_NAME));
    request.onsuccess = () => resolve(request.result as T);
    request.onerror = () => reject(request.error ?? new Error("Operazione font non riuscita"));
    transaction.oncomplete = () => database.close();
    transaction.onabort = () => {
      database.close();
      reject(transaction.error ?? new Error("Operazione font interrotta"));
    };
  });
}
