package main

import (
	"encoding/json"
	"testing"

	"github.com/crimsab/oneday/internal/game/contracts"
)

func TestGatewayLoadResponsePreservesSnapshotMetadata(t *testing.T) {
	contract := &contracts.BrowserLoadResponse{
		Save:           contracts.BrowserSaveView{ID: "save-1", Name: "Before the gate"},
		Legacy:         true,
		SnapshotState:  "complete",
		SnapshotDetail: "canonical snapshot restored",
	}

	payload, err := json.Marshal(gatewayLoadResponseFromContract(contract))
	if err != nil {
		t.Fatal(err)
	}
	var decoded gatewayLoadResponse
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Save == nil || decoded.Save.ID != contract.Save.ID {
		t.Fatalf("save lost in bridge response: %+v", decoded.Save)
	}
	if decoded.Legacy != contract.Legacy || decoded.SnapshotState != contract.SnapshotState || decoded.SnapshotDetail != contract.SnapshotDetail {
		t.Fatalf("snapshot metadata drifted: %+v", decoded)
	}
}
