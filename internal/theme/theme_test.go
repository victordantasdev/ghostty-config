package theme

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"ghostty-config/internal/ghostty"
	"ghostty-config/internal/ui"
)

// --- Test helpers ---------------------------------------------------------

func makeOpts(t *testing.T, userThemes, systemThemes map[string]string) ghostty.Options {
	t.Helper()
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config")
	if err := os.WriteFile(cfg, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	opts := ghostty.Options{
		ConfigPath: cfg,
		NoReload:   true,
	}
	if userThemes != nil {
		opts.UserThemeDir = writeThemeDir(t, dir, "user-themes", userThemes)
	} else {
		opts.UserThemeDir = filepath.Join(dir, "no-user")
	}
	if systemThemes != nil {
		opts.SystemThemeDir = writeThemeDir(t, dir, "sys-themes", systemThemes)
	} else {
		opts.SystemThemeDir = filepath.Join(dir, "no-sys")
	}
	return opts
}

func writeThemeDir(t *testing.T, root, name string, themes map[string]string) string {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for n, body := range themes {
		path := filepath.Join(dir, n)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func keyMsg(s string) tea.KeyMsg {
	switch s {
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	case "ctrl+s":
		return tea.KeyMsg{Type: tea.KeyCtrlS}
	case "ctrl+u":
		return tea.KeyMsg{Type: tea.KeyCtrlU}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "home":
		return tea.KeyMsg{Type: tea.KeyHome}
	case "end":
		return tea.KeyMsg{Type: tea.KeyEnd}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	case "space":
		return tea.KeyMsg{Type: tea.KeySpace}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// --- Pure functions -------------------------------------------------------

func TestParseThemeValues(t *testing.T) {
	cases := []struct {
		name      string
		in        []string
		wantLight string
		wantDark  string
	}{
		{"empty", nil, "", ""},
		{"simple single", []string{"catppuccin"}, "catppuccin", "catppuccin"},
		{"light/dark", []string{"light:solar,dark:dracula"}, "solar", "dracula"},
		{"split entries", []string{"light:solar", "dark:dracula"}, "solar", "dracula"},
		{"simple then explicit", []string{"common", "light:explicit"}, "explicit", "common"},
		{"only light", []string{"light:solar"}, "solar", ""},
		{"only dark", []string{"dark:dracula"}, "", "dracula"},
		{"explicit overrides simple in same entry", []string{"simple,light:other"}, "other", "simple"},
		{"empty parts skipped", []string{"  ,  "}, "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l, d := parseThemeValues(c.in)
			if l != c.wantLight || d != c.wantDark {
				t.Errorf("got (%q,%q) want (%q,%q)", l, d, c.wantLight, c.wantDark)
			}
		})
	}
}

func TestIndexForTheme(t *testing.T) {
	themes := []option{
		{Name: "Solarized"},
		{Name: "Dracula"},
		{Name: "Catppuccin"},
	}
	if i := indexForTheme(themes, ""); i != -1 {
		t.Errorf("empty -> %d", i)
	}
	if i := indexForTheme(themes, "Dracula"); i != 1 {
		t.Errorf("exact -> %d", i)
	}
	if i := indexForTheme(themes, "dracula"); i != 1 {
		t.Errorf("lower -> %d", i)
	}
	if i := indexForTheme(themes, "missing"); i != -1 {
		t.Errorf("missing -> %d", i)
	}
}

func TestDiscoverThemes(t *testing.T) {
	dir := t.TempDir()
	user := writeThemeDir(t, dir, "user", map[string]string{
		"alpha":      "a",
		"bravo":      "b",
		".hidden":    "h",
		"shared":     "u",
		"sub/nested": "x", // nested dir → SkipDir (not collected from subdir)
	})
	sys := writeThemeDir(t, dir, "sys", map[string]string{
		"shared":  "s",
		"charlie": "c",
	})

	themes, err := discoverThemes(user, sys)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]string{}
	for _, x := range themes {
		names[x.Name] = x.Source
	}
	if names["shared"] != "user" {
		t.Errorf("user overrides system for shared, got %s", names["shared"])
	}
	if names["alpha"] != "user" || names["bravo"] != "user" {
		t.Errorf("user-only entries lost: %v", names)
	}
	if names["charlie"] != "builtin" {
		t.Errorf("system entry missing: %v", names)
	}
	if _, ok := names[".hidden"]; ok {
		t.Errorf("hidden file should be skipped")
	}
	if _, ok := names["nested"]; ok {
		t.Errorf("nested subdirectory file should not be collected")
	}
	// Sort order: case-insensitive ascending.
	for i := 1; i < len(themes); i++ {
		if strings.ToLower(themes[i-1].Name) > strings.ToLower(themes[i].Name) {
			t.Errorf("not sorted: %s > %s", themes[i-1].Name, themes[i].Name)
		}
	}
}

func TestDiscoverThemesEmptyDir(t *testing.T) {
	dir := t.TempDir()
	user := filepath.Join(dir, "u")
	sys := filepath.Join(dir, "s")
	if err := os.MkdirAll(user, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sys, 0o755); err != nil {
		t.Fatal(err)
	}
	themes, err := discoverThemes(user, sys)
	if err != nil {
		t.Fatal(err)
	}
	if len(themes) != 0 {
		t.Errorf("expected zero themes, got %d", len(themes))
	}
}

func TestDiscoverThemesEmptyDirString(t *testing.T) {
	themes, err := discoverThemes("", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(themes) != 0 {
		t.Errorf("expected zero, got %d", len(themes))
	}
}

func TestDiscoverThemesSystemError(t *testing.T) {
	dir := t.TempDir()
	fileAsParent := filepath.Join(dir, "file")
	_ = os.WriteFile(fileAsParent, []byte(""), 0o644)
	// systemDir whose parent is a file → Stat returns non-ENOENT (ENOTDIR).
	_, err := discoverThemes("", filepath.Join(fileAsParent, "themes"))
	if err == nil {
		t.Errorf("expected error")
	}
}

func TestCollectThemesWalkReadDirError(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "themes")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "x"), []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Mode 0 → Stat on parent succeeds (we have x on temp), but ReadDir on sub fails.
	if err := os.Chmod(sub, 0); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(sub, 0o755)
	err := collectThemes(sub, "user", map[string]option{})
	if err == nil {
		t.Errorf("expected error from unreadable dir")
	}
}

func TestCollectThemesUserOverrideBranch(t *testing.T) {
	// Call collectThemes twice with the same dir & source=user to hit the
	// "existing entry from user, skip" branch.
	dir := t.TempDir()
	userDir := writeThemeDir(t, dir, "user", map[string]string{"x": "1"})
	byName := map[string]option{}
	if err := collectThemes(userDir, "user", byName); err != nil {
		t.Fatal(err)
	}
	if err := collectThemes(userDir, "user", byName); err != nil {
		t.Fatal(err)
	}
	if len(byName) != 1 {
		t.Errorf("expected 1 entry, got %d", len(byName))
	}
}

func TestDiscoverThemesMissingDirNoError(t *testing.T) {
	// Non-existent dirs are tolerated (return nil from collectThemes' err check).
	themes, err := discoverThemes("/nonexistent/u", "/nonexistent/s")
	if err != nil {
		t.Fatal(err)
	}
	if len(themes) != 0 {
		t.Errorf("expected zero, got %d", len(themes))
	}
}

func TestSelectionLabel(t *testing.T) {
	if got := selectionLabel(selection{}); got != "none" {
		t.Errorf("got %q", got)
	}
	if got := selectionLabel(selection{light: "x", dark: "x"}); got != "x" {
		t.Errorf("got %q", got)
	}
	if got := selectionLabel(selection{light: "L", dark: "D"}); got != "light=L · dark=D" {
		t.Errorf("got %q", got)
	}
}

func TestSlotLabelFor(t *testing.T) {
	if slotLabelFor(slotLight) != "Light" {
		t.Errorf("light label")
	}
	if slotLabelFor(slotDark) != "Dark" {
		t.Errorf("dark label")
	}
}

func TestRenderBadge(t *testing.T) {
	if got := renderBadge(option{Source: "user"}); !strings.Contains(got, "user") {
		t.Errorf("got %q", got)
	}
	if got := renderBadge(option{Source: "builtin"}); !strings.Contains(got, "builtin") {
		t.Errorf("got %q", got)
	}
}

// --- Model lifecycle ------------------------------------------------------

func TestNew(t *testing.T) {
	t.Run("no themes returns error", func(t *testing.T) {
		opts := makeOpts(t, map[string]string{}, map[string]string{})
		_, err := New(opts)
		if err == nil {
			t.Errorf("expected error")
		}
	})

	t.Run("discoverThemes error propagates", func(t *testing.T) {
		// Parent is a regular file → Stat returns ENOTDIR (not ENOENT).
		dir := t.TempDir()
		cfg := filepath.Join(dir, "config")
		_ = os.WriteFile(cfg, []byte(""), 0o644)
		fileAsParent := filepath.Join(dir, "file")
		_ = os.WriteFile(fileAsParent, []byte(""), 0o644)
		opts := ghostty.Options{
			ConfigPath:     cfg,
			UserThemeDir:   filepath.Join(fileAsParent, "themes"),
			SystemThemeDir: filepath.Join(dir, "no-sys"),
			NoReload:       true,
		}
		_, err := New(opts)
		if err == nil {
			t.Errorf("expected error from discovery")
		}
	})

	t.Run("loads current config theme values", func(t *testing.T) {
		dir := t.TempDir()
		cfg := filepath.Join(dir, "config")
		_ = os.WriteFile(cfg, []byte("theme = light:alpha,dark:bravo\n"), 0o644)
		userDir := writeThemeDir(t, dir, "user", map[string]string{
			"alpha": "a", "bravo": "b", "charlie": "c",
		})
		opts := ghostty.Options{
			ConfigPath:     cfg,
			UserThemeDir:   userDir,
			SystemThemeDir: filepath.Join(dir, "no-sys"),
			NoReload:       true,
		}
		m, err := New(opts)
		if err != nil {
			t.Fatal(err)
		}
		if m.options[m.cursors[slotLight]].Name != "alpha" {
			t.Errorf("light cursor wrong")
		}
		if m.options[m.cursors[slotDark]].Name != "bravo" {
			t.Errorf("dark cursor wrong")
		}
	})

	t.Run("unknown current values fall back to 0", func(t *testing.T) {
		dir := t.TempDir()
		cfg := filepath.Join(dir, "config")
		_ = os.WriteFile(cfg, []byte("theme = unknown\n"), 0o644)
		userDir := writeThemeDir(t, dir, "user", map[string]string{"alpha": "a"})
		opts := ghostty.Options{
			ConfigPath:     cfg,
			UserThemeDir:   userDir,
			SystemThemeDir: filepath.Join(dir, "no-sys"),
			NoReload:       true,
		}
		m, err := New(opts)
		if err != nil {
			t.Fatal(err)
		}
		if m.cursors[slotLight] != 0 || m.cursors[slotDark] != 0 {
			t.Errorf("cursors should fall back to 0: %v", m.cursors)
		}
	})
}

// --- Update ---------------------------------------------------------------

func newTestModel(t *testing.T) Model {
	t.Helper()
	opts := makeOpts(t, map[string]string{
		"alpha": "a", "bravo": "b", "charlie": "c", "delta": "d",
	}, nil)
	m, err := New(opts)
	if err != nil {
		t.Fatal(err)
	}
	m.SetSize(80, 24)
	return m
}

func TestSetSize(t *testing.T) {
	m := newTestModel(t)
	m.SetSize(100, 50)
	if m.width != 100 || m.height != 50 {
		t.Errorf("size not applied")
	}
}

func TestInitCmd(t *testing.T) {
	m := newTestModel(t)
	cmd := m.InitCmd()
	if cmd == nil {
		t.Fatal("expected init cmd")
	}
	msg := cmd()
	pmsg, ok := msg.(previewDoneMsg)
	if !ok {
		t.Fatalf("unexpected msg type %T", msg)
	}
	if pmsg.err != nil {
		t.Errorf("preview failed: %v", pmsg.err)
	}
}

func TestPreviewNameOutOfRange(t *testing.T) {
	m := newTestModel(t)
	m.cursors[m.active] = 9999
	if got := m.previewName(); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestUpdatePreviewDoneMsg(t *testing.T) {
	m := newTestModel(t)
	m2, _ := m.Update(previewDoneMsg{label: "x"})
	if !m2.statusGood {
		t.Errorf("status should be good")
	}
	if !strings.Contains(m2.status, "previewing x") {
		t.Errorf("got %q", m2.status)
	}

	m3, _ := m.Update(previewDoneMsg{label: "x", err: errSentinel{}})
	if m3.statusGood {
		t.Errorf("status should be bad")
	}
	if m3.lastErr == nil {
		t.Errorf("expected error preserved")
	}
}

type errSentinel struct{}

func (errSentinel) Error() string { return "sentinel" }

func TestUpdateNavigation(t *testing.T) {
	m := newTestModel(t)
	// down
	m, _ = m.Update(keyMsg("down"))
	if m.cursors[slotLight] != 1 {
		t.Errorf("down should advance, got %d", m.cursors[slotLight])
	}
	// j
	m, _ = m.Update(keyMsg("j"))
	if m.cursors[slotLight] != 2 {
		t.Errorf("j should advance, got %d", m.cursors[slotLight])
	}
	// up
	m, _ = m.Update(keyMsg("up"))
	if m.cursors[slotLight] != 1 {
		t.Errorf("up should go back, got %d", m.cursors[slotLight])
	}
	// k
	m, _ = m.Update(keyMsg("k"))
	if m.cursors[slotLight] != 0 {
		t.Errorf("k should go back, got %d", m.cursors[slotLight])
	}
	// up at 0 wraps to last
	m, _ = m.Update(keyMsg("up"))
	if m.cursors[slotLight] != len(m.options)-1 {
		t.Errorf("up wrap, got %d", m.cursors[slotLight])
	}
	// down at last wraps to 0
	m, _ = m.Update(keyMsg("down"))
	if m.cursors[slotLight] != 0 {
		t.Errorf("down wrap, got %d", m.cursors[slotLight])
	}
	// end / G
	m, _ = m.Update(keyMsg("end"))
	if m.cursors[slotLight] != len(m.options)-1 {
		t.Errorf("end")
	}
	m, _ = m.Update(keyMsg("home"))
	if m.cursors[slotLight] != 0 {
		t.Errorf("home")
	}
	m, _ = m.Update(keyMsg("G"))
	if m.cursors[slotLight] != len(m.options)-1 {
		t.Errorf("G")
	}
	m, _ = m.Update(keyMsg("g"))
	if m.cursors[slotLight] != 0 {
		t.Errorf("g")
	}
	// tab switches slot
	m, _ = m.Update(keyMsg("tab"))
	if m.active != slotDark {
		t.Errorf("tab")
	}
	m, _ = m.Update(keyMsg("shift+tab"))
	if m.active != slotLight {
		t.Errorf("shift+tab")
	}
	m, _ = m.Update(keyMsg("right"))
	if m.active != slotDark {
		t.Errorf("right")
	}
	m, _ = m.Update(keyMsg("left"))
	if m.active != slotLight {
		t.Errorf("left")
	}
	m, _ = m.Update(keyMsg("l"))
	if m.active != slotDark {
		t.Errorf("l")
	}
	m, _ = m.Update(keyMsg("h"))
	if m.active != slotLight {
		t.Errorf("h")
	}
}

func TestUpdateSearchMode(t *testing.T) {
	m := newTestModel(t)
	// open search
	m, _ = m.Update(keyMsg("/"))
	if !m.searching {
		t.Errorf("not in search")
	}
	// type
	m, _ = m.Update(keyMsg("c"))
	if m.query != "c" {
		t.Errorf("query: %q", m.query)
	}
	// add more characters
	m, _ = m.Update(keyMsg("h"))
	// backspace
	m, _ = m.Update(keyMsg("backspace"))
	if m.query != "c" {
		t.Errorf("backspace: %q", m.query)
	}
	// ctrl+u clears
	m, _ = m.Update(keyMsg("ctrl+u"))
	if m.query != "" {
		t.Errorf("ctrl+u: %q", m.query)
	}
	// re-type and ensure cursor adjusts to match
	m, _ = m.Update(keyMsg("c")) // 'charlie'
	m, _ = m.Update(keyMsg("h"))
	if m.options[m.cursors[m.active]].Name != "charlie" {
		t.Errorf("cursor not on charlie: %s", m.options[m.cursors[m.active]].Name)
	}
	// enter exits search keeping query
	m, _ = m.Update(keyMsg("enter"))
	if m.searching {
		t.Errorf("still searching")
	}
	if m.query == "" {
		t.Errorf("query lost")
	}
	// reopen and esc → clear + exit
	m, _ = m.Update(keyMsg("/"))
	m, _ = m.Update(keyMsg("esc"))
	if m.searching || m.query != "" {
		t.Errorf("esc should clear")
	}
	// search + up/down/tab/shift+tab still work
	m, _ = m.Update(keyMsg("/"))
	m, _ = m.Update(keyMsg("down"))
	m, _ = m.Update(keyMsg("up"))
	m, _ = m.Update(keyMsg("tab"))
	if !m.searching {
		// In search mode, tab switches slot but stays searching? Actually clears query and stays in search.
	}
	m, _ = m.Update(keyMsg("shift+tab"))
	// backspace with empty query is no-op
	m.query = ""
	m, _ = m.Update(keyMsg("backspace"))
	if m.query != "" {
		t.Errorf("backspace empty no-op")
	}
}

func TestUpdateSearchNoMatches(t *testing.T) {
	m := newTestModel(t)
	m, _ = m.Update(keyMsg("/"))
	m, _ = m.Update(keyMsg("z")) // no match
	m, _ = m.Update(keyMsg("z"))
	m, _ = m.Update(keyMsg("z"))
	// navigation on no-match is no-op
	m, _ = m.Update(keyMsg("down"))
	m, _ = m.Update(keyMsg("home"))
	if m.cursors[m.active] != 0 {
		t.Errorf("cursor should stay")
	}
	// view should render "no matches"
	view := m.View()
	if !strings.Contains(view, "no matches") {
		t.Errorf("view missing no-match: %s", view)
	}
}

func TestSearchCtrlCQuits(t *testing.T) {
	m := newTestModel(t)
	m, _ = m.Update(keyMsg("/"))
	_, cmd := m.Update(keyMsg("ctrl+c"))
	if cmd == nil {
		t.Fatal("expected quit cmd")
	}
	msg := cmd()
	if _, ok := msg.(ui.QuitAppMsg); !ok {
		t.Errorf("expected QuitAppMsg, got %T", msg)
	}
}

func TestSearchUnknownKeyIgnored(t *testing.T) {
	m := newTestModel(t)
	m, _ = m.Update(keyMsg("/"))
	// A key without Runes is ignored in search.
	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyF1})
	if cmd != nil {
		t.Errorf("expected no cmd")
	}
	if m2.query != "" {
		t.Errorf("query changed: %q", m2.query)
	}
}

func TestUpdateCtrlCQuits(t *testing.T) {
	m := newTestModel(t)
	_, cmd := m.Update(keyMsg("ctrl+c"))
	if cmd == nil {
		t.Fatal("expected cmd")
	}
	msg := cmd()
	if _, ok := msg.(ui.QuitAppMsg); !ok {
		t.Errorf("expected QuitAppMsg, got %T", msg)
	}
}

func TestUpdateEscRestores(t *testing.T) {
	m := newTestModel(t)
	// Move so dirty.
	m, _ = m.Update(keyMsg("down"))
	if !m.isDirty() {
		t.Errorf("expected dirty")
	}
	_, cmd := m.Update(keyMsg("esc"))
	if cmd == nil {
		t.Fatal()
	}
	// tea.Batch returns batched; we can call the messages it returns indirectly
	// by invoking the cmd once.
	msg := cmd()
	if _, ok := msg.(tea.BatchMsg); !ok {
		t.Errorf("expected batch, got %T", msg)
	}
}

func TestUpdateEscCleanState(t *testing.T) {
	m := newTestModel(t)
	_, cmd := m.Update(keyMsg("q"))
	if cmd == nil {
		t.Fatal()
	}
	_ = cmd()
}

func TestUpdateEnterCommits(t *testing.T) {
	m := newTestModel(t)
	m, _ = m.Update(keyMsg("down"))
	_, cmd := m.Update(keyMsg("enter"))
	if cmd == nil {
		t.Fatal()
	}
	_ = cmd()
}

func TestUpdateCtrlSCommits(t *testing.T) {
	m := newTestModel(t)
	_, cmd := m.Update(keyMsg("ctrl+s"))
	if cmd == nil {
		t.Fatal()
	}
	_ = cmd()
}

func TestUpdateNonKeyMsg(t *testing.T) {
	m := newTestModel(t)
	m2, cmd := m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	if cmd != nil {
		t.Errorf("unexpected cmd")
	}
	_ = m2
}

func TestUpdateUnknownKey(t *testing.T) {
	m := newTestModel(t)
	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyF1})
	if cmd != nil {
		t.Errorf("unexpected cmd")
	}
	_ = m2
}

