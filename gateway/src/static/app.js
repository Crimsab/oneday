const state = {
  stories: [],
  storyId: "",
  snapshot: null,
  tab: "overview",
  events: null,
  historyFilter: "",
  sending: false,
  talkMode: null,
};

const $ = (id) => document.getElementById(id);

function setSync(text, error) {
  const node = $("syncValue");
  node.textContent = text;
  node.classList.toggle("error-text", Boolean(error));
}

async function api(path, options = {}) {
  const response = await fetch(path, options);
  const payload = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error(payload.error || response.statusText);
  }
  return payload;
}

async function boot() {
  bindUI();
  applyTheme(localStorage.getItem("oneday-theme") || preferredTheme());
  await refreshHealth();
  await loadStories();
}

function bindUI() {
  $("refreshStories").addEventListener("click", loadStories);
  $("storySearch").addEventListener("input", renderStories);
  $("jumpBottom").addEventListener("click", jumpTranscriptBottom);
  $("themeToggle").addEventListener("click", () => {
    const next = document.documentElement.dataset.theme === "dark" ? "light" : "dark";
    applyTheme(next);
  });
  $("clearAction").addEventListener("click", () => {
    $("actionInput").value = "";
    $("actionInput").focus();
  });
  $("composer").addEventListener("submit", (event) => {
    event.preventDefault();
    submitComposer();
  });
  $("tabs").addEventListener("click", (event) => {
    const button = event.target.closest("button[data-tab]");
    if (!button) return;
    state.tab = button.dataset.tab;
    renderTabs();
    renderPanel();
  });
}

function preferredTheme() {
  return window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches
    ? "dark"
    : "light";
}

function applyTheme(theme) {
  document.documentElement.dataset.theme = theme;
  localStorage.setItem("oneday-theme", theme);
}

async function refreshHealth() {
  try {
    const health = await api("/api/health");
    $("healthLine").textContent = `${health.stories} stories - Rust gateway`;
  } catch (error) {
    $("healthLine").textContent = error.message;
  }
}

async function loadStories() {
  setSync("Loading");
  try {
    state.stories = await api("/api/stories");
    renderStories();
    setSync(state.storyId ? "Live" : "Idle");
  } catch (error) {
    setSync("Error", true);
    $("storyList").textContent = error.message;
  }
}

function renderStories() {
  const list = $("storyList");
  list.innerHTML = "";
  const query = $("storySearch").value.trim().toLowerCase();
  const stories = state.stories.filter((story) => {
    const haystack = `${story.name} ${story.description} ${story.genre} ${story.tone}`.toLowerCase();
    return !query || haystack.includes(query);
  });
  if (!stories.length) {
    list.textContent = "No matching stories.";
    list.classList.add("empty-state");
    return;
  }
  list.classList.remove("empty-state");
  stories.forEach((story) => {
    const button = document.createElement("button");
    button.type = "button";
    button.className = `story-card${story.id === state.storyId ? " active" : ""}`;
    button.addEventListener("click", () => selectStory(story.id));
    appendText(button, "strong", story.name || story.id);
    appendText(button, "span", [story.genre, story.tone, story.language].filter(Boolean).join(" / "), "story-meta");
    appendText(button, "span", story.description || story.id, "story-meta");
    list.appendChild(button);
  });
}

async function selectStory(storyId) {
  if (state.events) {
    state.events.close();
    state.events = null;
  }
  state.storyId = storyId;
  state.snapshot = null;
  renderStories();
  await loadSnapshot();
  openEvents();
}

async function loadSnapshot() {
  if (!state.storyId) return;
  setSync("Loading");
  try {
    const snapshot = await api(`/api/stories/${encodeURIComponent(state.storyId)}/snapshot`);
    applySnapshot(snapshot);
    setSync("Live");
  } catch (error) {
    setSync("Error", true);
    $("transcript").textContent = error.message;
  }
}

function openEvents() {
  if (!state.storyId) return;
  const source = new EventSource(`/api/stories/${encodeURIComponent(state.storyId)}/events`);
  state.events = source;
  source.addEventListener("open", () => setSync("Live"));
  source.addEventListener("snapshot", (event) => {
    applySnapshot(JSON.parse(event.data));
    setSync("Live");
  });
  source.addEventListener("error", () => setSync("Reconnecting", true));
}

