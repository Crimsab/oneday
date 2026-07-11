package audio

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/crimsab/oneday/internal/config"
	"github.com/crimsab/oneday/internal/storage"
)

type Service struct {
	db            *storage.DB
	cfg           config.TTSConfig
	providers     map[string]Provider
	providerOrder []string
}

func NewService(db *storage.DB, cfg config.TTSConfig) *Service {
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 45 * time.Second
	}
	return &Service{
		db: db, cfg: cfg, providerOrder: append([]string(nil), cfg.ProviderOrder...),
		providers: map[string]Provider{
			"cloud": NewCloudProvider(cfg.Cloud, timeout),
			"local": NewLocalProvider(cfg.Local, timeout),
		},
	}
}

func NewServiceWithProviders(db *storage.DB, cfg config.TTSConfig, providers ...Provider) *Service {
	service := NewService(db, cfg)
	for _, provider := range providers {
		if provider != nil {
			service.providers[provider.ID()] = provider
		}
	}
	return service
}

func (service *Service) ProviderStatuses(ctx context.Context) []ProviderStatus {
	statuses := make([]ProviderStatus, 0, len(service.providerOrder))
	seen := map[string]bool{}
	for _, id := range service.providerOrder {
		if provider := service.providers[id]; provider != nil {
			statuses = append(statuses, provider.Status(ctx))
			seen[id] = true
		}
	}
	for id, provider := range service.providers {
		if !seen[id] {
			statuses = append(statuses, provider.Status(ctx))
		}
	}
	sort.SliceStable(statuses, func(i, j int) bool { return statuses[i].ID < statuses[j].ID })
	return statuses
}

func (service *Service) EnsureConfiguredVoiceProfiles() ([]storage.VoiceProfile, error) {
	if service == nil || service.db == nil {
		return nil, errors.New("audio service database is unavailable")
	}
	profiles := []storage.VoiceProfile{}
	for _, item := range []struct {
		provider string
		cfg      config.TTSEndpoint
	}{
		{"local", service.cfg.Local}, {"cloud", service.cfg.Cloud},
	} {
		if !item.cfg.Enabled || strings.TrimSpace(item.cfg.Voice) == "" {
			continue
		}
		idHash := sha256.Sum256([]byte(strings.Join([]string{item.provider, item.cfg.Model, item.cfg.Voice, item.cfg.Version}, "\x00")))
		languageJSON, _ := json.Marshal(item.cfg.Languages)
		profile, err := service.db.UpsertVoiceProfile(storage.VoiceProfile{
			ID: "voice-" + hex.EncodeToString(idHash[:8]), Provider: item.provider,
			Model: item.cfg.Model, ProviderVoiceID: item.cfg.Voice,
			DisplayName:  strings.ToUpper(item.provider[:1]) + item.provider[1:] + " / " + item.cfg.Voice,
			LanguageTags: languageJSON, Traits: json.RawMessage(`{"source":"configured"}`),
			Rights: json.RawMessage(`{"operator_verified":false}`), Version: item.cfg.Version,
			StyleFamily: "neutral", Enabled: true,
		})
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, *profile)
	}
	return profiles, nil
}

