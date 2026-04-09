package views

import "strings"

func (m *NarrativeModel) recordInputHistory(value string) {
	if m == nil {
		return
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	if n := len(m.inputHistory); n > 0 && m.inputHistory[n-1] == value {
		m.resetInputHistoryNavigation()
		return
	}
	m.inputHistory = append(m.inputHistory, value)
	m.resetInputHistoryNavigation()
}

func (m *NarrativeModel) resetInputHistoryNavigation() {
	if m == nil {
		return
	}
	m.inputHistoryCursor = -1
	m.inputHistoryDraft = ""
}

func (m *NarrativeModel) canNavigateInputHistory() bool {
	if m == nil || !m.inputFocus || len(m.inputHistory) == 0 {
		return false
	}
	return !strings.Contains(m.input.Value(), "\n")
}

func (m *NarrativeModel) navigateInputHistory(delta int) bool {
	if !m.canNavigateInputHistory() || delta == 0 {
		return false
	}

	if m.inputHistoryCursor == -1 {
		if delta > 0 {
			return false
		}
		m.inputHistoryDraft = m.input.Value()
		m.inputHistoryCursor = len(m.inputHistory)
	}

	next := m.inputHistoryCursor + delta
	if next < 0 {
		next = 0
	}
	if next > len(m.inputHistory) {
		next = len(m.inputHistory)
	}

	if next == len(m.inputHistory) {
		m.input.SetValue(m.inputHistoryDraft)
		m.inputHistoryCursor = -1
		return true
	}

	m.inputHistoryCursor = next
	m.input.SetValue(m.inputHistory[m.inputHistoryCursor])
	return true
}
