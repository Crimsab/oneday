package engine

import "testing"

func TestGoldenMiniGameEvalCorpusPassesFairnessAndReplayGates(t *testing.T) {
	corpus, err := LoadMiniGameEvalCorpus("testdata/minigame-evals.json")
	if err != nil {
		t.Fatal(err)
	}
	report := RunMiniGameEvalCorpus(*corpus)
	if !report.Passed() {
		t.Fatalf("eval report failed: %+v", report)
	}
	if report.SuccessRate != 0.5 {
		t.Fatalf("success rate = %.2f, want 0.50", report.SuccessRate)
	}
}