function applySnapshot(snapshot) {
  state.snapshot = snapshot;
  $("storyTitle").textContent = snapshot.story.name || snapshot.story.id;
  $("turnValue").textContent = snapshot.world.current_turn;
  $("locationValue").textContent = snapshot.world.current_location || "-";
  $("chapterValue").textContent = snapshot.world.current_chapter || "-";
  renderTranscript();
  renderChoices();
  renderTabs();
  renderPanel();
  renderStories();
}

function renderTranscript() {
  const node = $("transcript");
  node.innerHTML = "";
  const messages = state.snapshot?.messages || [];
  if (!messages.length) {
    node.textContent = "No canonical messages yet.";
    node.classList.add("empty-state");
    return;
  }
  node.classList.remove("empty-state");
  messages.forEach((message) => {
    const card = document.createElement("article");
    card.className = `message ${message.role}`;
    appendText(card, "div", `turn ${message.turn} / ${message.role} / ${message.message_type}`, "message-meta");
    appendText(card, "div", message.content || "(empty)", "message-content");
    node.appendChild(card);
  });
  jumpTranscriptBottom();
}

function renderChoices() {
  const node = $("choices");
  node.innerHTML = "";
  const choices = state.snapshot?.choices || [];
  if (!choices.length) {
    node.textContent = "Free action available.";
    node.classList.add("empty-state");
    return;
  }
  node.classList.remove("empty-state");
  choices.forEach((choice) => {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "choice-button";
    button.addEventListener("click", () => sendChoice(choice));
    appendText(button, "strong", choice.text);
    const meta = [choice.intent, choice.risk, choice.scope, choice.certainty].filter(Boolean).join(" / ");
    appendText(button, "span", meta || `choice ${choice.id}`, "choice-meta");
    node.appendChild(button);
  });
}

function renderTabs() {
  document.querySelectorAll("#tabs button").forEach((button) => {
    button.classList.toggle("active", button.dataset.tab === state.tab);
  });
}

function renderPanel() {
  const body = $("panelBody");
  body.innerHTML = "";
  if (!state.snapshot) {
    body.textContent = "State panels appear after selecting a story.";
    body.classList.add("empty-state");
    return;
  }
  body.classList.remove("empty-state");
  const renderers = {
    overview: renderOverview,
    inventory: renderInventory,
    stats: renderStats,
    codex: renderCodex,
    fronts: renderFronts,
    investigations: renderInvestigations,
    projects: renderProjects,
    history: renderHistory,
    saves: renderSaves,
  };
  (renderers[state.tab] || renderOverview)(body);
}

function renderOverview(body) {
  const snap = state.snapshot;
  section(body, "Story", [
    ["Name", snap.story.name],
    ["Genre", snap.story.genre || "-"],
    ["Tone", snap.story.tone || "-"],
    ["Language", snap.story.language || "-"],
  ]);
  section(body, "Character", [
    ["Name", snap.character.name],
    ["Background", snap.character.fields.background || "-"],
  ]);
  section(body, "World", [
    ["Location", snap.world.current_location || "-"],
    ["Turn", String(snap.world.current_turn)],
    ["Chapter", String(snap.world.current_chapter)],
    ["Updated", snap.world.updated_at || "-"],
  ]);
}

function renderInventory(body) {
  section(body, "Inventory", jsonPairs(state.snapshot.character.fields.inventory));
  section(body, "Known Recipes", jsonPairs(state.snapshot.character.fields.known_recipes));
}

function renderStats(body) {
  section(body, "Stats", jsonPairs(state.snapshot.character.fields.stats));
  section(body, "Traits", jsonPairs(state.snapshot.character.fields.traits));
  section(body, "Skills", jsonPairs(state.snapshot.character.fields.skills));
}

function renderCodex(body) {
  section(body, "Chapters", state.snapshot.panels.chapters.map((chapter) => [
    `Chapter ${chapter.chapter_number}`,
    `${chapter.title || "Untitled"}\n${chapter.summary || ""}`.trim(),
  ]));
  section(body, "Characters", state.snapshot.panels.npcs.map((npc) => [
    npc.name,
    `${npc.fields.role || "NPC"}\n${npc.fields.appearance || ""}\nDisposition: ${npc.fields.disposition}`.trim(),
  ]));
  section(body, "Timeline", jsonPairs(state.snapshot.world.timeline));
}

function renderFronts(body) {
  section(body, "Open Hooks", jsonPairs(state.snapshot.world.story_hooks));
  section(body, "Fronts", jsonPairs(state.snapshot.world.fronts));
  section(body, "Recent Fallout", jsonPairs(state.snapshot.world.world_reactions));
  section(body, "Scene Contract", jsonPairs(state.snapshot.world.scene_contract));
}

