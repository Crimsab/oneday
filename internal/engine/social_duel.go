package engine

import (
	"fmt"
	"strings"

	"github.com/crimsab/oneday/internal/storage"
)

type SocialDuelStatus string

const (
	SocialDuelActive   SocialDuelStatus = "active"
	SocialDuelResolved SocialDuelStatus = "resolved"
)

type SocialStance string

const (
	SocialStanceMeasured SocialStance = "measured"
	SocialStanceBold     SocialStance = "bold"
	SocialStanceGuarded  SocialStance = "guarded"
)

type SocialAction string

const (
	SocialActionAppeal   SocialAction = "appeal"
	SocialActionPressure SocialAction = "pressure"
	SocialActionDeceive  SocialAction = "deceive"
	SocialActionConcede  SocialAction = "concede"
	SocialActionExpose   SocialAction = "expose"
	SocialActionWithdraw SocialAction = "withdraw"
	SocialActionEscalate SocialAction = "escalate"
)

type SocialLeverage struct {
	ID         string `json:"id,omitempty"`
	Label      string `json:"label"`
	Kind       string `json:"kind,omitempty"`
	Strength   int    `json:"strength,omitempty"`
	Consumable bool   `json:"consumable,omitempty"`
	Spent      bool   `json:"spent,omitempty"`
}

type SocialFailForward struct {
	Kind   string `json:"kind"`
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
}

type SocialDuelState struct {
	NPCName          string           `json:"npc_name"`
	Objective        string           `json:"objective"`
	Stakes           string           `json:"stakes,omitempty"`
	PlayerStance     SocialStance     `json:"player_stance"`
	NPCStance        SocialStance     `json:"npc_stance"`
	Status           SocialDuelStatus `json:"status"`
	Round            int              `json:"round"`
	PlayerComposure  int              `json:"player_composure"`
	NPCComposure     int              `json:"npc_composure"`
	PlayerPatience   int              `json:"player_patience"`
	NPCPatience      int              `json:"npc_patience"`
	Tempo            int              `json:"tempo"`
	PlayerLeverage   []SocialLeverage `json:"player_leverage,omitempty"`
	NPCLeverage      []SocialLeverage `json:"npc_leverage,omitempty"`
	Winner           string           `json:"winner,omitempty"`
	Outcome          string           `json:"outcome,omitempty"`
	LastExchangeNote string           `json:"last_exchange_note,omitempty"`
}

type SocialDuelSpec struct {
	NPCName        string           `json:"npc_name"`
	Objective      string           `json:"objective"`
	Stakes         string           `json:"stakes,omitempty"`
	PlayerStance   SocialStance     `json:"player_stance,omitempty"`
	NPCStance      SocialStance     `json:"npc_stance,omitempty"`
	PlayerLeverage []SocialLeverage `json:"player_leverage,omitempty"`
	NPCLeverage    []SocialLeverage `json:"npc_leverage,omitempty"`
}

type SocialRoundResult struct {
	PlayerAction  SocialAction       `json:"player_action"`
	NPCAction     SocialAction       `json:"npc_action"`
	PlayerRoll    int                `json:"player_roll"`
	NPCRoll       int                `json:"npc_roll"`
	PlayerScore   int                `json:"player_score"`
	NPCScore      int                `json:"npc_score"`
	PlayerDamage  int                `json:"player_damage"`
	NPCDamage     int                `json:"npc_damage"`
	TempoDelta    int                `json:"tempo_delta"`
	Consumed      *SocialLeverage    `json:"consumed,omitempty"`
	Resolved      bool               `json:"resolved"`
	Winner        string             `json:"winner,omitempty"`
	Outcome       string             `json:"outcome,omitempty"`
	FailForward   *SocialFailForward `json:"fail_forward,omitempty"`
	ExchangeLabel string             `json:"exchange_label"`
}

type SocialDuelEngine struct {
	rollFn func() int
}

func NewSocialDuelEngine() *SocialDuelEngine {
	return &SocialDuelEngine{rollFn: RollD20}
}

