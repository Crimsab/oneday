package gatewayprotocol

import (
	"encoding/json"
	"testing"
)

func TestFailureSerializesVersionedTypedError(t *testing.T) {
	response := TurnResponse{
		ResponseMeta: Failure("turn_failed", "stale revision"),
		Error:        "stale revision",
	}
	payload, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if decoded["protocol_version"] != float64(Version) {
		t.Fatalf("protocol_version = %v, want %d", decoded["protocol_version"], Version)
	}
	detail, ok := decoded["error_detail"].(map[string]any)
	if !ok {
		t.Fatalf("error_detail missing from %s", payload)
	}
	if detail["code"] != "turn_failed" || detail["message"] != "stale revision" {
		t.Fatalf("unexpected error_detail: %#v", detail)
	}
	if decoded["error"] != "stale revision" {
		t.Fatalf("legacy error missing from %s", payload)
	}
}
