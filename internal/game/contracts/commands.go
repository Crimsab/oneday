package contracts

type CommandGroup string

const (
	CommandGroupPlay   CommandGroup = "play"
	CommandGroupTalk   CommandGroup = "talk"
	CommandGroupState  CommandGroup = "state"
	CommandGroupSave   CommandGroup = "save"
	CommandGroupMeta   CommandGroup = "meta"
	CommandGroupSystem CommandGroup = "system"
	CommandGroupDebug  CommandGroup = "debug"
)

type CommandParity string

const (
	CommandParityShared       CommandParity = "shared"
	CommandParityTerminalOnly CommandParity = "terminal_only"
	CommandParityBrowserOnly  CommandParity = "browser_only"
)

type CommandBehavior string

const (
	CommandBehaviorSubmitAction   CommandBehavior = "submit_action"
	CommandBehaviorSubmitMeta     CommandBehavior = "submit_meta"
	CommandBehaviorOpenPanel      CommandBehavior = "open_panel"
	CommandBehaviorSaveCreate     CommandBehavior = "save_create"
	CommandBehaviorSaveLoad       CommandBehavior = "save_load"
	CommandBehaviorSaveDelete     CommandBehavior = "save_delete"
	CommandBehaviorInsertTemplate CommandBehavior = "insert_template"
	CommandBehaviorLocalOnly      CommandBehavior = "local_only"
	CommandBehaviorTimeline       CommandBehavior = "timeline"
)

type CommandArgDescriptor struct {
	Name        string `json:"name"`
	Label       string `json:"label,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Variadic    bool   `json:"variadic,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
	Description string `json:"description,omitempty"`
}

type CommandDescriptor struct {
	ID                 string                 `json:"id"`
	Canonical          string                 `json:"canonical"`
	Aliases            []string               `json:"aliases,omitempty"`
	Title              string                 `json:"title"`
	Description        string                 `json:"description"`
	Group              CommandGroup           `json:"group"`
	Parity             CommandParity          `json:"parity"`
	Behavior           CommandBehavior        `json:"behavior"`
	Args               []CommandArgDescriptor `json:"args,omitempty"`
	CompletionProvider string                 `json:"completion_provider,omitempty"`
	TrailingSpace      bool                   `json:"trailing_space,omitempty"`
	Examples           []string               `json:"examples,omitempty"`
	EnabledWhen        string                 `json:"enabled_when,omitempty"`
}

