import "./styles.css";
import {
  desktopBridge,
  friendlyError,
  type ClaudeStatus,
  type CodexStatus,
  type DesktopState,
  type StorySummary,
  type UpdateCheck,
  type UpdaterStatus,
} from "./bridge";
import { isServerReady, lifecycleLabel, profileLabel } from "./profile";

const brandIcon = new URL("../src-tauri/icons/128x128.png", import.meta.url).href;
const app = document.querySelector<HTMLElement>("#app");
if (!app) throw new Error("Missing desktop application root");

app.innerHTML = `
  <header class="brand">
    <img src="${brandIcon}" alt="" width="52" height="52" />
    <div><p class="eyebrow">OneDay Desktop</p><h1>Set up this device</h1></div>
  </header>

  <main>
    <section class="panel primary-panel" aria-labelledby="profile-title">
      <div class="section-heading">
        <div><p class="step">1 · Connection</p><h2 id="profile-title">Where should OneDay run?</h2><p id="profile-description">Run it privately on this device or connect to an existing server.</p></div>
        <span id="connection-status" class="status" role="status">Stopped</span>
      </div>
      <div class="profile-grid" aria-describedby="profile-description">
        <button id="choose-standalone" class="profile-choice" type="button" aria-pressed="false">
          <span class="choice-top"><strong>Run on this device</strong><span class="recommendation">Recommended</span></span>
          <small>One click. Stories stay in an isolated local profile.</small>
        </button>
        <button id="choose-remote" class="profile-choice" type="button" aria-pressed="false">
          <span class="choice-top"><strong>Connect to a server</strong></span>
          <small>Use a OneDay gateway you already operate.</small>
        </button>
      </div>
      <form id="server-form" class="inset" hidden>
        <label for="server-url">OneDay server URL</label>
        <div class="field-row"><input id="server-url" name="serverUrl" type="url" autocomplete="url" spellcheck="false" placeholder="https://oneday.example.com" required /><button type="submit">Connect</button></div>
        <p class="hint">HTTPS is required, except for localhost development. No story database is created on this device.</p>
      </form>
      <div id="standalone-controls" class="local-actions" hidden>
        <button id="restart-standalone" class="secondary" type="button">Restart local service</button>
        <button id="stop-standalone" class="tertiary" type="button">Stop local service</button>
      </div>
      <div id="connection-message" class="message" role="status" aria-live="polite"></div>
      <button id="open-story" class="wide-action" type="button" disabled>Open OneDay</button>
    </section>

    <section id="provider-panel" class="panel" aria-labelledby="provider-title" hidden>
      <div class="section-heading">
        <div><p class="step">2 · AI connections</p><h2 id="provider-title">Choose any supported provider</h2><p>Subscriptions and API providers can be mixed. OneDay never copies subscription credentials into its configuration.</p></div>
        <button id="open-provider-setup" class="secondary heading-action" type="button">Configure models</button>
      </div>
      <div class="provider-grid">
        <article class="provider-card" aria-labelledby="codex-title">
          <div class="provider-heading"><div><span class="provider-kind">Subscription</span><h3 id="codex-title">Codex</h3></div><span id="codex-status" class="status">Checking</span></div>
          <p id="codex-version">Use an existing Codex login or install OneDay’s verified component.</p>
          <div class="provider-actions"><button id="install-codex" class="secondary" type="button" hidden>Install Codex</button><button id="login-codex" type="button" hidden>Sign in</button><button id="refresh-codex" class="tertiary" type="button">Refresh</button></div>
          <div id="codex-message" class="message compact" role="status" aria-live="polite"></div>
        </article>
        <article class="provider-card" aria-labelledby="claude-title">
          <div class="provider-heading"><div><span class="provider-kind">Subscription</span><h3 id="claude-title">Claude Code</h3></div><span id="claude-status" class="status">Checking</span></div>
          <p id="claude-version">Use a Claude subscription already signed in on this device.</p>
          <div class="provider-actions"><button id="install-claude" class="secondary" type="button" hidden>Install Claude</button><button id="claude-guide" class="secondary" type="button" hidden>Installation guide</button><button id="login-claude" type="button" hidden>Sign in</button><button id="refresh-claude" class="tertiary" type="button">Refresh</button></div>
          <div id="claude-message" class="message compact" role="status" aria-live="polite"></div>
        </article>
        <article class="provider-card api-provider" aria-labelledby="openrouter-title">
          <div class="provider-heading"><div><span class="provider-kind">API</span><h3 id="openrouter-title">OpenRouter</h3></div><span class="status neutral">In Setup</span></div>
          <p>Enter an endpoint and API key in OneDay. The key is write-only after saving.</p>
        </article>
        <article class="provider-card api-provider" aria-labelledby="litellm-title">
          <div class="provider-heading"><div><span class="provider-kind">API or local</span><h3 id="litellm-title">LiteLLM compatible</h3></div><span class="status neutral">In Setup</span></div>
          <p>Connect LiteLLM or another OpenAI-compatible endpoint, including a local network service.</p>
        </article>
      </div>
      <p class="provider-footnote">Provider credentials remain where they belong: subscription sessions stay in Codex or Claude Code; API keys stay in the protected gateway configuration.</p>
    </section>

    <section class="panel update-panel" aria-labelledby="update-title">
      <div class="section-heading">
        <div><p class="step">3 · Updates</p><h2 id="update-title">OneDay updates</h2><p id="update-trust">Checking build configuration…</p></div>
        <div class="version-stack"><span id="current-version">Version —</span><span id="update-channel" class="status neutral">—</span></div>
      </div>
      <div id="update-result" class="update-result" hidden>
        <strong id="update-version"></strong><p id="update-notes"></p>
      </div>
      <div id="update-message" class="message" role="status" aria-live="polite"></div>
      <div class="update-actions"><button id="check-updates" class="secondary" type="button">Check now</button><button id="install-update" type="button" hidden>Install and restart</button></div>
    </section>

    <details class="panel utility-panel">
      <summary><span><strong>Desktop behavior</strong><small>Autostart and native notifications</small></span></summary>
      <div class="details-body"><label class="toggle"><input id="autostart" type="checkbox" /><span><strong>Start with the computer</strong><small>Launch quietly in the system tray.</small></span></label><label class="toggle"><input id="notifications" type="checkbox" /><span><strong>Native notifications</strong><small>The operating system asks before enabling them.</small></span></label><button id="test-notification" class="tertiary" type="button" disabled>Send test notification</button></div>
    </details>

    <details class="panel utility-panel">
      <summary><span><strong>Story import and export</strong><small>Portable OneDay files</small></span></summary>
      <div class="details-body"><button id="import-package" class="secondary" type="button" disabled>Import OneDay package…</button><div class="export-row"><select id="story-select" aria-label="Story to export" disabled><option value="">Choose a story</option></select><button id="export-archive" class="secondary" type="button" disabled>Export complete ZIP…</button><button id="export-world" class="tertiary" type="button" disabled>Export world…</button></div><div id="transfer-message" class="message" role="status" aria-live="polite"></div><button id="refresh-stories" class="text-button" type="button" disabled>Refresh stories</button></div>
    </details>
  </main>`;

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
const claudeStatus = element<HTMLElement>("claude-status");
const claudeVersion = element<HTMLElement>("claude-version");
const claudeMessage = element<HTMLElement>("claude-message");
const installClaude = element<HTMLButtonElement>("install-claude");
const claudeGuide = element<HTMLButtonElement>("claude-guide");
const loginClaude = element<HTMLButtonElement>("login-claude");
const refreshClaude = element<HTMLButtonElement>("refresh-claude");
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
const updateTrust = element<HTMLElement>("update-trust");
const currentVersion = element<HTMLElement>("current-version");
const updateChannel = element<HTMLElement>("update-channel");
const updateMessage = element<HTMLElement>("update-message");
const updateResult = element<HTMLElement>("update-result");
const updateVersion = element<HTMLElement>("update-version");
const updateNotes = element<HTMLElement>("update-notes");
const checkUpdates = element<HTMLButtonElement>("check-updates");
const installUpdate = element<HTMLButtonElement>("install-update");
const refreshStories = element<HTMLButtonElement>("refresh-stories");
let connected = false;

