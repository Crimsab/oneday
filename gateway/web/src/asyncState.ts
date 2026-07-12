export function isCurrentAsyncSelection(
  requestedStoryId: string,
  currentStoryId: string,
  requestVersion: number,
  latestVersion: number,
): boolean {
  return requestedStoryId === currentStoryId && requestVersion === latestVersion;
}