func CommandDescriptors() []CommandDescriptor {
	return []CommandDescriptor{
		{
			ID:          "inventory",
			Canonical:   "inventory",
			Aliases:     []string{"i"},
			Title:       "Inventory",
			Description: "Open inventory and crafting context.",
			Group:       CommandGroupState,
			Parity:      CommandParityShared,
			Behavior:    CommandBehaviorOpenPanel,
			Examples:    []string{"/inventory", "/i"},
		},
		{
			ID:          "stats",
			Canonical:   "stats",
			Aliases:     []string{"s"},
			Title:       "Stats",
			Description: "Open the character sheet.",
			Group:       CommandGroupState,
			Parity:      CommandParityShared,
			Behavior:    CommandBehaviorOpenPanel,
			Examples:    []string{"/stats", "/s"},
		},
		{
			ID:          "map",
			Canonical:   "map",
			Aliases:     []string{"m"},
			Title:       "Map",
			Description: "Open known locations and travel context.",
			Group:       CommandGroupState,
			Parity:      CommandParityShared,
			Behavior:    CommandBehaviorOpenPanel,
			Examples:    []string{"/map", "/m"},
		},
		{
			ID:          "journal",
			Canonical:   "journal",
			Aliases:     []string{"j"},
			Title:       "Journal",
			Description: "Open chapter journal and story notes.",
			Group:       CommandGroupState,
			Parity:      CommandParityShared,
			Behavior:    CommandBehaviorOpenPanel,
			Examples:    []string{"/journal", "/j"},
		},
		{
			ID:          "thoughts",
			Canonical:   "thoughts",
			Title:       "Thoughts",
			Description: "Inspect saved NPC private thoughts when enabled.",
			Group:       CommandGroupDebug,
			Parity:      CommandParityShared,
			Behavior:    CommandBehaviorOpenPanel,
			EnabledWhen: "visible_private_thoughts",
			Examples:    []string{"/thoughts"},
		},
		{
			ID:          "codex",
			Canonical:   "codex",
			Title:       "Codex",
			Description: "Open the story codex.",
			Group:       CommandGroupState,
			Parity:      CommandParityShared,
			Behavior:    CommandBehaviorOpenPanel,
			Examples:    []string{"/codex"},
		},
		{
			ID:          "characters",
			Canonical:   "characters",
			Title:       "Characters",
			Description: "Open character records.",
			Group:       CommandGroupState,
			Parity:      CommandParityShared,
			Behavior:    CommandBehaviorOpenPanel,
			Examples:    []string{"/characters"},
		},
		{
			ID:          "fronts",
			Canonical:   "hooks",
			Aliases:     []string{"fronts", "front"},
			Title:       "Fronts",
			Description: "Open fronts, hooks, fallout, and pressure clocks.",
			Group:       CommandGroupState,
			Parity:      CommandParityShared,
			Behavior:    CommandBehaviorOpenPanel,
			Examples:    []string{"/fronts", "/hooks"},
		},
		{
			ID:          "investigations",
			Canonical:   "investigations",
			Aliases:     []string{"investigation"},
			Title:       "Investigations",
			Description: "Open the investigation workspace.",
			Group:       CommandGroupState,
			Parity:      CommandParityShared,
			Behavior:    CommandBehaviorOpenPanel,
			Examples:    []string{"/investigations"},
		},
		{
			ID:          "projects",
			Canonical:   "projects",
			Aliases:     []string{"project"},
			Title:       "Projects",
			Description: "Open downtime projects and progress clocks.",
			Group:       CommandGroupState,
			Parity:      CommandParityShared,
			Behavior:    CommandBehaviorOpenPanel,
			Examples:    []string{"/projects"},
		},
		{
			ID:          "achievements",
			Canonical:   "achievements",
			Aliases:     []string{"a"},
			Title:       "Achievements",
			Description: "Show earned achievements.",
			Group:       CommandGroupState,
			Parity:      CommandParityShared,
			Behavior:    CommandBehaviorOpenPanel,
			Examples:    []string{"/achievements", "/a"},
		},
		{
			ID:          "craft",
			Canonical:   "craft",
			Aliases:     []string{"crafting"},
			Title:       "Craft",
			Description: "Open the crafting station.",
			Group:       CommandGroupPlay,
			Parity:      CommandParityShared,
			Behavior:    CommandBehaviorOpenPanel,
			Examples:    []string{"/craft"},
		},
		{
			ID:          "history",
			Canonical:   "history",
			Title:       "History",
			Description: "Open transcript and session history.",
			Group:       CommandGroupState,
			Parity:      CommandParityShared,
			Behavior:    CommandBehaviorOpenPanel,
			Examples:    []string{"/history"},
		},
		{ID: "branches", Canonical: "branches", Aliases: []string{"branch"}, Title: "Branches", Description: "List and navigate alternate story branches.", Group: CommandGroupState, Parity: CommandParityShared, Behavior: CommandBehaviorTimeline, Examples: []string{"/branches"}},
		{ID: "fork", Canonical: "fork", Title: "Fork branch", Description: "Fork the current story head into a named alternate.", Group: CommandGroupState, Parity: CommandParityShared, Behavior: CommandBehaviorTimeline, TrailingSpace: true, Args: []CommandArgDescriptor{{Name: "name", Label: "Name", Required: true, Variadic: true}}, Examples: []string{"/fork What if we stayed"}},
		{ID: "branch-rename", Canonical: "branch-rename", Aliases: []string{"rename-branch"}, Title: "Rename branch", Description: "Rename the active story branch.", Group: CommandGroupState, Parity: CommandParityShared, Behavior: CommandBehaviorTimeline, TrailingSpace: true, Args: []CommandArgDescriptor{{Name: "name", Label: "Name", Required: true, Variadic: true}}, Examples: []string{"/branch-rename Harbor route"}},
		{ID: "checkout", Canonical: "checkout", Title: "Checkout branch", Description: "Switch to a named branch without deleting the current one.", Group: CommandGroupState, Parity: CommandParityShared, Behavior: CommandBehaviorTimeline, TrailingSpace: true, Args: []CommandArgDescriptor{{Name: "branch", Label: "Branch", Required: true, Variadic: true}}, Examples: []string{"/checkout main"}},
		{
			ID:                 "talk",
			Canonical:          "talk",
			Title:              "Talk",
			Description:        "Talk to a nearby NPC with an optional intent and message.",
			Group:              CommandGroupTalk,
			Parity:             CommandParityShared,
			Behavior:           CommandBehaviorSubmitAction,
			CompletionProvider: "nearby_npcs",
			TrailingSpace:      true,
			Args: []CommandArgDescriptor{
				{Name: "npc", Label: "NPC", Required: true, Placeholder: "Maren Lo", Description: "Nearby NPC name."},
				{Name: "intent", Label: "Intent", Placeholder: "ask", Description: "ask, probe, bond, bargain, threaten, promise, lie, or confess."},
				{Name: "message", Label: "Message", Variadic: true, Placeholder: "about the ledger", Description: "What you say or ask."},
			},
			Examples: []string{"/talk Maren Lo probe the ledger"},
		},
		{
			ID:            "btw",
			Canonical:     "btw",
			Title:         "BTW",
			Description:   "Ask a contextual side question without advancing the turn.",
			Group:         CommandGroupMeta,
			Parity:        CommandParityShared,
			Behavior:      CommandBehaviorSubmitMeta,
			TrailingSpace: true,
			Args:          []CommandArgDescriptor{{Name: "question", Label: "Question", Required: true, Variadic: true}},
			Examples:      []string{"/btw what did I miss?"},
		},
		{
			ID:            "guide",
			Canonical:     "guide",
			Title:         "Guide",
			Description:   "Store soft future-facing story guidance.",
			Group:         CommandGroupMeta,
			Parity:        CommandParityShared,
			Behavior:      CommandBehaviorSubmitMeta,
			TrailingSpace: true,
			Args:          []CommandArgDescriptor{{Name: "guidance", Label: "Guidance", Required: true, Variadic: true}},
			Examples:      []string{"/guide make the next scene tenser"},
		},
		{
			ID:            "narrator",
			Canonical:     "narrator",
			Aliases:       []string{"n"},
			Title:         "Narrator Control",
			Description:   "Direct narrator canon or correct world state.",
			Group:         CommandGroupMeta,
			Parity:        CommandParityShared,
			Behavior:      CommandBehaviorSubmitMeta,
			TrailingSpace: true,
			Args:          []CommandArgDescriptor{{Name: "instruction", Label: "Instruction", Required: true, Variadic: true}},
			Examples:      []string{"/n the office is locked, not open"},
		},
		{
			ID:            "advance",
			Canonical:     "advance",
			Title:         "Advance",
			Description:   "Push to the next meaningful beat without replaying filler.",
			Group:         CommandGroupPlay,
			Parity:        CommandParityShared,
			Behavior:      CommandBehaviorSubmitAction,
			TrailingSpace: true,
			Args:          []CommandArgDescriptor{{Name: "hint", Label: "Hint", Variadic: true, Placeholder: "toward the docks"}},
			Examples:      []string{"/advance toward the next clue"},
		},
		{
			ID:            "timeskip",
			Canonical:     "timeskip",
			Title:         "Time Skip",
			Description:   "Jump ahead to a later meaningful moment.",
			Group:         CommandGroupPlay,
			Parity:        CommandParityShared,
			Behavior:      CommandBehaviorSubmitAction,
			TrailingSpace: true,
			Args:          []CommandArgDescriptor{{Name: "destination", Label: "Destination", Variadic: true, Placeholder: "tomorrow morning"}},
			Examples:      []string{"/timeskip after the stakeout"},
		},
		{
			ID:            "downtime",
			Canonical:     "downtime",
			Title:         "Downtime",
			Description:   "Request a quieter scene around a focus.",
			Group:         CommandGroupPlay,
			Parity:        CommandParityShared,
			Behavior:      CommandBehaviorSubmitAction,
			TrailingSpace: true,
			Args:          []CommandArgDescriptor{{Name: "focus", Label: "Focus", Required: true, Variadic: true}},
			Examples:      []string{"/downtime repair gear"},
		},
		{
			ID:            "save",
			Canonical:     "save",
			Title:         "Save",
			Description:   "Create a manual save.",
			Group:         CommandGroupSave,
			Parity:        CommandParityShared,
			Behavior:      CommandBehaviorSaveCreate,
			TrailingSpace: true,
			Args:          []CommandArgDescriptor{{Name: "name", Label: "Name", Variadic: true, Placeholder: "Before the docks"}},
			Examples:      []string{"/save Before the docks"},
		},
		{
			ID:                 "load",
			Canonical:          "load",
			Aliases:            []string{"saves"},
			Title:              "Load",
			Description:        "Open or filter saved snapshots.",
			Group:              CommandGroupSave,
			Parity:             CommandParityShared,
			Behavior:           CommandBehaviorSaveLoad,
			CompletionProvider: "saves",
			TrailingSpace:      true,
			Args:               []CommandArgDescriptor{{Name: "filter", Label: "Filter", Variadic: true}},
			Examples:           []string{"/load docks"},
		},
		{
			ID:                 "delete-save",
			Canonical:          "delete-save",
			Aliases:            []string{"delete"},
			Title:              "Delete Save",
			Description:        "Filter saves and delete one through the browser confirmation flow.",
			Group:              CommandGroupSave,
			Parity:             CommandParityBrowserOnly,
			Behavior:           CommandBehaviorSaveDelete,
			CompletionProvider: "saves",
			TrailingSpace:      true,
			Args:               []CommandArgDescriptor{{Name: "filter", Label: "Filter", Variadic: true}},
			Examples:           []string{"/delete-save docks"},
		},
		{
			ID:          "help",
			Canonical:   "help",
			Title:       "Help",
			Description: "Show available commands.",
			Group:       CommandGroupSystem,
			Parity:      CommandParityShared,
			Behavior:    CommandBehaviorLocalOnly,
			Examples:    []string{"/help"},
		},
		{
			ID:          "quit",
			Canonical:   "quit",
			Aliases:     []string{"q"},
			Title:       "Quit",
			Description: "Save and leave the terminal session.",
			Group:       CommandGroupSystem,
			Parity:      CommandParityTerminalOnly,
			Behavior:    CommandBehaviorLocalOnly,
			Examples:    []string{"/quit", "/q"},
		},
	}
}

func CommandAliasRegistry() map[string]string {
	registry := make(map[string]string)
	for _, descriptor := range CommandDescriptors() {
		if descriptor.Parity == CommandParityBrowserOnly {
			continue
		}
		registry[descriptor.Canonical] = descriptor.Canonical
		registry[descriptor.ID] = descriptor.Canonical
		for _, alias := range descriptor.Aliases {
			registry[alias] = descriptor.Canonical
		}
	}
	return registry
}
