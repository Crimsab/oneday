package engine

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/crimsab/oneday/internal/storage"
)

const (
	NPCStageRumor       = "rumor"
	NPCStageObserved    = "observed"
	NPCStageIdentified  = "identified"
	NPCStageEstablished = "established"
	NPCStageDismissed   = "dismissed"

	NPCVisualNone       = "none"
	NPCVisualSilhouette = "silhouette"
	NPCVisualDraft      = "draft"
	NPCVisualCanonical  = "canonical"
)

type NPCDiscovery struct {
	Stage               string                 `json:"stage"`
	PublicLabel         string                 `json:"public_label,omitempty"`
	Aliases             []string               `json:"aliases,omitempty"`
	Source              string                 `json:"source,omitempty"`
	Confidence          string                 `json:"confidence,omitempty"`
	ProfileCompleteness int                    `json:"profile_completeness"`
	VisualCompleteness  int                    `json:"visual_completeness"`
	VisualReadiness     string                 `json:"visual_readiness"`
	FirstMentionedTurn  int                    `json:"first_mentioned_turn,omitempty"`
	FirstObservedTurn   int                    `json:"first_observed_turn,omitempty"`
	IdentifiedTurn      int                    `json:"identified_turn,omitempty"`
	EstablishedTurn     int                    `json:"established_turn,omitempty"`
	LastEvidenceTurn    int                    `json:"last_evidence_turn,omitempty"`
	FieldFacts          map[string]NPCFact     `json:"field_facts,omitempty"`
	VisualFacts         NPCVisualFacts         `json:"visual_facts,omitempty"`
	Contradictions      []NPCContradiction     `json:"contradictions,omitempty"`
	ManualProfileLock   bool                   `json:"manual_profile_lock,omitempty"`
	ManualVisualLock    bool                   `json:"manual_visual_lock,omitempty"`
	VisualFingerprint   string                 `json:"visual_fingerprint,omitempty"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
}

type NPCFact struct {
	Value      string `json:"value"`
	Confidence string `json:"confidence,omitempty"`
	Source     string `json:"source,omitempty"`
	Turn       int    `json:"turn,omitempty"`
	Public     bool   `json:"public"`
	Confirmed  bool   `json:"confirmed,omitempty"`
	Locked     bool   `json:"locked,omitempty"`
}

type NPCVisualFacts struct {
	Silhouette        string   `json:"silhouette,omitempty"`
	ApparentAge       string   `json:"apparent_age,omitempty"`
	Build             string   `json:"build,omitempty"`
	Face              string   `json:"face,omitempty"`
	Hair              string   `json:"hair,omitempty"`
	Clothing          string   `json:"clothing,omitempty"`
	Distinguishing    []string `json:"distinguishing,omitempty"`
	Palette           []string `json:"palette,omitempty"`
	NonVisualIdentity []string `json:"non_visual_identity,omitempty"`
}

type NPCContradiction struct {
	Field string `json:"field"`
	Old   string `json:"old"`
	New   string `json:"new"`
	Turn  int    `json:"turn,omitempty"`
}

type npcDiscoveryMerge struct {
	Changed  bool
	Promoted bool
}

var confidenceRank = map[string]int{
	"inferred":  0,
	"rumored":   1,
	"observed":  2,
	"confirmed": 3,
	"manual":    4,
}

var stageRank = map[string]int{
	NPCStageDismissed:   0,
	NPCStageRumor:       1,
	NPCStageObserved:    2,
	NPCStageIdentified:  3,
	NPCStageEstablished: 4,
}

func npcDiscoveryFromStorage(npc *storage.NPC) NPCDiscovery {
	if npc == nil {
		return NPCDiscovery{}
	}
	var discovery NPCDiscovery
	if raw := strings.TrimSpace(npc.DiscoveryJSON); raw != "" && raw != "{}" && raw != "null" {
		_ = json.Unmarshal([]byte(raw), &discovery)
	}
	if discovery.Stage == "" {
		discovery = inferredDiscoveryForNPC(npc)
	}
	return normalizeNPCDiscovery(discovery, npc.Name, npc.FirstAppearedTurn)
}

func setNPCDiscovery(npc *storage.NPC, discovery NPCDiscovery) {
	if npc == nil {
		return
	}
	discovery = normalizeNPCDiscovery(discovery, npc.Name, npc.FirstAppearedTurn)
	discovery.VisualFingerprint = npcVisualFingerprintFromDiscovery(npc, discovery)
	bytes, err := json.Marshal(discovery)
	if err != nil {
		npc.DiscoveryJSON = "{}"
		return
	}
	npc.DiscoveryJSON = string(bytes)
}

func inferredDiscoveryForNPC(npc *storage.NPC) NPCDiscovery {
	discovery := NPCDiscovery{
		Stage:               NPCStageIdentified,
		PublicLabel:         npc.Name,
		Source:              "inference",
		Confidence:          "inferred",
		ProfileCompleteness: 25,
		VisualCompleteness:  0,
		VisualReadiness:     NPCVisualNone,
		FirstMentionedTurn:  npc.FirstAppearedTurn,
		LastEvidenceTurn:    npc.LastSeenTurn,
	}
	if !isPlaceholderNPC(npc) {
		discovery.Stage = NPCStageEstablished
		discovery.Source = "migration"
		discovery.Confidence = "confirmed"
		discovery.ProfileCompleteness = profileCompletenessForNPC(npc)
		discovery.VisualCompleteness = visualCompletenessForNPC(npc, NPCVisualFacts{})
		if discovery.VisualCompleteness >= 65 {
			discovery.VisualReadiness = NPCVisualCanonical
		}
	}
	return discovery
}

func normalizeNPCDiscovery(discovery NPCDiscovery, fallbackName string, currentTurn int) NPCDiscovery {
	discovery.Stage = normalizedStage(discovery.Stage, NPCStageIdentified)
	discovery.Confidence = normalizedConfidence(discovery.Confidence, "inferred")
	discovery.VisualReadiness = normalizedVisualReadiness(discovery.VisualReadiness)
	if discovery.PublicLabel == "" {
		discovery.PublicLabel = fallbackName
	}
	if discovery.FirstMentionedTurn == 0 {
		discovery.FirstMentionedTurn = currentTurn
	}
	if discovery.LastEvidenceTurn == 0 {
		discovery.LastEvidenceTurn = currentTurn
	}
	discovery.ProfileCompleteness = clampPercent(discovery.ProfileCompleteness)
	discovery.VisualCompleteness = clampPercent(discovery.VisualCompleteness)
	discovery.Aliases = cleanUniqueStrings(discovery.Aliases)
	return discovery
}

func discoveryForNewNPCData(data *NPCData, turn int) NPCDiscovery {
	discovery := data.Discovery
	if discovery.Stage == "" {
		discovery.Stage = NPCStageEstablished
	}
	if discovery.Confidence == "" {
		discovery.Confidence = "confirmed"
	}
	if discovery.Source == "" {
		discovery.Source = "narrator"
	}
	if discovery.ProfileCompleteness == 0 {
		discovery.ProfileCompleteness = profileCompletenessForData(data)
	}
	if discovery.VisualCompleteness == 0 {
		discovery.VisualCompleteness = visualCompletenessForAppearanceAndFacts(data.Appearance, discovery.VisualFacts)
	}
	if discovery.VisualReadiness == "" {
		if discovery.Stage == NPCStageEstablished && discovery.VisualCompleteness >= 65 && !isPlaceholderAppearance(data.Appearance) {
			discovery.VisualReadiness = NPCVisualCanonical
		} else if (discovery.Stage == NPCStageObserved || discovery.Stage == NPCStageIdentified) && discovery.VisualCompleteness >= 45 {
			discovery.VisualReadiness = NPCVisualDraft
		} else {
			discovery.VisualReadiness = NPCVisualNone
		}
	}
	discovery.Aliases = append(discovery.Aliases, data.Aliases...)
	discovery.PublicLabel = firstNonEmpty(discovery.PublicLabel, data.Name)
	discovery.LastEvidenceTurn = turn
	if discovery.Stage == NPCStageEstablished {
		discovery.EstablishedTurn = firstNonZero(discovery.EstablishedTurn, turn)
	}
	if discovery.Stage == NPCStageIdentified || discovery.Stage == NPCStageEstablished {
		discovery.IdentifiedTurn = firstNonZero(discovery.IdentifiedTurn, turn)
	}
	return normalizeNPCDiscovery(discovery, data.Name, turn)
}

func MergeNPCProfile(npc *storage.NPC, data *NPCData, turn int) npcDiscoveryMerge {
	if npc == nil || data == nil {
		return npcDiscoveryMerge{}
	}
	before := npcDiscoveryFromStorage(npc)
	after := before
	incoming := discoveryForNewNPCData(data, turn)
	changed := false

	if rankStage(incoming.Stage) > rankStage(after.Stage) {
		after.Stage = incoming.Stage
		changed = true
	}
	if rankConfidence(incoming.Confidence) >= rankConfidence(after.Confidence) {
		after.Confidence = incoming.Confidence
	}
	after.Source = firstNonEmpty(incoming.Source, after.Source)
	after.PublicLabel = firstNonEmpty(incoming.PublicLabel, after.PublicLabel, data.Name)
	after.Aliases = cleanUniqueStrings(append(append(after.Aliases, incoming.Aliases...), data.Aliases...))
	after.ProfileCompleteness = maxInt(after.ProfileCompleteness, incoming.ProfileCompleteness)
	after.VisualCompleteness = maxInt(after.VisualCompleteness, incoming.VisualCompleteness)
	after.VisualReadiness = strongerVisualReadiness(after.VisualReadiness, incoming.VisualReadiness)
	after.VisualFacts = mergeVisualFacts(after.VisualFacts, incoming.VisualFacts)
	after.LastEvidenceTurn = turn
	if after.Stage == NPCStageEstablished && after.EstablishedTurn == 0 {
		after.EstablishedTurn = turn
	}
	if rankStage(after.Stage) >= rankStage(NPCStageIdentified) && after.IdentifiedTurn == 0 {
		after.IdentifiedTurn = turn
	}

	if updateNPCStringField(&npc.Name, data.Name) {
		changed = true
	}
	if shouldReplaceNPCRole(npc.Role, data.Role) {
		npc.Role = strings.TrimSpace(data.Role)
		changed = true
	}
	if shouldReplaceNPCField(npc.Appearance, data.Appearance) {
		npc.Appearance = strings.TrimSpace(data.Appearance)
		changed = true
	}
	if !isZeroPersonality(data.Personality) {
		bytes, err := json.Marshal(data.Personality)
		if err == nil && string(bytes) != npc.PersonalityJSON {
			npc.PersonalityJSON = string(bytes)
			changed = true
		}
	}
	if len(data.PrivateThoughts) > 0 {
		npc.PrivateThoughts = mergeJSONStringSlice(npc.PrivateThoughts, data.PrivateThoughts)
		changed = true
	}
	if len(data.Desires) > 0 {
		bytes, err := json.Marshal(data.Desires)
		if err == nil && string(bytes) != npc.Desires {
			npc.Desires = string(bytes)
			changed = true
		}
	}
	if data.Disposition != 0 && data.Disposition != npc.Disposition {
		npc.Disposition = data.Disposition
		changed = true
	}
	if data.CanHelp && data.CanHelp != npc.CanHelp {
		npc.CanHelp = data.CanHelp
		changed = true
	}

	setNPCDiscovery(npc, after)
	return npcDiscoveryMerge{
		Changed:  changed || npc.DiscoveryJSON != "",
		Promoted: rankStage(after.Stage) > rankStage(before.Stage),
	}
}

func applyNPCReference(npc *storage.NPC, ref map[string]interface{}, turn int) bool {
	if npc == nil || ref == nil {
		return false
	}
	discovery := npcDiscoveryFromStorage(npc)
	before := discovery
	stage := stringValue(ref["stage"])
	if stage != "" {
		nextStage := normalizedStage(stage, discovery.Stage)
		if rankStage(nextStage) > rankStage(discovery.Stage) || discovery.Source == "inference" || discovery.Confidence == "inferred" {
			discovery.Stage = nextStage
		}
	}
	confidence := stringValue(ref["confidence"])
	if confidence != "" && rankConfidence(confidence) >= rankConfidence(discovery.Confidence) {
		discovery.Confidence = normalizedConfidence(confidence, discovery.Confidence)
	}
	discovery.Source = firstNonEmpty(stringValue(ref["source"]), discovery.Source)
	discovery.PublicLabel = firstNonEmpty(stringValue(ref["public_label"]), discovery.PublicLabel, npc.Name)
	discovery.Aliases = cleanUniqueStrings(append(discovery.Aliases, toStringSlice(ref["aliases"])...))
	if facts, ok := ref["visual_facts"].(map[string]interface{}); ok {
		discovery.VisualFacts = mergeVisualFacts(discovery.VisualFacts, parseNPCVisualFacts(facts))
	}
	discovery.VisualCompleteness = maxInt(discovery.VisualCompleteness, visualCompletenessForNPC(npc, discovery.VisualFacts))
	if discovery.VisualReadiness == NPCVisualNone && (discovery.Stage == NPCStageObserved || discovery.Stage == NPCStageIdentified) && discovery.VisualCompleteness >= 45 {
		discovery.VisualReadiness = NPCVisualDraft
	}
	discovery.LastEvidenceTurn = turn
	if discovery.Stage == NPCStageObserved && discovery.FirstObservedTurn == 0 {
		discovery.FirstObservedTurn = turn
	}
	setNPCDiscovery(npc, discovery)
	return discovery.Stage != before.Stage ||
		discovery.PublicLabel != before.PublicLabel ||
		discovery.VisualCompleteness != before.VisualCompleteness ||
		discovery.VisualReadiness != before.VisualReadiness
}

func applyNPCDiscoveryUpdate(npc *storage.NPC, update map[string]interface{}, turn int) bool {
	if npc == nil || update == nil {
		return false
	}
	discovery := npcDiscoveryFromStorage(npc)
	beforeStage := discovery.Stage
	stage := normalizedStage(stringValue(update["stage"]), discovery.Stage)
	if rankStage(stage) > rankStage(discovery.Stage) {
		discovery.Stage = stage
	}
	confidence := normalizedConfidence(stringValue(update["confidence"]), discovery.Confidence)
	if rankConfidence(confidence) >= rankConfidence(discovery.Confidence) {
		discovery.Confidence = confidence
	}
	discovery.Aliases = cleanUniqueStrings(append(discovery.Aliases, toStringSlice(update["aliases"])...))
	if canonical := strings.TrimSpace(stringValue(update["canonical_name"])); canonical != "" {
		if !strings.EqualFold(canonical, npc.Name) {
			discovery.Aliases = cleanUniqueStrings(append(discovery.Aliases, npc.Name))
			npc.Name = canonical
		}
		discovery.PublicLabel = canonical
	}
	for _, fact := range toObjectMaps(update["facts"]) {
		field := strings.TrimSpace(stringValue(fact["field"]))
		value := strings.TrimSpace(stringValue(fact["value"]))
		if field == "" || value == "" {
			continue
		}
		switch field {
		case "name":
			if !strings.EqualFold(npc.Name, value) {
				discovery.Aliases = cleanUniqueStrings(append(discovery.Aliases, npc.Name))
				npc.Name = value
			}
		case "role":
			if shouldReplaceNPCRole(npc.Role, value) {
				npc.Role = value
			}
		case "appearance":
			if shouldReplaceNPCField(npc.Appearance, value) {
				npc.Appearance = value
			}
		}
		if discovery.FieldFacts == nil {
			discovery.FieldFacts = map[string]NPCFact{}
		}
		factConfidence := normalizedConfidence(stringValue(fact["confidence"]), confidence)
		discovery.FieldFacts[field] = NPCFact{
			Value:      value,
			Confidence: factConfidence,
			Source:     firstNonEmpty(stringValue(fact["source"]), "narrator"),
			Turn:       turn,
			Public:     boolValueWithDefault(fact["public"], true),
			Confirmed:  factConfidence == "confirmed",
		}
	}
	if facts, ok := update["visual_facts"].(map[string]interface{}); ok {
		discovery.VisualFacts = mergeVisualFacts(discovery.VisualFacts, parseNPCVisualFacts(facts))
	}
	discovery.ProfileCompleteness = maxInt(discovery.ProfileCompleteness, profileCompletenessForNPC(npc))
	discovery.VisualCompleteness = maxInt(discovery.VisualCompleteness, visualCompletenessForNPC(npc, discovery.VisualFacts))
	if discovery.VisualReadiness == NPCVisualNone {
		if discovery.Stage == NPCStageEstablished && discovery.VisualCompleteness >= 65 {
			discovery.VisualReadiness = NPCVisualCanonical
		} else if rankStage(discovery.Stage) >= rankStage(NPCStageObserved) && discovery.VisualCompleteness >= 45 {
			discovery.VisualReadiness = NPCVisualDraft
		}
	}
	discovery.LastEvidenceTurn = turn
	setNPCDiscovery(npc, discovery)
	return rankStage(discovery.Stage) > rankStage(beforeStage)
}

func isPlaceholderNPC(npc *storage.NPC) bool {
	if npc == nil {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(npc.Role), "person of interest") ||
		isPlaceholderAppearance(npc.Appearance) ||
		strings.Contains(strings.ToLower(npc.PersonalityJSON), `"unknown"`)
}

func isPlaceholderAppearance(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "" ||
		strings.Contains(value, "unidentified figure") ||
		strings.Contains(value, "derive from story context") ||
		strings.Contains(value, "not established")
}

func shouldReplaceNPCField(existing, incoming string) bool {
	incoming = strings.TrimSpace(incoming)
	if incoming == "" {
		return false
	}
	existing = strings.TrimSpace(existing)
	return existing == "" || isPlaceholderAppearance(existing) || strings.EqualFold(existing, "unknown")
}

func shouldReplaceNPCRole(existing, incoming string) bool {
	incoming = strings.TrimSpace(incoming)
	if incoming == "" {
		return false
	}
	existing = strings.TrimSpace(existing)
	return existing == "" ||
		strings.EqualFold(existing, "unknown") ||
		strings.EqualFold(existing, "person of interest")
}

func updateNPCStringField(target *string, incoming string) bool {
	incoming = strings.TrimSpace(incoming)
	if target == nil || incoming == "" || strings.EqualFold(strings.TrimSpace(*target), incoming) {
		return false
	}
	if strings.TrimSpace(*target) == "" || strings.EqualFold(strings.TrimSpace(*target), "unknown") {
		*target = incoming
		return true
	}
	return false
}

func profileCompletenessForData(data *NPCData) int {
	if data == nil {
		return 0
	}
	score := 0
	if data.Name != "" {
		score += 15
	}
	if data.Role != "" {
		score += 15
	}
	if !isPlaceholderAppearance(data.Appearance) {
		score += 20
	}
	if len(data.Personality.Traits) > 0 {
		score += 15
	}
	if data.Personality.SpeechStyle != "" {
		score += 10
	}
	if len(data.Desires) > 0 {
		score += 10
	}
	if len(data.PrivateThoughts) > 0 {
		score += 5
	}
	if data.CanHelp {
		score += 5
	}
	return clampPercent(score)
}

func profileCompletenessForNPC(npc *storage.NPC) int {
	if npc == nil {
		return 0
	}
	score := 0
	if npc.Name != "" {
		score += 15
	}
	if npc.Role != "" && !strings.EqualFold(npc.Role, "person of interest") {
		score += 15
	}
	if !isPlaceholderAppearance(npc.Appearance) {
		score += 20
	}
	if npc.PersonalityJSON != "" && npc.PersonalityJSON != "{}" && !strings.Contains(strings.ToLower(npc.PersonalityJSON), `"unknown"`) {
		score += 20
	}
	if npc.Desires != "" && npc.Desires != "[]" {
		score += 10
	}
	if npc.NotesOnProtagonist != "" && npc.NotesOnProtagonist != "[]" {
		score += 5
	}
	if npc.CanHelp {
		score += 5
	}
	return clampPercent(score)
}

func visualCompletenessForNPC(npc *storage.NPC, facts NPCVisualFacts) int {
	if npc == nil {
		return 0
	}
	return visualCompletenessForAppearanceAndFacts(npc.Appearance, facts)
}

func visualCompletenessForAppearanceAndFacts(appearance string, facts NPCVisualFacts) int {
	score := 0
	if !isPlaceholderAppearance(appearance) {
		score += 65
	}
	anchors := 0
	for _, value := range []string{facts.Silhouette, facts.ApparentAge, facts.Build, facts.Face, facts.Hair, facts.Clothing} {
		if strings.TrimSpace(value) != "" {
			anchors++
		}
	}
	anchors += len(facts.Distinguishing)
	anchors += len(facts.Palette)
	score += anchors * 12
	return clampPercent(score)
}

func parseNPCVisualFacts(raw map[string]interface{}) NPCVisualFacts {
	return NPCVisualFacts{
		Silhouette:        stringValue(raw["silhouette"]),
		ApparentAge:       stringValue(raw["apparent_age"]),
		Build:             stringValue(raw["build"]),
		Face:              stringValue(raw["face"]),
		Hair:              stringValue(raw["hair"]),
		Clothing:          stringValue(raw["clothing"]),
		Distinguishing:    toStringSlice(raw["distinguishing"]),
		Palette:           toStringSlice(raw["palette"]),
		NonVisualIdentity: toStringSlice(raw["non_visual_identity"]),
	}
}

func mergeVisualFacts(existing, incoming NPCVisualFacts) NPCVisualFacts {
	if incoming.Silhouette != "" {
		existing.Silhouette = incoming.Silhouette
	}
	if incoming.ApparentAge != "" {
		existing.ApparentAge = incoming.ApparentAge
	}
	if incoming.Build != "" {
		existing.Build = incoming.Build
	}
	if incoming.Face != "" {
		existing.Face = incoming.Face
	}
	if incoming.Hair != "" {
		existing.Hair = incoming.Hair
	}
	if incoming.Clothing != "" {
		existing.Clothing = incoming.Clothing
	}
	existing.Distinguishing = cleanUniqueStrings(append(existing.Distinguishing, incoming.Distinguishing...))
	existing.Palette = cleanUniqueStrings(append(existing.Palette, incoming.Palette...))
	existing.NonVisualIdentity = cleanUniqueStrings(append(existing.NonVisualIdentity, incoming.NonVisualIdentity...))
	return existing
}

func npcVisualFingerprintFromDiscovery(npc *storage.NPC, discovery NPCDiscovery) string {
	if npc == nil {
		return ""
	}
	parts := []string{
		strings.TrimSpace(npc.Name),
		strings.TrimSpace(npc.Role),
		strings.TrimSpace(npc.Appearance),
		discovery.Stage,
		discovery.VisualReadiness,
		discovery.VisualFacts.Silhouette,
		discovery.VisualFacts.ApparentAge,
		discovery.VisualFacts.Build,
		discovery.VisualFacts.Face,
		discovery.VisualFacts.Hair,
		discovery.VisualFacts.Clothing,
		strings.Join(cleanUniqueStrings(discovery.VisualFacts.Distinguishing), "|"),
		strings.Join(cleanUniqueStrings(discovery.VisualFacts.Palette), "|"),
	}
	return strings.Join(parts, "\n")
}

func normalizedStage(stage, fallback string) string {
	stage = strings.ToLower(strings.TrimSpace(stage))
	if _, ok := stageRank[stage]; ok {
		return stage
	}
	return fallback
}

func normalizedConfidence(confidence, fallback string) string {
	confidence = strings.ToLower(strings.TrimSpace(confidence))
	if _, ok := confidenceRank[confidence]; ok {
		return confidence
	}
	return fallback
}

func normalizedVisualReadiness(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case NPCVisualSilhouette, NPCVisualDraft, NPCVisualCanonical:
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return NPCVisualNone
	}
}

func rankStage(stage string) int {
	return stageRank[normalizedStage(stage, NPCStageIdentified)]
}

func rankConfidence(confidence string) int {
	return confidenceRank[normalizedConfidence(confidence, "inferred")]
}

func strongerVisualReadiness(a, b string) string {
	order := map[string]int{NPCVisualNone: 0, NPCVisualSilhouette: 1, NPCVisualDraft: 2, NPCVisualCanonical: 3}
	a = normalizedVisualReadiness(a)
	b = normalizedVisualReadiness(b)
	if order[b] > order[a] {
		return b
	}
	return a
}

func boolValueWithDefault(value interface{}, fallback bool) bool {
	if b, ok := value.(bool); ok {
		return b
	}
	return fallback
}

func cleanUniqueStrings(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func mergeJSONStringSlice(existing string, incoming []string) string {
	var values []string
	_ = json.Unmarshal([]byte(existing), &values)
	values = cleanUniqueStrings(append(values, incoming...))
	bytes, err := json.Marshal(values)
	if err != nil {
		return existing
	}
	return string(bytes)
}

func clampPercent(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func firstNonZero(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func isZeroPersonality(value NPCPersonality) bool {
	return len(value.Traits) == 0 &&
		value.SpeechStyle == "" &&
		len(value.Quirks) == 0 &&
		len(value.Values) == 0 &&
		len(value.Fears) == 0
}
