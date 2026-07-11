package storage

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/crimsab/oneday/internal/ai"
)

type AITelemetryRecorder struct {
	db *DB
}

func NewAITelemetryRecorder(db *DB) *AITelemetryRecorder {
	return &AITelemetryRecorder{db: db}
}

func (r *AITelemetryRecorder) StartRun(_ context.Context, input ai.TelemetryRunStart) (ai.TelemetryRef, error) {
	if r == nil || r.db == nil {
		return ai.TelemetryRef{}, nil
	}
	profileName := strings.TrimSpace(input.Metadata.PromptProfile)
	if profileName == "" {
		profileName = strings.TrimSpace(input.Metadata.Stage)
	}
	if profileName == "" {
		profileName = "completion"
	}
	revision, err := r.db.EnsurePromptRevision(PromptRevisionInput{
		ProfileName:        profileName,
		Description:        "Redacted reproducibility profile for " + profileName,
		TemplateVersion:    input.Metadata.PromptTemplate,
		PromptHash:         input.PromptHash,
		ResponseSchemaHash: input.ResponseSchemaHash,
		ConfigJSON:         input.RequestConfigJSON,
	})
	if err != nil {
		return ai.TelemetryRef{}, err
	}
	metadataJSON, _ := json.Marshal(ai.RedactTelemetryPayload(stringMapToAny(input.Metadata.SafeMetadata)))
	run := &GenerationRun{
		TraceID:            input.Metadata.TraceID,
		ParentRunID:        input.Metadata.ParentRunID,
		StoryID:            input.Metadata.StoryID,
		BranchID:           input.Metadata.BranchID,
		SourceCommitID:     input.Metadata.SourceCommitID,
		MessageID:          input.Metadata.MessageID,
		Stage:              defaultString(input.Metadata.Stage, "completion"),
		PromptRevisionID:   revision.ID,
		PromptHash:         input.PromptHash,
		RequestConfigJSON:  input.RequestConfigJSON,
		RequestedStreaming: input.RequestedStreaming,
		MetadataJSON:       string(metadataJSON),
		CreatedAt:          input.StartedAt,
	}
	if err := r.db.StartGenerationRun(run); err != nil {
		return ai.TelemetryRef{}, err
	}
	return ai.TelemetryRef{RunID: run.ID, TraceID: run.TraceID}, nil
}

func (r *AITelemetryRecorder) StartAttempt(_ context.Context, input ai.TelemetryAttemptStart) (string, error) {
	if r == nil || r.db == nil || input.RunID == "" {
		return "", nil
	}
	attempt := &GenerationAttempt{
		RunID:               input.RunID,
		Sequence:            input.Sequence,
		Provider:            input.Provider,
		RequestedModel:      input.RequestedModel,
		ReasoningConfigJSON: input.ReasoningConfigJSON,
		RequestedStreaming:  input.RequestedStreaming,
		RetryReason:         input.RetryReason,
		CreatedAt:           input.StartedAt,
	}
	if err := r.db.StartGenerationAttempt(attempt); err != nil {
		return "", err
	}
	return attempt.ID, nil
}

func (r *AITelemetryRecorder) FinishAttempt(_ context.Context, id string, completion ai.TelemetryCompletion) error {
	if r == nil || r.db == nil || id == "" {
		return nil
	}
	return r.db.FinishGenerationAttempt(id, storageCompletion(completion))
}

func (r *AITelemetryRecorder) FinishRun(_ context.Context, id string, completion ai.TelemetryCompletion) error {
	if r == nil || r.db == nil || id == "" {
		return nil
	}
	return r.db.FinishGenerationRun(id, storageCompletion(completion))
}

func (r *AITelemetryRecorder) Event(_ context.Context, runID, attemptID, eventType string, payload map[string]any) error {
	if r == nil || r.db == nil || runID == "" {
		return nil
	}
	encoded, _ := json.Marshal(ai.RedactTelemetryPayload(payload))
	return r.db.AppendGenerationEvent(runID, attemptID, eventType, string(encoded))
}

func storageCompletion(completion ai.TelemetryCompletion) GenerationCompletion {
	return GenerationCompletion{
		Status:            completion.Status,
		ResolvedModel:     completion.ResolvedModel,
		ObservedStreaming: completion.ObservedStreaming,
		TTFTMs:            completion.TTFTMs,
		DurationMs:        completion.DurationMs,
		Usage: GenerationUsage{
			InputTokens:       completion.Usage.PromptTokens,
			OutputTokens:      completion.Usage.CompletionTokens,
			ReasoningTokens:   completion.Usage.ReasoningTokens,
			CachedInputTokens: completion.Usage.CachedPromptTokens,
			TotalTokens:       completion.Usage.TotalTokens,
			CostUSD:           completion.Usage.CostUSD,
		},
		ErrorClass:   completion.ErrorClass,
		ErrorSummary: completion.ErrorSummary,
	}
}

func stringMapToAny(values map[string]string) map[string]any {
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
