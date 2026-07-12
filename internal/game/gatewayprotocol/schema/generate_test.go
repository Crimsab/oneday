package schema

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCheckedInSchemaIsCurrent(t *testing.T) {
	generated, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	_, sourceFile, _, _ := runtime.Caller(0)
	contractPath := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "../../../../contracts/gateway-v1.schema.json"))
	checkedIn, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(generated, checkedIn) {
		t.Fatal("contracts/gateway-v1.schema.json is stale; run go run ./cmd/oneday-gateway-schema")
	}
}