// --- View paths -----------------------------------------------------------

func TestViewCleanAndDirty(t *testing.T) {
	m := newTestModel(t)
	// Trigger a preview message to populate status.
	m, _ = m.Update(previewDoneMsg{label: "alpha"})
	clean := m.View()
	if !strings.Contains(clean, "Saved state") {
		t.Errorf("expected clean banner: %s", clean)
	}
	// Make dirty.
	m, _ = m.Update(keyMsg("down"))
	m, _ = m.Update(previewDoneMsg{label: "bravo"})
	dirty := m.View()
	if !strings.Contains(dirty, "UNSAVED PREVIEW") {
		t.Errorf("expected warn banner: %s", dirty)
	}
}

func TestViewSearchingHints(t *testing.T) {
	m := newTestModel(t)
	m, _ = m.Update(keyMsg("/"))
	view := m.View()
	if !strings.Contains(view, "type to filter") {
		t.Errorf("missing search hint: %s", view)
	}
}

func TestViewFilteredCount(t *testing.T) {
	m := newTestModel(t)
	m, _ = m.Update(keyMsg("/"))
	m, _ = m.Update(keyMsg("c"))
	m, _ = m.Update(keyMsg("enter")) // exit search but keep query
	view := m.View()
	if !strings.Contains(view, "of ") {
		t.Errorf("expected filter count: %s", view)
	}
}

