package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"ghostty-config/internal/ui"
)

type helpEntry struct {
	keys string
	desc string
}

type helpSection struct {
	title   string
	entries []helpEntry
}

var helpSections = []helpSection{
	{
		title: "How it works",
		entries: []helpEntry{
			{"", "Every selection is applied LIVE to Ghostty so you can audition combinations."},
			{"", "Changes are NOT saved until you press Enter."},
			{"", "Pressing Esc / q / Ctrl+C REVERTS to your previous state."},
		},
	},
	{
		title: "Selection screens (Themes / Shaders)",
		entries: []helpEntry{
			{"↑/↓  j/k", "move cursor (also previews on single-select slots)"},
			{"Home/g  End/G", "jump to first / last"},
			{"Tab  Shift+Tab", "switch slot"},
			{"Space  x", "toggle item (multi-select slot only — Shaders › Global)"},
			{"/", "open search"},
			{"Enter  Ctrl+S", "SAVE selection and return to menu"},
			{"Esc  q  Ctrl+C", "CANCEL & REVERT to previous state"},
			{"?", "toggle this help"},
		},
	},
	{
		title: "Menu",
		entries: []helpEntry{
			{"↑/↓  j/k", "navigate"},
			{"1-9", "jump to entry"},
			{"Enter  Space  →  l", "open selected entry"},
			{"?", "toggle this help"},
			{"Esc  q  Ctrl+C", "quit"},
		},
	},
}

type helpModel struct {
	width int
}

func (h helpModel) Update(msg tea.Msg) (helpModel, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return h, nil
	}
	switch key.String() {
	case "?", "esc", "q":
		return h, func() tea.Msg { return ui.CloseHelpMsg{} }
	case "ctrl+c":
		return h, func() tea.Msg { return ui.QuitAppMsg{} }
	}
	return h, nil
}

func (h helpModel) View() string {
	var b strings.Builder
	b.WriteString(ui.RenderBreadcrumb("Ghostty configurator", "Help"))
	b.WriteByte('\n')
	b.WriteByte('\n')

	for _, sec := range helpSections {
		b.WriteString(ui.HelpHeaderStyle.Render(sec.title))
		b.WriteByte('\n')
		keyCol := 0
		for _, e := range sec.entries {
			if len(e.keys) > keyCol {
				keyCol = len(e.keys)
			}
		}
		for _, e := range sec.entries {
			if e.keys == "" {
				b.WriteString("  ")
				b.WriteString(ui.MutedStyle.Render(e.desc))
				b.WriteByte('\n')
				continue
			}
			pad := strings.Repeat(" ", keyCol-len(e.keys))
			b.WriteString("  ")
			b.WriteString(ui.HelpKeyStyle.Render(e.keys))
			b.WriteString(pad)
			b.WriteString("   ")
			b.WriteString(ui.FooterLabelStyle.Render(e.desc))
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
	}

	width := h.width
	if width < 1 {
		width = 80
	}
	footer := ui.RenderFooter(width, []ui.FooterAction{
		{Key: "ESC", Label: "Close help", Variant: ui.FooterCancel},
	})
	b.WriteString(footer)
	b.WriteByte('\n')
	return b.String()
}