func (service *Service) QueueCommittedMessage(ctx context.Context, storyID string, messageID int64) ([]storage.AudioAsset, error) {
	if service == nil || service.db == nil {
		return nil, errors.New("audio service database is unavailable")
	}
	message, err := service.db.GetCommittedAssistantMessage(storyID, messageID)
	if err != nil {
		return nil, err
	}
	settings, err := service.db.GetStoryTTSSettings(storyID)
	if err != nil || settings.Mode == "off" {
		return nil, err
	}
	story, err := service.db.GetStory(storyID)
	if err != nil {
		return nil, err
	}
	segments, lineage := SegmentCommittedMessage(*message)
	assignments, err := service.db.ListVoiceAssignments(storyID)
	if err != nil {
		return nil, err
	}
	profiles, err := service.db.ListVoiceProfiles(true)
	if err != nil {
		return nil, err
	}
	profilesByID := map[string]storage.VoiceProfile{}
	for _, profile := range profiles {
		profilesByID[profile.ID] = profile
	}
	queued := []storage.AudioAsset{}
	for _, segment := range segments {
		if !modeAllows(settings.Mode, segment.Kind) {
			continue
		}
		formID := ""
		if segment.SpeakerEntityID != "" {
			formID = service.db.GetActiveEntityFormID(storyID, segment.SpeakerEntityID)
		}
		assignment := resolveAssignment(assignments, segment, formID)
		if assignment == nil || assignment.EnabledMode == "off" {
			continue
		}
		profile, ok := profilesByID[assignment.VoiceProfileID]
		if !ok {
			continue
		}
		language := firstNonEmpty(normalizeLanguageTag(segment.LanguageTag), normalizeLanguageTag(assignment.LanguageTag), normalizeLanguageTag(settings.DefaultLanguage), normalizeLanguageTag(story.Language))
		if language == "" || !voiceSupportsLanguage(profile, language) {
			continue
		}
		style := assignment.Style
		if len(style) == 0 {
			style = json.RawMessage(`{}`)
		}
		lexicon, err := service.db.ListPronunciations(storyID, language)
		if err != nil {
			return nil, err
		}
		pronunciationRevision := maxPronunciationRevision(lexicon)
		textHash := hashText(normalizeSpeechText(segment.Text))
		cacheKey, _, canonicalStyle, err := CacheIdentity(profile, language, segment.Text, style, 1, outputFormatForProvider(profile.Provider), pronunciationRevision)
		if err != nil {
			return nil, err
		}
		asset := storage.AudioAsset{
			StoryID: storyID, SourceMessageID: message.ID, SegmentIndex: segment.Index,
			SegmentKind: segment.Kind, SpeakerEntityID: segment.SpeakerEntityID, FormID: formID,
			VoiceProfileID: profile.ID, Provider: profile.Provider, Model: profile.Model,
			ProviderVoiceID: profile.ProviderVoiceID, VoiceVersion: profile.Version,
			LanguageTag: language, PronunciationRevision: pronunciationRevision,
			Text: segment.Text, TextHash: textHash, CacheKey: cacheKey, Style: canonicalStyle,
			Speed: 1, OutputFormat: outputFormatForProvider(profile.Provider), Status: "queued", Timings: json.RawMessage(`[]`),
		}
		if cached, cacheErr := service.db.GetTTSCacheEntry(cacheKey); cacheErr == nil && cached.Status == "ready" && safeRegularFile(cached.FilePath) {
			asset.Status, asset.FilePath, asset.DurationMS, asset.Timings = "ready", cached.FilePath, cached.DurationMS, cached.Timings
		}
		job := storage.TTSJob{Provider: profile.Provider, Status: "queued", MaxAttempts: 3, TraceID: lineage.TraceID, ParentRunID: lineage.RunID}
		stored, _, err := service.db.QueueAudioAsset(asset, job)
		if err != nil {
			return nil, err
		}
		queued = append(queued, *stored)
	}
	return queued, nil
}

func CacheIdentity(profile storage.VoiceProfile, language, text string, style json.RawMessage, speed float64, format string, pronunciationRevision int) (cacheKey, styleHash string, canonicalStyle json.RawMessage, err error) {
	var value any
	if len(style) == 0 {
		style = json.RawMessage(`{}`)
	}
	if err = json.Unmarshal(style, &value); err != nil {
		return "", "", nil, fmt.Errorf("invalid TTS style: %w", err)
	}
	canonicalStyle, err = json.Marshal(value)
	if err != nil {
		return "", "", nil, err
	}
	styleDigest := sha256.Sum256(append(canonicalStyle, []byte(fmt.Sprintf("\x00pronunciation:%d", pronunciationRevision))...))
	styleHash = hex.EncodeToString(styleDigest[:])
	textHash := hashText(normalizeSpeechText(text))
	parts := []string{profile.Provider, profile.Model, profile.ProviderVoiceID, profile.Version, language, textHash, styleHash, fmt.Sprintf("%.3f", speed), format}
	digest := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(digest[:]), styleHash, canonicalStyle, nil
}

