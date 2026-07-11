package engine

import (
	"fmt"
	"strings"

	"github.com/crimsab/oneday/internal/storage"
)

func ApplyInvestigationUpdate(world *storage.WorldState, raw map[string]interface{}, currentTurn int) []StateChange {
	if world == nil || len(raw) == 0 {
		return nil
	}

	board := loadInvestigationBoard(world)
	caseIdx := ensureInvestigationCase(&board, raw, currentTurn)
	if caseIdx < 0 {
		return nil
	}
	invCase := &board.Cases[caseIdx]
	changes := []StateChange{}

	if summary := strings.TrimSpace(stringValue(raw["summary"])); summary != "" {
		invCase.Summary = summary
	}
	if status := strings.TrimSpace(stringValue(raw["status"])); status != "" {
		invCase.Status = status
	}
	invCase.UpdatedTurn = currentTurn
	invCase.Links = append(invCase.Links, normalizeInvestigationLinks(toInvestigationLinks(raw["links"]))...)
	invCase.Links = normalizeInvestigationLinks(invCase.Links)

	changes = append(changes, applyInvestigationClueUpdates(invCase, toObjectMaps(raw["clues"]), currentTurn)...)
	changes = append(changes, applyInvestigationSuspectUpdates(invCase, toObjectMaps(raw["suspects"]))...)
	changes = append(changes, applyInvestigationClaimUpdates(invCase, toObjectMaps(raw["claims"]))...)
	changes = append(changes, applyInvestigationContradictionUpdates(invCase, toObjectMaps(raw["contradictions"]))...)
	changes = append(changes, applyInvestigationLeadUpdates(invCase, toObjectMaps(raw["leads"]))...)
	changes = append(changes, applyInvestigationTheoryUpdates(invCase, toObjectMaps(raw["theories"]))...)

	if len(changes) == 0 && strings.TrimSpace(invCase.Summary) == "" && strings.TrimSpace(invCase.Status) == "" {
		return nil
	}

	storeInvestigationBoard(world, board)
	changes = append([]StateChange{{
		Target:      "world",
		Field:       fmt.Sprintf("investigation.%s", invCase.Title),
		New:         invCase.Status,
		Description: fmt.Sprintf("Investigation updated: %s", invCase.Title),
	}}, changes...)
	return changes
}

func ensureInvestigationCase(board *InvestigationBoard, raw map[string]interface{}, currentTurn int) int {
	if board == nil {
		return -1
	}

	caseID := strings.TrimSpace(stringValue(raw["case_id"]))
	caseTitle := strings.TrimSpace(stringValue(raw["case_title"]))
	if caseMap, ok := raw["case"].(map[string]interface{}); ok {
		caseID = firstNonEmpty(caseID, strings.TrimSpace(stringValue(caseMap["id"])))
		caseTitle = firstNonEmpty(caseTitle, strings.TrimSpace(stringValue(caseMap["title"])))
	}

	if idx := findInvestigationCaseIndex(board.Cases, caseID, caseTitle); idx >= 0 {
		return idx
	}
	if caseTitle == "" {
		return -1
	}

	newCase := InvestigationCase{
		ID:          firstNonEmpty(caseID, "case:"+slugKey(caseTitle)),
		Title:       caseTitle,
		Summary:     strings.TrimSpace(stringValue(raw["summary"])),
		Status:      firstNonEmpty(strings.TrimSpace(stringValue(raw["status"])), "open"),
		UpdatedTurn: currentTurn,
		Links:       normalizeInvestigationLinks(toInvestigationLinks(raw["links"])),
	}
	board.Cases = append(board.Cases, newCase)
	return len(board.Cases) - 1
}

