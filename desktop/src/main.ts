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
        <div><p class="step">2 · AI connections</p><h2 id="provider-title">Connect your preferred AI</h2><p>Codex is recommended for the complete OneDay experience, including subscription access and image generation. Other providers remain available.</p></div>
        <button id="open-provider-setup" class="secondary heading-action" type="button">Configure models</button>
      </div>
      <div class="provider-grid">
        <article class="provider-card recommended-provider" aria-labelledby="codex-title">
          <div class="provider-heading"><div><span class="provider-kind">Subscription</span><div class="provider-title-row"><h3 id="codex-title">Codex</h3><span class="recommendation">Recommended</span></div></div><span id="codex-status" class="status" role="status" aria-live="polite">Checking</span></div>
          <p id="codex-version">OneDay checks the Codex app, the official CLI, sign-in, and PATH separately.</p>
          <div class="provider-actions"><button id="install-codex" class="secondary" type="button" hidden>Install Codex</button><button id="login-codex" type="button" hidden>Sign in</button><button id="refresh-codex" class="tertiary" type="button">Refresh</button></div>
          <div id="codex-message" class="message compact" role="status" aria-live="polite"></div>
          <details id="codex-diagnostic" class="diagnostic" hidden>
            <summary><span>Codex diagnostics</span><span id="codex-diagnostic-shell" class="shell-badge">Terminal</span></summary>
            <div class="diagnostic-body">
              <p>Run this only when troubleshooting. It checks the command path, installed app, version, and sign-in status; it does not include saved API keys.</p>
              <pre><code id="codex-diagnostic-command"></code></pre>
              <div class="diagnostic-actions"><button id="copy-codex-diagnostic" class="secondary" type="button">Copy command</button><output id="codex-diagnostic-feedback" role="status" aria-live="polite"></output></div>
            </div>
          </details>
        </article>
        <article class="provider-card" aria-labelledby="claude-title">
          <div class="provider-heading"><div><span class="provider-kind">Subscription</span><h3 id="claude-title">Claude Code</h3></div><span id="claude-status" class="status" role="status" aria-live="polite">Checking</span></div>
          <p id="claude-version">Use a Claude subscription already signed in on this device.</p>
          <div class="provider-actions"><button id="install-claude" class="secondary" type="button" hidden>Install Claude</button><button id="claude-guide" class="secondary" type="button" hidden>Installation guide</button><button id="login-claude" type="button" hidden>Sign in</button><button id="refresh-claude" class="tertiary" type="button">Refresh</button></div>
          <div id="claude-message" class="message compact" role="status" aria-live="polite"></div>
        </article>
        <article class="provider-card api-provider" aria-labelledby="openrouter-title">
          <div class="provider-heading"><div><span class="provider-kind">API</span><h3 id="openrouter-title">OpenRouter</h3></div><span class="status neutral">In Setup</span></div>
          <p>Enter an endpoint and API key in OneDay. The key is write-only after saving.</p>
        </article>
        <article class="provider-card api-provider" aria-labelledby="openai-compatible-title">
          <div class="provider-heading"><div><span class="provider-kind">API or local</span><h3 id="openai-compatible-title">OpenAI-compatible</h3></div><span class="status neutral">In Setup</span></div>
          <p>Connect LiteLLM or any OpenAI-compatible endpoint, including a local network service.</p>
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
      <div class="details-body"><button id="import-package" class="secondary" type="button" disabled>Import OneDay package…</button><div class="export-row"><div id="story-select" class="desktop-select"><button id="story-select-trigger" class="desktop-select-trigger" type="button" role="combobox" aria-label="Story to export" aria-controls="story-select-menu" aria-expanded="false" aria-haspopup="listbox" disabled><span id="story-select-value">Choose a story</span></button><div id="story-select-menu" class="desktop-select-menu" role="listbox" aria-label="Story to export" hidden></div></div><button id="export-archive" class="secondary" type="button" disabled>Export complete ZIP…</button><button id="export-world" class="tertiary" type="button" disabled>Export world…</button></div><div id="transfer-message" class="message" role="status" aria-live="polite"></div><button id="refresh-stories" class="text-button" type="button" disabled>Refresh stories</button></div>
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
const codexDiagnostic = element<HTMLDetailsElement>("codex-diagnostic");
const codexDiagnosticShell = element<HTMLElement>("codex-diagnostic-shell");
const codexDiagnosticCommand = element<HTMLElement>("codex-diagnostic-command");
const copyCodexDiagnostic = element<HTMLButtonElement>("copy-codex-diagnostic");
const codexDiagnosticFeedback = element<HTMLOutputElement>("codex-diagnostic-feedback");
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
const storySelect = element<HTMLElement>("story-select");
const storySelectTrigger = element<HTMLButtonElement>("story-select-trigger");
const storySelectValue = element<HTMLElement>("story-select-value");
const storySelectMenu = element<HTMLElement>("story-select-menu");
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
let storyOptions: StorySummary[] = [];
let selectedStoryId = "";
let activeStoryIndex = 0;

