package providers

import (
	"bytes"
	"strings"
	"testing"
)

func TestReadResponseBodyAcceptsBoundedPayload(t *testing.T) {
	want := []byte(`{"ok":true}`)
	got, err := readResponseBody(bytes.NewReader(want))
	if err != nil {
		t.Fatalf("read bounded response: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("response = %q, want %q", got, want)
	}
}

func TestReadResponseBodyRejectsOversizedPayload(t *testing.T) {
	_, err := readResponseBody(strings.NewReader(strings.Repeat("x", int(maxProviderResponseBytes)+1)))
	if err == nil || !strings.Contains(err.Error(), "exceeds 16 MiB") {
		t.Fatalf("expected bounded response error, got %v", err)
	}
}
