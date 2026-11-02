// Copyright (c) 2026 J
// All rights reserved.
// Released under the GNU Affero General Public License v3.0.

// Package tui implements the Bubble Tea terminal user interface for
// audit2fivew, providing a master-detail view of AVC denials.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"audit2fivew/internal/avc"
	"audit2fivew/internal/reader"
	"audit2fivew/internal/store"
)

// pane identifies which panel currently has keyboard focus.
type pane int

const (
	paneList   pane = iota
	paneDetail
)

// refreshMsg is sent when background log re-ingestion completes.
type refreshMsg struct {
	denials []*avc.Denial
}

// Model is the top-level Bubble Tea model.
type Model struct {
	denials  []*avc.Denial // full set from store
	filtered []*avc.Denial // after search filter
	store    *store.Store
	logPath  string

	cursor int // selected index in filtered
	offset int // scroll offset for list pane

	focus pane

	viewport viewport.Model

	width  int
	height int

	filterText string
	filtering  bool // true while the user is typing a filter

	ready bool
}

// NewModel creates a Model pre-loaded with the given denials.
func NewModel(denials []*avc.Denial, s *store.Store, logPath string) Model {
	return Model{
		denials:  denials,
		filtered: denials,
		store:    s,
		logPath:  logPath,
		focus:    paneList,
	}
}

// ── Bubble Tea interface ────────────────────────────────────────────

// Init initializes the Bubble Tea runtime. Returns nil as no initial command is needed.
func (m Model) Init() tea.Cmd { return nil }

// Update handles incoming Bubble Tea messages (keys, window resize, async refresh).
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// ── Window resize ───────────────────────────────────────────
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		vpW := m.detailWidth() - 6 // padding + border
		vpH := m.height - 7
		if vpW < 10 {
			vpW = 10
		}
		if vpH < 4 {
			vpH = 4
		}

		if !m.ready {
			m.viewport = viewport.New(vpW, vpH)
			m.viewport.SetContent(m.renderDetail())
			m.ready = true
		} else {
			m.viewport.Width = vpW
			m.viewport.Height = vpH
			m.viewport.SetContent(m.renderDetail())
		}
		return m, nil

	// ── Keyboard ────────────────────────────────────────────────
	case tea.KeyMsg:
		return m.handleKey(msg)

	// ── Mouse (forwarded to viewport when detail is focused) ───
	case tea.MouseMsg:
		if m.focus == paneDetail {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}

	// ── Async refresh result ────────────────────────────────────
	case refreshMsg:
		m.denials = msg.denials
		m.applyFilter()
		m.clampCursor()
		m.viewport.SetContent(m.renderDetail())
		return m, nil
	}

	return m, nil
}

// handleKey processes all keyboard input.
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// ── Filter input mode ───────────────────────────────────────
	if m.filtering {
		switch key {
		case "enter", "esc":
			m.filtering = false
		case "backspace":
			if len(m.filterText) > 0 {
				m.filterText = m.filterText[:len(m.filterText)-1]
				m.applyFilter()
				m.clampCursor()
				m.viewport.SetContent(m.renderDetail())
			}
		default:
			if len(key) == 1 || key == " " {
				m.filterText += key
				m.applyFilter()
				m.clampCursor()
				m.viewport.SetContent(m.renderDetail())
			}
		}
		return m, nil
	}

	// ── Global keys ─────────────────────────────────────────────
	switch key {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "tab":
		if m.focus == paneList {
			m.focus = paneDetail
		} else {
			m.focus = paneList
		}
		return m, nil
	case "/":
		m.filtering = true
		m.filterText = ""
		return m, nil
	case "esc":
		if m.filterText != "" {
			m.filterText = ""
			m.applyFilter()
			m.clampCursor()
			m.viewport.SetContent(m.renderDetail())
		}
		return m, nil
	case "r":
		return m, m.refreshCmd()
	}

	// ── Pane-specific keys ──────────────────────────────────────
	if m.focus == paneList {
		return m.handleListKey(key)
	}

	// Detail pane: handle g/G ourselves, delegate the rest to viewport.
	switch key {
	case "g":
		m.viewport.GotoTop()
		return m, nil
	case "G":
		m.viewport.GotoBottom()
		return m, nil
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// handleListKey navigates the denial list.
func (m Model) handleListKey(key string) (tea.Model, tea.Cmd) {
	moved := false
	switch key {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			moved = true
		}
	case "down", "j":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
			moved = true
		}
	case "home", "g":
		m.cursor = 0
		moved = true
	case "end", "G":
		m.cursor = max(len(m.filtered)-1, 0)
		moved = true
	case "pgup":
		m.cursor -= m.visibleListRows()
		if m.cursor < 0 {
			m.cursor = 0
		}
		moved = true
	case "pgdown":
		m.cursor += m.visibleListRows()
		m.cursor = min(m.cursor, max(len(m.filtered)-1, 0))
		moved = true
	}

	if moved {
		m.adjustScroll()
		m.viewport.SetContent(m.renderDetail())
		m.viewport.GotoTop()
	}
	return m, nil
}

