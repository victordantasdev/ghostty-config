package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type FooterVariant int

const (
	FooterDefault FooterVariant = iota
	FooterSave
	FooterCancel
)

type FooterAction struct {
	Key     string
	Label   string
	Variant FooterVariant
}

// RenderFooter draws a horizontal divider followed by a row of
// [KEY] Label chips. Width controls the divider length; if the
// rendered actions exceed the width, they wrap to the next line.
func RenderFooter(width int, actions []FooterAction) string {
	if width < 1 {
		width = 80
	}
	if len(actions) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(FooterDividerStyle.Render(strings.Repeat("─", width)))
	b.WriteByte('\n')

	chips := make([]string, len(actions))
	for i, a := range actions {
		chips[i] = renderActionChip(a)
	}

	// Greedy line-wrap by visible width.
	separator := "   "
	sepWidth := lipgloss.Width(separator)
	lineWidth := 0
	for i, chip := range chips {
		cw := lipgloss.Width(chip)
		if i == 0 {
			b.WriteString(chip)
			lineWidth = cw
			continue
		}
		if lineWidth+sepWidth+cw > width {
			b.WriteByte('\n')
			b.WriteString(chip)
			lineWidth = cw
			continue
		}
		b.WriteString(separator)
		b.WriteString(chip)
		lineWidth += sepWidth + cw
	}
	return b.String()
}

func renderActionChip(a FooterAction) string {
	var key string
	switch a.Variant {
	case FooterSave:
		key = FooterSaveKeyStyle.Render(a.Key)
	case FooterCancel:
		key = FooterCancelKeyStyle.Render(a.Key)
	default:
		key = FooterKeyStyle.Render(a.Key)
	}
	return key + " " + FooterLabelStyle.Render(a.Label)
}
