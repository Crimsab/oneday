package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/config"
	"github.com/crimsab/oneday/internal/game/contracts"
	gameservice "github.com/crimsab/oneday/internal/game/service"
	"github.com/crimsab/oneday/internal/storage"
)

type gatewayTurnResponse struct {
	Events []contracts.TurnEvent `json:"events,omitempty"`
	Error  string                `json:"error,omitempty"`
}

func runGatewayTurn(ctx context.Context, cfg config.Config, db *storage.DB, router *ai.Router, in io.Reader, out io.Writer) error {
	var req contracts.SubmitActionRequest
	if err := json.NewDecoder(in).Decode(&req); err != nil {
		return writeGatewayTurnError(out, fmt.Errorf("invalid gateway-turn JSON: %w", err))
	}

	turns := gameservice.NewInProcessTurnService(cfg, db, router)
	stream, err := turns.SubmitAction(ctx, req)
	if err != nil {
		return writeGatewayTurnError(out, err)
	}

	events := make([]contracts.TurnEvent, 0, 8)
	for event := range stream {
		events = append(events, event)
	}
	if err := json.NewEncoder(out).Encode(gatewayTurnResponse{Events: events}); err != nil {
		return fmt.Errorf("writing gateway-turn response: %w", err)
	}
	return nil
}

func writeGatewayTurnError(out io.Writer, err error) error {
	_ = json.NewEncoder(out).Encode(gatewayTurnResponse{Error: err.Error()})
	return err
}