func applyInvestigationClueUpdates(invCase *InvestigationCase, updates []map[string]interface{}, currentTurn int) []StateChange {
	if invCase == nil || len(updates) == 0 {
		return nil
	}
	changes := []StateChange{}
	for _, update := range updates {
		action := normalizedEvidenceAction(update["action"], "add")
		clue, ok := buildInvestigationClue(invCase, update, currentTurn)
		if !ok {
			continue
		}
		idx := findInvestigationClueIndex(invCase.Clues, clue.ID, clue.Label)
		switch action {
		case "discredit":
			clue.Status = "discredited"
		case "reveal":
			clue.Status = "known"
		case "revise":
			if clue.Status == "" {
				clue.Status = "known"
			}
		}
		if idx >= 0 {
			mergeInvestigationClue(&invCase.Clues[idx], clue)
		} else {
			invCase.Clues = append(invCase.Clues, clue)
		}
		changes = append(changes, StateChange{
			Target:      "world",
			Field:       fmt.Sprintf("investigation.%s.clue", invCase.Title),
			New:         clue.Label,
			Description: fmt.Sprintf("Clue %s: %s", action, clue.Label),
		})
	}
	invCase.Clues = normalizeInvestigationClues(invCase.Clues)
	return changes
}

func applyInvestigationSuspectUpdates(invCase *InvestigationCase, updates []map[string]interface{}) []StateChange {
	if invCase == nil || len(updates) == 0 {
		return nil
	}
	changes := []StateChange{}
	for _, update := range updates {
		action := normalizedEvidenceAction(update["action"], "add")
		suspect := InvestigationSuspect{
			ID:     strings.TrimSpace(stringValue(update["id"])),
			Name:   strings.TrimSpace(stringValue(update["name"])),
			Detail: strings.TrimSpace(stringValue(update["detail"])),
			Status: strings.TrimSpace(stringValue(update["status"])),
			Links:  normalizeInvestigationLinks(toInvestigationLinks(update["links"])),
		}
		if suspect.Name == "" {
			continue
		}
		if suspect.ID == "" {
			suspect.ID = "suspect:" + slugKey(suspect.Name)
		}
		if suspect.Status == "" {
			suspect.Status = "person_of_interest"
		}
		if action == "discredit" || action == "collapse" {
			suspect.Status = "ruled_out"
		}
		idx := findInvestigationSuspectIndex(invCase.Suspects, suspect.ID, suspect.Name)
		if idx >= 0 {
			mergeInvestigationSuspect(&invCase.Suspects[idx], suspect)
		} else {
			invCase.Suspects = append(invCase.Suspects, suspect)
		}
		changes = append(changes, StateChange{
			Target:      "world",
			Field:       fmt.Sprintf("investigation.%s.suspect", invCase.Title),
			New:         suspect.Name,
			Description: fmt.Sprintf("Suspect %s: %s", action, suspect.Name),
		})
	}
	invCase.Suspects = normalizeInvestigationSuspects(invCase.Suspects)
	return changes
}

func applyInvestigationClaimUpdates(invCase *InvestigationCase, updates []map[string]interface{}) []StateChange {
	if invCase == nil || len(updates) == 0 {
		return nil
	}
	changes := []StateChange{}
	for _, update := range updates {
		action := normalizedEvidenceAction(update["action"], "add")
		claim, ok := buildInvestigationClaim(invCase, update)
		if !ok {
			continue
		}
		switch action {
		case "discredit":
			claim.Status = "disputed"
			claim.Confidence = "weak"
		case "strengthen", "confirm", "reveal":
			claim.Status = "supported"
			if claim.Confidence == "" || claim.Confidence == "uncertain" {
				claim.Confidence = "likely"
			}
		case "collapse":
			claim.Status = "collapsed"
			claim.Confidence = "weak"
		}
		idx := findInvestigationClaimIndex(invCase.Claims, claim.ID, claim.Statement)
		if idx >= 0 {
			mergeInvestigationClaim(&invCase.Claims[idx], claim)
		} else {
			invCase.Claims = append(invCase.Claims, claim)
		}
		changes = append(changes, StateChange{
			Target:      "world",
			Field:       fmt.Sprintf("investigation.%s.claim", invCase.Title),
			New:         claim.Statement,
			Description: fmt.Sprintf("Claim %s: %s", action, claim.Statement),
		})
	}
	invCase.Claims = normalizeInvestigationClaims(invCase.Claims)
	return changes
}

