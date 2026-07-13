package assets

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type ImageProvider interface {
	Generate(ctx context.Context, job ImageJob) ([]byte, string, error)
}

type CommandImageProvider struct {
	Command   string
	Args      []string
	Extension string
	Timeout   time.Duration
}

func (p CommandImageProvider) Generate(ctx context.Context, job ImageJob) ([]byte, string, error) {
	if strings.TrimSpace(p.Command) == "" {
		return nil, "", fmt.Errorf("imagegen command is empty")
	}
	if p.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.Timeout)
		defer cancel()
	}

	payload, err := json.Marshal(struct {
		Job ImageJob `json:"job"`
	}{
		Job: job,
	})
	if err != nil {
		return nil, "", fmt.Errorf("marshaling imagegen command payload: %w", err)
	}

	cmd := exec.CommandContext(ctx, p.Command, p.Args...)
	cmd.Stdin = bytes.NewReader(payload)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail != "" {
			return nil, "", fmt.Errorf("imagegen command failed: %w: %s", err, detail)
		}
		return nil, "", fmt.Errorf("imagegen command failed: %w", err)
	}
	if len(out) == 0 {
		return nil, "", fmt.Errorf("imagegen command returned empty output")
	}
	return out, firstNonEmptyString(p.Extension, "png"), nil
}

type ImageAssetStore interface {
	Save(ctx context.Context, job ImageJob, data []byte, extension string) (ImageAsset, error)
}

type ImageWorker struct {
	Queue    ImageQueue
	Provider ImageProvider
	Store    ImageAssetStore
}

func (w ImageWorker) RunOnce(ctx context.Context) (bool, error) {
	if w.Queue == nil {
		return false, fmt.Errorf("image worker queue is nil")
	}
	if w.Provider == nil {
		return false, fmt.Errorf("image worker provider is nil")
	}
	if w.Store == nil {
		return false, fmt.Errorf("image worker store is nil")
	}
	job, err := w.Queue.Claim(ctx)
	if err != nil || job == nil {
		return false, err
	}

	data, extension, err := w.Provider.Generate(ctx, *job)
	if err != nil {
		_ = w.Queue.Fail(ctx, job.ID, err, true)
		return true, err
	}
	asset, err := w.Store.Save(ctx, *job, data, extension)
	if err != nil {
		_ = w.Queue.Fail(ctx, job.ID, err, true)
		return true, err
	}
	if err := w.Queue.Complete(ctx, job.ID, asset); err != nil {
		return true, err
	}
	return true, nil
}

type FileAssetStore struct {
	Root string
}

func (s FileAssetStore) Save(ctx context.Context, job ImageJob, data []byte, extension string) (ImageAsset, error) {
	if err := ctx.Err(); err != nil {
		return ImageAsset{}, err
	}
	if len(data) == 0 {
		return ImageAsset{}, fmt.Errorf("image asset data is empty")
	}
	root := strings.TrimSpace(s.Root)
	if root == "" {
		root = "oneday_data/assets"
	}
	extension = normalizeExtension(extension)
	assetID := assetIDForJob(job, data)
	dir := filepath.Join(root, safePathPart(job.StoryID), "images")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return ImageAsset{}, fmt.Errorf("creating image asset dir: %w", err)
	}
	path := filepath.Join(dir, assetID+extension)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return ImageAsset{}, fmt.Errorf("writing image asset: %w", err)
	}
	return ImageAsset{
		ID:        assetID,
		JobID:     job.ID,
		Path:      path,
		CreatedAt: time.Now().UTC(),
	}, nil
}

func assetIDForJob(job ImageJob, data []byte) string {
	sum := sha256.Sum256(append([]byte(job.ID+"|"+job.CueHash+"|"), data...))
	return "img_" + hex.EncodeToString(sum[:8])
}

func normalizeExtension(extension string) string {
	extension = strings.ToLower(strings.TrimSpace(extension))
	extension = strings.TrimPrefix(extension, ".")
	switch extension {
	case "jpg", "jpeg":
		return ".jpg"
	case "webp":
		return ".webp"
	default:
		return ".png"
	}
}

var unsafePathPart = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func safePathPart(value string) string {
	value = strings.Trim(unsafePathPart.ReplaceAllString(value, "-"), "-.")
	if value == "" {
		return "unknown"
	}
	return value
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
