package ai

import (
	"context"
	"fmt"
	"testing"
)

// mockStreamProvider is a Provider + StreamProvider test double.
type mockStreamProvider struct {
	name    string
	content string
	err     error    // error returned from Stream
	chunks  []string // if set, send these chunks (instead of one big content)
}

func (m *mockStreamProvider) Name() string { return m.name }

func (m *mockStreamProvider) Complete(_ context.Context, _ Request) (Response, error) {
	if m.err != nil {
		return Response{}, m.err
	}
	return Response{Content: m.content, Provider: m.name}, nil
}

func (m *mockStreamProvider) Stream(_ context.Context, _ Request) (<-chan StreamChunk, error) {
	if m.err != nil {
		return nil, m.err
	}
	texts := m.chunks
	if len(texts) == 0 {
		texts = []string{m.content}
	}
	ch := make(chan StreamChunk, len(texts)+1)
	go func() {
		defer close(ch)
		for _, c := range texts {
			ch <- StreamChunk{Content: c}
		}
		ch <- StreamChunk{Done: true}
	}()
	return ch, nil
}

// collectChunks drains a StreamChunk channel and returns concatenated content.
func collectChunks(ch <-chan StreamChunk) (string, error) {
	var buf string
	for chunk := range ch {
		if chunk.Error != nil {
			return buf, chunk.Error
		}
		if chunk.Done {
			break
		}
		buf += chunk.Content
	}
	return buf, nil
}

// --- Router.Stream tests ---

func TestRouterStreamUsesStreamProvider(t *testing.T) {
	sp := &mockStreamProvider{name: "streaming", content: "streamed text"}
	r, err := NewRouter([]Provider{sp})
	if err != nil {
		t.Fatal(err)
	}

	ch, providerName, err := r.Stream(context.Background(), Request{
		Messages: []Message{{Role: RoleUser, Content: "test"}},
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if providerName != "streaming" {
		t.Errorf("providerName = %q, want %q", providerName, "streaming")
	}

	text, err := collectChunks(ch)
	if err != nil {
		t.Fatalf("collecting chunks: %v", err)
	}
	if text != "streamed text" {
		t.Errorf("content = %q, want %q", text, "streamed text")
	}
}

func TestRouterStreamMultipleChunks(t *testing.T) {
	sp := &mockStreamProvider{
		name:   "streaming",
		chunks: []string{"Hello", ", ", "world", "!"},
	}
	r, _ := NewRouter([]Provider{sp})

	ch, _, err := r.Stream(context.Background(), Request{
		Messages: []Message{{Role: RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	text, err := collectChunks(ch)
	if err != nil {
		t.Fatal(err)
	}
	if text != "Hello, world!" {
		t.Errorf("content = %q, want %q", text, "Hello, world!")
	}
}

func TestRouterStreamFallsBackToComplete(t *testing.T) {
	// Non-streaming provider: only implements Provider, not StreamProvider.
	plain := &mockProvider{name: "plain", content: "fallback text"}
	r, _ := NewRouter([]Provider{plain})

	ch, providerName, err := r.Stream(context.Background(), Request{
		Messages: []Message{{Role: RoleUser, Content: "test"}},
	})
	if err != nil {
		t.Fatalf("Stream fallback: %v", err)
	}
	if providerName != "plain" {
		t.Errorf("providerName = %q, want %q", providerName, "plain")
	}
	text, _ := collectChunks(ch)
	if text != "fallback text" {
		t.Errorf("content = %q, want %q", text, "fallback text")
	}
}

func TestRouterStreamPrefersStreamProviderOverPlain(t *testing.T) {
	plain := &mockProvider{name: "plain", content: "plain"}
	sp := &mockStreamProvider{name: "streaming", content: "from-stream"}
	// streaming provider listed second — it should still win the first pass.
	r, _ := NewRouter([]Provider{plain, sp})

	ch, providerName, err := r.Stream(context.Background(), Request{
		Messages: []Message{{Role: RoleUser, Content: "test"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if providerName != "streaming" {
		t.Errorf("providerName = %q, want %q", providerName, "streaming")
	}
	text, _ := collectChunks(ch)
	if text != "from-stream" {
		t.Errorf("content = %q, want %q", text, "from-stream")
	}
}

func TestRouterStreamFallsBackWhenStreamProviderFails(t *testing.T) {
	failing := &mockStreamProvider{name: "broken", err: fmt.Errorf("timeout")}
	good := &mockStreamProvider{name: "good", content: "recovered"}
	r, _ := NewRouter([]Provider{failing, good})

	ch, providerName, err := r.Stream(context.Background(), Request{
		Messages: []Message{{Role: RoleUser, Content: "test"}},
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if providerName != "good" {
		t.Errorf("providerName = %q, want %q", providerName, "good")
	}
	text, _ := collectChunks(ch)
	if text != "recovered" {
		t.Errorf("content = %q, want %q", text, "recovered")
	}
}

func TestRouterStreamAllFail(t *testing.T) {
	r, _ := NewRouter([]Provider{
		&mockStreamProvider{name: "s1", err: fmt.Errorf("err1")},
		&mockProvider{name: "p1", err: fmt.Errorf("err2")},
	})

	_, _, err := r.Stream(context.Background(), Request{
		Messages: []Message{{Role: RoleUser, Content: "test"}},
	})
	if err == nil {
		t.Error("expected error when all providers fail")
	}
}
