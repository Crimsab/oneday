import type { ManifestOptions } from "vite-plugin-pwa";

export const NETWORK_ONLY_PATH_PATTERNS = [
  /^\/api(?:\/|$)/,
  /^\/generated(?:\/|$)/,
] as const;

export const NETWORK_ONLY_URL_PATTERNS = [
  /^https?:\/\/[^/]+\/api(?:\/|$)/,
  /^https?:\/\/[^/]+\/generated(?:\/|$)/,
] as const;

export const pwaManifest = {
  id: "/",
  name: "OneDay",
  short_name: "OneDay",
  description: "Persistent interactive stories that remember, branch, and evolve.",
  lang: "en",
  start_url: "/",
  scope: "/",
  display: "standalone",
  orientation: "any",
  background_color: "#070807",
  theme_color: "#071c3a",
  categories: ["entertainment", "games", "books"],
  icons: [
    {
      src: "/brand/oneday-icon-192.png",
      sizes: "192x192",
      type: "image/png",
      purpose: "any",
    },
    {
      src: "/brand/oneday-mark.png",
      sizes: "512x512",
      type: "image/png",
      purpose: "any",
    },
  ],
} satisfies Partial<ManifestOptions>;

export const pwaWorkbox = {
  cleanupOutdatedCaches: true,
  clientsClaim: true,
  skipWaiting: false,
  globPatterns: ["**/*.{html,js,css,woff2,png,svg,ico,webmanifest}"],
  navigateFallback: "index.html",
  navigateFallbackDenylist: [...NETWORK_ONLY_PATH_PATTERNS],
  runtimeCaching: NETWORK_ONLY_URL_PATTERNS.map((urlPattern) => ({
    urlPattern,
    handler: "NetworkOnly" as const,
    method: "GET" as const,
  })),
};

export function isServerCanonicalPath(pathname: string): boolean {
  return NETWORK_ONLY_PATH_PATTERNS.some((pattern) => pattern.test(pathname));
}
