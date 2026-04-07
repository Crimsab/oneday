package ai

import (
	"context"
	"fmt"
	"testing"
)

// mockProvider is a test double for ai.Provider.
type mockProvider struct {
	name    string
	err     error
	content string
}

func (m *mockProvider) Name() string { return m.name }

func (m *mockProvider) Complete(_ context.Context, _ Request) (Response, error) {
	if m.err != nil {
		return Response{}, m.err
	}
	return Response{
		Content:   m.content,
		Model:     "test-model",
		Provider:  m.name,
		LatencyMs: 100,
	}, nil
}

func TestRouterFirstProviderSucceeds(t *testing.T) {
	r, err := NewRouter([]Provider{
		&mockProvider{name: "primary", content: "hello"},
		&mockProvider{name: "fallback", content: "world"},
	})
	if err != nil {
		t.Fatal(err)
	}

	resp, err := r.Complete(context.Background(), Request{
		Messages: []Message{{Role: RoleUser, Content: "test"}},
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Provider != "primary" {
		t.Errorf("Provider = %q, want %q", resp.Provider, "primary")
	}
	if resp.Content != "hello" {
		t.Errorf("Content = %q, want %q", resp.Content, "hello")
	}
}

func TestRouterFallsBack(t *testing.T) {
	r, err := NewRouter([]Provider{
		&mockProvider{name: "primary", err: fmt.Errorf("connection refused")},
		&mockProvider{name: "fallback", content: "recovered"},
	})
	if err != nil {
		t.Fatal(err)
	}

	resp, err := r.Complete(context.Background(), Request{
		Messages: []Message{{Role: RoleUser, Content: "test"}},
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Provider != "fallback" {
		t.Errorf("Provider = %q, want %q", resp.Provider, "fallback")
	}
	if resp.Content != "recovered" {
		t.Errorf("Content = %q, want %q", resp.Content, "recovered")
	}
}

func TestRouterAllFail(t *testing.T) {
	r, err := NewRouter([]Provider{
		&mockProvider{name: "p1", err: fmt.Errorf("fail 1")},
		&mockProvider{name: "p2", err: fmt.Errorf("fail 2")},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = r.Complete(context.Background(), Request{
		Messages: []Message{{Role: RoleUser, Content: "test"}},
	})
	if err == nil {
		t.Error("expected error when all providers fail")
	}
}

func TestRouterNoProviders(t *testing.T) {
	_, err := NewRouter([]Provider{})
	if err == nil {
		t.Error("expected error for empty provider list")
	}
}

func TestRouterProviderNames(t *testing.T) {
	r, _ := NewRouter([]Provider{
		&mockProvider{name: "a"},
		&mockProvider{name: "b"},
		&mockProvider{name: "c"},
	})
	names := r.ProviderNames()
	if len(names) != 3 {
		t.Fatalf("ProviderNames length = %d, want 3", len(names))
	}
	if names[0] != "a" || names[1] != "b" || names[2] != "c" {
		t.Errorf("ProviderNames = %v, want [a b c]", names)
	}
}
