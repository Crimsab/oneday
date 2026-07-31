import "./styles.css";
import { desktopBridge, friendlyError, type CodexStatus, type DesktopState, type StorySummary } from "./bridge";
import { isServerReady, lifecycleLabel, profileLabel } from "./profile";

const brandIcon = new URL("../../docs/assets/oneday-icon.png", import.meta.url).href;
const app = document.querySelector<HTMLElement>("#app");
if (!app) throw new Error("Missing desktop application root");

app.innerHTML = `
  <header class="brand"><img src="${brandIcon}" alt="" width="56" height="56" /><div><p>OneDay</p><h1>Desktop profile</h1></div></header>
  <section class="panel" aria-labelledby="profile-title">
    <div class="section-heading"><div><h2 id="profile-title">Where should OneDay run?</h2><p id="profile-description">Choose once to get started. You can safely switch profiles later.</p></div><span id="connection-status" class="status">Stopped</span></div>
    <div class="profile-actions"><button id="choose-remote" type="button">Connect to a server</button><button id="choose-standalone" class="secondary" type="button">Run on this device</button></div>
    <form id="server-form" hidden><label for="server-url">Server URL</label><div class="field-row"><input id="server-url" name="serverUrl" type="url" autocomplete="url" spellcheck="false" placeholder="https://oneday.example.com" required /><button type="submit">Connect</button></div><p class="hint">Remote mode stores no story database here. HTTPS is required except for localhost development.</p></form>
    <div id="standalone-controls" class="local-actions" hidden><p class="hint">Standalone data stays in this app’s isolated profile. The local gateway uses a fresh loopback endpoint each launch.</p><button id="restart-standalone" class="secondary" type="button">Restart local gateway</button><button id="stop-standalone" class="tertiary" type="button">Stop local gateway</button></div>
    <div id="connection-message" class="message" role="status" aria-live="polite"></div><button id="open-story" class="secondary" type="button" disabled>Open OneDay</button>
  </section>
  <section id="provider-panel" class="panel" aria-labelledby="provider-title" hidden>
    <div class="section-heading"><div><h2 id="provider-title">AI provider</h2><p>Codex is optional. OneDay checks an existing installation first and downloads nothing until you choose Install.</p></div><span id="codex-status" class="status">Checking</span></div>
    <div class="component-row">
      <div class="component-copy"><strong>Codex subscription</strong><span id="codex-version">Use an existing Codex login or install OneDay’s verified private component.</span></div>
      <div class="component-actions"><button id="install-codex" class="secondary" type="button" hidden>Install Codex</button><button id="login-codex" type="button" hidden>Sign in</button><button id="open-provider-setup" class="secondary" type="button">Open Setup</button><button id="refresh-codex" class="tertiary" type="button">Refresh</button></div>
    </div>
    <div id="codex-message" class="message" role="status" aria-live="polite"></div>
    <p class="hint">After sign-in, open OneDay Setup and select Codex plus the model you want. Other providers remain available there and never require this component.</p>
  </section>
  <section class="panel" aria-labelledby="native-title"><div class="section-heading"><div><h2 id="native-title">Desktop behavior</h2><p>These permissions are opt-in and only available to this local settings window.</p></div></div><label class="toggle"><input id="autostart" type="checkbox" /><span><strong>Start with the computer</strong><small>Launch quietly in the system tray.</small></span></label><label class="toggle"><input id="notifications" type="checkbox" /><span><strong>Native notifications</strong><small>Ask the operating system before enabling them.</small></span></label><button id="test-notification" class="tertiary" type="button" disabled>Send test notification</button></section>
  <section class="panel" aria-labelledby="files-title"><div class="section-heading"><div><h2 id="files-title">Story files</h2><p>Native dialogs only read or write the file you select.</p></div></div><button id="import-package" class="secondary" type="button" disabled>Import OneDay package…</button><div class="export-row"><select id="story-select" aria-label="Story to export" disabled><option value="">Choose a story</option></select><button id="export-archive" class="secondary" type="button" disabled>Export complete ZIP…</button><button id="export-world" class="tertiary" type="button" disabled>Export world…</button></div><div id="transfer-message" class="message" role="status" aria-live="polite"></div></section>
  <footer><span id="updater-status">Signed updates are not configured.</span><div><button id="check-updates" class="text-button" type="button" hidden>Check for updates</button><button id="refresh-stories" class="text-button" type="button" disabled>Refresh stories</button></div></footer>`;

