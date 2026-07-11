package audio

import (
	"testing"

	"github.com/crimsab/oneday/internal/storage"
)

func TestSegmentCommittedMessagePreservesDialogueOrderAndSpeakerIDs(t *testing.T) {
	message := storage.ChatMessage{
		Role: "assistant", SourceCommitID: "commit-1",
		Content:      "Rain needles the glass. Keep your voice down. The seal answers with a low hum.",
		MetadataJSON: `{"generation":{"run_id":"run-1","trace_id":"trace-1"},"output":{"dialogue_blocks":[{"speaker_id":"npc-lyanna","speaker":"Lyanna","role":"npc","text":"Keep your voice down."}]}}`,
	}
	segments, lineage := SegmentCommittedMessage(message)
	if len(segments) != 3 {
		t.Fatalf("segments=%+v", segments)
	}
	if segments[0].Kind != "narrator" || segments[1].SpeakerEntityID != "npc-lyanna" || segments[2].Kind != "narrator" {
		t.Fatalf("ordered segmentation lost semantics: %+v", segments)
	}
	if lineage.RunID != "run-1" || lineage.TraceID != "trace-1" {
		t.Fatalf("lineage=%+v", lineage)
	}
}

func TestSegmentCommittedMessageRejectsProvisionalText(t *testing.T) {
	segments, _ := SegmentCommittedMessage(storage.ChatMessage{Role: "assistant", Content: "Not committed"})
	if len(segments) != 0 {
		t.Fatalf("provisional message produced segments: %+v", segments)
	}
}

func TestSpeechChunksStayWithinProviderLimit(t *testing.T) {
	text := ""
	for index := 0; index < 900; index++ {
		text += "Una frase multilingue. "
	}
	for _, chunk := range splitSpeechChunks(text) {
		if len(chunk) > 3500 {
			t.Fatalf("chunk length=%d", len(chunk))
		}
	}
}
