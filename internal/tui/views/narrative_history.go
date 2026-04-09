package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/crimsab/oneday/internal/storage"
	"github.com/crimsab/oneday/internal/tui/theme"
)

func (m NarrativeModel) showHistory(args []string) (NarrativeModel, tea.Cmd) {
	query := strings.TrimSpace(strings.Join(args, " "))
	if m.narrator == nil || m.narrator.DB() == nil || strings.TrimSpace(m.narrator.SessionID()) == "" {
		m.showOverlay("History", "No active session history yet.")
		return m, nil
	}

	msgs, err := m.narrator.DB().GetSessionMessages(m.narrator.SessionID())
	if err != nil {
		m.errMsg = fmt.Sprintf("History error: %v", err)
		return m, nil
	}

	m.showOverlay("History", formatHistoryOverlay(msgs, query))
	return m, nil
}

func formatHistoryOverlay(messages []storage.ChatMessage, query string) string {
	if len(messages) == 0 {
		return "No messages in this session yet."
	}

	query = strings.TrimSpace(strings.ToLower(query))
	lines := []string{}
	if query != "" {
		lines = append(lines, fmt.Sprintf("Filtered by: %q", query), "")
	}

	matches := 0
	for _, msg := range messages {
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(content), query) {
			continue
		}
		matches++
		role := historyRoleLabel(msg.Role)
		lines = append(lines, fmt.Sprintf("Turn %d · %s", msg.Turn, role))
		lines = append(lines, indentWrappedHistory(content))
		lines = append(lines, "")
	}

	if matches == 0 {
		return fmt.Sprintf("No history entries matched %q.", query)
	}

	lines = append(lines, theme.MutedText.Render("Use /history <text> to filter the current session."))
	return strings.Join(lines, "\n")
}

func historyRoleLabel(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "assistant":
		return "Narrator"
	case "user":
		return "Player"
	case "system":
		return "System"
	default:
		if role == "" {
			return "Unknown"
		}
		return strings.ToUpper(role[:1]) + role[1:]
	}
}

func indentWrappedHistory(text string) string {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	for i, line := range lines {
		lines[i] = "  " + strings.TrimRight(line, " ")
	}
	return strings.Join(lines, "\n")
}
