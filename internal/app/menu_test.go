package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"ghostty-config/internal/ui"
)

func TestNewMenuModel(t *testing.T) {
	m := newMenuModel("/cfg", "1.2.3")
	if len(m.entries) != 2 {
		t.Errorf("expected 2 entries")
	}
	if m.configPath != "/cfg" {
		t.Errorf("config")
	}
	if m.version != "1.2.3" {
		t.Errorf("version")
	}
}

func TestMenuUpdateNonKeyMsg(t *testing.T) {
	m := newMenuModel("/cfg", "")
	m2, cmd := m.Update(tea.WindowSizeMsg{Width: 80})
	if cmd != nil {
		t.Errorf("unexpected cmd")
	}
	if m2.cursor != 0 {
		t.Errorf("unchanged")
	}
}

func TestMenuNavigation(t *testing.T) {
	m := newMenuModel("/cfg", "")
	// down moves cursor.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 1 {
		t.Errorf("down")
	}
	// down at last stays (no wrap).
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 1 {
		t.Errorf("stay last")
	}
	// j alias
	m.cursor = 0
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.cursor != 1 {
		t.Errorf("j")
	}
	// up
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != 0 {
		t.Errorf("up")
	}
	// up at 0 stays.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != 0 {
		t.Errorf("up stay")
	}
	// k alias
	m.cursor = 1
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.cursor != 0 {
		t.Errorf("k")
	}
	// end / G
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	if m.cursor != len(m.entries)-1 {
		t.Errorf("end")
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyHome})
	if m.cursor != 0 {
		t.Errorf("home")
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	if m.cursor != len(m.entries)-1 {
		t.Errorf("G")
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	if m.cursor != 0 {
		t.Errorf("g")
	}
}

func TestMenuQuit(t *testing.T) {
	m := newMenuModel("/cfg", "")
	for _, k := range []tea.KeyMsg{
		{Type: tea.KeyCtrlC},
		{Type: tea.KeyEsc},
		{Type: tea.KeyRunes, Runes: []rune("q")},
	} {
		_, cmd := m.Update(k)
		if cmd == nil {
			t.Fatalf("expected cmd for %v", k)
		}
		if _, ok := cmd().(ui.QuitAppMsg); !ok {
			t.Errorf("expected quit msg")
		}
	}
}

func TestMenuOpen(t *testing.T) {
	m := newMenuModel("/cfg", "")
	// enter opens.
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal()
	}
	if msg, ok := cmd().(ui.SwitchScreenMsg); !ok || msg.Target != ui.ScreenShader {
		t.Errorf("got %T %+v", cmd(), msg)
	}
	// space opens.
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	if cmd == nil {
		t.Fatal()
	}
	// right
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if cmd == nil {
		t.Fatal()
	}
	// l
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	if cmd == nil {
		t.Fatal()
	}
}

func TestMenuJumpDigits(t *testing.T) {
	m := newMenuModel("/cfg", "")
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	if cmd == nil {
		t.Fatal()
	}
	if msg, ok := cmd().(ui.SwitchScreenMsg); !ok || msg.Target != ui.ScreenTheme {
		t.Errorf("got %+v", msg)
	}
	// 9 is out of range — no cmd.
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("9")})
	if cmd != nil {
		t.Errorf("expected nil cmd for out-of-range")
	}
}

func TestMenuUnknownKey(t *testing.T) {
	m := newMenuModel("/cfg", "")
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("z")})
	if cmd != nil {
		t.Errorf("expected nil")
	}
}

func TestMenuView(t *testing.T) {
	m := newMenuModel("/cfg/path", "0.1.0")
	v := m.View()
	if !strings.Contains(v, "Shaders") || !strings.Contains(v, "Themes") {
		t.Errorf("missing entries: %s", v)
	}
	if !strings.Contains(v, "/cfg/path") {
		t.Errorf("missing config path: %s", v)
	}
	if !strings.Contains(v, "v0.1.0") {
		t.Errorf("missing version: %s", v)
	}

	// With error.
	m.errorMsg = "boom"
	v = m.View()
	if !strings.Contains(v, "boom") {
		t.Errorf("missing error: %s", v)
	}

	// Empty version (no "v" suffix line)
	m2 := newMenuModel("/cfg", "")
	v = m2.View()
	if strings.Contains(v, "  ·  v") {
		t.Errorf("should omit version: %s", v)
	}

	// Zero width fallback
	m3 := newMenuModel("/cfg", "")
	m3.width = 0
	_ = m3.View()
}

func TestWrap(t *testing.T) {
	if wrap("hello world", 0) != "hello world" {
		t.Errorf("width 0 unchanged")
	}
	got := wrap("hello world foo bar", 10)
	lines := strings.Split(got, "\n")
	if len(lines) < 2 {
		t.Errorf("expected wrapping, got %q", got)
	}
	// word longer than width still on its own line.
	got = wrap("supercalifragilistic something", 5)
	if !strings.Contains(got, "supercalifragilistic") {
		t.Errorf("got %q", got)
	}
	// Empty input.
	if got := wrap("", 5); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestIndent(t *testing.T) {
	got := indent("a\nb", "> ")
	if got != "> a\n> b" {
		t.Errorf("got %q", got)
	}
}
