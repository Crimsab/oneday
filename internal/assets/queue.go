package assets

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync"
	"time"
)

type ImageJobStatus string

const (
	ImageJobQueued    ImageJobStatus = "queued"
	ImageJobRunning   ImageJobStatus = "running"
	ImageJobDone      ImageJobStatus = "done"
	ImageJobFailed    ImageJobStatus = "failed"
	ImageJobCancelled ImageJobStatus = "cancelled"
)

type VisualCue struct {
	Kind        string `json:"kind"`
	Subject     string `json:"subject"`
	Mood        string `json:"mood,omitempty"`
	Composition string `json:"composition,omitempty"`
	StylePreset string `json:"style_preset,omitempty"`
	Negative    string `json:"negative,omitempty"`
}

type ImageJob struct {
	ID          string         `json:"id"`
	StoryID     string         `json:"story_id"`
	Turn        int            `json:"turn"`
	SceneID     string         `json:"scene_id,omitempty"`
	Status      ImageJobStatus `json:"status"`
	Provider    string         `json:"provider"`
	StylePreset string         `json:"style_preset"`
	Cue         VisualCue      `json:"cue,omitempty"`
	CueHash     string         `json:"cue_hash"`
	AssetID     string         `json:"asset_id,omitempty"`
	Error       string         `json:"error,omitempty"`
	Attempts    int            `json:"attempts"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type ImageAsset struct {
	ID        string    `json:"id"`
	JobID     string    `json:"job_id"`
	Path      string    `json:"path"`
	CreatedAt time.Time `json:"created_at"`
}

type ImageQueue interface {
	Enqueue(ctx context.Context, job ImageJob) (ImageJob, bool, error)
	Claim(ctx context.Context) (*ImageJob, error)
	Complete(ctx context.Context, jobID string, asset ImageAsset) error
	Fail(ctx context.Context, jobID string, err error, retryable bool) error
}

type MemoryImageQueue struct {
	mu       sync.Mutex
	seq      int
	jobs     []ImageJob
	byHash   map[string]int
	byJobID  map[string]int
	failedAt map[string]time.Time
}

func NewMemoryImageQueue() *MemoryImageQueue {
	return &MemoryImageQueue{
		byHash:   map[string]int{},
		byJobID:  map[string]int{},
		failedAt: map[string]time.Time{},
	}
}

func (q *MemoryImageQueue) Enqueue(ctx context.Context, job ImageJob) (ImageJob, bool, error) {
	if err := ctx.Err(); err != nil {
		return ImageJob{}, false, err
	}
	q.mu.Lock()
	defer q.mu.Unlock()

	if job.CueHash == "" {
		job.CueHash = HashVisualCue(job.StoryID, job.SceneID, job.Turn, job.StylePreset, job.Provider, job)
	}
	if idx, ok := q.byHash[job.CueHash]; ok {
		return q.jobs[idx], true, nil
	}
	q.seq++
	now := time.Now().UTC()
	if job.ID == "" {
		job.ID = "imgjob_" + hex.EncodeToString([]byte{byte(q.seq >> 8), byte(q.seq)})
	}
	job.Status = ImageJobQueued
	job.CreatedAt = now
	job.UpdatedAt = now
	q.jobs = append(q.jobs, job)
	idx := len(q.jobs) - 1
	q.byHash[job.CueHash] = idx
	q.byJobID[job.ID] = idx
	return job, false, nil
}

func (q *MemoryImageQueue) Claim(ctx context.Context) (*ImageJob, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	q.mu.Lock()
	defer q.mu.Unlock()

	for i := range q.jobs {
		if q.jobs[i].Status == ImageJobQueued {
			q.jobs[i].Status = ImageJobRunning
			q.jobs[i].Attempts++
			q.jobs[i].UpdatedAt = time.Now().UTC()
			job := q.jobs[i]
			return &job, nil
		}
	}
	return nil, nil
}

func (q *MemoryImageQueue) Complete(ctx context.Context, jobID string, asset ImageAsset) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	q.mu.Lock()
	defer q.mu.Unlock()

	idx, ok := q.byJobID[jobID]
	if !ok {
		return errors.New("image job not found")
	}
	q.jobs[idx].Status = ImageJobDone
	q.jobs[idx].AssetID = asset.ID
	q.jobs[idx].UpdatedAt = time.Now().UTC()
	return nil
}

func (q *MemoryImageQueue) Fail(ctx context.Context, jobID string, err error, retryable bool) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	q.mu.Lock()
	defer q.mu.Unlock()

	idx, ok := q.byJobID[jobID]
	if !ok {
		return errors.New("image job not found")
	}
	q.jobs[idx].Status = ImageJobFailed
	if err != nil {
		q.jobs[idx].Error = err.Error()
	}
	q.jobs[idx].UpdatedAt = time.Now().UTC()
	q.failedAt[jobID] = q.jobs[idx].UpdatedAt
	return nil
}

func HashVisualCue(storyID, sceneID string, turn int, stylePreset, provider string, cue any) string {
	payload := struct {
		StoryID     string `json:"story_id"`
		SceneID     string `json:"scene_id"`
		Turn        int    `json:"turn"`
		StylePreset string `json:"style_preset"`
		Provider    string `json:"provider"`
		Cue         any    `json:"cue"`
	}{
		StoryID:     storyID,
		SceneID:     sceneID,
		Turn:        turn,
		StylePreset: stylePreset,
		Provider:    provider,
		Cue:         cue,
	}
	data, _ := json.Marshal(payload)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
