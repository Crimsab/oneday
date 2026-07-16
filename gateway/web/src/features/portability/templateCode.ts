import { gunzipSync, gzipSync, strFromU8, strToU8 } from "fflate";

const PREFIX = "OD1";
const MAX_TEMPLATE_BYTES = 4 * 1024 * 1024;

function base64Url(bytes: Uint8Array): string {
  let binary = "";
  for (const byte of bytes) binary += String.fromCharCode(byte);
  return btoa(binary).replaceAll("+", "-").replaceAll("/", "_").replace(/=+$/, "");
}

function fromBase64Url(value: string): Uint8Array {
  const normalized = value.replaceAll("-", "+").replaceAll("_", "/");
  const binary = atob(normalized + "=".repeat((4 - normalized.length % 4) % 4));
  return Uint8Array.from(binary, (character) => character.charCodeAt(0));
}

async function digest(bytes: Uint8Array): Promise<string> {
  const hash = new Uint8Array(await crypto.subtle.digest("SHA-256", Uint8Array.from(bytes).buffer));
  return base64Url(hash.slice(0, 12));
}

export async function encodeTemplateCode(template: string): Promise<string> {
  const raw = strToU8(template);
  if (raw.length > MAX_TEMPLATE_BYTES) throw new Error("Template is too large for a share code.");
  JSON.parse(template);
  const compressed = gzipSync(raw, { level: 9, mtime: 0 });
  return `${PREFIX}:${base64Url(compressed)}.${await digest(compressed)}`;
}

export async function decodeTemplateCode(code: string): Promise<string> {
  const match = code.trim().match(/^OD1:([A-Za-z0-9_-]+)\.([A-Za-z0-9_-]+)$/);
  if (!match) throw new Error("Invalid OneDay share code.");
  const compressed = fromBase64Url(match[1]);
  if (await digest(compressed) !== match[2]) throw new Error("Share code checksum mismatch.");
  const raw = gunzipSync(compressed);
  if (raw.length > MAX_TEMPLATE_BYTES) throw new Error("Decoded template exceeds the size limit.");
  const text = strFromU8(raw);
  JSON.parse(text);
  return text;
}
