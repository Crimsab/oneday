package service

import (
	"context"

	"github.com/crimsab/oneday/internal/game/contracts"
)

// TurnService is the shared boundary intended for both TUI and browser clients.
type TurnService interface {
	SubmitAction(ctx context.Context, req contracts.SubmitActionRequest) (<-chan contracts.TurnEvent, error)
	SubmitMeta(ctx context.Context, req contracts.BrowserMetaRequest) (*contracts.BrowserMetaResponse, error)
	CreateSave(ctx context.Context, req contracts.BrowserSaveRequest) (*contracts.BrowserSaveResponse, error)
	LoadSave(ctx context.Context, req contracts.BrowserLoadRequest) (*contracts.BrowserLoadResponse, error)
	DeleteSave(ctx context.Context, req contracts.BrowserDeleteSaveRequest) (*contracts.BrowserDeleteSaveResponse, error)
	Snapshot(ctx context.Context, storyID string) (*contracts.GameSnapshot, error)
}