function setStatus(node: HTMLElement, label: string, kind: "ready" | "failed" | "neutral" = "neutral") {
  node.textContent = label;
  node.classList.toggle("connected", kind === "ready");
  node.classList.toggle("failed", kind === "failed");
}

function setConnected(value: boolean) {
  connected = value;
  openStory.disabled = !value;
  openProviderSetup.disabled = !value;
  for (const control of [importPackage, storySelect, refreshStories]) control.disabled = !value;
  updateExportButtons();
}

function renderState(state: DesktopState) {
  const standalone = state.profile?.mode === "standalone";
  const ready = isServerReady(state.lifecycle);
  setStatus(connectionStatus, lifecycleLabel(state.lifecycle), ready ? "ready" : state.lifecycle.state === "failed" ? "failed" : "neutral");
  profileDescription.textContent = `${profileLabel(state.profile)}. ${standalone ? "Stories remain in this device’s isolated OneDay profile." : "Remote mode creates no local story database."}`;
  serverForm.hidden = standalone || !state.profile;
  standaloneControls.hidden = !standalone;
  providerPanel.hidden = !standalone;
  chooseRemote.classList.toggle("selected", state.profile?.mode === "remote");
  chooseStandalone.classList.toggle("selected", standalone);
  chooseRemote.setAttribute("aria-pressed", String(state.profile?.mode === "remote"));
  chooseStandalone.setAttribute("aria-pressed", String(standalone));
  if (state.profile?.mode === "remote") serverUrl.value = state.profile.serverUrl;
  if (state.lifecycle.state === "failed") connectionMessage.textContent = state.lifecycle.message;
  setConnected(ready);
}