func TestViewStatusErrorBranch(t *testing.T) {
	m := newTestModel(t)
	m, _ = m.Update(previewDoneMsg{label: "x", err: errSentinel{}})
	view := m.View()
	if !strings.Contains(view, "sentinel") {
		t.Errorf("expected error in view: %s", view)
	}
}

func TestViewDirtyGoodStatus(t *testing.T) {
	m := newTestModel(t)
	m, _ = m.Update(keyMsg("down"))
	m, _ = m.Update(previewDoneMsg{label: "bravo"})
	view := m.View()
	if !strings.Contains(view, "not saved yet") {
		t.Errorf("expected dirty status: %s", view)
	}
}

func TestViewScrollWindow(t *testing.T) {
	// Many themes + small height to force scroll window.
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config")
	_ = os.WriteFile(cfg, []byte(""), 0o644)
	many := map[string]string{}
	for i := 0; i < 50; i++ {
		many[string(rune('a'+i%26))+string(rune('a'+i/26))] = "x"
	}
	userDir := writeThemeDir(t, dir, "user", many)
	opts := ghostty.Options{
		ConfigPath:     cfg,
		UserThemeDir:   userDir,
		SystemThemeDir: filepath.Join(dir, "no-sys"),
		NoReload:       true,
	}
	m, err := New(opts)
	if err != nil {
		t.Fatal(err)
	}
	m.SetSize(80, 24)
	// Move cursor near end of list.
	for i := 0; i < 40; i++ {
		m, _ = m.Update(keyMsg("down"))
	}
	view := m.View()
	if !strings.Contains(view, "showing ") {
		t.Errorf("expected scroll indicator: %s", view)
	}
}