func NewSocialDuelEngineWithRollFn(rollFn func() int) *SocialDuelEngine {
	if rollFn == nil {
		rollFn = RollD20
	}
	return &SocialDuelEngine{rollFn: rollFn}
}

func (e *SocialDuelEngine) Start(spec SocialDuelSpec, char *storage.Character, npc *storage.NPC) (*SocialDuelState, error) {
	if strings.TrimSpace(spec.NPCName) == "" {
		return nil, fmt.Errorf("social duel requires npc name")
	}
	if strings.TrimSpace(spec.Objective) == "" {
		return nil, fmt.Errorf("social duel requires objective")
	}
	if char == nil || npc == nil {
		return nil, fmt.Errorf("social duel requires character and npc")
	}

	playerStance := normalizeSocialStance(spec.PlayerStance, SocialStanceMeasured)
	npcStance := normalizeSocialStance(spec.NPCStance, SocialStanceGuarded)
	axes := loadRelationshipAxes(npc)

	state := &SocialDuelState{
		NPCName:         spec.NPCName,
		Objective:       spec.Objective,
		Stakes:          strings.TrimSpace(spec.Stakes),
		PlayerStance:    playerStance,
		NPCStance:       npcStance,
		Status:          SocialDuelActive,
		Round:           1,
		PlayerComposure: 8 + socialBestStat(char, "wil", "will", "cha", "presence")/3 + clampSocialBonus((axes.Respect+axes.Trust)/25),
		NPCComposure:    7 + maxInt(0, npc.Disposition)/20 + clampSocialBonus((axes.Fear+axes.Debt)/30),
		PlayerPatience:  3 + clampSocialBonus((axes.Respect+axes.Trust)/40),
		NPCPatience:     3 + clampSocialBonus((axes.Fear+maxInt(0, npc.Disposition))/45),
		PlayerLeverage:  normalizeSocialLeverage(spec.PlayerLeverage),
		NPCLeverage:     normalizeSocialLeverage(spec.NPCLeverage),
	}
	if state.PlayerComposure < 6 {
		state.PlayerComposure = 6
	}
	if state.NPCComposure < 6 {
		state.NPCComposure = 6
	}
	if state.PlayerPatience < 2 {
		state.PlayerPatience = 2
	}
	if state.NPCPatience < 2 {
		state.NPCPatience = 2
	}

	return state, nil
}

