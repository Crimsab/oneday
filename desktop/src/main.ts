import "./styles.css";
import { desktopBridge, friendlyError, type StorySummary } from "./bridge";

const brandIcon = new URL("../../docs/assets/oneday-icon.png", import.meta.url).href;

const app = document.querySelector<HTMLElement>("#app");
if (!app) throw new Error("Missing desktop application root");

app.innerHTML = `
  <header class="brand">
    <img src="${brandIcon}" alt="" width="56" height="56" />
    <div><p>OneDay</p><h1>Desktop connection</h1></div>
  </header>
  <section class="panel" aria-labelledby="server-title">
    <div class="section-heading"><div><h2 id="server-title">Story server</h2><p>Your server remains the only source of truth. This app stores no story database.</p></div><span id="connection-status" class="status">Not connected</span></div>
    <form id="server-form">
      <label for="server-url">Server URL</label>
      <div class="field-row"><input id="server-url" name="serverUrl" type="url" autocomplete="url" spellcheck="false" placeholder="https://oneday.example.com" required /><button type="submit">Connect</button></div>
      <p class="hint">HTTPS is required. Plain HTTP is accepted only for localhost development.</p>
    </form>
    <div id="connection-message" class="message" role="status" aria-live="polite"></div>
    <button id="open-story" class="secondary" type="button" disabled>Open OneDay</button>
  </section>
  <section class="panel" aria-labelledby="native-title">
    <div class="section-heading"><div><h2 id="native-title">Desktop behavior</h2><p>These permissions are opt-in and only available to this local settings window.</p></div></div>
    <label class="toggle"><input id="autostart" type="checkbox" /><span><strong>Start with the computer</strong><small>Launch quietly in the system tray.</small></span></label>
    <label class="toggle"><input id="notifications" type="checkbox" /><span><strong>Native notifications</strong><small>Ask the operating system before enabling them.</small></span></label>
    <button id="test-notification" class="tertiary" type="button" disabled>Send test notification</button>
  </section>
  <section class="panel" aria-labelledby="files-title">
    <div class="section-heading"><div><h2 id="files-title">Story files</h2><p>Native dialogs only read or write the file you select.</p></div></div>
    <button id="import-package" class="secondary" type="button" disabled>Import OneDay package…</button>
    <div class="export-row"><select id="story-select" aria-label="Story to export" disabled><option value="">Choose a story</option></select><button id="export-archive" class="secondary" type="button" disabled>Export complete ZIP…</button><button id="export-world" class="tertiary" type="button" disabled>Export world…</button></div>
    <div id="transfer-message" class="message" role="status" aria-live="polite"></div>
  </section>
  <footer><span id="updater-status">Signed updates are not configured.</span><div><button id="check-updates" class="text-button" type="button" hidden>Check for updates</button><button id="refresh-stories" class="text-button" type="button" disabled>Refresh stories</button></div></footer>
`;

const serverForm = element<HTMLFormElement>("server-form");
const serverUrl = element<HTMLInputElement>("server-url");
const connectionStatus = element<HTMLElement>("connection-status");
const connectionMessage = element<HTMLElement>("connection-message");
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

function element<T extends HTMLElement>(id: string): T {
  const value = document.getElementById(id);
  if (!value) throw new Error(`Missing element: ${id}`);
  return value as T;
}

function setConnected(value: boolean) {
  connected = value;
  connectionStatus.textContent = value ? "Connected" : "Not connected";
  connectionStatus.classList.toggle("connected", value);
  for (const control of [openStory, importPackage, storySelect, refreshStories]) control.disabled = !value;
  updateExportButtons();
}

function setBusy(busy: boolean, message = "") {
  serverUrl.disabled = busy;
  serverForm.querySelector<HTMLButtonElement>("button")!.disabled = busy;
  connectionMessage.textContent = message;
}

function updateExportButtons() {
  const disabled = !connected || !storySelect.value;
  exportArchive.disabled = disabled;
  exportWorld.disabled = disabled;
}

async function loadStories() {
  transferMessage.textContent = "Loading stories…";
  try {
    const stories = await desktopBridge.stories();
    renderStories(stories);
    transferMessage.textContent = stories.length ? "" : "No stories are available on this server.";
  } catch (error) {
    transferMessage.textContent = friendlyError(error);
  }
}

function renderStories(stories: StorySummary[]) {
  storySelect.replaceChildren(new Option("Choose a story", ""));
  for (const story of stories) storySelect.add(new Option(story.name, story.id));
  updateExportButtons();
}

async function connect() {
  setBusy(true, "Checking the server…");
  try {
    await desktopBridge.connect(serverUrl.value);
    setConnected(true);
    connectionMessage.textContent = "Connected securely. The OneDay window is ready.";
    await loadStories();
  } catch (error) {
    setConnected(false);
    connectionMessage.textContent = friendlyError(error);
  } finally {
    setBusy(false);
  }
}

serverForm.addEventListener("submit", (event) => { event.preventDefault(); void connect(); });
openStory.addEventListener("click", () => void desktopBridge.showStoryWindow().catch((error) => { connectionMessage.textContent = friendlyError(error); }));
refreshStories.addEventListener("click", () => void loadStories());
checkUpdates.addEventListener("click", async () => {
  checkUpdates.disabled = true;
  updaterStatus.textContent = "Checking the signed release feed…";
  try {
    const result = await desktopBridge.checkAndInstallUpdate();
    updaterStatus.textContent = result.message;
  } catch (error) {
    updaterStatus.textContent = friendlyError(error);
  } finally {
    checkUpdates.disabled = false;
  }
});
storySelect.addEventListener("change", updateExportButtons);

autostart.addEventListener("change", async () => {
  const requested = autostart.checked;
  try { await desktopBridge.setAutostart(requested); }
  catch (error) { autostart.checked = !requested; connectionMessage.textContent = friendlyError(error); }
});

notifications.addEventListener("change", async () => {
  if (!notifications.checked) return;
  try {
    notifications.checked = await desktopBridge.requestNotifications();
    testNotification.disabled = !notifications.checked;
  } catch (error) {
    notifications.checked = false;
    connectionMessage.textContent = friendlyError(error);
  }
});
testNotification.addEventListener("click", () => void desktopBridge.testNotification());

importPackage.addEventListener("click", async () => {
  transferMessage.textContent = "Waiting for a file…";
  try {
    const result = await desktopBridge.importPackage();
    transferMessage.textContent = result.message;
    if (!result.cancelled) await loadStories();
  } catch (error) { transferMessage.textContent = friendlyError(error); }
});

async function exportStory(kind: "archive" | "world") {
  transferMessage.textContent = "Preparing export…";
  try {
    const result = await desktopBridge.exportPackage(storySelect.value, kind);
    transferMessage.textContent = result.message;
  } catch (error) { transferMessage.textContent = friendlyError(error); }
}
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
    if (state.serverUrl) serverUrl.value = state.serverUrl;
    if (state.serverUrl && !state.startedMinimized) await connect();
  } catch (error) { connectionMessage.textContent = friendlyError(error); }
}

void bootstrap();
