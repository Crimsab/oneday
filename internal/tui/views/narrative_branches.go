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
		m.errMsg = m.loc.T("branches.error", err)
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
	out.WriteString(m.loc.T("branches.commands"))
	m.showOverlay(m.loc.T("branches.title"), out.String())
	return m, nil
}

func (m NarrativeModel) forkBranch(args []string) (NarrativeModel, tea.Cmd) {
	name := strings.TrimSpace(strings.Join(args, " "))
	if name == "" {
		m.errMsg = m.loc.T("branches.fork_usage")
		return m, nil
	}
	story := m.narrator.Story()
	head, err := m.narrator.DB().GetActiveTimeline(story.ID)
	if err == nil {
		_, err = m.narrator.DB().ForkStoryBranch(story.ID, head.Commit.ID, name, story.Revision)
	}
	if err != nil {
		m.errMsg = m.loc.T("branches.fork_error", err)
		return m, nil
	}
	_ = m.narrator.RefreshFromDB()
	m.statusMsg = m.loc.T("branches.created", name)
	m.statusExpiry = time.Now().Add(8 * time.Second)
	return m.showBranches()
}

func (m NarrativeModel) renameActiveBranch(args []string) (NarrativeModel, tea.Cmd) {
	name := strings.TrimSpace(strings.Join(args, " "))
	if name == "" {
		m.errMsg = m.loc.T("branches.rename_usage")
		return m, nil
	}
	story := m.narrator.Story()
	if err := m.narrator.DB().RenameStoryBranch(story.ID, story.ActiveBranchID, name, story.Revision); err != nil {
		m.errMsg = m.loc.T("branches.rename_error", err)
		return m, nil
	}
	_ = m.narrator.RefreshFromDB()
	m.statusMsg = m.loc.T("branches.renamed", name)
	m.statusExpiry = time.Now().Add(8 * time.Second)
	return m.showBranches()
}

func (m NarrativeModel) checkoutBranch(args []string) (NarrativeModel, tea.Cmd) {
	query := strings.TrimSpace(strings.Join(args, " "))
	if query == "" {
		m.errMsg = m.loc.T("branches.checkout_usage")
		return m, nil
	}
	story := m.narrator.Story()
	branches, err := m.narrator.DB().ListStoryBranches(story.ID)
	if err != nil {
		m.errMsg = m.loc.T("branches.checkout_error", err)
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
		m.errMsg = m.loc.T("branches.not_found", query)
		return m, nil
	}
	if _, err := m.narrator.DB().CheckoutStoryBranch(story.ID, branchID, story.Revision); err != nil {
		m.errMsg = m.loc.T("branches.checkout_error", err)
		return m, nil
	}
	if err := m.narrator.RefreshFromDB(); err != nil {
		m.errMsg = m.loc.T("branches.refresh_error", err)
		return m, nil
	}
	m.statusMsg = m.loc.T("branches.checked_out", branchName)
	m.statusExpiry = time.Now().Add(10 * time.Second)
	m.historyBrowser = nil
	return m.showHistory(nil)
}

func (m NarrativeModel) retryLatestDecision() (NarrativeModel, tea.Cmd) {
	story := m.narrator.Story()
	head, err := m.narrator.DB().GetActiveTimeline(story.ID)
	if err != nil {
		m.errMsg = m.loc.T("branches.retry_error", err)
		return m, nil
	}
	decisionID := strings.TrimSpace(head.Commit.ParentCommitID)
	if decisionID == "" {
		m.errMsg = m.loc.T("branches.no_decision")
		return m, nil
	}
	name := fmt.Sprintf("Turn %d alternative", head.Commit.CanonicalTurn)
	if _, err := m.narrator.DB().ForkAndCheckoutStoryBranch(story.ID, decisionID, name, story.Revision); err != nil {
		m.errMsg = m.loc.T("branches.retry_error", err)
		return m, nil
	}
	if err := m.narrator.RefreshFromDB(); err != nil {
		m.errMsg = m.loc.T("branches.refresh_error", err)
		return m, nil
	}
	m.statusMsg = m.loc.T("branches.retry_ready", name)
	m.statusExpiry = time.Now().Add(10 * time.Second)
	m.historyBrowser = nil
	return m.showHistory(nil)
}
