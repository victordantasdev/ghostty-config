package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"ghostty-config/internal/ghostty"
	"ghostty-config/internal/ui"
)

// --- Helpers --------------------------------------------------------------

func setupOpts(t *testing.T) ghostty.Options {
	t.Helper()
	root := t.TempDir()
	cfg := filepath.Join(root, "config")
	_ = os.WriteFile(cfg, []byte(""), 0o644)
	shaderDir := filepath.Join(root, "shaders")
	_ = os.MkdirAll(shaderDir, 0o755)
	_ = os.WriteFile(filepath.Join(shaderDir, "crt.glsl"), []byte("x"), 0o644)
	_ = os.WriteFile(filepath.Join(shaderDir, "cursor_blaze.glsl"), []byte("y"), 0o644)
	userTheme := filepath.Join(root, "themes")
	_ = os.MkdirAll(userTheme, 0o755)
	_ = os.WriteFile(filepath.Join(userTheme, "alpha"), []byte("a"), 0o644)
	return ghostty.Options{
		ConfigPath:     cfg,
		ShaderDir:      shaderDir,
		UserThemeDir:   userTheme,
		SystemThemeDir: filepath.Join(root, "no-sys"),
		NoReload:       true,
	}
}

// --- App ------------------------------------------------------------------

func TestNewApp(t *testing.T) {
	opts := setupOpts(t)
	a := New(opts, "1.0")
	if a.version != "1.0" {
		t.Errorf("version")
	}
	if a.current != ui.ScreenMenu {
		t.Errorf("default screen")
	}
}

func TestAppInit(t *testing.T) {
	a := New(setupOpts(t), "")
	if cmd := a.Init(); cmd != nil {
		t.Errorf("expected nil init")
	}
}

func TestAppWindowSize(t *testing.T) {
	a := New(setupOpts(t), "")
	updated, _ := a.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app := updated.(App)
	if app.width != 100 || app.height != 30 {
		t.Errorf("size not stored")
	}
}

func TestAppWindowSizeWithShaderAndTheme(t *testing.T) {
	opts := setupOpts(t)
	a := New(opts, "")
	// Switch to shader screen.
	updated, _ := a.Update(ui.SwitchScreenMsg{Target: ui.ScreenShader})
	a = updated.(App)
	updated, _ = a.Update(ui.SwitchScreenMsg{Target: ui.ScreenTheme})
	a = updated.(App)
	updated, _ = a.Update(tea.WindowSizeMsg{Width: 90, Height: 25})
	a = updated.(App)
	if a.width != 90 {
		t.Errorf("width")
	}
}

func TestAppSwitchScreens(t *testing.T) {
	opts := setupOpts(t)
	a := New(opts, "")
	updated, cmd := a.Update(ui.SwitchScreenMsg{Target: ui.ScreenShader})
	a = updated.(App)
	if a.current != ui.ScreenShader {
		t.Errorf("not shader screen")
	}
	if cmd == nil {
		t.Errorf("expected init cmd")
	}
	// Reset to menu.
	updated, _ = a.Update(ui.SwitchScreenMsg{Target: ui.ScreenMenu})
	a = updated.(App)
	if a.shader != nil {
		t.Errorf("shader should be cleared")
	}
	if a.current != ui.ScreenMenu {
		t.Errorf("not menu")
	}
}

func TestAppSwitchToTheme(t *testing.T) {
	a := New(setupOpts(t), "")
	updated, _ := a.Update(ui.SwitchScreenMsg{Target: ui.ScreenTheme})
	a = updated.(App)
	if a.current != ui.ScreenTheme {
		t.Errorf("not theme")
	}
}

func TestAppSwitchUnknownTarget(t *testing.T) {
	a := New(setupOpts(t), "")
	updated, cmd := a.Update(ui.SwitchScreenMsg{Target: ui.Screen(99)})
	if cmd != nil {
		t.Errorf("expected nil cmd")
	}
	_ = updated
}

