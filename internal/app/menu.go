package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"ghostty-config/internal/ui"
)

type menuEntry struct {
	title       string
	description string
	target      ui.Screen
}

type menuModel struct {
	entries    []menuEntry
	cursor     int
	errorMsg   string
	configPath string
	version    string
	width      int
}

func newMenuModel(configPath, version string) menuModel {
	return menuModel{
		entries: []menuEntry{
			{
				title:       "Shaders",
				description: "Choose Ghostty's custom-shader pipeline: zero or more global GLSL post-processing effects (CRT, glow, animated gradients) plus an optional cursor effect (warp, blaze, tail). Changes preview live; press Enter to save.",
				target:      ui.ScreenShader,
			},
			{
				title:       "Themes",
				description: "Pick a light and a dark color theme for Ghostty from your user themes folder and the themes bundled with the Ghostty app. Changes preview live; press Enter to save.",
				target:      ui.ScreenTheme,
			},
		},
		configPath: configPath,
		version:    version,
	}
}

func (m menuModel) Update(msg tea.Msg) (menuModel, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "ctrl+c", "esc", "q":
		return m, func() tea.Msg { return ui.QuitAppMsg{} }
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case "down", "j":
		if m.cursor < len(m.entries)-1 {
			m.cursor++
		}
		return m, nil
	case "home", "g":
		m.cursor = 0
		return m, nil
	case "end", "G":
		m.cursor = len(m.entries) - 1
		return m, nil
	case "enter", " ", "right", "l":
		target := m.entries[m.cursor].target
		return m, func() tea.Msg { return ui.SwitchScreenMsg{Target: target} }
	}
	switch key.String() {
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		idx := int(key.String()[0] - '1')
		if idx >= 0 && idx < len(m.entries) {
			m.cursor = idx
			target := m.entries[idx].target
			return m, func() tea.Msg { return ui.SwitchScreenMsg{Target: target} }
		}
	}
	return m, nil
}

func (m menuModel) View() string {
	width := m.width
	if width < 1 {
		width = 80
	}

	var b strings.Builder
	b.WriteString(ui.RenderBreadcrumb("Ghostty configurator", "Menu"))
	b.WriteByte('\n')
	b.WriteString(ui.MutedStyle.Render("Choose what to configure. Every change is reloaded live in this terminal."))
	b.WriteString("\n\n")

	for i, e := range m.entries {
		prefix := fmt.Sprintf("  %d. ", i+1)
		header := prefix + e.title
		if i == m.cursor {
			b.WriteString(ui.SelectedStyle.Render("▸ " + header))
		} else {
			b.WriteString("  " + header)
		}
		b.WriteByte('\n')
		b.WriteString(indent(wrap(e.description, 78), "      "))
		b.WriteString("\n\n")
	}

	if m.errorMsg != "" {
		b.WriteString(ui.ErrorStyle.Render("error: " + m.errorMsg))
		b.WriteByte('\n')
	}

	b.WriteString(ui.RenderFooter(width, []ui.FooterAction{
		{Key: "ENTER", Label: "Open", Variant: ui.FooterSave},
		{Key: "1-9", Label: "Jump", Variant: ui.FooterDefault},
		{Key: "?", Label: "Help", Variant: ui.FooterDefault},
		{Key: "ESC", Label: "Quit", Variant: ui.FooterCancel},
	}))
	b.WriteByte('\n')

	footerInfo := "config: " + m.configPath
	if m.version != "" {
		footerInfo += "  ·  v" + m.version
	}
	b.WriteString(ui.MutedStyle.Render(footerInfo))
	b.WriteByte('\n')
	return b.String()
}

func wrap(text string, width int) string {
	if width <= 0 {
		return text
	}
	words := strings.Fields(text)
	var lines []string
	var current strings.Builder
	for _, w := range words {
		if current.Len() == 0 {
			current.WriteString(w)
			continue
		}
		if current.Len()+1+len(w) > width {
			lines = append(lines, current.String())
			current.Reset()
			current.WriteString(w)
			continue
		}
		current.WriteByte(' ')
		current.WriteString(w)
	}
	if current.Len() > 0 {
		lines = append(lines, current.String())
	}
	return strings.Join(lines, "\n")
}

func indent(text, prefix string) string {
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		lines[i] = prefix + l
	}
	return strings.Join(lines, "\n")
}