// ── Commands ────────────────────────────────────────────────────────

// refreshCmd re-reads the audit log and ausearch, upserts into the
// store, and returns the updated list.
func (m Model) refreshCmd() tea.Cmd {
	s := m.store
	logPath := m.logPath
	return func() tea.Msg {
		if denials, err := reader.Read(logPath); err == nil {
			for _, d := range denials {
				_ = s.Upsert(d)
			}
		}
		if denials, err := reader.ReadAusearch(); err == nil {
			for _, d := range denials {
				_ = s.Upsert(d)
			}
		}
		all, err := s.List()
		if err != nil {
			return nil
		}
		return refreshMsg{denials: all}
	}
}

// ── Filtering ───────────────────────────────────────────────────────

func (m *Model) applyFilter() {
	if m.filterText == "" {
		m.filtered = m.denials
		return
	}
	q := strings.ToLower(m.filterText)
	var result []*avc.Denial
	for _, d := range m.denials {
		if strings.Contains(strings.ToLower(d.SContext), q) ||
			strings.Contains(strings.ToLower(d.TContext), q) ||
			strings.Contains(strings.ToLower(d.Comm), q) ||
			strings.Contains(strings.ToLower(d.TClass), q) ||
			strings.Contains(strings.ToLower(d.Name), q) ||
			strings.Contains(strings.ToLower(d.Path), q) {
			result = append(result, d)
		}
	}
	m.filtered = result
}

// ── Layout helpers ──────────────────────────────────────────────────

func (m Model) listWidth() int {
	w := int(float64(m.width) * 0.38)
	if w < 20 {
		w = 20
	}
	return w
}

func (m Model) detailWidth() int {
	return m.width - m.listWidth()
}

func (m Model) visibleListRows() int {
	h := m.height - 6 // borders + title + status
	if h < 1 {
		h = 1
	}
	return h
}

func (m *Model) adjustScroll() {
	vis := m.visibleListRows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+vis {
		m.offset = m.cursor - vis + 1
	}
}

func (m *Model) clampCursor() {
	if m.cursor >= len(m.filtered) {
		m.cursor = max(len(m.filtered)-1, 0)
	}
}

// ── View rendering ──────────────────────────────────────────────────

// View renders the master-detail UI including left list, right detail, and footer status bar.
func (m Model) View() string {
	if !m.ready {
		return "\n  Initializing…"
	}

	left := m.renderListPane()
	right := m.renderDetailPane()
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	status := m.renderStatusBar()

	return lipgloss.JoinVertical(lipgloss.Left, body, status)
}

// ── List pane ───────────────────────────────────────────────────────

func (m Model) renderListPane() string {
	lw := m.listWidth()
	lh := m.height - 3

	title := titleStyle.Render(" ☰ AVC Denials ")

	var rows []string
	vis := m.visibleListRows()
	end := m.offset + vis
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	innerW := lw - 4 // border + padding
	if innerW < 10 {
		innerW = 10
	}

	for i := m.offset; i < end; i++ {
		d := m.filtered[i]
		ts := d.Timestamp.Format("01/02 15:04")
		line := fmt.Sprintf("%s  %s → %s", ts, d.SType(), d.TType())
		if d.Count > 1 {
			line += fmt.Sprintf(" (%d×)", d.Count)
		}
		if len(line) > innerW {
			line = line[:innerW-1] + "…"
		}

		if i == m.cursor {
			rows = append(rows, selectedItemStyle.Width(innerW).Render(line))
		} else {
			style := normalItemStyle
			if d.Permissive {
				style = style.Copy().Foreground(lipgloss.Color("#FFA500"))
			}
			rows = append(rows, style.Width(innerW).Render(line))
		}
	}

	if len(m.filtered) == 0 {
		rows = append(rows, helpStyle.Render("  No denials found"))
		rows = append(rows, helpStyle.Render("  Press r to refresh"))
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		append([]string{title, ""}, rows...)...)

	ps := listPaneStyle
	if m.focus == paneList {
		ps = activeListPaneStyle
	}
	return ps.Width(lw).Height(lh).Render(content)
}

// ── Detail pane ─────────────────────────────────────────────────────

func (m Model) renderDetailPane() string {
	dw := m.detailWidth()
	dh := m.height - 3

	title := titleStyle.Render(" ⓘ  Detail ")

	scrollPct := ""
	if m.viewport.TotalLineCount() > m.viewport.Height {
		scrollPct = fmt.Sprintf(" %d%%", int(m.viewport.ScrollPercent()*100))
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		title+helpStyle.Render(scrollPct), "", m.viewport.View())

	ps := detailPaneStyle
	if m.focus == paneDetail {
		ps = activeDetailPaneStyle
	}
	return ps.Width(dw).Height(dh).Render(content)
}