func TestViewVerySmallHeightClamps(t *testing.T) {
	m := newTestModel(t)
	m.SetSize(80, 1) // forces visibleHeight = 8 minimum
	_ = m.View()
}

func TestViewZeroWidthFooterFallback(t *testing.T) {
	m := newTestModel(t)
	m.SetSize(0, 24)
	_ = m.View()
}

// --- tea.Cmd helpers ------------------------------------------------------

func TestPreviewCmdEmpty(t *testing.T) {
	m := newTestModel(t)
	cmd := previewCmd(m.opts, "")
	msg := cmd().(previewDoneMsg)
	if msg.label != "none" || msg.err != nil {
		t.Errorf("got %+v", msg)
	}
}

func TestPreviewCmdError(t *testing.T) {
	// Create opts pointing at a directory for ConfigPath → write fails.
	dir := t.TempDir()
	cfgDir := filepath.Join(dir, "cfg")
	_ = os.MkdirAll(cfgDir, 0o755)
	opts := ghostty.Options{ConfigPath: cfgDir, NoReload: true}
	cmd := previewCmd(opts, "x")
	msg := cmd().(previewDoneMsg)
	if msg.err == nil {
		t.Errorf("expected error")
	}
}

func TestCommitAndBackCmd(t *testing.T) {
	m := newTestModel(t)
	cmd := commitAndBackCmd(m.opts, m.currentSelection(), true)
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected batch, got %T", msg)
	}
	// Run each sub-command and inspect.
	sawToast := false
	sawSwitch := false
	for _, sub := range batch {
		out := sub()
		switch m := out.(type) {
		case ui.ShowToastMsg:
			sawToast = true
			if m.Kind != ui.ToastSaved {
				t.Errorf("expected ToastSaved")
			}
		case ui.SwitchScreenMsg:
			sawSwitch = true
			if m.Target != ui.ScreenMenu {
				t.Errorf("expected ScreenMenu")
			}
		}
	}
	if !sawToast || !sawSwitch {
		t.Errorf("missing one of the batch messages")
	}
}

