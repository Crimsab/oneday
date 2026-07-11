package audio

import (
	"encoding/json"
	"sort"
	"strings"
	"unicode"

	"github.com/crimsab/oneday/internal/storage"
)

type Segment struct {
	Index           int    `json:"index"`
	Kind            string `json:"kind"`
	SpeakerEntityID string `json:"speaker_entity_id,omitempty"`
	Speaker         string `json:"speaker,omitempty"`
	Role            string `json:"role,omitempty"`
	LanguageTag     string `json:"language_tag,omitempty"`
	Text            string `json:"text"`
}

type MessageGenerationLineage struct {
	RunID   string
	TraceID string
}

type dialogueBlock struct {
	SpeakerID   string `json:"speaker_id"`
	Speaker     string `json:"speaker"`
	Role        string `json:"role"`
	LanguageTag string `json:"language_tag"`
	Text        string `json:"text"`
}

type messageMetadata struct {
	Output struct {
		DialogueBlocks []dialogueBlock `json:"dialogue_blocks"`
	} `json:"output"`
	Generation struct {
		RunID   string `json:"run_id"`
		TraceID string `json:"trace_id"`
	} `json:"generation"`
}

func SegmentCommittedMessage(message storage.ChatMessage) ([]Segment, MessageGenerationLineage) {
	if message.Role != "assistant" || strings.TrimSpace(message.SourceCommitID) == "" {
		return nil, MessageGenerationLineage{}
	}
	var metadata messageMetadata
	_ = json.Unmarshal([]byte(message.MetadataJSON), &metadata)
	lineage := MessageGenerationLineage{RunID: metadata.Generation.RunID, TraceID: metadata.Generation.TraceID}
	narrative := normalizeSpeechText(message.Content)
	blocks := make([]dialogueBlock, 0, len(metadata.Output.DialogueBlocks))
	for _, block := range metadata.Output.DialogueBlocks {
		block.Text = normalizeSpeechText(block.Text)
		if block.Text != "" && block.Role != "meta" && block.Role != "system" {
			blocks = append(blocks, block)
		}
	}
	if len(blocks) == 0 {
		return indexedNarration(splitSpeechChunks(narrative)), lineage
	}

	type occurrence struct {
		start int
		end   int
		block dialogueBlock
	}
	occurrences := make([]occurrence, 0, len(blocks))
	unmatched := make([]dialogueBlock, 0)
	searchFrom := 0
	for _, block := range blocks {
		relative := strings.Index(narrative[searchFrom:], block.Text)
		if relative < 0 {
			unmatched = append(unmatched, block)
			continue
		}
		start := searchFrom + relative
		occurrences = append(occurrences, occurrence{start: start, end: start + len(block.Text), block: block})
		searchFrom = start + len(block.Text)
	}
	sort.SliceStable(occurrences, func(i, j int) bool { return occurrences[i].start < occurrences[j].start })
	segments := []Segment{}
	cursor := 0
	appendNarration := func(text string) {
		for _, chunk := range splitSpeechChunks(text) {
			segments = append(segments, Segment{Kind: "narrator", Role: "narrator", Text: chunk})
		}
	}
	for _, found := range occurrences {
		if found.start > cursor {
			appendNarration(narrative[cursor:found.start])
		}
		segments = append(segments, Segment{Kind: "dialogue", SpeakerEntityID: strings.TrimSpace(found.block.SpeakerID), Speaker: strings.TrimSpace(found.block.Speaker), Role: normalizedDialogueRole(found.block.Role), LanguageTag: normalizeLanguageTag(found.block.LanguageTag), Text: found.block.Text})
		cursor = found.end
	}
	if cursor < len(narrative) {
		appendNarration(narrative[cursor:])
	}
	for _, block := range unmatched {
		segments = append(segments, Segment{Kind: "dialogue", SpeakerEntityID: strings.TrimSpace(block.SpeakerID), Speaker: strings.TrimSpace(block.Speaker), Role: normalizedDialogueRole(block.Role), LanguageTag: normalizeLanguageTag(block.LanguageTag), Text: block.Text})
	}
	for index := range segments {
		segments[index].Index = index
	}
	return segments, lineage
}

func indexedNarration(chunks []string) []Segment {
	segments := make([]Segment, 0, len(chunks))
	for index, chunk := range chunks {
		segments = append(segments, Segment{Index: index, Kind: "narrator", Role: "narrator", Text: chunk})
	}
	return segments
}

func splitSpeechChunks(text string) []string {
	text = normalizeSpeechText(text)
	if text == "" {
		return nil
	}
	const maxInput = 3500
	chunks := []string{}
	start := 0
	for start < len(text) {
		end := start + maxInput
		if end >= len(text) {
			chunks = append(chunks, strings.TrimSpace(text[start:]))
			break
		}
		cut := -1
		for index := end; index > start+maxInput/2; index-- {
			if unicode.IsSpace(rune(text[index])) && strings.ContainsRune(".!?;:", rune(text[index-1])) {
				cut = index
				break
			}
		}
		if cut < 0 {
			cut = strings.LastIndex(text[start:end], " ")
			if cut > 0 {
				cut += start
			} else {
				cut = end
				for cut > start && text[cut]&0xc0 == 0x80 {
					cut--
				}
			}
		}
		chunks = append(chunks, strings.TrimSpace(text[start:cut]))
		start = cut
	}
	return chunks
}

func normalizeSpeechText(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func normalizedDialogueRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "player", "protagonist":
		return "protagonist"
	default:
		return "npc"
	}
}

func normalizeLanguageTag(tag string) string {
	tag = strings.TrimSpace(strings.ReplaceAll(tag, "_", "-"))
	if tag == "" {
		return ""
	}
	parts := strings.Split(tag, "-")
	parts[0] = strings.ToLower(parts[0])
	for index := 1; index < len(parts); index++ {
		if len(parts[index]) == 2 {
			parts[index] = strings.ToUpper(parts[index])
		} else {
			parts[index] = strings.ToLower(parts[index])
		}
	}
	return strings.Join(parts, "-")
}
