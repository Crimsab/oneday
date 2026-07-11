package engine

import "testing"

func TestNormalizeDialogueBlocksPreservesCanonicalSpeakerID(t *testing.T) {
	blocks := normalizeDialogueBlocks([]DialogueBlock{{SpeakerID: " npc-mira ", Speaker: "Mira", Role: "npc", Text: " Speak. "}})
	if len(blocks) != 1 || blocks[0].SpeakerID != "npc-mira" || blocks[0].Text != "Speak." {
		t.Fatalf("normalized blocks=%+v", blocks)
	}
}