func TestAppShaderNewError(t *testing.T) {
	opts := setupOpts(t)
	opts.ShaderDir = "/no/such/dir"
	a := New(opts, "")
	updated, _ := a.Update(ui.SwitchScreenMsg{Target: ui.ScreenShader})
	a = updated.(App)
	if a.menu.errorMsg == "" {
		t.Errorf("error should be reported on menu")
	}
	if a.current != ui.ScreenMenu {
		t.Errorf("should stay at menu on shader error")
	}
}

func TestAppThemeNewError(t *testing.T) {
	opts := setupOpts(t)
	opts.UserThemeDir = filepath.Join(t.TempDir(), "missing-too")
	opts.SystemThemeDir = filepath.Join(t.TempDir(), "missing-too")
	a := New(opts, "")
	updated, _ := a.Update(ui.SwitchScreenMsg{Target: ui.ScreenTheme})
	a = updated.(App)
	if a.menu.errorMsg == "" {
		t.Errorf("error should be set")
	}
}

func TestAppQuit(t *testing.T) {
	a := New(setupOpts(t), "")
	updated, cmd := a.Update(ui.QuitAppMsg{})
	a = updated.(App)
	if !a.quitting {
		t.Errorf("not quitting")
	}
	if cmd == nil {
		t.Errorf("expected quit cmd")
	}
}

func TestAppHelpOpenAndClose(t *testing.T) {
	a := New(setupOpts(t), "")
	updated, _ := a.Update(ui.OpenHelpMsg{})
	a = updated.(App)
	if !a.helpOpen {
		t.Errorf("not open")
	}
	updated, _ = a.Update(ui.CloseHelpMsg{})
	a = updated.(App)
	if a.helpOpen {
		t.Errorf("still open")
	}
}

func TestAppHelpQuestionKey(t *testing.T) {
	a := New(setupOpts(t), "")
	updated, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	a = updated.(App)
	if !a.helpOpen {
		t.Errorf("? should open help")
	}
	if cmd != nil {
		t.Errorf("expected nil cmd")
	}
}

func TestAppHelpRoutesMessages(t *testing.T) {
	a := New(setupOpts(t), "")
	a.helpOpen = true
	updated, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	a = updated.(App)
	// "?" inside help closes via CloseHelpMsg via cmd.
	if cmd == nil {
		t.Fatal()
	}
	if _, ok := cmd().(ui.CloseHelpMsg); !ok {
		t.Errorf("got %T", cmd())
	}
}

func TestAppToast(t *testing.T) {
	a := New(setupOpts(t), "")
	updated, cmd := a.Update(ui.ShowToastMsg{Text: "hello", Kind: ui.ToastSaved})
	a = updated.(App)
	if a.toast == nil || a.toast.text != "hello" {
		t.Errorf("toast missing")
	}
	if cmd == nil {
		t.Errorf("expected clear cmd")
	}

	updated, _ = a.Update(ui.ClearToastMsg{Token: a.toast.token})
	a = updated.(App)
	if a.toast != nil {
		t.Errorf("toast not cleared")
	}
}

func TestAppToastClearWithStaleToken(t *testing.T) {
	a := New(setupOpts(t), "")
	updated, _ := a.Update(ui.ShowToastMsg{Text: "hello", Kind: ui.ToastSaved})
	a = updated.(App)
	stale := a.toast.token
	// Show a newer toast.
	updated, _ = a.Update(ui.ShowToastMsg{Text: "newer"})
	a = updated.(App)
	// Stale clear should not affect current.
	updated, _ = a.Update(ui.ClearToastMsg{Token: stale})
	a = updated.(App)
	if a.toast == nil {
		t.Errorf("current toast lost")
	}
}

func TestAppRoutesMessagesToScreens(t *testing.T) {
	a := New(setupOpts(t), "")
	// menu by default — send arbitrary key.
	updated, _ := a.Update(tea.KeyMsg{Type: tea.KeyDown})
	a = updated.(App)
	// switch to shader and send key.
	updated, _ = a.Update(ui.SwitchScreenMsg{Target: ui.ScreenShader})
	a = updated.(App)
	updated, _ = a.Update(tea.KeyMsg{Type: tea.KeyDown})
	a = updated.(App)
	// switch to theme and send key.
	updated, _ = a.Update(ui.SwitchScreenMsg{Target: ui.ScreenTheme})
	a = updated.(App)
	updated, _ = a.Update(tea.KeyMsg{Type: tea.KeyDown})
	a = updated.(App)
	_ = a
}

