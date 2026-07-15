import type { FontSourcePreference } from "./types";

const DATABASE_NAME = "oneday-font-library";
const DATABASE_VERSION = 1;
const STORE_NAME = "fonts";

export interface FontChoice {
  id: string;
  family: string;
  label: string;
  source: FontSourcePreference;
  detail?: string;
}

export interface ImportedFontRecord extends FontChoice {
  source: "imported";
  fileName: string;
  mimeType: string;
  createdAt: string;
  data: Blob;
}

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

export function cssFontFamily(family: string): string {
  if (family === "system-ui" || family === "ui-serif" || family === "ui-monospace") return family;
  return `${JSON.stringify(family)}, system-ui, sans-serif`;
}

export async function importFontFile(file: File): Promise<ImportedFontRecord> {
  if (!isSupportedFontFile(file)) throw new Error("Formato font non supportato");
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
  await registerImportedFont(record);
  await withStore("readwrite", (store) => store.put(record));
  return record;
}

export async function listImportedFonts(): Promise<ImportedFontRecord[]> {
  const records = await withStore<ImportedFontRecord[]>("readonly", (store) => store.getAll());
  return records.sort((left, right) => right.createdAt.localeCompare(left.createdAt));
}

export async function loadImportedFonts(): Promise<ImportedFontRecord[]> {
  const records = await listImportedFonts();
  await Promise.all(records.map(registerImportedFont));
  return records;
}

export async function deleteImportedFont(id: string): Promise<void> {
  const face = registeredFaces.get(id);
  if (face && typeof document !== "undefined") document.fonts.delete(face);
  registeredFaces.delete(id);
  await withStore("readwrite", (store) => store.delete(id));
}

export async function registerImportedFont(record: ImportedFontRecord): Promise<void> {
  if (registeredFaces.has(record.id) || typeof document === "undefined") return;
  const buffer = await record.data.arrayBuffer();
  const face = new FontFace(record.family, buffer, { display: "swap" });
  await face.load();
  document.fonts.add(face);
  registeredFaces.set(record.id, face);
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
    };
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error ?? new Error("Impossibile aprire la libreria font"));
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
