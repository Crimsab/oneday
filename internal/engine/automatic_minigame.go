package engine

import (
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/crimsab/oneday/internal/game/contracts"
)

// AutomaticMiniGamePolicy controls browser-selectable challenge behavior while
// keeping the terminal and older clients on the established defaults.
type AutomaticMiniGamePolicy struct {
	Enabled        bool
	TimingFreeOnly bool
	UseCooldowns   bool
}

func DefaultAutomaticMiniGamePolicy() AutomaticMiniGamePolicy {
	return AutomaticMiniGamePolicy{Enabled: true, TimingFreeOnly: true, UseCooldowns: true}
}

func (n *Narrator) prepareAutomaticMiniGame(narrative *NarrativeResponse, turn int) (*MiniGameInstance, error) {
	if narrative == nil || n.story == nil || !n.automaticMiniGamePolicy.Enabled {
		return nil, nil
	}
	var intent *ChallengeSpec
	for _, challenge := range narrative.Challenges {
		if challenge != nil && challenge.Type == ChallengeMiniGame {
			intent = challenge
			break
		}
	}
	if intent == nil {
		return nil, nil
	}
	if n.persistMiniGames && n.db != nil {
		if _, err := n.db.GetActiveMiniGameInstance(n.story.ID); err == nil {
			return nil, nil
		} else if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}
	branchID := n.story.ActiveBranchID
	if branchID == "" && n.db != nil {
		head, err := n.db.GetActiveTimeline(n.story.ID)
		if err != nil {
			return nil, err
		}
		branchID = head.Branch.ID
	}
	recent := []MiniGameUsage{}
	if n.automaticMiniGamePolicy.UseCooldowns && n.db != nil {
		records, err := n.db.ListRecentMiniGameInstances(n.story.ID, 20)
		if err != nil {
			return nil, err
		}
		for _, record := range records {
			recent = append(recent, MiniGameUsage{Kind: MiniGameType(record.Kind), Turn: record.Turn})
		}
	}
	difficulty := intent.Difficulty
	if difficulty <= 0 {
		difficulty = 50
	}
	selection, err := SelectMiniGame(DefaultMiniGameCandidates(), MiniGameSelectionContext{
		NarrativeTags: automaticMiniGameTags(n.story.Genre, n.story.Tone, narrative.SceneType, intent.Description, intent.NPCName),
		CurrentTurn:   turn, Difficulty: difficulty, TimingFreeOnly: n.automaticMiniGamePolicy.TimingFreeOnly, Recent: recent,
	})
	if err != nil {
		return nil, err
	}
	definition := selection.Definition
	definition.Difficulty = difficulty
	if strings.TrimSpace(intent.Description) != "" {
		definition.Prompt = strings.TrimSpace(intent.Description)
	}
	intent.MiniGame = string(definition.Kind)
	identity := fmt.Sprintf("%s\x00%s\x00%d\x00%s", n.story.ID, branchID, turn, intent.Description)
	digest := sha256.Sum256([]byte(identity))
	id := fmt.Sprintf("mini-auto-%x", digest[:8])
	seed := int64(binary.BigEndian.Uint64(digest[8:16]) & uint64(contracts.MaxPortableChallengeSeed))
	instance := NewMiniGameInstance(id, n.story.ID, branchID, turn, seed, definition)
	if err := NewMiniGameHost().Start(&instance); err != nil {
		return nil, err
	}
	if !n.persistMiniGames {
		return nil, nil
	}
	return &instance, nil
}

func automaticMiniGameTags(values ...string) []string {
	words := map[string]bool{}
	for _, value := range values {
		for _, token := range strings.FieldsFunc(strings.ToLower(value), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) }) {
			words[token] = true
		}
	}
	tags := map[string]bool{}
	add := func(values ...string) {
		for _, value := range values {
			tags[value] = true
		}
	}
	for word := range words {
		switch word {
		case "clue", "clues", "deduce", "deduction", "identity", "mystery", "investigate", "investigation", "evidence", "contradiction":
			add("mystery", "investigation", "evidence", "identity")
		case "convince", "persuade", "persuasion", "bargain", "bargaining", "negotiate", "negotiation", "leverage", "diplomacy", "guard":
			add("social", "diplomacy", "conflict")
		case "sequence", "pattern", "decode", "decoding", "mechanism", "ritual", "puzzle", "cipher":
			add("puzzle", "ritual", "technology", "discovery")
		case "bid", "bidding", "auction", "offer", "price", "market", "resource":
			add("trade", "auction", "resource", "social")
		case "court", "courtroom", "trial", "testimony", "witness", "procedure", "accuse", "accusation":
			add("courtroom", "trial", "evidence", "social")
		case "joke", "comedy", "banter", "perform", "performance", "embarrass", "embarrassment", "callback":
			add("comedy", "social", "performance", "zero-combat")
		default:
			tags[word] = true
		}
	}
	result := make([]string, 0, len(tags))
	for tag := range tags {
		result = append(result, tag)
	}
	return result
}
