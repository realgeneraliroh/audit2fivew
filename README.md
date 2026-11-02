# audit2fivew

A non-destructive **Terminal UI (TUI)** for SELinux AVC denial troubleshooting, built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea).

```
┌─ ☰ AVC Denials ─────────────┬─ ⓘ  Detail ───────────────────────────────┐
│                              │                                           │
│ ▸ 10/31 09:14  httpd_t → …   │ ▸ When                                    │
│   10/30 17:02  sshd_t → …    │   2026-10-31 09:14:22 EDT                 │
│   10/29 11:45  nginx_t → …   │                                           │
│                              │ ▸ Who                                     │
│                              │   Source: system_u:system_r:httpd_t:s0    │
│                              │   Process: httpd (PID 1234)               │
│                              │                                           │
│                              │ ▸ Where                                   │
│                              │   Target: …:object_r:user_home_t:s0       │
│                              │   File: index.html                        │
│                              │                                           │
│                              │ ▸ What                                    │
│                              │   Class:       file                       │
│                              │   Permissions: { read }                   │
│                              │                                           │
│                              │ ▸ Way to Fix                              │
│                              │   allow httpd_t user_home_t:file { … };   │
└──────────────────────────────┴───────────────────────────────────────────┘
 3 denial(s)            q quit │ ↑↓/jk navigate │ tab switch │ / filter
```

## Features

- **Master-detail TUI** — browse AVC denials in a scrollable list; the selected denial is immediately expanded into a human-readable breakdown.
- **When / Who / Where / What / Way to Fix** — every denial is translated from raw audit gibberish into plain-language fields.
- **Auto-generated `.te` rules** — replicates `audit2allow` output with the exact `checkmodule` / `semodule_package` / `semodule` commands you need.
- **SQLite history** — all denials are persisted locally so you never lose track of past failures, even after log rotation.
- **Fully offline** — no network access required. Ever.
- **Static binary** — single `CGO_ENABLED=0` binary with zero runtime dependencies beyond the Go standard library.

## Requirements

- **Go 1.21+** (build time only)
- **Linux** with audit subsystem (`/var/log/audit/audit.log`)
- Read access to the audit log (typically requires `root` or the `audit` group)

## Quick Start

```bash
# Build for current host
make build

# Cross-compile static Linux ELF binary from macOS (x86_64 / amd64)
make build-linux-amd64

# Cross-compile for Linux ARM64 (aarch64)
make build-linux-arm64

# Run on the Linux host (needs read access to the audit log)
sudo ./audit2fivew

# Or point at a different log file
./audit2fivew -log /path/to/audit.log

# Custom database location
./audit2fivew -db /tmp/test-denials.db
```

## Keyboard Shortcuts

| Key           | Action                          |
|---------------|---------------------------------|
| `↑` / `k`    | Move up in list / scroll detail |
| `↓` / `j`    | Move down / scroll detail       |
| `Tab`         | Switch focus between panes      |
| `/`           | Start filtering                 |
| `Esc`         | Clear filter                    |
| `r`           | Refresh from audit log          |
| `g` / `Home`  | Jump to top                     |
| `G` / `End`   | Jump to bottom                  |
| `PgUp`/`PgDn` | Page up / down                  |
| `q`           | Quit                            |

## Database

Denials are stored in a SQLite database at:

```
~/.local/share/audit2fivew/denials.db
```

Override with `-db /path/to/file.db`. The database is created automatically on first run.

Duplicate denials (same source context, target context, class, permissions, and command) are de-duplicated — the `count` and `last_seen` fields are updated instead of creating new rows.

## RPM Packaging (EPEL)

```bash
# Vendor dependencies, build binary, create RPM
make rpm
```

This produces an RPM under `rpmbuild/RPMS/` that installs the static binary to `/usr/bin/audit2fivew`. The spec file targets RHEL 8/9 and Fedora via EPEL.

### Build Dependencies

- `golang >= 1.21`
- `make`
- `rpm-build` (for `make rpm`)

### Runtime Dependencies

- `audit` (for `ausearch`)
- `policycoreutils-python-utils` (recommended, for `audit2allow` comparison)

## How It Works

1. **Ingest** — reads `/var/log/audit/audit.log` line-by-line and also runs `ausearch -m avc --raw` (if available) to capture AVC denial entries.
2. **Parse** — regex-based parser extracts timestamp, serial, permissions, class, source/target contexts, process info, and file paths.
3. **Store** — parsed denials are upserted into SQLite with content-hash-based deduplication.
4. **Display** — Bubble Tea TUI renders a master-detail layout with Lipgloss styling.
5. **Fix generation** — for each denial, generates the exact Type Enforcement (`.te`) module and `checkmodule` / `semodule` commands needed to allow the action.

## Documentation

API and package documentation is generated directly from source code comments (Go doc comments) using [`doc2go`](https://github.com/abhinav/doc2go).

### Local Generation

```bash
# Install doc2go once
go install go.abhg.dev/doc2go@latest

# Build HTML docs in public/
make docs

# Or build and serve locally at http://localhost:8080
make docs-serve
```

You can also browse CLI documentation in terminal:

```bash
go doc ./...
go doc audit2fivew/internal/avc Denial
```

### Automated CI Documentation (Forgejo / GitHub)

Workflows are included for both CI platforms:
- **Forgejo Actions**: [`.forgejo/workflows/docs.yml`](.forgejo/workflows/docs.yml) builds docs in a container on push to `main` and publishes them as an artifact / Pages.
- **GitHub Actions**: [`.github/workflows/docs.yml`](.github/workflows/docs.yml) builds docs on push to `main` and deploys directly to GitHub Pages.

## License

`audit2fivew` is released under the **GNU Affero General Public License v3.0 or later** ([`AGPL-3.0-or-later`](LICENSE)).

Bundled dependencies are distributed under their respective upstream permissive licenses:
- **MIT**: [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Lipgloss](https://github.com/charmbracelet/lipgloss), [Bubbles](https://github.com/charmbracelet/bubbles), and related terminal formatting utilities
- **BSD-3-Clause**: [`modernc.org/sqlite`](https://gitlab.com/cznic/sqlite), [`modernc.org/libc`](https://gitlab.com/cznic/libc), `google/uuid`, `golang.org/x/sys`
- **blessing** (Public Domain dedication): [SQLite](https://sqlite.org/copyright.html)

