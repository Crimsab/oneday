export interface ServiceWorkerUpdateRegistration {
  installing: ServiceWorker | null;
  update(): Promise<unknown>;
}

export async function checkForServiceWorkerUpdate(
  swUrl: string,
  registration: ServiceWorkerUpdateRegistration,
  online = navigator.onLine,
  request: typeof fetch = fetch,
): Promise<boolean> {
  if (!online || registration.installing) return false;
  try {
    const response = await request(swUrl, {
      cache: "no-store",
      headers: {
        Cache: "no-store",
        "Cache-Control": "no-cache",
      },
    });
    if (!response.ok) return false;
    await registration.update();
    return true;
  } catch {
    return false;
  }
}
