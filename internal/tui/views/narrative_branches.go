package views

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m NarrativeModel) showBranches() (NarrativeModel, tea.Cmd) {
	story := m.narrator.Story()
	branches, err := m.narrator.DB().ListStoryBranches(story.ID)
	if err != nil {
		m.errMsg = fmt.Sprintf("Branches error: %v", err)
		return m, nil
	}
	var out strings.Builder
	out.WriteString("\n")
	for _, branch := range branches {
		marker := "  "
		if branch.ID == story.ActiveBranchID {
			marker = "* "
		}
		fmt.Fprintf(&out, "%s%s\n  id: %s\n  head: %s\n\n", marker, branch.Name, branch.ID, branch.HeadCommitID)
	}
	out.WriteString("Commands: /fork <name>, /branch-rename <name>, /checkout <name-or-id>")
	m.showOverlay("Story Branches", out.String())
	return m, nil
}

func (m NarrativeModel) forkBranch(args []string) (NarrativeModel, tea.Cmd) {
	name := strings.TrimSpace(strings.Join(args, " "))
	if name == "" {
		m.errMsg = "Usage: /fork <branch name>"
		return m, nil
	}
	story := m.narrator.Story()
	head, err := m.narrator.DB().GetActiveTimeline(story.ID)
	if err == nil {
		_, err = m.narrator.DB().ForkStoryBranch(story.ID, head.Commit.ID, name, story.Revision)
	}
	if err != nil {
		m.errMsg = fmt.Sprintf("Fork error: %v", err)
		return m, nil
	}
	_ = m.narrator.RefreshFromDB()
	m.statusMsg = "Created branch “" + name + "”."
	m.statusExpiry = time.Now().Add(8 * time.Second)
	return m.showBranches()
}

func (m NarrativeModel) renameActiveBranch(args []string) (NarrativeModel, tea.Cmd) {
	name := strings.TrimSpace(strings.Join(args, " "))
	if name == "" {
		m.errMsg = "Usage: /branch-rename <name>"
		return m, nil
	}
	story := m.narrator.Story()
	if err := m.narrator.DB().RenameStoryBranch(story.ID, story.ActiveBranchID, name, story.Revision); err != nil {
		m.errMsg = fmt.Sprintf("Rename error: %v", err)
		return m, nil
	}
	_ = m.narrator.RefreshFromDB()
	m.statusMsg = "Active branch renamed to “" + name + "”."
	m.statusExpiry = time.Now().Add(8 * time.Second)
	return m.showBranches()
}

func (m NarrativeModel) checkoutBranch(args []string) (NarrativeModel, tea.Cmd) {
	query := strings.TrimSpace(strings.Join(args, " "))
	if query == "" {
		m.errMsg = "Usage: /checkout <branch name or id>"
		return m, nil
	}
	story := m.narrator.Story()
	branches, err := m.narrator.DB().ListStoryBranches(story.ID)
	if err != nil {
		m.errMsg = fmt.Sprintf("Checkout error: %v", err)
		return m, nil
	}
	branchID, branchName := "", ""
	for _, branch := range branches {
		if branch.ID == query || strings.EqualFold(branch.Name, query) {
			branchID, branchName = branch.ID, branch.Name
			break
		}
	}
	if branchID == "" {
		m.errMsg = "Branch not found: " + query
		return m, nil
	}
	if _, err := m.narrator.DB().CheckoutStoryBranch(story.ID, branchID, story.Revision); err != nil {
		m.errMsg = fmt.Sprintf("Checkout error: %v", err)
		return m, nil
	}
	if err := m.narrator.RefreshFromDB(); err != nil {
		m.errMsg = fmt.Sprintf("Refresh error: %v", err)
		return m, nil
	}
	m.statusMsg = "Checked out “" + branchName + "”; previous branch preserved."
	m.statusExpiry = time.Now().Add(10 * time.Second)
	m.historyBrowser = nil
	return m.showHistory(nil)
}
