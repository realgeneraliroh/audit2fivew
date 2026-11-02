// Copyright (c) 2026 J
// All rights reserved.
// Released under the GNU Affero General Public License v3.0.

/*
Package main implements audit2fivew, a non-destructive Terminal UI (TUI)
for SELinux AVC denial troubleshooting.

It reads AVC denial events from /var/log/audit/audit.log (or via
ausearch(8)), parses them into structured records, de-duplicates and
persists them in a local SQLite database, and presents them in an
interactive master-detail view built with Bubble Tea.

For every denial the tool generates the Type Enforcement (.te) policy
module and the exact checkmodule / semodule_package / semodule commands
needed to allow the blocked action — essentially what audit2allow does,
but in a fully offline, browsable, persistent interface.

# Usage and Flags

	audit2fivew [flags]

Flags:

	-log string   Path to the audit log file (default "/var/log/audit/audit.log")
	-db  string   Path to the SQLite database (default "~/.local/share/audit2fivew/denials.db")
	-version      Print version and exit

# Quick Start

Build and run on a Linux host with audit log access:

	make build
	sudo ./audit2fivew

Point at a different log file:

	./audit2fivew -log /path/to/audit.log

Use a custom database location:

	./audit2fivew -db /tmp/test-denials.db

# Architecture

The codebase is organised into four internal packages:

  - [audit2fivew/internal/avc] — parsing and representation of AVC denials
  - [audit2fivew/internal/reader] — log file and ausearch ingestion
  - [audit2fivew/internal/store] — SQLite-backed persistence layer
  - [audit2fivew/internal/tui] — Bubble Tea terminal user interface
*/
package main