func applyInvestigationContradictionUpdates(invCase *InvestigationCase, updates []map[string]interface{}) []StateChange {
	if invCase == nil || len(updates) == 0 {
		return nil
	}
	changes := []StateChange{}
	for _, update := range updates {
		action := normalizedEvidenceAction(update["action"], "add")
		item := InvestigationContradiction{
			ID:     strings.TrimSpace(stringValue(update["id"])),
			Label:  strings.TrimSpace(stringValue(update["label"])),
			Detail: strings.TrimSpace(stringValue(update["detail"])),
			Status: strings.TrimSpace(stringValue(update["status"])),
			Links:  normalizeInvestigationLinks(toInvestigationLinks(update["links"])),
		}
		if item.Label == "" {
			continue
		}
		if item.ID == "" {
			item.ID = "contradiction:" + slugKey(item.Label)
		}
		if item.Status == "" {
			item.Status = "open"
		}
		if action == "resolve" {
			item.Status = "resolved"
		}
		idx := findInvestigationContradictionIndex(invCase.Contradictions, item.ID, item.Label)
		if idx >= 0 {
			mergeInvestigationContradiction(&invCase.Contradictions[idx], item)
		} else {
			invCase.Contradictions = append(invCase.Contradictions, item)
		}
		changes = append(changes, StateChange{
			Target:      "world",
			Field:       fmt.Sprintf("investigation.%s.contradiction", invCase.Title),
			New:         item.Label,
			Description: fmt.Sprintf("Contradiction %s: %s", action, item.Label),
		})
	}
	invCase.Contradictions = normalizeInvestigationContradictions(invCase.Contradictions)
	return changes
}

func applyInvestigationLeadUpdates(invCase *InvestigationCase, updates []map[string]interface{}) []StateChange {
	if invCase == nil || len(updates) == 0 {
		return nil
	}
	changes := []StateChange{}
	for _, update := range updates {
		action := normalizedEvidenceAction(update["action"], "add")
		item := InvestigationLead{
			ID:     strings.TrimSpace(stringValue(update["id"])),
			Title:  strings.TrimSpace(stringValue(update["title"])),
			Detail: strings.TrimSpace(stringValue(update["detail"])),
			Status: strings.TrimSpace(stringValue(update["status"])),
			Links:  normalizeInvestigationLinks(toInvestigationLinks(update["links"])),
		}
		if item.Title == "" {
			continue
		}
		if item.ID == "" {
			item.ID = "lead:" + slugKey(item.Title)
		}
		if item.Status == "" {
			item.Status = "open"
		}
		switch action {
		case "progress":
			item.Status = "pursued"
		case "collapse", "discredit":
			item.Status = "cold"
		}
		idx := findInvestigationLeadIndex(invCase.Leads, item.ID, item.Title)
		if idx >= 0 {
			mergeInvestigationLead(&invCase.Leads[idx], item)
		} else {
			invCase.Leads = append(invCase.Leads, item)
		}
		changes = append(changes, StateChange{
			Target:      "world",
			Field:       fmt.Sprintf("investigation.%s.lead", invCase.Title),
			New:         item.Title,
			Description: fmt.Sprintf("Lead %s: %s", action, item.Title),
		})
	}
	invCase.Leads = normalizeInvestigationLeads(invCase.Leads)
	return changes
}

