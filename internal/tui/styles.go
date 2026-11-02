// Copyright (c) 2026 J
// All rights reserved.
// Released under the GNU Affero General Public License v3.0.

package tui

import "github.com/charmbracelet/lipgloss"

// Adaptive colour pairs that work on both light and dark terminals.
var (
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#585858"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
	danger    = lipgloss.AdaptiveColor{Light: "#FF6347", Dark: "#FF6347"}
	dimText   = lipgloss.AdaptiveColor{Light: "#999999", Dark: "#666666"}

	// ── Pane borders ────────────────────────────────────────────────
	listPaneStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(subtle).
			Padding(0, 1)

	activeListPaneStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(highlight).
				Padding(0, 1)

	detailPaneStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(subtle).
			Padding(1, 2)

	activeDetailPaneStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(highlight).
				Padding(1, 2)

	// ── Title bar ───────────────────────────────────────────────────
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#6124DF")).
			Padding(0, 1).
			Bold(true)

	// ── Detail view ─────────────────────────────────────────────────
	labelStyle = lipgloss.NewStyle().
			Foreground(highlight).
			Bold(true)

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#1A1A1A", Dark: "#FAFAFA"})

	codeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A6E22E"))

	// ── List items ──────────────────────────────────────────────────
	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(highlight).
				Padding(0, 1)

	normalItemStyle = lipgloss.NewStyle().
			Padding(0, 1)

	// ── Footer / help ───────────────────────────────────────────────
	helpStyle = lipgloss.NewStyle().
			Foreground(dimText)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#6124DF")).
			Padding(0, 1)

	// ── Badges ──────────────────────────────────────────────────────
	permissiveBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFA500")).
			Bold(true)

	enforcingBadge = lipgloss.NewStyle().
			Foreground(danger).
			Bold(true)
)