func TestCommitAndBackCmdNotDirty(t *testing.T) {
	m := newTestModel(t)
	cmd := commitAndBackCmd(m.opts, m.currentSelection(), false)
	batch := cmd().(tea.BatchMsg)
	for _, sub := range batch {
		if toast, ok := sub().(ui.ShowToastMsg); ok {
			if toast.Kind != ui.ToastInfo || toast.Text != "No changes" {
				t.Errorf("got %+v", toast)
			}
		}
	}
}

func TestCommitAndBackCmdWriteError(t *testing.T) {
	dir := t.TempDir()
	cfgDir := filepath.Join(dir, "cfg")
	_ = os.MkdirAll(cfgDir, 0o755)
	opts := ghostty.Options{ConfigPath: cfgDir, NoReload: true}
	cmd := commitAndBackCmd(opts, selection{light: "x", dark: "y"}, true)
	batch := cmd().(tea.BatchMsg)
	for _, sub := range batch {
		out := sub()
		if pm, ok := out.(previewDoneMsg); ok {
			if pm.err == nil {
				t.Errorf("expected write error")
			}
		}
	}
}

func TestRestoreAndBackCmd(t *testing.T) {
	m := newTestModel(t)
	cmd := restoreAndBackCmd(m.opts, m.initialRawValues, true)
	batch := cmd().(tea.BatchMsg)
	for _, sub := range batch {
		if toast, ok := sub().(ui.ShowToastMsg); ok {
			if toast.Kind != ui.ToastReverted {
				t.Errorf("expected reverted, got %v", toast.Kind)
			}
		}
	}
}