// renderDetail builds the formatted detail text for the currently
// selected denial: When, Who, Where, What, Way to Fix.
func (m Model) renderDetail() string {
	if len(m.filtered) == 0 || m.cursor >= len(m.filtered) {
		return helpStyle.Render(
			"No denial selected.\n\n" +
				"Press r to refresh from the audit log.\n" +
				"Press / to search.")
	}

	d := m.filtered[m.cursor]
	dw := m.detailWidth() - 10
	if dw < 30 {
		dw = 30
	}

	var b strings.Builder

	// ── When ────────────────────────────────────────────────────
	b.WriteString(labelStyle.Render("▸ When") + "\n")
	b.WriteString(fmt.Sprintf("  %s\n",
		valueStyle.Render(d.Timestamp.Format("2006-01-02 15:04:05 MST"))))
	if d.Count > 1 {
		b.WriteString(fmt.Sprintf("  Seen %s times  (first: %s, last: %s)\n",
			valueStyle.Bold(true).Render(fmt.Sprintf("%d", d.Count)),
			d.FirstSeen.Format("2006-01-02 15:04"),
			d.LastSeen.Format("2006-01-02 15:04")))
	}
	b.WriteString("\n")

	// ── Who ─────────────────────────────────────────────────────
	b.WriteString(labelStyle.Render("▸ Who") + "\n")
	b.WriteString(fmt.Sprintf("  Source:   %s\n", valueStyle.Render(d.SContext)))
	if d.Comm != "" {
		proc := valueStyle.Bold(true).Render(d.Comm)
		if d.PID > 0 {
			proc += fmt.Sprintf(" (PID %d)", d.PID)
		}
		b.WriteString(fmt.Sprintf("  Process:  %s\n", proc))
	}
	b.WriteString("\n")

	// ── Where ───────────────────────────────────────────────────
	b.WriteString(labelStyle.Render("▸ Where") + "\n")
	b.WriteString(fmt.Sprintf("  Target:   %s\n", valueStyle.Render(d.TContext)))
	if d.Path != "" {
		b.WriteString(fmt.Sprintf("  Path:     %s\n", valueStyle.Bold(true).Render(d.Path)))
	} else if d.Name != "" {
		b.WriteString(fmt.Sprintf("  File:     %s\n", valueStyle.Bold(true).Render(d.Name)))
	}
	if d.Dev != "" {
		b.WriteString(fmt.Sprintf("  Device:   %s", d.Dev))
		if d.Ino != "" {
			b.WriteString(fmt.Sprintf("  inode: %s", d.Ino))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// ── What ────────────────────────────────────────────────────
	b.WriteString(labelStyle.Render("▸ What") + "\n")
	b.WriteString(fmt.Sprintf("  Class:        %s\n",
		valueStyle.Bold(true).Render(d.TClass)))
	b.WriteString(fmt.Sprintf("  Permissions:  %s\n",
		valueStyle.Bold(true).Render("{ "+d.PermissionsStr()+" }")))
	if d.Permissive {
		b.WriteString(fmt.Sprintf("  Mode:         %s\n",
			permissiveBadge.Render("PERMISSIVE (would deny in enforcing)")))
	} else {
		b.WriteString(fmt.Sprintf("  Mode:         %s\n",
			enforcingBadge.Render("ENFORCING (access denied)")))
	}
	b.WriteString("\n")

	// ── Way to Fix ──────────────────────────────────────────────
	b.WriteString(labelStyle.Render("▸ Way to Fix") + "\n")
	b.WriteString(helpStyle.Render("  Type Enforcement rule:") + "\n")
	for _, line := range strings.Split(d.GenerateTE(), "\n") {
		b.WriteString("  " + codeStyle.Render(line) + "\n")
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  Build & install commands:") + "\n")
	for _, line := range strings.Split(d.GenerateCommands(), "\n") {
		if line != "" {
			b.WriteString("  " + codeStyle.Render(line) + "\n")
		}
	}
	b.WriteString("\n")

	// ── Raw log ─────────────────────────────────────────────────
	b.WriteString(labelStyle.Render("▸ Raw Log") + "\n")
	for _, chunk := range wrapText(d.RawLog, dw) {
		b.WriteString("  " + helpStyle.Render(chunk) + "\n")
	}

	return b.String()
}

// ── Status bar ──────────────────────────────────────────────────────

func (m Model) renderStatusBar() string {
	left := fmt.Sprintf(" %d denial(s)", len(m.filtered))
	if len(m.filtered) != len(m.denials) {
		left += fmt.Sprintf(" of %d", len(m.denials))
	}
	if m.filtering {
		left += fmt.Sprintf("  filter: %s▏", m.filterText)
	} else if m.filterText != "" {
		left += fmt.Sprintf("  [filter: %s]", m.filterText)
	}

	right := " q quit │ ↑↓/jk navigate │ tab switch │ / filter │ esc clear │ r refresh "

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
		right = ""
	}

	return statusBarStyle.Width(m.width).Render(
		left + strings.Repeat(" ", gap) + right)
}

// ── Utilities ───────────────────────────────────────────────────────

// wrapText hard-wraps s at width characters.
func wrapText(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}
	var lines []string
	for len(s) > width {
		lines = append(lines, s[:width])
		s = s[width:]
	}
	if s != "" {
		lines = append(lines, s)
	}
	return lines
}