function setStatus(node: HTMLElement, label: string, kind: "ready" | "failed" | "neutral" = "neutral") {
  node.textContent = label;
  node.classList.toggle("connected", kind === "ready");
  node.classList.toggle("failed", kind === "failed");
}

function setConnected(value: boolean) {
  connected = value;
  openStory.disabled = !value;
  openProviderSetup.disabled = !value;
  for (const control of [importPackage, storySelectTrigger, refreshStories]) control.disabled = !value;
  if (!value) closeStorySelect();
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
  else if (state.startupWarning) connectionMessage.textContent = state.startupWarning;
  else connectionMessage.textContent = "";
  setConnected(ready);
  openProviderSetup.disabled = !standalone;
  openStory.disabled = !(ready || standalone);
}

function renderCodex(status: CodexStatus) {
  const stateLabel: Record<CodexStatus["state"], string> = {
    ready: "Ready",
    signed_out: "Sign in",
    app_only: "App found",
    legacy_cli: "Legacy CLI",
    unusable: "Repair needed",
    missing: "Not installed",
  };
  setStatus(codexStatus, stateLabel[status.state], status.authenticated ? "ready" : status.state === "unusable" ? "failed" : "neutral");
  codexVersion.textContent = status.version
    ? `${status.version} · ${status.source === "global" ? "global CLI" : status.source === "managed" ? "private OneDay component" : "found on this device"}`
    : status.state === "app_only"
      ? "Codex app detected · official CLI not available yet"
      : status.state === "legacy_cli"
        ? "codex-cli detected · current codex command missing"
        : status.state === "unusable"
          ? "A Codex command was found but could not start"
    : status.managedVersion
      ? status.installScope === "global"
        ? `Verified CLI available: ${status.managedVersion} · OneDay can install it globally for this Windows user.`
        : `Managed component available: ${status.managedVersion} · install it to use your Codex subscription in OneDay.`
      : "No Codex installation found on this device.";
  codexMessage.textContent = status.message || (status.version
    ? "OneDay can use this local Codex installation after sign-in."
    : "Codex is optional. Install the verified CLI, then sign in to use the Codex subscription.");
  codexDiagnostic.hidden = !status.diagnosticCommand;
  codexDiagnostic.open = false;
  codexDiagnosticShell.textContent = status.diagnosticShell === "powershell" ? "PowerShell" : "Terminal";
  codexDiagnosticCommand.textContent = status.diagnosticCommand;
  codexDiagnosticFeedback.textContent = "";
  const canMigratePrivateWindowsInstall = status.installScope === "global" && status.source === "managed";
  installCodex.textContent = canMigratePrivateWindowsInstall
    ? "Make CLI global"
    : status.state === "app_only"
      ? "Add Codex CLI"
      : status.state === "legacy_cli"
        ? "Install current CLI"
        : status.state === "unusable"
          ? "Repair Codex CLI"
          : status.installScope === "global"
            ? "Install recommended"
            : "Install Codex";
  installCodex.hidden = status.available && !canMigratePrivateWindowsInstall;
  loginCodex.hidden = !status.available || status.authenticated;
}