function element<T extends HTMLElement>(id: string): T {
  const value = document.getElementById(id);
  if (!value) throw new Error(`Missing element: ${id}`);
  return value as T;
}

const serverForm = element<HTMLFormElement>("server-form");
const serverUrl = element<HTMLInputElement>("server-url");
const connectionStatus = element<HTMLElement>("connection-status");
const connectionMessage = element<HTMLElement>("connection-message");
const profileDescription = element<HTMLElement>("profile-description");
const chooseRemote = element<HTMLButtonElement>("choose-remote");
const chooseStandalone = element<HTMLButtonElement>("choose-standalone");
const standaloneControls = element<HTMLElement>("standalone-controls");
const providerPanel = element<HTMLElement>("provider-panel");
const codexStatus = element<HTMLElement>("codex-status");
const codexVersion = element<HTMLElement>("codex-version");
const codexMessage = element<HTMLElement>("codex-message");
const installCodex = element<HTMLButtonElement>("install-codex");
const loginCodex = element<HTMLButtonElement>("login-codex");
const refreshCodex = element<HTMLButtonElement>("refresh-codex");
const openProviderSetup = element<HTMLButtonElement>("open-provider-setup");
const restartStandalone = element<HTMLButtonElement>("restart-standalone");
const stopStandalone = element<HTMLButtonElement>("stop-standalone");
const openStory = element<HTMLButtonElement>("open-story");
const autostart = element<HTMLInputElement>("autostart");
const notifications = element<HTMLInputElement>("notifications");
const testNotification = element<HTMLButtonElement>("test-notification");
const importPackage = element<HTMLButtonElement>("import-package");
const storySelect = element<HTMLSelectElement>("story-select");
const exportArchive = element<HTMLButtonElement>("export-archive");
const exportWorld = element<HTMLButtonElement>("export-world");
const transferMessage = element<HTMLElement>("transfer-message");
const updaterStatus = element<HTMLElement>("updater-status");
const checkUpdates = element<HTMLButtonElement>("check-updates");
const refreshStories = element<HTMLButtonElement>("refresh-stories");
let connected = false;

function setConnected(value: boolean) {
  connected = value;
  openStory.disabled = !value;
  for (const control of [importPackage, storySelect, refreshStories]) control.disabled = !value;
  updateExportButtons();
}

function renderState(state: DesktopState) {
  const standalone = state.profile?.mode === "standalone";
  const ready = isServerReady(state.lifecycle);
  connectionStatus.textContent = lifecycleLabel(state.lifecycle);
  connectionStatus.classList.toggle("connected", ready);
  connectionStatus.classList.toggle("failed", state.lifecycle.state === "failed");
  profileDescription.textContent = `${profileLabel(state.profile)}. ${standalone ? "Standalone data remains isolated from other profiles." : "Remote mode creates no local story database."}`;
  serverForm.hidden = standalone || !state.profile;
  standaloneControls.hidden = !standalone;
  providerPanel.hidden = !standalone;
  openProviderSetup.disabled = !standalone || !ready;
  chooseRemote.classList.toggle("selected", state.profile?.mode === "remote");
  chooseStandalone.classList.toggle("selected", standalone);
  if (state.profile?.mode === "remote") serverUrl.value = state.profile.serverUrl;
  if (state.lifecycle.state === "failed") connectionMessage.textContent = state.lifecycle.message;
  setConnected(ready);
}