func applyInvestigationTheoryUpdates(invCase *InvestigationCase, updates []map[string]interface{}) []StateChange {
	if invCase == nil || len(updates) == 0 {
		return nil
	}
	changes := []StateChange{}
	for _, update := range updates {
		action := normalizedEvidenceAction(update["action"], "add")
		theory, ok := buildInvestigationTheory(invCase, update)
		if !ok {
			continue
		}
		switch action {
		case "strengthen", "confirm", "reveal":
			theory.Status = "likely"
			if theory.Confidence == "" || theory.Confidence == "fragile" {
				theory.Confidence = "supported"
			}
		case "collapse", "discredit":
			theory.Status = "collapsed"
			theory.Confidence = "weak"
		}
		idx := findInvestigationTheoryIndex(invCase.Theories, theory.ID, theory.Statement)
		if idx >= 0 {
			mergeInvestigationTheory(&invCase.Theories[idx], theory)
		} else {
			invCase.Theories = append(invCase.Theories, theory)
		}
		changes = append(changes, StateChange{
			Target:      "world",
			Field:       fmt.Sprintf("investigation.%s.theory", invCase.Title),
			New:         theory.Statement,
			Description: fmt.Sprintf("Theory %s: %s", action, theory.Statement),
		})
	}
	invCase.Theories = normalizeInvestigationTheories(invCase.Theories)
	return changes
}

func buildInvestigationClue(invCase *InvestigationCase, update map[string]interface{}, currentTurn int) (InvestigationClue, bool) {
	clue := InvestigationClue{
		ID:             strings.TrimSpace(stringValue(update["id"])),
		Label:          strings.TrimSpace(stringValue(update["label"])),
		Detail:         strings.TrimSpace(stringValue(update["detail"])),
		Source:         strings.TrimSpace(stringValue(update["source"])),
		DiscoveredTurn: currentTurn,
		Status:         strings.TrimSpace(stringValue(update["status"])),
		Links:          normalizeInvestigationLinks(toInvestigationLinks(update["links"])),
	}
	if fromTruth := revealedHiddenTruth(invCase, update); fromTruth != nil {
		clue.Label = firstNonEmpty(clue.Label, fromTruth.Label)
		clue.Detail = firstNonEmpty(clue.Detail, fromTruth.Detail)
		if clue.Status == "" {
			clue.Status = "known"
		}
	}
	if clue.Label == "" {
		return InvestigationClue{}, false
	}
	if clue.ID == "" {
		clue.ID = "clue:" + slugKey(clue.Label)
	}
	if clue.Status == "" {
		clue.Status = "known"
	}
	return clue, true
}

func buildInvestigationClaim(invCase *InvestigationCase, update map[string]interface{}) (InvestigationClaim, bool) {
	claim := InvestigationClaim{
		ID:         strings.TrimSpace(stringValue(update["id"])),
		Statement:  strings.TrimSpace(stringValue(update["statement"])),
		Confidence: strings.TrimSpace(stringValue(update["confidence"])),
		Status:     strings.TrimSpace(stringValue(update["status"])),
		Links:      normalizeInvestigationLinks(toInvestigationLinks(update["links"])),
	}
	if fromTruth := revealedHiddenTruth(invCase, update); fromTruth != nil {
		claim.Statement = firstNonEmpty(claim.Statement, fromTruth.Label)
		claim.Links = append(claim.Links, InvestigationLink{Kind: "truth", RefID: fromTruth.ID, Label: fromTruth.Label})
	}
	if claim.Statement == "" {
		return InvestigationClaim{}, false
	}
	if claim.ID == "" {
		claim.ID = "claim:" + slugKey(claim.Statement)
	}
	if claim.Status == "" {
		claim.Status = "open"
	}
	if claim.Confidence == "" {
		claim.Confidence = "uncertain"
	}
	claim.Links = normalizeInvestigationLinks(claim.Links)
	return claim, true
}

