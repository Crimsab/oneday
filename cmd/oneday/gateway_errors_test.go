package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"github.com/crimsab/oneday/internal/config"
	"github.com/crimsab/oneday/internal/game/contracts"
	"github.com/crimsab/oneday/internal/game/gatewayprotocol"
	gameservice "github.com/crimsab/oneday/internal/game/service"
	"github.com/crimsab/oneday/internal/storage"
)

func TestGatewayCauseCodesCoverInvalidMissingStaleAndInternalErrors(t *testing.T) {
	db, err := storage.Open(filepath.Join(t.TempDir(), "gateway-errors.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	t.Run("invalid request", func(t *testing.T) {
		var output bytes.Buffer
		_ = runGatewayLoad(context.Background(), config.Config{}, db, nil, bytes.NewBufferString("{"), &output)
		response := decodeGatewayLoadError(t, output.Bytes())
		assertGatewayCode(t, response.ResponseMeta, gatewayCodeInvalidRequest)
	})

	t.Run("missing story", func(t *testing.T) {
		response := runGatewayLoadErrorCase(t, db, contracts.BrowserLoadRequest{
			StoryID: "missing-story", SessionID: "session", SaveID: "save",
		})
		assertGatewayCode(t, response.ResponseMeta, gatewayCodeNotFound)
	})

	if _, err := db.Conn().Exec(`INSERT INTO stories(id,name) VALUES('story-errors','Story')`); err != nil {
		t.Fatal(err)
	}
	head, err := db.EnsureStoryTimeline("story-errors")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Conn().Exec(`INSERT INTO world_state(id,story_id) VALUES('world-errors','story-errors')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Conn().Exec(`INSERT INTO sessions(id,story_id,branch_id,source_commit_id) VALUES('session-errors','story-errors',?,?)`, head.Branch.ID, head.Commit.ID); err != nil {
		t.Fatal(err)
	}

	t.Run("missing save", func(t *testing.T) {
		response := runGatewayLoadErrorCase(t, db, contracts.BrowserLoadRequest{
			StoryID: "story-errors", SessionID: "session-errors", SaveID: "missing-save",
		})
		assertGatewayCode(t, response.ResponseMeta, gatewayCodeNotFound)
	})

	t.Run("stale revision", func(t *testing.T) {
		response := runGatewayLoadErrorCase(t, db, contracts.BrowserLoadRequest{
			StoryID: "story-errors", SessionID: "session-errors", SaveID: "missing-save", ClientRevision: 1,
		})
		assertGatewayCode(t, response.ResponseMeta, gatewayCodeStaleRequest)
	})

	t.Run("committed turn retry bypasses stale preflight", func(t *testing.T) {
		if _, err := db.Conn().Exec(
			`INSERT INTO turn_idempotency(story_id,idempotency_key,events_json,status,request_hash) VALUES(?,?,?,?,?)`,
			"story-errors", "retry-key", `[{"type":"turn.committed"}]`, "committed", "existing-hash",
		); err != nil {
			t.Fatal(err)
		}
		turns := gameservice.NewInProcessTurnService(config.Config{}, db, nil)
		err := gatewayTurnPreflight(context.Background(), db, turns, contracts.SubmitActionRequest{
			StoryID: "story-errors", SessionID: "session-errors", ClientRevision: 99,
			IdempotencyKey: "retry-key", Action: contracts.PlayerAction{Text: "look"},
		})
		if err != nil {
			t.Fatalf("committed retry should be classified by the idempotency service: %v", err)
		}
	})

	t.Run("internal redaction", func(t *testing.T) {
		var output bytes.Buffer
		_ = writeGatewayLoadError(&output, errors.New("database /private/path failed"))
		response := decodeGatewayLoadError(t, output.Bytes())
		assertGatewayCode(t, response.ResponseMeta, gatewayCodeInternal)
		if response.Error != "An internal gateway error occurred." {
			t.Fatalf("internal error was not redacted: %q", response.Error)
		}
	})
}

func TestGatewayStreamErrorEventIsCodedAndRedacted(t *testing.T) {
	event, err := contracts.NewTurnEvent(
		"event-error", "story", "session", 1, contracts.EventError,
		map[string]string{"message": "provider token leaked from /private/path"},
	)
	if err != nil {
		t.Fatal(err)
	}

	redacted := gatewayStreamEvent(event)
	var payload map[string]string
	if err := json.Unmarshal(redacted.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["code"] != gatewayCodeInternal || payload["message"] != "An internal gateway error occurred." {
		t.Fatalf("unexpected stream error payload: %#v", payload)
	}
}

func runGatewayLoadErrorCase(t *testing.T, db *storage.DB, request contracts.BrowserLoadRequest) gatewayLoadResponse {
	t.Helper()
	payload, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	_ = runGatewayLoad(context.Background(), config.Config{}, db, nil, bytes.NewReader(payload), &output)
	return decodeGatewayLoadError(t, output.Bytes())
}

func decodeGatewayLoadError(t *testing.T, payload []byte) gatewayLoadResponse {
	t.Helper()
	var response gatewayLoadResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("decode gateway response %q: %v", payload, err)
	}
	return response
}

func assertGatewayCode(t *testing.T, meta gatewayprotocol.ResponseMeta, want string) {
	t.Helper()
	if meta.ErrorDetail == nil || meta.ErrorDetail.Code != want {
		t.Fatalf("error detail = %+v, want code %q", meta.ErrorDetail, want)
	}
}