func (service *Service) ProcessJob(ctx context.Context, jobID string) error {
	job, err := service.db.ClaimTTSJob(jobID)
	if err != nil {
		return err
	}
	asset, err := service.db.GetAudioAsset(job.StoryID, job.AudioAssetID)
	if err != nil {
		return err
	}
	provider := service.providers[asset.Provider]
	if provider == nil {
		_ = service.db.FailTTSJob(job.ID, "unavailable", "TTS provider is not registered", "", false)
		return errors.New("TTS provider is not registered")
	}
	status := provider.Status(ctx)
	if !status.Available {
		detail := "TTS provider unavailable: " + status.Reason
		_ = service.db.FailTTSJob(job.ID, "unavailable", detail, "", false)
		return errors.New(detail)
	}
	run, attempt, started, err := service.startTTSRun(*job, *asset)
	if err != nil {
		_ = service.db.FailTTSJob(job.ID, "telemetry", err.Error(), "", true)
		return err
	}
	lexicon, _ := service.db.ListPronunciations(asset.StoryID, asset.LanguageTag)
	request := SynthesisRequest{Text: asset.Text, Model: asset.Model, Voice: asset.ProviderVoiceID, Speed: asset.Speed, Format: asset.OutputFormat}
	if asset.Provider == "cloud" {
		request.Instructions = pronunciationInstructions(lexicon)
	} else {
		request.Text = applyPronunciations(asset.Text, lexicon)
	}
	result, synthesisErr := provider.Synthesize(ctx, request)
	duration := time.Since(started).Milliseconds()
	if synthesisErr != nil {
		service.finishTTSRun(run, attempt, "failed", duration, "provider", synthesisErr.Error(), asset.Model)
		_ = service.db.FailTTSJob(job.ID, "provider", synthesisErr.Error(), run.ID, job.Attempts < job.MaxAttempts)
		return synthesisErr
	}
	filePath, err := service.writeCacheAudio(asset.CacheKey, result.Format, result.Audio)
	if err != nil {
		service.finishTTSRun(run, attempt, "failed", duration, "storage", err.Error(), asset.Model)
		_ = service.db.FailTTSJob(job.ID, "storage", err.Error(), run.ID, true)
		return err
	}
	service.finishTTSRun(run, attempt, "succeeded", duration, "", "", asset.Model)
	cache := storage.TTSCacheEntry{CacheKey: asset.CacheKey, Provider: asset.Provider, Model: asset.Model, ProviderVoiceID: asset.ProviderVoiceID, VoiceVersion: asset.VoiceVersion, LanguageTag: asset.LanguageTag, TextHash: asset.TextHash, Style: asset.Style, Speed: asset.Speed, OutputFormat: result.Format, Status: "ready"}
	_, styleHash, _, _ := CacheIdentity(storage.VoiceProfile{Provider: asset.Provider, Model: asset.Model, ProviderVoiceID: asset.ProviderVoiceID, Version: asset.VoiceVersion}, asset.LanguageTag, asset.Text, asset.Style, asset.Speed, result.Format, asset.PronunciationRevision)
	cache.StyleHash = styleHash
	return service.db.CompleteTTSJob(job.ID, cache, filePath, result.DurationMS, result.Timings, run.ID)
}

func (service *Service) startTTSRun(job storage.TTSJob, asset storage.AudioAsset) (*storage.GenerationRun, *storage.GenerationAttempt, time.Time, error) {
	started := time.Now().UTC()
	revision, err := service.db.EnsurePromptRevision(storage.PromptRevisionInput{ProfileName: "tts", Description: "Canonical committed-text speech synthesis", TemplateVersion: "tts-v1", PromptHash: asset.TextHash, ResponseSchemaHash: "audio-binary-v1", ConfigJSON: string(asset.Style)})
	if err != nil {
		return nil, nil, started, err
	}
	messageID := asset.SourceMessageID
	run := &storage.GenerationRun{TraceID: firstNonEmpty(job.TraceID, uuid.NewString()), ParentRunID: job.ParentRunID, StoryID: asset.StoryID, BranchID: asset.BranchID, SourceCommitID: asset.SourceCommitID, MessageID: &messageID, Stage: "tts", PromptRevisionID: revision.ID, PromptHash: asset.TextHash, RequestConfigJSON: string(asset.Style), MetadataJSON: fmt.Sprintf(`{"audio_asset_id":%q,"voice_profile_id":%q}`, asset.ID, asset.VoiceProfileID), CreatedAt: started}
	if err := service.db.StartGenerationRun(run); err != nil {
		return nil, nil, started, err
	}
	attempt := &storage.GenerationAttempt{RunID: run.ID, Sequence: 1, Provider: asset.Provider, RequestedModel: asset.Model, ReasoningConfigJSON: `{}`, CreatedAt: started}
	if err := service.db.StartGenerationAttempt(attempt); err != nil {
		_ = service.db.FinishGenerationRun(run.ID, storage.GenerationCompletion{Status: "failed", ErrorClass: "telemetry"})
		return nil, nil, started, err
	}
	return run, attempt, started, nil
}

