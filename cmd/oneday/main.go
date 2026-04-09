package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/crimsab/oneday/internal/aifactory"
	"github.com/crimsab/oneday/internal/config"
	"github.com/crimsab/oneday/internal/storage"
	"github.com/crimsab/oneday/internal/tui"
)

func main() {
	// Load config
	cfg, err := config.Load(resolveConfigPath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Open database
	dbPath := filepath.Join(cfg.DataDir, "oneday.db")
	db, err := storage.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Create AI router
	router, err := aifactory.NewRouterFromConfig(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating AI router: %v\n", err)
		os.Exit(1)
	}

	// Start TUI
	app := tui.New(cfg, db, router)
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}

func resolveConfigPath() string {
	const configName = "config.yaml"

	if _, err := os.Stat(configName); err == nil {
		return configName
	}

	exePath, err := os.Executable()
	if err != nil {
		return configName
	}

	exeConfig := filepath.Join(filepath.Dir(exePath), configName)
	if _, err := os.Stat(exeConfig); err == nil {
		return exeConfig
	}

	return configName
}