function renderClaude(status: ClaudeStatus) {
  setStatus(claudeStatus, status.authenticated ? "Ready" : status.available ? "Sign in" : "Not installed", status.authenticated ? "ready" : "neutral");
  claudeVersion.textContent = status.version
    ? `${status.version} · found on this device`
    : status.installMethod
      ? `Official installation available through ${status.installMethod}`
      : "Install Claude Code with Anthropic’s official instructions.";
  claudeMessage.textContent = status.message || (status.authenticated
    ? "Claude Code is signed in locally and can be selected as a narrative provider."
    : "Claude Code is optional. Install it and sign in before selecting it in OneDay Setup.");
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
  catch (error) { setStatus(codexStatus, "Check failed", "failed"); codexMessage.textContent = friendlyError(error); codexDiagnostic.hidden = true; codexDiagnostic.open = false; codexDiagnosticCommand.textContent = ""; }
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

async function copyDiagnosticCommand() {
  const command = codexDiagnosticCommand.textContent?.trim() ?? "";
  if (!command) return;
  try {
    if (!navigator.clipboard?.writeText) throw new Error("Clipboard API unavailable");
    await navigator.clipboard.writeText(command);
    codexDiagnosticFeedback.textContent = "Command copied.";
  } catch {
    const field = document.createElement("textarea");
    field.value = command;
    field.setAttribute("readonly", "");
    field.style.position = "fixed";
    field.style.opacity = "0";
    document.body.append(field);
    field.select();
    const copied = document.execCommand("copy");
    field.remove();
    codexDiagnosticFeedback.textContent = copied ? "Command copied." : "Could not copy. Select the command above manually.";
  }
}

function renderUpdater(status: UpdaterStatus) {
  currentVersion.textContent = `Version ${status.currentVersion}`;
  updateChannel.textContent = status.channel;
  updateTrust.textContent = status.reason;
  checkUpdates.hidden = !status.enabled;
  if (!status.enabled) updateMessage.textContent = status.reason;
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
  const disabled = !connected || !selectedStoryId;
  exportArchive.disabled = disabled;
  exportWorld.disabled = disabled;
}

function renderStories(stories: StorySummary[]) {
  storyOptions = stories;
  if (!stories.some((story) => story.id === selectedStoryId)) selectedStoryId = "";
  renderStorySelect();
  updateExportButtons();
}

function renderStorySelect() {
  storySelectValue.textContent = storyOptions.find((story) => story.id === selectedStoryId)?.name || "Choose a story";
  storySelectMenu.replaceChildren();
  const options = [{ id: "", name: "Choose a story" }, ...storyOptions];
  options.forEach((story, index) => {
    const option = document.createElement("button");
    option.type = "button";
    option.id = `story-select-option-${index}`;
    option.setAttribute("role", "option");
    option.setAttribute("aria-selected", String(story.id === selectedStoryId));
    option.classList.toggle("active", index === activeStoryIndex);
    option.textContent = story.name;
    option.addEventListener("pointermove", () => setActiveStoryIndex(index));
    option.addEventListener("click", () => chooseStory(story.id));
    storySelectMenu.append(option);
  });
  if (storySelect.classList.contains("open")) {
    storySelectTrigger.setAttribute("aria-activedescendant", `story-select-option-${activeStoryIndex}`);
  } else {
    storySelectTrigger.removeAttribute("aria-activedescendant");
  }
}

function setActiveStoryIndex(index: number) {
  const optionCount = storyOptions.length + 1;
  activeStoryIndex = Math.max(0, Math.min(index, optionCount - 1));
  [...storySelectMenu.children].forEach((element, optionIndex) => element.classList.toggle("active", optionIndex === activeStoryIndex));
  storySelectTrigger.setAttribute("aria-activedescendant", `story-select-option-${activeStoryIndex}`);
  storySelectMenu.children[activeStoryIndex]?.scrollIntoView({ block: "nearest" });
}

function placeStorySelect() {
  const bounds = storySelectTrigger.getBoundingClientRect();
  storySelectMenu.style.left = `${Math.max(8, Math.min(bounds.left, innerWidth - bounds.width - 8))}px`;
  storySelectMenu.style.top = `${bounds.bottom + 5}px`;
  storySelectMenu.style.width = `${bounds.width}px`;
}

function openStorySelect() {
  if (storySelectTrigger.disabled) return;
  activeStoryIndex = Math.max(0, storyOptions.findIndex((story) => story.id === selectedStoryId) + 1);
  storySelect.classList.add("open");
  storySelectMenu.hidden = false;
  renderStorySelect();
  storySelectTrigger.setAttribute("aria-expanded", "true");
  setActiveStoryIndex(activeStoryIndex);
  placeStorySelect();
}

function closeStorySelect(restoreFocus = false) {
  storySelect.classList.remove("open");
  storySelectMenu.hidden = true;
  storySelectTrigger.setAttribute("aria-expanded", "false");
  storySelectTrigger.removeAttribute("aria-activedescendant");
  if (restoreFocus) storySelectTrigger.focus();
}

function chooseStory(id: string) {
  selectedStoryId = id;
  renderStorySelect();
  closeStorySelect(true);
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
    await refreshState();
    connectionMessage.textContent = "Connected securely. OneDay is ready.";
  } catch (error) {
    connectionMessage.textContent = friendlyError(error);
    await refreshState(false).catch(() => undefined);
  } finally { setBusy(false); }
}

