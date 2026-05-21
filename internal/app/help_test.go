package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"ghostty-config/internal/ui"
)

func TestHelpUpdateNonKeyMsg(t *testing.T) {
	h := helpModel{}
	_, cmd := h.Update(tea.WindowSizeMsg{Width: 80})
	if cmd != nil {
		t.Errorf("unexpected")
	}
}

func TestHelpClose(t *testing.T) {
	h := helpModel{}
	for _, k := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune("?")},
		{Type: tea.KeyEsc},
		{Type: tea.KeyRunes, Runes: []rune("q")},
	} {
		_, cmd := h.Update(k)
		if cmd == nil {
			t.Fatalf("expected cmd for %v", k)
		}
		if _, ok := cmd().(ui.CloseHelpMsg); !ok {
			t.Errorf("expected CloseHelpMsg, got %T", cmd())
		}
	}
}

func TestHelpQuit(t *testing.T) {
	h := helpModel{}
	_, cmd := h.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal()
	}
	if _, ok := cmd().(ui.QuitAppMsg); !ok {
		t.Errorf("expected quit")
	}
}

func TestHelpUnknownKey(t *testing.T) {
	h := helpModel{}
	_, cmd := h.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("z")})
	if cmd != nil {
		t.Errorf("unexpected")
	}
}

func TestHelpView(t *testing.T) {
	h := helpModel{width: 80}
	v := h.View()
	if !strings.Contains(v, "How it works") {
		t.Errorf("missing section: %s", v)
	}
	if !strings.Contains(v, "Selection screens") {
		t.Errorf("missing section: %s", v)
	}
	if !strings.Contains(v, "ESC") {
		t.Errorf("missing footer key: %s", v)
	}
}

func TestHelpViewZeroWidth(t *testing.T) {
	h := helpModel{width: 0}
	_ = h.View()
}
