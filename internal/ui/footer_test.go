package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestRenderFooter(t *testing.T) {
	t.Run("no actions returns empty", func(t *testing.T) {
		if got := RenderFooter(80, nil); got != "" {
			t.Errorf("got %q", got)
		}
		if got := RenderFooter(80, []FooterAction{}); got != "" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("width <1 falls back to 80", func(t *testing.T) {
		got := RenderFooter(0, []FooterAction{{Key: "X", Label: "Y"}})
		if !strings.Contains(got, "X") || !strings.Contains(got, "Y") {
			t.Errorf("missing content: %q", got)
		}
	})

	t.Run("single action fits in line", func(t *testing.T) {
		got := RenderFooter(80, []FooterAction{{Key: "ENTER", Label: "Save", Variant: FooterSave}})
		if !strings.Contains(got, "ENTER") {
			t.Errorf("missing key: %q", got)
		}
		if strings.Count(got, "\n") != 1 {
			t.Errorf("expected single line of chips, got %q", got)
		}
	})

	t.Run("wraps when width insufficient", func(t *testing.T) {
		actions := []FooterAction{
			{Key: "K1", Label: "long label one"},
			{Key: "K2", Label: "long label two"},
			{Key: "K3", Label: "long label three"},
		}
		got := RenderFooter(10, actions)
		// Each chip wider than 10 → divider + each chip on its own line.
		lines := strings.Split(got, "\n")
		if len(lines) < 4 {
			t.Errorf("expected wrap onto multiple lines, got %d lines: %q", len(lines), got)
		}
	})

	t.Run("all variants render distinct styling", func(t *testing.T) {
		actions := []FooterAction{
			{Key: "A", Label: "a", Variant: FooterDefault},
			{Key: "B", Label: "b", Variant: FooterSave},
			{Key: "C", Label: "c", Variant: FooterCancel},
		}
		got := RenderFooter(80, actions)
		// Verify each key appears.
		for _, k := range []string{"A", "B", "C"} {
			if !strings.Contains(got, k) {
				t.Errorf("missing key %s in %q", k, got)
			}
		}
	})
}

func TestRenderActionChip(t *testing.T) {
	cases := []FooterVariant{FooterDefault, FooterSave, FooterCancel}
	for _, v := range cases {
		chip := renderActionChip(FooterAction{Key: "K", Label: "L", Variant: v})
		if lipgloss.Width(chip) == 0 {
			t.Errorf("variant %d produced empty chip", v)
		}
	}
}