func buildInvestigationTheory(invCase *InvestigationCase, update map[string]interface{}) (InvestigationTheory, bool) {
	theory := InvestigationTheory{
		ID:         strings.TrimSpace(stringValue(update["id"])),
		Statement:  strings.TrimSpace(stringValue(update["statement"])),
		Confidence: strings.TrimSpace(stringValue(update["confidence"])),
		Status:     strings.TrimSpace(stringValue(update["status"])),
		Links:      normalizeInvestigationLinks(toInvestigationLinks(update["links"])),
	}
	if fromTruth := revealedHiddenTruth(invCase, update); fromTruth != nil {
		theory.Statement = firstNonEmpty(theory.Statement, fromTruth.Label)
		theory.Links = append(theory.Links, InvestigationLink{Kind: "truth", RefID: fromTruth.ID, Label: fromTruth.Label})
	}
	if theory.Statement == "" {
		return InvestigationTheory{}, false
	}
	if theory.ID == "" {
		theory.ID = "theory:" + slugKey(theory.Statement)
	}
	if theory.Status == "" {
		theory.Status = "forming"
	}
	if theory.Confidence == "" {
		theory.Confidence = "fragile"
	}
	theory.Links = normalizeInvestigationLinks(theory.Links)
	return theory, true
}

func revealedHiddenTruth(invCase *InvestigationCase, update map[string]interface{}) *InvestigationHiddenTruth {
	if invCase == nil {
		return nil
	}
	action := normalizedEvidenceAction(update["action"], "add")
	if action != "reveal" {
		return nil
	}
	truthID := strings.TrimSpace(stringValue(update["hidden_truth_id"]))
	truthLabel := strings.TrimSpace(stringValue(update["hidden_truth"]))
	for i := range invCase.HiddenTruths {
		if truthID != "" && invCase.HiddenTruths[i].ID != truthID {
			continue
		}
		if truthID == "" && truthLabel != "" && !strings.EqualFold(invCase.HiddenTruths[i].Label, truthLabel) {
			continue
		}
		invCase.HiddenTruths[i].Status = "revealed"
		return &invCase.HiddenTruths[i]
	}
	return nil
}

func normalizedEvidenceAction(raw interface{}, fallback string) string {
	action := strings.ToLower(strings.TrimSpace(stringValue(raw)))
	switch action {
	case "add", "revise", "discredit", "reveal", "strengthen", "confirm", "collapse", "resolve", "progress":
		return action
	default:
		return fallback
	}
}

func toInvestigationLinks(raw interface{}) []InvestigationLink {
	items := toObjectMaps(raw)
	if len(items) == 0 {
		return nil
	}
	links := make([]InvestigationLink, 0, len(items))
	for _, item := range items {
		links = append(links, InvestigationLink{
			Kind:  strings.TrimSpace(stringValue(item["kind"])),
			RefID: strings.TrimSpace(stringValue(item["ref_id"])),
			Label: strings.TrimSpace(stringValue(item["label"])),
		})
	}
	return links
}

func findInvestigationCaseIndex(cases []InvestigationCase, id, title string) int {
	id = strings.TrimSpace(id)
	title = strings.TrimSpace(title)
	for i, item := range cases {
		if id != "" && strings.EqualFold(item.ID, id) {
			return i
		}
		if title != "" && strings.EqualFold(item.Title, title) {
			return i
		}
	}
	return -1
}

func findInvestigationClueIndex(items []InvestigationClue, id, label string) int {
	id = strings.TrimSpace(id)
	label = strings.TrimSpace(label)
	for i, item := range items {
		if id != "" && strings.EqualFold(item.ID, id) {
			return i
		}
		if label != "" && strings.EqualFold(item.Label, label) {
			return i
		}
	}
	return -1
}

func findInvestigationSuspectIndex(items []InvestigationSuspect, id, name string) int {
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	for i, item := range items {
		if id != "" && strings.EqualFold(item.ID, id) {
			return i
		}
		if name != "" && strings.EqualFold(item.Name, name) {
			return i
		}
	}
	return -1
}