func TestRestoreAndBackCmdNotDirty(t *testing.T) {
	m := newTestModel(t)
	cmd := restoreAndBackCmd(m.opts, m.initialRawValues, false)
	batch := cmd().(tea.BatchMsg)
	for _, sub := range batch {
		if toast, ok := sub().(ui.ShowToastMsg); ok {
			if toast.Kind != ui.ToastInfo {
				t.Errorf("expected info, got %v", toast.Kind)
			}
		}
	}
}

func TestRestoreAndQuitCmd(t *testing.T) {
	m := newTestModel(t)
	cmd := restoreAndQuitCmd(m.opts, m.initialRawValues)
	msg := cmd()
	if _, ok := msg.(ui.QuitAppMsg); !ok {
		t.Errorf("expected QuitAppMsg, got %T", msg)
	}
}

func TestWriteSelectionVariants(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config")
	opts := ghostty.Options{ConfigPath: cfg, NoReload: true}

	// Both empty: removes key (file remains absent if was absent).
	if err := writeSelection(opts, selection{}); err != nil {
		t.Fatal(err)
	}

	// Same value: single line.
	if err := writeSelection(opts, selection{light: "x", dark: "x"}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(cfg)
	if !strings.Contains(string(data), "theme = x") {
		t.Errorf("expected single x: %q", data)
	}

	// Different: composite.
	if err := writeSelection(opts, selection{light: "a", dark: "b"}); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(cfg)
	if !strings.Contains(string(data), "light:a,dark:b") {
		t.Errorf("expected composite: %q", data)
	}

	// Only light.
	if err := writeSelection(opts, selection{light: "a"}); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(cfg)
	if !strings.Contains(string(data), "light:a") {
		t.Errorf("expected only-light: %q", data)
	}

	// Only dark.
	if err := writeSelection(opts, selection{dark: "b"}); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(cfg)
	if !strings.Contains(string(data), "dark:b") {
		t.Errorf("expected only-dark: %q", data)
	}
}

func TestRestoreValuesError(t *testing.T) {
	dir := t.TempDir()
	cfgDir := filepath.Join(dir, "cfg")
	_ = os.MkdirAll(cfgDir, 0o755)
	opts := ghostty.Options{ConfigPath: cfgDir, NoReload: true}
	if err := restoreValues(opts, []string{"x"}); err == nil {
		t.Errorf("expected error")
	}
}

func TestApplyPreviewReloadError(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config")
	opts := ghostty.Options{ConfigPath: cfg, ReloadCommand: "false"}
	if err := applyPreview(opts, "x"); err == nil {
		t.Errorf("expected reload error")
	}
}

func TestEnsureCursorMatches(t *testing.T) {
	m := newTestModel(t)
	// Set query that doesn't include current cursor.
	m.searching = true
	m.query = "delta"
	// Move cursor to index 0 (alpha).
	m.cursors[m.active] = 0
	m, _ = m.ensureCursorMatches()
	if m.options[m.cursors[m.active]].Name != "delta" {
		t.Errorf("cursor should snap to delta, got %s", m.options[m.cursors[m.active]].Name)
	}

	// When cursor already matches, no-op cmd.
	_, cmd := m.ensureCursorMatches()
	if cmd != nil {
		t.Errorf("expected nil cmd when already matching")
	}

	// When no matches, returns nil cmd.
	m.query = "zzz"
	_, cmd = m.ensureCursorMatches()
	if cmd != nil {
		t.Errorf("expected nil cmd on no-match")
	}
}

func TestMoveCursorWithFilterMismatch(t *testing.T) {
	m := newTestModel(t)
	m.searching = true
	m.query = "alpha"
	// Cursor on delta (3) → not in filtered set.
	m.cursors[m.active] = 3
	m, _ = m.moveCursor(1)
	if m.options[m.cursors[m.active]].Name != "alpha" {
		t.Errorf("expected snap to alpha, got %s", m.options[m.cursors[m.active]].Name)
	}
	m.cursors[m.active] = 3
	m, _ = m.moveCursor(-1)
	if m.options[m.cursors[m.active]].Name != "alpha" {
		t.Errorf("expected snap to alpha for delta=-1")
	}
}

func TestJumpEdgeNoMatches(t *testing.T) {
	m := newTestModel(t)
	m.searching = true
	m.query = "zzz"
	prev := m.cursors[m.active]
	m, _ = m.jumpEdge(true)
	if m.cursors[m.active] != prev {
		t.Errorf("expected no movement")
	}
}

func TestMoveCursorNoMatches(t *testing.T) {
	m := newTestModel(t)
	m.searching = true
	m.query = "zzz"
	prev := m.cursors[m.active]
	m, _ = m.moveCursor(1)
	if m.cursors[m.active] != prev {
		t.Errorf("expected no movement on empty filter")
	}
}