function renderCodex(status: CodexStatus) {
  codexStatus.textContent = status.authenticated ? "Ready" : status.available ? "Sign-in needed" : "Not installed";
  codexStatus.classList.toggle("connected", status.authenticated);
  codexStatus.classList.remove("failed");
  codexVersion.textContent = status.version
    ? `${status.version} · ${status.source === "managed" ? "managed by OneDay" : "found on this device"}`
    : `Managed component available: ${status.managedVersion}`;
  codexMessage.textContent = status.message;
  installCodex.hidden = status.available;
  loginCodex.hidden = !status.available || status.authenticated;
}

function setCodexBusy(busy: boolean, message = "") {
  providerPanel.setAttribute("aria-busy", String(busy));
  [installCodex, loginCodex, refreshCodex].forEach((button) => { button.disabled = busy; });
  if (message) codexMessage.textContent = message;
}

async function refreshCodexStatus() {
  setCodexBusy(true, "Checking Codex on this device…");
  try {
    renderCodex(await desktopBridge.codexStatus());
  } catch (error) {
    codexStatus.textContent = "Check failed";
    codexStatus.classList.add("failed");
    codexMessage.textContent = friendlyError(error);
  } finally { setCodexBusy(false); }
}

function setBusy(busy: boolean, message = "") {
  serverUrl.disabled = busy;
  serverForm.querySelector<HTMLButtonElement>("button")!.disabled = busy;
  [chooseRemote, chooseStandalone, restartStandalone, stopStandalone].forEach((button) => { button.disabled = busy; });
  if (message) connectionMessage.textContent = message;
}

function updateExportButtons() {
  const disabled = !connected || !storySelect.value;
  exportArchive.disabled = disabled;
  exportWorld.disabled = disabled;
}

function renderStories(stories: StorySummary[]) {
  storySelect.replaceChildren(new Option("Choose a story", ""));
  for (const story of stories) storySelect.add(new Option(story.name, story.id));
  updateExportButtons();
}

async function loadStories() {
  transferMessage.textContent = "Loading stories…";
  try {
    const stories = await desktopBridge.stories();
    renderStories(stories);
    transferMessage.textContent = stories.length ? "" : "No stories are available on this server.";
  } catch (error) { transferMessage.textContent = friendlyError(error); }
}

async function refreshState(load = true) {
  const state = await desktopBridge.state();
  renderState(state);
  if (load && isServerReady(state.lifecycle)) await loadStories();
}

async function connect() {
  setBusy(true, "Checking the server…");
  try {
    await desktopBridge.connect(serverUrl.value);
    connectionMessage.textContent = "Connected securely. The OneDay window is ready.";
    await refreshState();
  } catch (error) {
    connectionMessage.textContent = friendlyError(error);
    await refreshState(false).catch(() => undefined);
  } finally { setBusy(false); }
}

async function startStandalone(restart = false) {
  setBusy(true, restart ? "Restarting the local gateway…" : "Starting the local gateway…");
  try {
    if (restart) await desktopBridge.restartStandalone(); else await desktopBridge.startStandalone();
    connectionMessage.textContent = "The local OneDay gateway is ready.";
    await refreshState();
  } catch (error) {
    connectionMessage.textContent = friendlyError(error);
    await refreshState(false).catch(() => undefined);
  } finally { setBusy(false); }
}