func TestAppRoutesMessagesToShaderNil(t *testing.T) {
	a := New(setupOpts(t), "")
	a.current = ui.ScreenShader
	a.shader = nil
	updated, cmd := a.Update(tea.KeyMsg{Type: tea.KeyDown})
	a = updated.(App)
	if cmd != nil {
		t.Errorf("unexpected cmd")
	}
}

func TestAppRoutesMessagesToThemeNil(t *testing.T) {
	a := New(setupOpts(t), "")
	a.current = ui.ScreenTheme
	a.theme = nil
	updated, cmd := a.Update(tea.KeyMsg{Type: tea.KeyDown})
	a = updated.(App)
	if cmd != nil {
		t.Errorf("unexpected cmd")
	}
}

func TestAppViewQuitting(t *testing.T) {
	a := New(setupOpts(t), "")
	a.quitting = true
	if v := a.View(); v != "" {
		t.Errorf("expected empty: %q", v)
	}
}

func TestAppViewHelp(t *testing.T) {
	a := New(setupOpts(t), "")
	a.helpOpen = true
	v := a.View()
	if !strings.Contains(v, "Help") {
		t.Errorf("expected help view: %s", v)
	}
}

func TestAppViewBranches(t *testing.T) {
	a := New(setupOpts(t), "")
	if v := a.View(); !strings.Contains(v, "Ghostty configurator") {
		t.Errorf("menu view: %s", v)
	}

	// Open shader.
	updated, _ := a.Update(ui.SwitchScreenMsg{Target: ui.ScreenShader})
	a = updated.(App)
	if v := a.View(); !strings.Contains(v, "Shaders") {
		t.Errorf("shader view: %s", v)
	}

	// Shader screen with nil shader renders empty body.
	a.shader = nil
	if v := a.View(); strings.Contains(v, "▸") {
		t.Errorf("nil shader: %s", v)
	}

	// Open theme.
	updated, _ = a.Update(ui.SwitchScreenMsg{Target: ui.ScreenMenu})
	a = updated.(App)
	updated, _ = a.Update(ui.SwitchScreenMsg{Target: ui.ScreenTheme})
	a = updated.(App)
	if v := a.View(); !strings.Contains(v, "Themes") {
		t.Errorf("theme view: %s", v)
	}

	// Theme screen with nil renders empty body.
	a.theme = nil
	_ = a.View()
}

func TestAppViewUnknownScreen(t *testing.T) {
	a := New(setupOpts(t), "")
	a.current = ui.Screen(99)
	_ = a.View()
}

func TestAppToastTickFiresClearMsg(t *testing.T) {
	prev := toastDuration
	toastDuration = time.Microsecond
	defer func() { toastDuration = prev }()

	a := New(setupOpts(t), "")
	_, cmd := a.Update(ui.ShowToastMsg{Text: "hi"})
	if cmd == nil {
		t.Fatal()
	}
	msg := cmd()
	if _, ok := msg.(ui.ClearToastMsg); !ok {
		t.Errorf("expected ClearToastMsg, got %T", msg)
	}
}

func TestAppUpdateUnknownCurrentScreen(t *testing.T) {
	a := New(setupOpts(t), "")
	a.current = ui.Screen(99)
	updated, cmd := a.Update(tea.KeyMsg{Type: tea.KeyDown})
	if cmd != nil {
		t.Errorf("expected nil cmd")
	}
	_ = updated
}

func TestAppViewWithToast(t *testing.T) {
	a := New(setupOpts(t), "")
	updated, _ := a.Update(ui.ShowToastMsg{Text: "hi", Kind: ui.ToastSaved})
	a = updated.(App)
	v := a.View()
	if !strings.Contains(v, "hi") {
		t.Errorf("toast not rendered: %s", v)
	}
}