function renderCodex(status: CodexStatus) {
  setStatus(codexStatus, status.authenticated ? "Ready" : status.available ? "Sign in" : "Not installed", status.authenticated ? "ready" : "neutral");
  codexVersion.textContent = status.version
    ? `${status.version} · ${status.source === "managed" ? "managed by OneDay" : "found on this device"}`
    : `Verified component available: ${status.managedVersion}`;
  codexMessage.textContent = status.message;
  installCodex.hidden = status.available;
  loginCodex.hidden = !status.available || status.authenticated;
}

function renderClaude(status: ClaudeStatus) {
  setStatus(claudeStatus, status.authenticated ? "Ready" : status.available ? "Sign in" : "Not installed", status.authenticated ? "ready" : "neutral");
  claudeVersion.textContent = status.version
    ? `${status.version} · found on this device`
    : status.installMethod
      ? `Official installation available through ${status.installMethod}`
      : "Install Claude Code with Anthropic’s official instructions.";
  claudeMessage.textContent = status.message;
  installClaude.hidden = status.available || !status.installSupported;
  claudeGuide.hidden = status.available || status.installSupported;
  loginClaude.hidden = !status.available || status.authenticated;
}

function setProviderBusy(provider: "codex" | "claude", busy: boolean, message = "") {
  const controls = provider === "codex"
    ? [installCodex, loginCodex, refreshCodex]
    : [installClaude, claudeGuide, loginClaude, refreshClaude];
  controls.forEach((button) => { button.disabled = busy; });
  if (message) (provider === "codex" ? codexMessage : claudeMessage).textContent = message;
}

async function refreshCodexStatus() {
  setProviderBusy("codex", true, "Checking Codex on this device…");
  try { renderCodex(await desktopBridge.codexStatus()); }
  catch (error) { setStatus(codexStatus, "Check failed", "failed"); codexMessage.textContent = friendlyError(error); }
  finally { setProviderBusy("codex", false); }
}

async function refreshClaudeStatus() {
  setProviderBusy("claude", true, "Checking Claude Code on this device…");
  try { renderClaude(await desktopBridge.claudeStatus()); }
  catch (error) { setStatus(claudeStatus, "Check failed", "failed"); claudeMessage.textContent = friendlyError(error); }
  finally { setProviderBusy("claude", false); }
}

async function refreshProviderStatuses() {
  await Promise.all([refreshCodexStatus(), refreshClaudeStatus()]);
}

function renderUpdater(status: UpdaterStatus) {
  currentVersion.textContent = `Version ${status.currentVersion}`;
  updateChannel.textContent = status.channel;
  updateTrust.textContent = status.reason;
  checkUpdates.hidden = !status.enabled;
  if (!status.enabled) updateMessage.textContent = "Install a signed release to enable automatic update checks.";
}

function renderUpdateCheck(result: UpdateCheck) {
  updateMessage.textContent = result.message;
  updateResult.hidden = !result.available;
  installUpdate.hidden = !result.available;
  updateVersion.textContent = result.version ? `Version ${result.version}` : "";
  updateNotes.textContent = result.notes?.trim() || (result.available ? "A signed OneDay release is ready." : "");
}

async function checkForUpdates(automatic = false) {
  checkUpdates.disabled = true;
  updateMessage.textContent = automatic ? "Checking the signed release feed…" : "Checking for a signed update…";
  try { renderUpdateCheck(await desktopBridge.checkUpdate()); }
  catch (error) { updateMessage.textContent = friendlyError(error); }
  finally { checkUpdates.disabled = false; }
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
    connectionMessage.textContent = "Connected securely. OneDay is ready.";
    await refreshState();
  } catch (error) {
    connectionMessage.textContent = friendlyError(error);
    await refreshState(false).catch(() => undefined);
  } finally { setBusy(false); }
}