func (e *SocialDuelEngine) ResolveRound(state *SocialDuelState, char *storage.Character, npc *storage.NPC, playerAction, npcAction SocialAction) (*SocialRoundResult, error) {
	if state == nil {
		return nil, fmt.Errorf("social duel state is nil")
	}
	if state.Status != SocialDuelActive {
		return nil, fmt.Errorf("social duel is not active")
	}
	if char == nil || npc == nil {
		return nil, fmt.Errorf("social duel requires character and npc")
	}

	playerAction = normalizeSocialAction(playerAction)
	npcAction = normalizeSocialAction(npcAction)

	if playerAction == SocialActionWithdraw {
		state.Status = SocialDuelResolved
		state.Winner = npc.Name
		state.Outcome = "withdrawal"
		fail := &SocialFailForward{
			Kind:   "concession",
			Title:  "You yield the field",
			Detail: "The exchange stays playable, but the other side gets to define the next beat.",
		}
		state.LastExchangeNote = fail.Title
		return &SocialRoundResult{
			PlayerAction:  playerAction,
			NPCAction:     npcAction,
			Resolved:      true,
			Winner:        state.Winner,
			Outcome:       state.Outcome,
			FailForward:   fail,
			ExchangeLabel: fail.Title,
		}, nil
	}

	playerRoll := e.roll()
	npcRoll := e.roll()
	playerConsumed := consumeSocialLeverage(&state.PlayerLeverage, playerAction)
	npcConsumed := consumeSocialLeverage(&state.NPCLeverage, npcAction)
	playerScore := socialActionScore(state, char, npc, playerAction, state.PlayerStance, playerRoll, playerConsumed, true)
	npcScore := socialActionScore(state, char, npc, npcAction, state.NPCStance, npcRoll, npcConsumed, false)

	playerDamage := 0
	npcDamage := 0
	if diff := playerScore - npcScore; diff > 0 {
		npcDamage = 1 + diff/3
	} else if diff < 0 {
		playerDamage = 1 + (-diff)/3
	}

	tempoDelta := clampRange((playerScore-npcScore)/3, -2, 2)
	state.Tempo = clampRange(state.Tempo+tempoDelta, -5, 5)
	state.PlayerComposure = maxInt(0, state.PlayerComposure-playerDamage)
	state.NPCComposure = maxInt(0, state.NPCComposure-npcDamage)

	if playerAction == SocialActionConcede {
		state.PlayerPatience++
		state.Tempo = clampRange(state.Tempo-1, -5, 5)
	}
	if npcAction == SocialActionConcede {
		state.NPCPatience++
		state.Tempo = clampRange(state.Tempo+1, -5, 5)
	}
	if playerDamage > 0 && socialActionCostsPatience(playerAction) {
		state.PlayerPatience = maxInt(0, state.PlayerPatience-1)
	}
	if npcDamage > 0 && socialActionCostsPatience(npcAction) {
		state.NPCPatience = maxInt(0, state.NPCPatience-1)
	}

	result := &SocialRoundResult{
		PlayerAction: playerAction,
		NPCAction:    npcAction,
		PlayerRoll:   playerRoll,
		NPCRoll:      npcRoll,
		PlayerScore:  playerScore,
		NPCScore:     npcScore,
		PlayerDamage: playerDamage,
		NPCDamage:    npcDamage,
		TempoDelta:   tempoDelta,
		ExchangeLabel: fmt.Sprintf(
			"%s vs %s",
			strings.Title(string(playerAction)),
			strings.Title(string(npcAction)),
		),
	}
	if playerConsumed != nil {
		result.Consumed = playerConsumed
	}

	switch {
	case state.NPCComposure <= 0 || state.NPCPatience <= 0:
		state.Status = SocialDuelResolved
		state.Winner = char.Name
		state.Outcome = "objective_secured"
		result.Resolved = true
		result.Winner = state.Winner
		result.Outcome = state.Outcome
	case state.PlayerComposure <= 0 || state.PlayerPatience <= 0:
		state.Status = SocialDuelResolved
		state.Winner = npc.Name
		state.Outcome = "fail_forward"
		result.Resolved = true
		result.Winner = state.Winner
		result.Outcome = state.Outcome
		result.FailForward = socialFailForwardFromLoss(playerAction, state, npc)
	default:
		state.Round++
	}

	state.LastExchangeNote = result.ExchangeLabel
	_ = npcConsumed
	return result, nil
}

func normalizeSocialStance(stance SocialStance, fallback SocialStance) SocialStance {
	switch stance {
	case SocialStanceMeasured, SocialStanceBold, SocialStanceGuarded:
		return stance
	default:
		return fallback
	}
}

func normalizeSocialAction(action SocialAction) SocialAction {
	switch action {
	case SocialActionAppeal, SocialActionPressure, SocialActionDeceive, SocialActionConcede, SocialActionExpose, SocialActionWithdraw, SocialActionEscalate:
		return action
	default:
		return SocialActionAppeal
	}
}

func normalizeSocialLeverage(items []SocialLeverage) []SocialLeverage {
	out := make([]SocialLeverage, 0, len(items))
	for _, item := range items {
		item.Label = strings.TrimSpace(item.Label)
		item.Kind = strings.TrimSpace(item.Kind)
		if item.Label == "" {
			continue
		}
		if item.Strength <= 0 {
			item.Strength = 1
		}
		out = append(out, item)
	}
	return out
}

func consumeSocialLeverage(pool *[]SocialLeverage, action SocialAction) *SocialLeverage {
	if pool == nil || action != SocialActionExpose {
		return nil
	}
	bestIdx := -1
	bestStrength := 0
	for i, item := range *pool {
		if item.Spent {
			continue
		}
		if item.Strength > bestStrength {
			bestIdx = i
			bestStrength = item.Strength
		}
	}
	if bestIdx < 0 {
		return nil
	}
	item := (*pool)[bestIdx]
	if item.Consumable {
		(*pool)[bestIdx].Spent = true
	}
	return &item
}

