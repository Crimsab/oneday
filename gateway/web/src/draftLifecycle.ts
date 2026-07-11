export function restoreFailedDraft(currentDraft: string, submittedDraft: string): string {
  return currentDraft === "" ? submittedDraft : currentDraft;
}
