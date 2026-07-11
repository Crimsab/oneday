package assets

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestImageJobDoesNotBlockTurnCommit(t *testing.T) {
	q := NewMemoryImageQueue()
	start := time.Now()
	_, _, err := q.Enqueue(context.Background(), ImageJob{
		StoryID:     "story-1",
		Turn:        7,
		Provider:    "codex-imagegen",
		StylePreset: "ink-noir",
	})
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if time.Since(start) > 50*time.Millisecond {
		t.Fatal("enqueue should be a quick local operation, not image generation")
	}
}

func TestImageJobDedupesByCueHashAndStyle(t *testing.T) {
	q := NewMemoryImageQueue()
	job := ImageJob{
		StoryID:     "story-1",
		Turn:        7,
		Provider:    "codex-imagegen",
		StylePreset: "ink-noir",
		CueHash:     "same-cue",
	}
	first, deduped, err := q.Enqueue(context.Background(), job)
	if err != nil || deduped {
		t.Fatalf("first enqueue = %+v deduped=%v err=%v", first, deduped, err)
	}
	second, deduped, err := q.Enqueue(context.Background(), job)
	if err != nil || !deduped {
		t.Fatalf("second enqueue = %+v deduped=%v err=%v", second, deduped, err)
	}
	if first.ID != second.ID {
		t.Fatalf("deduped job id = %q, want %q", second.ID, first.ID)
	}
}

func TestImageFailureEmitsAssetFailedWithoutRetryStorm(t *testing.T) {
	q := NewMemoryImageQueue()
	job, _, err := q.Enqueue(context.Background(), ImageJob{StoryID: "story-1", Turn: 7})
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	claimed, err := q.Claim(context.Background())
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if claimed == nil || claimed.ID != job.ID {
		t.Fatalf("claimed = %+v, want %s", claimed, job.ID)
	}
	if err := q.Fail(context.Background(), job.ID, errors.New("provider unavailable"), false); err != nil {
		t.Fatalf("Fail: %v", err)
	}
	claimed, err = q.Claim(context.Background())
	if err != nil {
		t.Fatalf("Claim after fail: %v", err)
	}
	if claimed != nil {
		t.Fatalf("failed non-retryable job should not be claimed again: %+v", claimed)
	}
}

func TestImageWorkerGeneratesAndStoresAsset(t *testing.T) {
	q := NewMemoryImageQueue()
	job, _, err := q.Enqueue(context.Background(), ImageJob{
		StoryID:  "story:one",
		Turn:     3,
		Provider: "fake-imagegen",
		Cue:      VisualCue{Kind: "scene", Subject: "Harbor at dawn"},
	})
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	worker := ImageWorker{
		Queue:    q,
		Provider: fakeImageProvider{},
		Store:    FileAssetStore{Root: t.TempDir()},
	}
	worked, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if !worked {
		t.Fatal("RunOnce worked = false, want true")
	}

	claimed, err := q.Claim(context.Background())
	if err != nil {
		t.Fatalf("Claim after worker: %v", err)
	}
	if claimed != nil {
		t.Fatalf("done job should not be claimable: %+v", claimed)
	}
	q.mu.Lock()
	stored := q.jobs[q.byJobID[job.ID]]
	q.mu.Unlock()
	if stored.Status != ImageJobDone || stored.AssetID == "" {
		t.Fatalf("stored job = %+v, want done with asset id", stored)
	}
	files, err := os.ReadDir(filepath.Join(worker.Store.(FileAssetStore).Root, "story-one", "images"))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("stored files = %d, want 1", len(files))
	}
}

func TestCommandImageProviderReturnsCommandBytes(t *testing.T) {
	provider := CommandImageProvider{
		Command:   "sh",
		Args:      []string{"-c", "printf command-image-bytes"},
		Extension: "webp",
	}
	data, ext, err := provider.Generate(context.Background(), ImageJob{
		ID:      "job-1",
		StoryID: "story-1",
		Cue:     VisualCue{Subject: "Harbor"},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if string(data) != "command-image-bytes" {
		t.Fatalf("data = %q, want command bytes", string(data))
	}
	if ext != "webp" {
		t.Fatalf("extension = %q, want webp", ext)
	}
}

type fakeImageProvider struct{}

func (fakeImageProvider) Name() string { return "fake-imagegen" }

func (fakeImageProvider) Generate(context.Context, ImageJob) ([]byte, string, error) {
	return []byte("fake-png-bytes"), "png", nil
}