func findInvestigationClaimIndex(items []InvestigationClaim, id, statement string) int {
	id = strings.TrimSpace(id)
	statement = strings.TrimSpace(statement)
	for i, item := range items {
		if id != "" && strings.EqualFold(item.ID, id) {
			return i
		}
		if statement != "" && strings.EqualFold(item.Statement, statement) {
			return i
		}
	}
	return -1
}

func findInvestigationContradictionIndex(items []InvestigationContradiction, id, label string) int {
	id = strings.TrimSpace(id)
	label = strings.TrimSpace(label)
	for i, item := range items {
		if id != "" && strings.EqualFold(item.ID, id) {
			return i
		}
		if label != "" && strings.EqualFold(item.Label, label) {
			return i
		}
	}
	return -1
}

func findInvestigationLeadIndex(items []InvestigationLead, id, title string) int {
	id = strings.TrimSpace(id)
	title = strings.TrimSpace(title)
	for i, item := range items {
		if id != "" && strings.EqualFold(item.ID, id) {
			return i
		}
		if title != "" && strings.EqualFold(item.Title, title) {
			return i
		}
	}
	return -1
}

func findInvestigationTheoryIndex(items []InvestigationTheory, id, statement string) int {
	id = strings.TrimSpace(id)
	statement = strings.TrimSpace(statement)
	for i, item := range items {
		if id != "" && strings.EqualFold(item.ID, id) {
			return i
		}
		if statement != "" && strings.EqualFold(item.Statement, statement) {
			return i
		}
	}
	return -1
}

func mergeInvestigationClue(dst *InvestigationClue, src InvestigationClue) {
	if dst == nil {
		return
	}
	if src.Detail != "" {
		dst.Detail = src.Detail
	}
	if src.Source != "" {
		dst.Source = src.Source
	}
	if src.Status != "" {
		dst.Status = src.Status
	}
	if src.DiscoveredTurn > 0 {
		dst.DiscoveredTurn = src.DiscoveredTurn
	}
	dst.Links = normalizeInvestigationLinks(append(dst.Links, src.Links...))
}

func mergeInvestigationSuspect(dst *InvestigationSuspect, src InvestigationSuspect) {
	if dst == nil {
		return
	}
	if src.Detail != "" {
		dst.Detail = src.Detail
	}
	if src.Status != "" {
		dst.Status = src.Status
	}
	dst.Links = normalizeInvestigationLinks(append(dst.Links, src.Links...))
}

func mergeInvestigationClaim(dst *InvestigationClaim, src InvestigationClaim) {
	if dst == nil {
		return
	}
	if src.Status != "" {
		dst.Status = src.Status
	}
	if src.Confidence != "" {
		dst.Confidence = src.Confidence
	}
	dst.Links = normalizeInvestigationLinks(append(dst.Links, src.Links...))
}

func mergeInvestigationContradiction(dst *InvestigationContradiction, src InvestigationContradiction) {
	if dst == nil {
		return
	}
	if src.Detail != "" {
		dst.Detail = src.Detail
	}
	if src.Status != "" {
		dst.Status = src.Status
	}
	dst.Links = normalizeInvestigationLinks(append(dst.Links, src.Links...))
}

func mergeInvestigationLead(dst *InvestigationLead, src InvestigationLead) {
	if dst == nil {
		return
	}
	if src.Detail != "" {
		dst.Detail = src.Detail
	}
	if src.Status != "" {
		dst.Status = src.Status
	}
	dst.Links = normalizeInvestigationLinks(append(dst.Links, src.Links...))
}

func mergeInvestigationTheory(dst *InvestigationTheory, src InvestigationTheory) {
	if dst == nil {
		return
	}
	if src.Status != "" {
		dst.Status = src.Status
	}
	if src.Confidence != "" {
		dst.Confidence = src.Confidence
	}
	dst.Links = normalizeInvestigationLinks(append(dst.Links, src.Links...))
}
