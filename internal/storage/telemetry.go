package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	GenerationStatusRunning   = "running"
	GenerationStatusSucceeded = "succeeded"
	GenerationStatusFailed    = "failed"
	GenerationStatusCancelled = "cancelled"
)

type GenerationUsage struct {
	InputTokens       int     `json:"input_tokens"`
	OutputTokens      int     `json:"output_tokens"`
	ReasoningTokens   int     `json:"reasoning_tokens"`
	CachedInputTokens int     `json:"cached_input_tokens"`
	TotalTokens       int     `json:"total_tokens"`
	CostUSD           float64 `json:"cost_usd"`
}

type PromptRevision struct {
	ID                 string    `json:"id"`
	ProfileID          string    `json:"profile_id"`
	ProfileName        string    `json:"profile_name"`
	Version            int       `json:"version"`
	TemplateVersion    string    `json:"template_version"`
	PromptHash         string    `json:"prompt_hash"`
	ResponseSchemaHash string    `json:"response_schema_hash"`
	ConfigJSON         string    `json:"config_json"`
	CreatedAt          time.Time `json:"created_at"`
}

type PromptRevisionInput struct {
	ProfileName        string
	Description        string
	TemplateVersion    string
	PromptHash         string
	ResponseSchemaHash string
	ConfigJSON         string
	RedactionPolicy    string
	RetentionDays      int
}

type GenerationRun struct {
	ID                 string          `json:"id"`
	TraceID            string          `json:"trace_id"`
	ParentRunID        string          `json:"parent_run_id,omitempty"`
	StoryID            string          `json:"story_id,omitempty"`
	BranchID           string          `json:"branch_id,omitempty"`
	SourceCommitID     string          `json:"source_commit_id,omitempty"`
	MessageID          *int64          `json:"message_id,omitempty"`
	Stage              string          `json:"stage"`
	Status             string          `json:"status"`
	PromptRevisionID   string          `json:"prompt_revision_id,omitempty"`
	PromptHash         string          `json:"prompt_hash"`
	RequestConfigJSON  string          `json:"request_config_json"`
	RequestedStreaming bool            `json:"requested_streaming"`
	ObservedStreaming  bool            `json:"observed_streaming"`
	Usage              GenerationUsage `json:"usage"`
	TTFTMs             int64           `json:"ttft_ms"`
	DurationMs         int64           `json:"duration_ms"`
	ErrorClass         string          `json:"error_class,omitempty"`
	MetadataJSON       string          `json:"metadata_json"`
	CreatedAt          time.Time       `json:"created_at"`
	FinishedAt         *time.Time      `json:"finished_at,omitempty"`
}

type GenerationAttempt struct {
	ID                  string          `json:"id"`
	RunID               string          `json:"run_id"`
	Sequence            int             `json:"sequence"`
	Provider            string          `json:"provider"`
	RequestedModel      string          `json:"requested_model,omitempty"`
	ResolvedModel       string          `json:"resolved_model,omitempty"`
	ReasoningConfigJSON string          `json:"reasoning_config_json"`
	RequestedStreaming  bool            `json:"requested_streaming"`
	ObservedStreaming   bool            `json:"observed_streaming"`
	Status              string          `json:"status"`
	TTFTMs              int64           `json:"ttft_ms"`
	DurationMs          int64           `json:"duration_ms"`
	Usage               GenerationUsage `json:"usage"`
	RetryReason         string          `json:"retry_reason,omitempty"`
	ErrorClass          string          `json:"error_class,omitempty"`
	ErrorSummary        string          `json:"error_summary,omitempty"`
	CreatedAt           time.Time       `json:"created_at"`
	FinishedAt          *time.Time      `json:"finished_at,omitempty"`
}

type GenerationCompletion struct {
	Status            string
	ObservedStreaming bool
	Usage             GenerationUsage
	TTFTMs            int64
	DurationMs        int64
	ErrorClass        string
	ErrorSummary      string
	ResolvedModel     string
}