async function startStandalone(restart = false) {
  setBusy(true, restart ? "Restarting the local OneDay service…" : "Starting OneDay on this device…");
  try {
    if (restart) await desktopBridge.restartStandalone(); else await desktopBridge.startStandalone();
    await refreshState();
    connectionMessage.textContent = "OneDay is running privately on this device.";
  } catch (error) {
    connectionMessage.textContent = friendlyError(error);
    await refreshState(false).catch(() => undefined);
  } finally { setBusy(false); }
}

chooseRemote.addEventListener("click", () => {
  serverForm.hidden = false;
  chooseRemote.classList.add("selected");
  chooseStandalone.classList.remove("selected");
  chooseRemote.setAttribute("aria-pressed", "true");
  chooseStandalone.setAttribute("aria-pressed", "false");
  serverUrl.focus();
});
chooseStandalone.addEventListener("click", () => void (async () => { await startStandalone(); await refreshProviderStatuses(); })());
serverForm.addEventListener("submit", (event) => { event.preventDefault(); void connect(); });
restartStandalone.addEventListener("click", () => void startStandalone(true));
stopStandalone.addEventListener("click", () => void (async () => { setBusy(true, "Stopping OneDay on this device…"); try { await desktopBridge.stopStandalone(); await refreshState(false); connectionMessage.textContent = "The local OneDay service is stopped. Your stories remain on this device."; } catch (error) { connectionMessage.textContent = friendlyError(error); } finally { setBusy(false); } })());
openStory.addEventListener("click", () => void (async () => {
  openStory.disabled = true;
  connectionMessage.textContent = "Checking the local service before opening OneDay…";
  try {
    await desktopBridge.showStoryWindow();
    await refreshState(false);
    connectionMessage.textContent = "OneDay is running privately on this device.";
  } catch (error) {
    await refreshState(false).catch(() => undefined);
    connectionMessage.textContent = friendlyError(error);
  }
})());
refreshStories.addEventListener("click", () => void loadStories());
checkUpdates.addEventListener("click", () => void checkForUpdates());
installUpdate.addEventListener("click", () => void (async () => { checkUpdates.disabled = true; installUpdate.disabled = true; updateMessage.textContent = "Downloading and verifying the signed update. OneDay will restart when it is safe…"; try { await desktopBridge.installUpdate(); } catch (error) { updateMessage.textContent = friendlyError(error); checkUpdates.disabled = false; installUpdate.disabled = false; } })());
refreshCodex.addEventListener("click", () => void refreshCodexStatus());
copyCodexDiagnostic.addEventListener("click", () => void copyDiagnosticCommand());
refreshClaude.addEventListener("click", () => void refreshClaudeStatus());
openProviderSetup.addEventListener("click", () => void desktopBridge.showProviderSetup().catch((error) => { codexMessage.textContent = friendlyError(error); }));
installCodex.addEventListener("click", () => void (async () => { setProviderBusy("codex", true, "Downloading the pinned Codex release and verifying its SHA-256 digest…"); try { renderCodex(await desktopBridge.installCodex()); } catch (error) { setStatus(codexStatus, "Install failed", "failed"); codexMessage.textContent = friendlyError(error); } finally { setProviderBusy("codex", false); } })());
loginCodex.addEventListener("click", () => void (async () => { setProviderBusy("codex", true, "Complete Codex sign-in in the browser…"); try { renderCodex(await desktopBridge.loginCodex()); await desktopBridge.restartStandalone(); await refreshState(false); codexMessage.textContent = "Codex is ready. OneDay restarted its local services; choose a model in OneDay Setup."; } catch (error) { setStatus(codexStatus, "Sign in", "failed"); codexMessage.textContent = friendlyError(error); } finally { setProviderBusy("codex", false); } })());
installClaude.addEventListener("click", () => void (async () => { setProviderBusy("claude", true, "Installing the official Claude Code package with your system package manager…"); try { renderClaude(await desktopBridge.installClaude()); } catch (error) { setStatus(claudeStatus, "Install failed", "failed"); claudeMessage.textContent = friendlyError(error); } finally { setProviderBusy("claude", false); } })());
claudeGuide.addEventListener("click", () => void desktopBridge.openClaudeInstallGuide().catch((error) => { claudeMessage.textContent = friendlyError(error); }));
loginClaude.addEventListener("click", () => void (async () => { setProviderBusy("claude", true, "Complete Claude sign-in in the browser…"); try { renderClaude(await desktopBridge.loginClaude()); await startStandalone(true); claudeMessage.textContent = "Claude Code is ready. Choose its model in OneDay Setup."; } catch (error) { setStatus(claudeStatus, "Sign in", "failed"); claudeMessage.textContent = friendlyError(error); } finally { setProviderBusy("claude", false); } })());
storySelectTrigger.addEventListener("click", () => {
  if (storySelect.classList.contains("open")) closeStorySelect(); else openStorySelect();
});
storySelectTrigger.addEventListener("keydown", (event) => {
  const optionCount = storyOptions.length + 1;
  if (event.key === "Escape" && storySelect.classList.contains("open")) {
    event.preventDefault();
    closeStorySelect(true);
    return;
  }
  if (event.key === "ArrowDown" || event.key === "ArrowUp") {
    event.preventDefault();
    if (!storySelect.classList.contains("open")) openStorySelect();
    setActiveStoryIndex((activeStoryIndex + (event.key === "ArrowDown" ? 1 : -1) + optionCount) % optionCount);
    return;
  }
  if (event.key === "Home" || event.key === "End") {
    event.preventDefault();
    if (!storySelect.classList.contains("open")) openStorySelect();
    setActiveStoryIndex(event.key === "Home" ? 0 : optionCount - 1);
    return;
  }
  if ((event.key === "Enter" || event.key === " ") && storySelect.classList.contains("open")) {
    event.preventDefault();
    chooseStory(activeStoryIndex === 0 ? "" : storyOptions[activeStoryIndex - 1]?.id || "");
    return;
  }
  if (event.key === "Tab") closeStorySelect();
});
document.addEventListener("pointerdown", (event) => {
  if (!storySelect.contains(event.target as Node) && !storySelectMenu.contains(event.target as Node)) closeStorySelect();
}, true);
window.addEventListener("resize", () => {
  if (storySelect.classList.contains("open")) placeStorySelect();
});
window.addEventListener("scroll", () => {
  if (storySelect.classList.contains("open")) placeStorySelect();
}, true);
autostart.addEventListener("change", () => void (async () => { const requested = autostart.checked; try { await desktopBridge.setAutostart(requested); } catch (error) { autostart.checked = !requested; connectionMessage.textContent = friendlyError(error); } })());
notifications.addEventListener("change", () => void (async () => {
  testNotification.disabled = !notifications.checked;
  if (!notifications.checked) return;
  try {
    notifications.checked = await desktopBridge.requestNotifications();
    testNotification.disabled = !notifications.checked;
  } catch (error) {
    notifications.checked = false;
    testNotification.disabled = true;
    connectionMessage.textContent = friendlyError(error);
  }
})());
testNotification.addEventListener("click", () => void desktopBridge.testNotification());
importPackage.addEventListener("click", () => void (async () => { transferMessage.textContent = "Waiting for a file…"; try { const result = await desktopBridge.importPackage(); transferMessage.textContent = result.message; if (!result.cancelled) await loadStories(); } catch (error) { transferMessage.textContent = friendlyError(error); } })());
async function exportStory(kind: "archive" | "world") { transferMessage.textContent = "Preparing export…"; try { transferMessage.textContent = (await desktopBridge.exportPackage(selectedStoryId, kind)).message; } catch (error) { transferMessage.textContent = friendlyError(error); } }
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