func (service *Service) finishTTSRun(run *storage.GenerationRun, attempt *storage.GenerationAttempt, status string, duration int64, errorClass, detail, model string) {
	completion := storage.GenerationCompletion{Status: status, DurationMs: duration, ErrorClass: errorClass, ErrorSummary: detail, ResolvedModel: model}
	_ = service.db.FinishGenerationAttempt(attempt.ID, completion)
	_ = service.db.FinishGenerationRun(run.ID, completion)
}

func (service *Service) writeCacheAudio(cacheKey, format string, audio []byte) (string, error) {
	if len(audio) == 0 || len(cacheKey) < 4 {
		return "", errors.New("audio bytes and a valid cache key are required")
	}
	root := service.cfg.OutputDir
	if strings.TrimSpace(root) == "" {
		root = "./oneday_data/audio"
	}
	directory := filepath.Join(root, cacheKey[:2])
	if err := os.MkdirAll(directory, 0o750); err != nil {
		return "", err
	}
	path := filepath.Join(directory, cacheKey+"."+format)
	temp, err := os.CreateTemp(directory, cacheKey+"-*.tmp")
	if err != nil {
		return "", err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if err := temp.Chmod(0o640); err != nil {
		temp.Close()
		return "", err
	}
	if _, err := temp.Write(audio); err != nil {
		temp.Close()
		return "", err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return "", err
	}
	if err := temp.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(tempName, path); err != nil {
		return "", err
	}
	return path, nil
}

func resolveAssignment(assignments []storage.VoiceAssignment, segment Segment, formID string) *storage.VoiceAssignment {
	role := segment.Role
	if segment.Kind == "narrator" {
		role = "narrator"
	}
	var best *storage.VoiceAssignment
	bestScore := -1
	for index := range assignments {
		assignment := &assignments[index]
		if assignment.Role != role {
			continue
		}
		score := 0
		if assignment.EntityID != "" {
			if assignment.EntityID != segment.SpeakerEntityID {
				continue
			}
			score += 4
		}
		if assignment.FormID != "" {
			if assignment.FormID != formID {
				continue
			}
			score += 2
		}
		if assignment.IdentityID != "" {
			continue
		}
		if score > bestScore {
			best, bestScore = assignment, score
		}
	}
	if best == nil && role != "narrator" {
		fallback := Segment{Kind: "narrator", Role: "narrator"}
		return resolveAssignment(assignments, fallback, "")
	}
	return best
}

func voiceSupportsLanguage(profile storage.VoiceProfile, language string) bool {
	if len(profile.LanguageTags) == 0 || string(profile.LanguageTags) == "[]" {
		return true
	}
	var tags []string
	if json.Unmarshal(profile.LanguageTags, &tags) != nil {
		return false
	}
	base := strings.Split(language, "-")[0]
	for _, tag := range tags {
		normalized := normalizeLanguageTag(tag)
		if normalized == language || strings.Split(normalized, "-")[0] == base {
			return true
		}
	}
	return false
}

func modeAllows(mode, kind string) bool {
	return mode == "all" || mode == "narrator" && kind == "narrator" || mode == "dialogue" && kind == "dialogue"
}

func maxPronunciationRevision(entries []storage.PronunciationEntry) int {
	max := 0
	for _, entry := range entries {
		if entry.Revision > max {
			max = entry.Revision
		}
	}
	return max
}

func pronunciationInstructions(entries []storage.PronunciationEntry) string {
	if len(entries) == 0 {
		return ""
	}
	parts := make([]string, 0, len(entries))
	for _, entry := range entries {
		parts = append(parts, fmt.Sprintf("Pronounce %q as %q (%s).", entry.SourceText, entry.Pronunciation, entry.Alphabet))
	}
	return strings.Join(parts, " ")
}

func applyPronunciations(text string, entries []storage.PronunciationEntry) string {
	result := text
	for _, entry := range entries {
		if entry.CaseSensitive {
			result = strings.ReplaceAll(result, entry.SourceText, entry.Pronunciation)
			continue
		}
		pattern := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(entry.SourceText))
		result = pattern.ReplaceAllStringFunc(result, func(string) string { return entry.Pronunciation })
	}
	return result
}

func outputFormatForProvider(provider string) string {
	if provider == "local" {
		return "wav"
	}
	return "mp3"
}

func hashText(text string) string {
	digest := sha256.Sum256([]byte(text))
	return hex.EncodeToString(digest[:])
}

func safeRegularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular() && info.Size() > 0
}