function renderInvestigations(body) {
  section(body, "Investigations", jsonPairs(state.snapshot.world.investigations));
}

function renderProjects(body) {
  section(body, "Projects", jsonPairs(state.snapshot.world.projects));
  section(body, "Guidance", jsonPairs(state.snapshot.world.guidance));
}

function renderHistory(body) {
  const input = document.createElement("input");
  input.className = "search-input history-search";
  input.type = "search";
  input.placeholder = "Search history";
  input.value = state.historyFilter;
  input.addEventListener("input", () => {
    state.historyFilter = input.value;
    renderPanel();
  });
  body.appendChild(input);
  const query = state.historyFilter.trim().toLowerCase();
  const rows = (state.snapshot.messages || [])
    .filter((message) => !query || `${message.role} ${message.content} ${message.message_type} ${message.turn}`.toLowerCase().includes(query))
    .map((message) => [`${message.turn} ${message.role}`, message.content]);
  section(body, "Messages", rows);
}

function renderSaves(body) {
  section(body, "Saves", state.snapshot.panels.saves.map((save) => [
    save.name,
    `turn ${save.turn}, chapter ${save.chapter}, ${save.location || "unknown"}\n${save.created_at}`,
  ]));
  section(body, "Sessions", state.snapshot.panels.sessions.map((session) => [
    session.id,
    `${session.started_at}\n${session.ended_at ? `ended ${session.ended_at}` : "active"}`,
  ]));
  section(body, "Achievements", state.snapshot.panels.achievements.map((item) => [
    item.name,
    `${item.rarity} / ${item.category}\n${item.description || item.context || ""}`.trim(),
  ]));
}

function section(parent, title, rows) {
  const box = document.createElement("section");
  box.className = "panel-section";
  appendText(box, "h4", title);
  if (!rows || !rows.length) {
    appendText(box, "p", "No data yet.", "muted");
    parent.appendChild(box);
    return;
  }
  rows.forEach(([key, value]) => {
    const row = document.createElement("div");
    row.className = "kv";
    appendText(row, "strong", key);
    appendText(row, "span", normalizeValue(value));
    box.appendChild(row);
  });
  parent.appendChild(box);
}

function jsonPairs(value) {
  if (value == null) return [];
  if (Array.isArray(value)) {
    return value.map((item, index) => [labelFor(item, index), detailFor(item)]);
  }
  if (typeof value === "object") {
    return Object.entries(value).map(([key, item]) => [key, detailFor(item)]);
  }
  return [["Value", String(value)]];
}

function labelFor(item, index) {
  if (item && typeof item === "object") {
    return item.title || item.name || item.id || `Item ${index + 1}`;
  }
  return `Item ${index + 1}`;
}

function detailFor(item) {
  if (item == null) return "";
  if (typeof item === "string" || typeof item === "number" || typeof item === "boolean") {
    return String(item);
  }
  return JSON.stringify(item, null, 2);
}

function normalizeValue(value) {
  if (value == null || value === "") return "-";
  return String(value);
}

async function submitComposer() {
  const input = $("actionInput");
  const text = input.value.trim();
  if (!text || !state.snapshot || state.sending) return;
  const commandResult = commandToAction(text);
  if (commandResult.handled) {
    input.value = "";
    return;
  }
  const actionText = commandResult.text || wrapTalk(text);
  await sendAction({ kind: "free_text", text: actionText });
  input.value = "";
}

async function sendChoice(choice) {
  await sendAction({
    kind: "choice",
    choice_id: choice.id,
    text: `[Choice ${choice.id}] ${choice.text}`,
  });
}

async function sendAction(action) {
  if (!state.snapshot || state.sending) return;
  state.sending = true;
  $("sendAction").disabled = true;
  setSync("Sending");
  try {
    const payload = await api(`/api/stories/${encodeURIComponent(state.storyId)}/actions`, {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({
        session_id: state.snapshot.active_session.id,
        client_turn: state.snapshot.world.current_turn,
        idempotency_key: crypto.randomUUID(),
        action,
        capabilities: { images: true, ascii: true, roll_log: true },
      }),
    });
    applySnapshot(payload.snapshot);
    setSync("Live");
  } catch (error) {
    setSync("Error", true);
    showInlineError(error.message);
    await loadSnapshot().catch(() => {});
  } finally {
    state.sending = false;
    $("sendAction").disabled = false;
  }
}

