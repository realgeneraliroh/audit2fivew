// Copyright (c) 2026 J
// All rights reserved.
// Released under the GNU Affero General Public License v3.0.

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"audit2fivew/internal/reader"
	"audit2fivew/internal/store"
	"audit2fivew/internal/tui"
)

// Version is set at build time via -ldflags "-X main.Version=...".
var Version = "dev"

func main() {
	var (
		logPath string
		dbPath  string
		showVer bool
	)

	home, _ := os.UserHomeDir()
	defaultDB := filepath.Join(home, ".local", "share", "audit2fivew", "denials.db")

	flag.StringVar(&logPath, "log", "/var/log/audit/audit.log",
		"path to the audit log file")
	flag.StringVar(&dbPath, "db", defaultDB,
		"path to the SQLite database")
	flag.BoolVar(&showVer, "version", false,
		"print version and exit")
	flag.Parse()

	if showVer {
		fmt.Printf("audit2fivew %s\n", Version)
		os.Exit(0)
	}

	// Ensure the database directory exists.
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o750); err != nil {
		log.Fatalf("cannot create database directory: %v", err)
	}

	// Open the local denial store (creates the DB if needed).
	s, err := store.New(dbPath)
	if err != nil {
		log.Fatalf("cannot open database: %v", err)
	}
	defer s.Close()

	// Ingest denials from the audit log (best-effort).
	if denials, err := reader.Read(logPath); err == nil {
		for _, d := range denials {
			_ = s.Upsert(d)
		}
	} else {
		fmt.Fprintf(os.Stderr, "audit2fivew: warning: %v\n", err)
	}

	// Also try ausearch for additional entries.
	if denials, err := reader.ReadAusearch(); err == nil {
		for _, d := range denials {
			_ = s.Upsert(d)
		}
	}

	// Load the full history into the TUI.
	all, err := s.List()
	if err != nil {
		log.Fatalf("cannot load denials: %v", err)
	}

	p := tea.NewProgram(
		tui.NewModel(all, s, logPath),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := p.Run(); err != nil {
		log.Fatalf("TUI error: %v", err)
	}
}