func socialActionScore(state *SocialDuelState, char *storage.Character, npc *storage.NPC, action SocialAction, stance SocialStance, roll int, leverage *SocialLeverage, playerSide bool) int {
	axes := loadRelationshipAxes(npc)
	base := 2 + roll/5
	base += socialStanceBonus(action, stance)
	base += state.Tempo
	if !playerSide {
		base -= state.Tempo
	}

	switch action {
	case SocialActionAppeal:
		base += socialBestStat(char, "cha", "presence", "emp", "heart") / 3
		base += clampSocialBonus((axes.Trust + axes.Respect - axes.Fear) / 25)
	case SocialActionPressure:
		base += socialBestStat(char, "wil", "will", "str", "cha") / 3
		base += clampSocialBonus((axes.Fear + axes.Debt) / 25)
	case SocialActionDeceive:
		base += socialBestStat(char, "int", "cun", "per", "dex") / 3
		base += clampSocialBonus((axes.Trust + axes.Intimacy) / 30)
	case SocialActionConcede:
		base += 1 + socialBestStat(char, "wil", "heart", "cha")/4
	case SocialActionExpose:
		base += socialBestStat(char, "int", "per", "reputation", "cha") / 3
		if leverage != nil {
			base += leverage.Strength * 2
		}
	case SocialActionEscalate:
		base += socialBestStat(char, "wil", "str", "cha")/3 + 2
		base += clampSocialBonus((axes.Fear - axes.Trust) / 25)
	}

	if action == SocialActionExpose && leverage == nil {
		base -= 2
	}
	if playerSide {
		return maxInt(1, base)
	}
	return maxInt(1, base+clampSocialBonus(npc.Disposition/25))
}

func socialBestStat(char *storage.Character, keys ...string) int {
	if char == nil {
		return 0
	}
	best := 0
	for _, key := range keys {
		for _, path := range []string{"attributes." + key, "secondary." + key} {
			value, err := getStatValue(char.StatsJSON, path)
			if err != nil {
				continue
			}
			if int(value) > best {
				best = int(value)
			}
		}
	}
	return best
}

func socialStanceBonus(action SocialAction, stance SocialStance) int {
	switch stance {
	case SocialStanceBold:
		switch action {
		case SocialActionPressure, SocialActionEscalate, SocialActionExpose:
			return 2
		case SocialActionConcede:
			return -1
		}
	case SocialStanceGuarded:
		switch action {
		case SocialActionAppeal, SocialActionConcede:
			return 1
		case SocialActionEscalate:
			return -1
		}
	default:
		switch action {
		case SocialActionAppeal, SocialActionDeceive:
			return 1
		}
	}
	return 0
}

func socialActionCostsPatience(action SocialAction) bool {
	switch action {
	case SocialActionPressure, SocialActionDeceive, SocialActionExpose, SocialActionEscalate:
		return true
	default:
		return false
	}
}

func socialFailForwardFromLoss(action SocialAction, state *SocialDuelState, npc *storage.NPC) *SocialFailForward {
	switch action {
	case SocialActionPressure, SocialActionEscalate:
		return &SocialFailForward{
			Kind:   "suspicion",
			Title:  "The room turns against you",
			Detail: fmt.Sprintf("%s gains the upper hand and your push leaves visible heat behind.", npc.Name),
		}
	case SocialActionDeceive, SocialActionExpose:
		return &SocialFailForward{
			Kind:   "exposure",
			Title:  "Your angle is exposed",
			Detail: "The confrontation stays live, but your leverage is compromised and the other side now knows where to press.",
		}
	default:
		return &SocialFailForward{
			Kind:   "concession",
			Title:  "You have to give ground",
			Detail: "You can keep playing the scene, but not on your original terms.",
		}
	}
}

func (e *SocialDuelEngine) roll() int {
	if e != nil && e.rollFn != nil {
		return e.rollFn()
	}
	return RollD20()
}

func clampSocialBonus(value int) int {
	switch {
	case value > 3:
		return 3
	case value < -3:
		return -3
	default:
		return value
	}
}

func clampRange(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