function commandToAction(text) {
  const lower = text.toLowerCase();
  const simpleTabs = {
    "/inventory": "inventory",
    "/i": "inventory",
    "/stats": "stats",
    "/s": "stats",
    "/characters": "codex",
    "/codex": "codex",
    "/fronts": "fronts",
    "/front": "fronts",
    "/hooks": "fronts",
    "/investigations": "investigations",
    "/investigation": "investigations",
    "/projects": "projects",
    "/project": "projects",
    "/history": "history",
    "/journal": "codex",
    "/j": "codex",
    "/achievements": "saves",
    "/a": "saves",
    "/map": "overview",
    "/m": "overview",
    "/help": "overview",
  };
  if (simpleTabs[lower]) {
    state.tab = simpleTabs[lower];
    renderTabs();
    renderPanel();
    return { handled: true };
  }
  if (lower.startsWith("/advance")) {
    return { text: buildAdvanceSceneAction(text.slice("/advance".length).trim()) };
  }
  if (lower.startsWith("/timeskip")) {
    return { text: buildTimeSkipAction(text.slice("/timeskip".length).trim()) };
  }
  if (lower.startsWith("/downtime")) {
    const hint = text.slice("/downtime".length).trim();
    if (!hint) {
      showInlineError("Usage: /downtime <focus>");
      return { handled: true };
    }
    return { text: `[Downtime Scene] ${hint}` };
  }
  if (lower.startsWith("/talk")) {
    return handleTalkCommand(text);
  }
  if (lower === "/talk off") {
    state.talkMode = null;
    setSync("Talk off");
    return { handled: true };
  }
  return {};
}

function handleTalkCommand(text) {
  const parts = text.trim().split(/\s+/).slice(1);
  if (!parts.length) {
    state.tab = "codex";
    renderTabs();
    renderPanel();
    showInlineError("Use /talk <npc> [intent] [message]. Known NPCs are in Codex.");
    return { handled: true };
  }
  const intents = new Set(["ask", "probe", "bond", "bargain", "threaten", "promise", "lie", "confess"]);
  const target = parts.shift();
  let intent = "ask";
  if (parts.length && intents.has(parts[0].toLowerCase())) {
    intent = parts.shift().toLowerCase();
  }
  const message = parts.join(" ").trim();
  if (!message) {
    state.talkMode = { target, intent };
    setSync(`Talk ${target}`);
    return { handled: true };
  }
  return { text: formatTalkAction(target, intent, message) };
}

function wrapTalk(text) {
  if (!state.talkMode) return text;
  return formatTalkAction(state.talkMode.target, state.talkMode.intent, text);
}

function formatTalkAction(target, intent, message) {
  return `[Talk to ${target} | intent:${intent || "ask"}] ${message}`;
}

function buildAdvanceSceneAction(hint) {
  let base = "[Advance Scene] Move to the next meaningful beat now. If this micro-scene is exhausted, do not replay it with near-identical prose or choices. Introduce a concrete change: reveal, consequence, interruption, pressure, location shift, or a natural time skip.";
  if (hint) {
    base += " Treat any extra text after this tag as the player's desired timing, destination, or arrival point for the next beat. Requested timing or destination: " + hint;
  }
  return base;
}

function buildTimeSkipAction(hint) {
  let base = "[Time Skip] Jump forward to a later meaningful moment instead of playing filler turn by turn. Keep continuity clear: show what changed, what stayed true, and why this later beat matters now. If exact age is unclear, use a life stage or milestone rather than inventing a precise number.";
  if (hint) {
    base += " Treat any extra text after this tag as the player's preferred arrival point, approximate age, or target moment. Requested destination: " + hint;
  }
  return base;
}

function showInlineError(message) {
  const card = document.createElement("article");
  card.className = "message";
  appendText(card, "div", "browser", "message-meta");
  appendText(card, "div", message, "message-content error-text");
  $("transcript").appendChild(card);
  jumpTranscriptBottom();
}

function jumpTranscriptBottom() {
  const node = $("transcript");
  node.scrollTop = node.scrollHeight;
}

function appendText(parent, tag, text, className) {
  const node = document.createElement(tag);
  if (className) node.className = className;
  node.textContent = text || "";
  parent.appendChild(node);
  return node;
}

boot().catch((error) => {
  $("healthLine").textContent = error.message;
  setSync("Error", true);
});