chooseRemote.addEventListener("click", () => { serverForm.hidden = false; serverUrl.focus(); });
chooseStandalone.addEventListener("click", () => void (async () => { await startStandalone(); await refreshCodexStatus(); })());
serverForm.addEventListener("submit", (event) => { event.preventDefault(); void connect(); });
restartStandalone.addEventListener("click", () => void startStandalone(true));
stopStandalone.addEventListener("click", () => void (async () => { setBusy(true, "Stopping the local gateway…"); try { await desktopBridge.stopStandalone(); await refreshState(false); } catch (error) { connectionMessage.textContent = friendlyError(error); } finally { setBusy(false); } })());
openStory.addEventListener("click", () => void desktopBridge.showStoryWindow().catch((error) => { connectionMessage.textContent = friendlyError(error); }));
refreshStories.addEventListener("click", () => void loadStories());
checkUpdates.addEventListener("click", () => void (async () => { checkUpdates.disabled = true; updaterStatus.textContent = "Checking the signed release feed…"; try { updaterStatus.textContent = (await desktopBridge.checkAndInstallUpdate()).message; } catch (error) { updaterStatus.textContent = friendlyError(error); } finally { checkUpdates.disabled = false; } })());
refreshCodex.addEventListener("click", () => void refreshCodexStatus());
openProviderSetup.addEventListener("click", () => void desktopBridge.showProviderSetup().catch((error) => { codexMessage.textContent = friendlyError(error); }));
installCodex.addEventListener("click", () => void (async () => {
  setCodexBusy(true, "Downloading the pinned Codex release and verifying its SHA-256 digest…");
  try { renderCodex(await desktopBridge.installCodex()); }
  catch (error) { codexStatus.textContent = "Install failed"; codexStatus.classList.add("failed"); codexMessage.textContent = friendlyError(error); }
  finally { setCodexBusy(false); }
})());
loginCodex.addEventListener("click", () => void (async () => {
  setCodexBusy(true, "Complete sign-in in the browser window opened by Codex…");
  try {
    renderCodex(await desktopBridge.loginCodex());
    await startStandalone(true);
    codexMessage.textContent = "Codex is ready. Open OneDay Setup and choose Codex and a model.";
  } catch (error) { codexStatus.textContent = "Sign-in needed"; codexStatus.classList.add("failed"); codexMessage.textContent = friendlyError(error); }
  finally { setCodexBusy(false); }
})());
storySelect.addEventListener("change", updateExportButtons);
autostart.addEventListener("change", () => void (async () => { const requested = autostart.checked; try { await desktopBridge.setAutostart(requested); } catch (error) { autostart.checked = !requested; connectionMessage.textContent = friendlyError(error); } })());
notifications.addEventListener("change", () => void (async () => { if (!notifications.checked) return; try { notifications.checked = await desktopBridge.requestNotifications(); testNotification.disabled = !notifications.checked; } catch (error) { notifications.checked = false; connectionMessage.textContent = friendlyError(error); } })());
testNotification.addEventListener("click", () => void desktopBridge.testNotification());
importPackage.addEventListener("click", () => void (async () => { transferMessage.textContent = "Waiting for a file…"; try { const result = await desktopBridge.importPackage(); transferMessage.textContent = result.message; if (!result.cancelled) await loadStories(); } catch (error) { transferMessage.textContent = friendlyError(error); } })());
async function exportStory(kind: "archive" | "world") { transferMessage.textContent = "Preparing export…"; try { transferMessage.textContent = (await desktopBridge.exportPackage(storySelect.value, kind)).message; } catch (error) { transferMessage.textContent = friendlyError(error); } }
exportArchive.addEventListener("click", () => void exportStory("archive"));
exportWorld.addEventListener("click", () => void exportStory("world"));

async function bootstrap() {
  try {
    const state = await desktopBridge.state();
    updaterStatus.textContent = state.updater.reason;
    checkUpdates.hidden = !state.updater.enabled;
    autostart.checked = await desktopBridge.autostartEnabled();
    notifications.checked = await desktopBridge.notificationsEnabled();
    testNotification.disabled = !notifications.checked;
    renderState(state);
    if (state.profile?.mode === "standalone") await refreshCodexStatus();
    if (state.profile?.mode === "remote" && !state.startedMinimized) await connect();
    if (state.profile?.mode === "standalone" && !state.startedMinimized) await startStandalone();
  } catch (error) { connectionMessage.textContent = friendlyError(error); }
}
void bootstrap();
