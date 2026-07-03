package main

import (
	"context"
	"encoding/json"
	"errors"
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
	Save  *contracts.BrowserSaveView `json:"save,omitempty"`
	Error string                     `json:"error,omitempty"`
}

type gatewayLoadResponse struct {
	Save   *contracts.BrowserSaveView `json:"save,omitempty"`
	Legacy bool                       `json:"legacy,omitempty"`
	Error  string                     `json:"error,omitempty"`
}

type gatewayDeleteSaveResponse struct {
	Save  *contracts.BrowserSaveView `json:"save,omitempty"`
	Error string                     `json:"error,omitempty"`
}

type gatewayCommandDescriptorsResponse struct {
	Commands []contracts.CommandDescriptor `json:"commands,omitempty"`
	Error    string                        `json:"error,omitempty"`
}

type gatewayModelSettingsResponse struct {
	Settings  *config.ModelRoutingSettings `json:"settings,omitempty"`
	Error     string                       `json:"error,omitempty"`
	ErrorCode string                       `json:"error_code,omitempty"`
}

func runGatewayCommandDescriptors(out io.Writer) error {
	if err := json.NewEncoder(out).Encode(gatewayCommandDescriptorsResponse{Commands: contracts.CommandDescriptors()}); err != nil {
		return fmt.Errorf("writing gateway-command-descriptors response: %w", err)
	}
	return nil
}

func runGatewayModelSettings(configPath string, out io.Writer) error {
	settings, err := config.ReadModelRoutingSettings(configPath)
	if err != nil {
		return writeGatewayModelSettingsError(out, err)
	}
	if err := json.NewEncoder(out).Encode(gatewayModelSettingsResponse{Settings: &settings}); err != nil {
		return fmt.Errorf("writing gateway-model-settings response: %w", err)
	}
	return nil
}

func runGatewayModelSettingsUpdate(configPath string, in io.Reader, out io.Writer) error {
	var update config.ModelRoutingUpdate
	if err := json.NewDecoder(in).Decode(&update); err != nil {
		return writeGatewayModelSettingsError(out, fmt.Errorf("invalid gateway-model-settings-update JSON: %w", err))
	}
	settings, err := config.UpdateModelRoutingSettings(configPath, update)
	if err != nil {
		return writeGatewayModelSettingsError(out, err)
	}
	if err := json.NewEncoder(out).Encode(gatewayModelSettingsResponse{Settings: &settings}); err != nil {
		return fmt.Errorf("writing gateway-model-settings-update response: %w", err)
	}
	return nil
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
	if err := json.NewEncoder(out).Encode(gatewaySaveResponse{Save: &resp.Save}); err != nil {
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
	if err := json.NewEncoder(out).Encode(gatewayLoadResponse{Save: &resp.Save, Legacy: resp.Legacy}); err != nil {
		return fmt.Errorf("writing gateway-load response: %w", err)
	}
	return nil
}

func runGatewayDeleteSave(ctx context.Context, cfg config.Config, db *storage.DB, router *ai.Router, in io.Reader, out io.Writer) error {
	var req contracts.BrowserDeleteSaveRequest
	if err := json.NewDecoder(in).Decode(&req); err != nil {
		return writeGatewayDeleteSaveError(out, fmt.Errorf("invalid gateway-delete-save JSON: %w", err))
	}

	turns := gameservice.NewInProcessTurnService(cfg, db, router)
	resp, err := turns.DeleteSave(ctx, req)
	if err != nil {
		return writeGatewayDeleteSaveError(out, err)
	}
	if err := json.NewEncoder(out).Encode(gatewayDeleteSaveResponse{Save: &resp.Save}); err != nil {
		return fmt.Errorf("writing gateway-delete-save response: %w", err)
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

func writeGatewayDeleteSaveError(out io.Writer, err error) error {
	_ = json.NewEncoder(out).Encode(gatewayDeleteSaveResponse{Error: err.Error()})
	return err
}

func writeGatewayModelSettingsError(out io.Writer, err error) error {
	code := config.ModelRoutingErrorWrite
	var routingErr config.ModelRoutingError
	if errors.As(err, &routingErr) && routingErr.Code != "" {
		code = routingErr.Code
	} else if err != nil && errors.Is(err, context.Canceled) {
		code = "cancelled"
	}
	_ = json.NewEncoder(out).Encode(gatewayModelSettingsResponse{Error: err.Error(), ErrorCode: code})
	return err
}