func (db *DB) EnsurePromptRevision(input PromptRevisionInput) (*PromptRevision, error) {
	input.ProfileName = strings.TrimSpace(input.ProfileName)
	input.PromptHash = strings.TrimSpace(input.PromptHash)
	if input.ProfileName == "" || input.PromptHash == "" {
		return nil, errors.New("prompt profile name and hash are required")
	}
	input.ConfigJSON = validJSONOrDefault(input.ConfigJSON, "{}")
	if input.RedactionPolicy == "" {
		input.RedactionPolicy = "secrets_and_reasoning"
	}
	if input.RetentionDays == 0 {
		input.RetentionDays = 30
	}
	if input.RetentionDays < 1 || input.RetentionDays > 3650 {
		return nil, errors.New("prompt retention days must be between 1 and 3650")
	}

	var revision *PromptRevision
	err := db.WithTx(func(tx *sql.Tx) error {
		profileID := ""
		err := tx.QueryRow(`SELECT id FROM prompt_profiles WHERE name=?`, input.ProfileName).Scan(&profileID)
		if errors.Is(err, sql.ErrNoRows) {
			profileID = uuid.NewString()
			if _, err := tx.Exec(`INSERT INTO prompt_profiles (id,name,description,redaction_policy,retention_days) VALUES (?,?,?,?,?)`, profileID, input.ProfileName, input.Description, input.RedactionPolicy, input.RetentionDays); err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else if _, err := tx.Exec(`UPDATE prompt_profiles SET description=?,redaction_policy=?,retention_days=?,updated_at=CURRENT_TIMESTAMP WHERE id=?`, input.Description, input.RedactionPolicy, input.RetentionDays, profileID); err != nil {
			return err
		}

		revision = &PromptRevision{}
		err = tx.QueryRow(`SELECT r.id,r.profile_id,p.name,r.version,r.template_version,r.prompt_hash,r.response_schema_hash,r.config_json,r.created_at FROM prompt_profile_revisions r JOIN prompt_profiles p ON p.id=r.profile_id WHERE r.profile_id=? AND r.prompt_hash=? AND r.response_schema_hash=?`, profileID, input.PromptHash, input.ResponseSchemaHash).Scan(
			&revision.ID, &revision.ProfileID, &revision.ProfileName, &revision.Version, &revision.TemplateVersion, &revision.PromptHash, &revision.ResponseSchemaHash, &revision.ConfigJSON, &revision.CreatedAt,
		)
		if err == nil {
			return nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if err := tx.QueryRow(`SELECT COALESCE(MAX(version),0)+1 FROM prompt_profile_revisions WHERE profile_id=?`, profileID).Scan(&revision.Version); err != nil {
			return err
		}
		revision.ID = uuid.NewString()
		revision.ProfileID = profileID
		revision.ProfileName = input.ProfileName
		revision.TemplateVersion = input.TemplateVersion
		revision.PromptHash = input.PromptHash
		revision.ResponseSchemaHash = input.ResponseSchemaHash
		revision.ConfigJSON = input.ConfigJSON
		revision.CreatedAt = time.Now().UTC()
		_, err = tx.Exec(`INSERT INTO prompt_profile_revisions (id,profile_id,version,template_version,prompt_hash,response_schema_hash,config_json,created_at) VALUES (?,?,?,?,?,?,?,?)`, revision.ID, revision.ProfileID, revision.Version, revision.TemplateVersion, revision.PromptHash, revision.ResponseSchemaHash, revision.ConfigJSON, revision.CreatedAt)
		return err
	})
	return revision, err
}

func (db *DB) StartGenerationRun(run *GenerationRun) error {
	if run == nil {
		return errors.New("generation run is required")
	}
	run.Stage = strings.TrimSpace(run.Stage)
	run.PromptHash = strings.TrimSpace(run.PromptHash)
	if run.Stage == "" || run.PromptHash == "" {
		return errors.New("generation stage and prompt hash are required")
	}
	if run.ID == "" {
		run.ID = uuid.NewString()
	}
	if run.TraceID == "" {
		run.TraceID = run.ID
	}
	if run.Status == "" {
		run.Status = GenerationStatusRunning
	}
	if run.Status != GenerationStatusRunning {
		return errors.New("generation run must start in running state")
	}
	run.RequestConfigJSON = validJSONOrDefault(run.RequestConfigJSON, "{}")
	run.MetadataJSON = validJSONOrDefault(run.MetadataJSON, "{}")
	if run.CreatedAt.IsZero() {
		run.CreatedAt = time.Now().UTC()
	}
	_, err := db.conn.Exec(`INSERT INTO generation_runs (id,trace_id,parent_run_id,story_id,branch_id,source_commit_id,message_id,stage,status,prompt_revision_id,prompt_hash,request_config_json,requested_streaming,metadata_json,created_at) VALUES (?,?,NULLIF(?,''),?,?,?,?,?,?,NULLIF(?,''),?,?,?,?,?)`,
		run.ID, run.TraceID, run.ParentRunID, run.StoryID, run.BranchID, run.SourceCommitID, run.MessageID, run.Stage, run.Status, run.PromptRevisionID, run.PromptHash, run.RequestConfigJSON, boolInt(run.RequestedStreaming), run.MetadataJSON, run.CreatedAt)
	return err
}

func (db *DB) StartGenerationAttempt(attempt *GenerationAttempt) error {
	if attempt == nil || strings.TrimSpace(attempt.RunID) == "" || strings.TrimSpace(attempt.Provider) == "" || attempt.Sequence < 1 {
		return errors.New("attempt run, provider, and positive sequence are required")
	}
	if attempt.ID == "" {
		attempt.ID = uuid.NewString()
	}
	if attempt.Status == "" {
		attempt.Status = GenerationStatusRunning
	}
	if attempt.Status != GenerationStatusRunning {
		return errors.New("generation attempt must start in running state")
	}
	attempt.ReasoningConfigJSON = validJSONOrDefault(attempt.ReasoningConfigJSON, "{}")
	if attempt.CreatedAt.IsZero() {
		attempt.CreatedAt = time.Now().UTC()
	}
	_, err := db.conn.Exec(`INSERT INTO generation_attempts (id,run_id,sequence,provider,requested_model,reasoning_config_json,requested_streaming,status,retry_reason,created_at) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		attempt.ID, attempt.RunID, attempt.Sequence, attempt.Provider, attempt.RequestedModel, attempt.ReasoningConfigJSON, boolInt(attempt.RequestedStreaming), attempt.Status, attempt.RetryReason, attempt.CreatedAt)
	return err
}

func (db *DB) FinishGenerationAttempt(id string, completion GenerationCompletion) error {
	if err := validateGenerationTerminalStatus(completion.Status); err != nil {
		return err
	}
	now := time.Now().UTC()
	result, err := db.conn.Exec(`UPDATE generation_attempts SET status=?,resolved_model=?,observed_streaming=?,ttft_ms=?,duration_ms=?,input_tokens=?,output_tokens=?,reasoning_tokens=?,cached_input_tokens=?,total_tokens=?,cost_usd=?,error_class=?,error_summary=?,finished_at=? WHERE id=? AND status='running'`,
		completion.Status, completion.ResolvedModel, boolInt(completion.ObservedStreaming), completion.TTFTMs, completion.DurationMs, completion.Usage.InputTokens, completion.Usage.OutputTokens, completion.Usage.ReasoningTokens, completion.Usage.CachedInputTokens, completion.Usage.TotalTokens, completion.Usage.CostUSD, completion.ErrorClass, completion.ErrorSummary, now, id)
	if err != nil {
		return err
	}
	if rows, _ := result.RowsAffected(); rows != 1 {
		return fmt.Errorf("generation attempt %s is missing or already finished", id)
	}
	return nil
}

func (db *DB) FinishGenerationRun(id string, completion GenerationCompletion) error {
	if err := validateGenerationTerminalStatus(completion.Status); err != nil {
		return err
	}
	now := time.Now().UTC()
	result, err := db.conn.Exec(`UPDATE generation_runs SET status=?,observed_streaming=?,ttft_ms=?,duration_ms=?,input_tokens=?,output_tokens=?,reasoning_tokens=?,cached_input_tokens=?,total_tokens=?,cost_usd=?,error_class=?,finished_at=? WHERE id=? AND status='running'`,
		completion.Status, boolInt(completion.ObservedStreaming), completion.TTFTMs, completion.DurationMs, completion.Usage.InputTokens, completion.Usage.OutputTokens, completion.Usage.ReasoningTokens, completion.Usage.CachedInputTokens, completion.Usage.TotalTokens, completion.Usage.CostUSD, completion.ErrorClass, now, id)
	if err != nil {
		return err
	}
	if rows, _ := result.RowsAffected(); rows != 1 {
		return fmt.Errorf("generation run %s is missing or already finished", id)
	}
	return nil
}

func (db *DB) AppendGenerationEvent(runID, attemptID, eventType, payloadJSON string) error {
	if strings.TrimSpace(runID) == "" || strings.TrimSpace(eventType) == "" {
		return errors.New("generation event run and type are required")
	}
	payloadJSON = validJSONOrDefault(payloadJSON, "{}")
	_, err := db.conn.Exec(`INSERT INTO generation_events (run_id,attempt_id,event_type,payload_json) VALUES (?,NULLIF(?,''),?,?)`, runID, attemptID, eventType, payloadJSON)
	return err
}

func (db *DB) BindGenerationRunMessage(runID string, messageID int64) error {
	if messageID < 1 {
		return errors.New("positive message id is required")
	}
	result, err := db.conn.Exec(`UPDATE generation_runs SET message_id=? WHERE id=? AND (message_id IS NULL OR message_id=?)`, messageID, runID, messageID)
	if err != nil {
		return err
	}
	if rows, _ := result.RowsAffected(); rows != 1 {
		return fmt.Errorf("generation run %s is missing or bound to another message", runID)
	}
	return nil
}

func validateGenerationTerminalStatus(status string) error {
	switch status {
	case GenerationStatusSucceeded, GenerationStatusFailed, GenerationStatusCancelled:
		return nil
	default:
		return fmt.Errorf("invalid terminal generation status %q", status)
	}
}

func validJSONOrDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	if json.Valid([]byte(value)) {
		return value
	}
	return fallback
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
