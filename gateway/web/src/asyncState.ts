export function isCurrentAsyncSelection(
  requestedStoryId: string,
  currentStoryId: string,
  requestVersion: number,
  latestVersion: number,
): boolean {
  return requestedStoryId === currentStoryId && requestVersion === latestVersion;
}

export function coalesceRequest<Key, Value>(
  inFlight: Map<Key, Promise<Value>>,
  key: Key,
  load: () => Promise<Value>,
): Promise<Value> {
  const existing = inFlight.get(key);
  if (existing) return existing;
  const request = load();
  inFlight.set(key, request);
  const clear = () => {
    if (inFlight.get(key) === request) inFlight.delete(key);
  };
  request.then(clear, clear);
  return request;
}