async function startStandalone(restart = false) {
  setBusy(true, restart ? "Restarting the local OneDay service…" : "Starting OneDay on this device…");
  try {
    if (restart) await desktopBridge.restartStandalone(); else await desktopBridge.startStandalone();
    connectionMessage.textContent = "OneDay is running privately on this device.";
    await refreshState();
  } catch (error) {
    connectionMessage.textContent = friendlyError(error);
    await refreshState(false).catch(() => undefined);
  } finally { setBusy(false); }
}

chooseRemote.addEventListener("click", () => { serverForm.hidden = false; serverUrl.focus(); });
chooseStandalone.addEventListener("click", () => void (async () => { await startStandalone(); await refreshProviderStatuses(); })());
serverForm.addEventListener("submit", (event) => { event.preventDefault(); void connect(); });
restartStandalone.addEventListener("click", () => void startStandalone(true));
stopStandalone.addEventListener("click", () => void (async () => { setBusy(true, "Stopping OneDay on this device…"); try { await desktopBridge.stopStandalone(); await refreshState(false); connectionMessage.textContent = "The local OneDay service is stopped. Your stories remain on this device."; } catch (error) { connectionMessage.textContent = friendlyError(error); } finally { setBusy(false); } })());
openStory.addEventListener("click", () => void desktopBridge.showStoryWindow().catch((error) => { connectionMessage.textContent = friendlyError(error); }));
refreshStories.addEventListener("click", () => void loadStories());
checkUpdates.addEventListener("click", () => void checkForUpdates());
installUpdate.addEventListener("click", () => void (async () => { checkUpdates.disabled = true; installUpdate.disabled = true; updateMessage.textContent = "Downloading and verifying the signed update. OneDay will restart when it is safe…"; try { await desktopBridge.installUpdate(); } catch (error) { updateMessage.textContent = friendlyError(error); checkUpdates.disabled = false; installUpdate.disabled = false; } })());
refreshCodex.addEventListener("click", () => void refreshCodexStatus());
refreshClaude.addEventListener("click", () => void refreshClaudeStatus());
openProviderSetup.addEventListener("click", () => void desktopBridge.showProviderSetup().catch((error) => { codexMessage.textContent = friendlyError(error); }));
installCodex.addEventListener("click", () => void (async () => { setProviderBusy("codex", true, "Downloading the pinned Codex release and verifying its SHA-256 digest…"); try { renderCodex(await desktopBridge.installCodex()); } catch (error) { setStatus(codexStatus, "Install failed", "failed"); codexMessage.textContent = friendlyError(error); } finally { setProviderBusy("codex", false); } })());
loginCodex.addEventListener("click", () => void (async () => { setProviderBusy("codex", true, "Complete Codex sign-in in the browser…"); try { renderCodex(await desktopBridge.loginCodex()); await startStandalone(true); codexMessage.textContent = "Codex is ready. Choose its model in OneDay Setup."; } catch (error) { setStatus(codexStatus, "Sign in", "failed"); codexMessage.textContent = friendlyError(error); } finally { setProviderBusy("codex", false); } })());
installClaude.addEventListener("click", () => void (async () => { setProviderBusy("claude", true, "Installing the official Claude Code package with your system package manager…"); try { renderClaude(await desktopBridge.installClaude()); } catch (error) { setStatus(claudeStatus, "Install failed", "failed"); claudeMessage.textContent = friendlyError(error); } finally { setProviderBusy("claude", false); } })());
claudeGuide.addEventListener("click", () => void desktopBridge.openClaudeInstallGuide().catch((error) => { claudeMessage.textContent = friendlyError(error); }));
loginClaude.addEventListener("click", () => void (async () => { setProviderBusy("claude", true, "Complete Claude sign-in in the browser…"); try { renderClaude(await desktopBridge.loginClaude()); await startStandalone(true); claudeMessage.textContent = "Claude Code is ready. Choose its model in OneDay Setup."; } catch (error) { setStatus(claudeStatus, "Sign in", "failed"); claudeMessage.textContent = friendlyError(error); } finally { setProviderBusy("claude", false); } })());
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
    renderUpdater(state.updater);
    autostart.checked = await desktopBridge.autostartEnabled();
    notifications.checked = await desktopBridge.notificationsEnabled();
    testNotification.disabled = !notifications.checked;
    renderState(state);
    if (state.profile?.mode === "standalone") await refreshProviderStatuses();
    if (state.updater.enabled) void checkForUpdates(true);
    if (state.profile?.mode === "remote" && !state.startedMinimized) await connect();
    if (state.profile?.mode === "standalone" && !state.startedMinimized) await startStandalone();
  } catch (error) { connectionMessage.textContent = friendlyError(error); }
}
void bootstrap();
