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

type gatewayMetaResponse struct {
	Meta  *contracts.BrowserMetaResponse `json:"meta,omitempty"`
	Error string                         `json:"error,omitempty"`
}

type gatewaySaveResponse struct {
	Save  *contracts.BrowserSaveResponse `json:"save,omitempty"`
	Error string                         `json:"error,omitempty"`
}

type gatewayLoadResponse struct {
	Load  *contracts.BrowserLoadResponse `json:"load,omitempty"`
	Error string                         `json:"error,omitempty"`
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

func runGatewayMeta(ctx context.Context, cfg config.Config, db *storage.DB, router *ai.Router, in io.Reader, out io.Writer) error {
	var req contracts.BrowserMetaRequest
	if err := json.NewDecoder(in).Decode(&req); err != nil {
		return writeGatewayMetaError(out, fmt.Errorf("invalid gateway-meta JSON: %w", err))
	}

	turns := gameservice.NewInProcessTurnService(cfg, db, router)
	resp, err := turns.SubmitMeta(ctx, req)
	if err != nil {
		return writeGatewayMetaError(out, err)
	}
	if err := json.NewEncoder(out).Encode(gatewayMetaResponse{Meta: resp}); err != nil {
		return fmt.Errorf("writing gateway-meta response: %w", err)
	}
	return nil
}

func runGatewaySave(ctx context.Context, cfg config.Config, db *storage.DB, router *ai.Router, in io.Reader, out io.Writer) error {
	var req contracts.BrowserSaveRequest
	if err := json.NewDecoder(in).Decode(&req); err != nil {
		return writeGatewaySaveError(out, fmt.Errorf("invalid gateway-save JSON: %w", err))
	}

	turns := gameservice.NewInProcessTurnService(cfg, db, router)
	resp, err := turns.CreateSave(ctx, req)
	if err != nil {
		return writeGatewaySaveError(out, err)
	}
	if err := json.NewEncoder(out).Encode(gatewaySaveResponse{Save: resp}); err != nil {
		return fmt.Errorf("writing gateway-save response: %w", err)
	}
	return nil
}

func runGatewayLoad(ctx context.Context, cfg config.Config, db *storage.DB, router *ai.Router, in io.Reader, out io.Writer) error {
	var req contracts.BrowserLoadRequest
	if err := json.NewDecoder(in).Decode(&req); err != nil {
		return writeGatewayLoadError(out, fmt.Errorf("invalid gateway-load JSON: %w", err))
	}

	turns := gameservice.NewInProcessTurnService(cfg, db, router)
	resp, err := turns.LoadSave(ctx, req)
	if err != nil {
		return writeGatewayLoadError(out, err)
	}
	if err := json.NewEncoder(out).Encode(gatewayLoadResponse{Load: resp}); err != nil {
		return fmt.Errorf("writing gateway-load response: %w", err)
	}
	return nil
}

func writeGatewayTurnError(out io.Writer, err error) error {
	_ = json.NewEncoder(out).Encode(gatewayTurnResponse{Error: err.Error()})
	return err
}

func writeGatewayMetaError(out io.Writer, err error) error {
	_ = json.NewEncoder(out).Encode(gatewayMetaResponse{Error: err.Error()})
	return err
}

func writeGatewaySaveError(out io.Writer, err error) error {
	_ = json.NewEncoder(out).Encode(gatewaySaveResponse{Error: err.Error()})
	return err
}

func writeGatewayLoadError(out io.Writer, err error) error {
	_ = json.NewEncoder(out).Encode(gatewayLoadResponse{Error: err.Error()})
	return err
}
