package shader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"ghostty-config/internal/ghostty"
	"ghostty-config/internal/ui"
)

// --- Helpers --------------------------------------------------------------

func writeShaderDir(t *testing.T, root, name string, shaders map[string]string) string {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for n, body := range shaders {
		path := filepath.Join(dir, n)
		_ = os.MkdirAll(filepath.Dir(path), 0o755)
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func makeOpts(t *testing.T, shaders map[string]string) ghostty.Options {
	t.Helper()
	root := t.TempDir()
	cfg := filepath.Join(root, "config")
	_ = os.WriteFile(cfg, []byte(""), 0o644)
	shaderDir := writeShaderDir(t, root, "shaders", shaders)
	return ghostty.Options{
		ConfigPath: cfg,
		ShaderDir:  shaderDir,
		NoReload:   true,
	}
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

func newTestModel(t *testing.T) Model {
	t.Helper()
	opts := makeOpts(t, map[string]string{
		"crt.glsl":            "g1",
		"bloom.glsl":          "g2",
		"dither.glsl":         "g3",
		"cursor_blaze.glsl":   "c1",
		"cursor_warp.glsl":    "c2",
		"notashader.txt":      "ignored",
		"sub/nested.glsl":     "nested-global",
		"sub/cursor_sub.glsl": "nested-cursor",
	})
	m, err := New(opts)
	if err != nil {
		t.Fatal(err)
	}
	m.SetSize(80, 24)
	return m
}

// --- Pure helpers ---------------------------------------------------------

func TestIsCursorShader(t *testing.T) {
	if !isCursorShader("cursor_blaze.glsl") {
		t.Error()
	}
	if !isCursorShader("CURSOR.glsl") {
		t.Error()
	}
	if isCursorShader("crt.glsl") {
		t.Error()
	}
}

func TestPathForConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config")
	shader := filepath.Join(dir, "shaders", "crt.glsl")
	got := pathForConfig(cfg, shader)
	if got != "shaders/crt.glsl" {
		t.Errorf("got %q", got)
	}

	// Outside config dir → absolute fallback.
	other := "/elsewhere/file.glsl"
	got = pathForConfig(cfg, other)
	if got != "/elsewhere/file.glsl" {
		t.Errorf("got %q", got)
	}
}

func TestIndexForShader(t *testing.T) {
	opts := []option{
		{Path: "/a/b.glsl"},
		{Path: ""},
		{Path: "/c.glsl"},
	}
	if i := indexForShader(opts, ""); i != -1 {
		t.Errorf("empty -> %d", i)
	}
	if i := indexForShader(opts, "/a/b.glsl"); i != 0 {
		t.Errorf("hit -> %d", i)
	}
	if i := indexForShader(opts, "/missing"); i != -1 {
		t.Errorf("miss -> %d", i)
	}
}

func TestReadCurrentShaders(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config")
	body := `custom-shader = shaders/crt.glsl
custom-shader = /abs/path/cursor_blaze.glsl
custom-shader = shaders/cursor_tail.glsl
custom-shader = shaders/bloom.glsl
`
	_ = os.WriteFile(cfg, []byte(body), 0o644)
	globals, cursor := readCurrentShaders(cfg)
	if len(globals) != 2 {
		t.Errorf("globals: %v", globals)
	}
	if !strings.HasSuffix(cursor, "cursor_blaze.glsl") {
		t.Errorf("cursor first wins: %q", cursor)
	}
}

func TestReadCurrentShadersNoConfig(t *testing.T) {
	g, c := readCurrentShaders("/no/file")
	if g != nil || c != "" {
		t.Errorf("expected nothing: %v %q", g, c)
	}
}

func TestSlotLabel(t *testing.T) {
	if slotLabel(0) != "Global" {
		t.Error()
	}
	if slotLabel(1) != "Cursor" {
		t.Error()
	}
}

func TestSelectionLabelShader(t *testing.T) {
	got := selectionLabel(selection{})
	if got != "global=none · cursor=none" {
		t.Errorf("got %q", got)
	}
	got = selectionLabel(selection{
		globals: []option{{Name: "a"}, {Name: "b"}},
		cursor:  option{Name: "c", Path: "/c.glsl"},
	})
	if !strings.Contains(got, "a → b") || !strings.Contains(got, "cursor=c") {
		t.Errorf("got %q", got)
	}
}

func TestSlotSummary(t *testing.T) {
	s := slotState{}
	if got := slotSummary(s); got != "none" {
		t.Errorf("got %q", got)
	}
	s = slotState{
		options:  []option{{Name: "a", Path: "/a"}, {Name: "b", Path: "/b"}},
		multi:    true,
		selected: []int{0, 1},
	}
	if got := slotSummary(s); got != "a → b" {
		t.Errorf("got %q", got)
	}
}

func TestSlotStateEffective(t *testing.T) {
	s := slotState{
		options:  []option{{Name: "a", Path: "/a"}, {Name: "", Path: ""}, {Name: "b", Path: "/b"}},
		multi:    true,
		selected: []int{0, 1, 2, 99}, // 99 invalid, 1 empty path
	}
	got := s.effective()
	if len(got) != 2 {
		t.Errorf("got %d: %v", len(got), got)
	}

	// Single-select with empty path returns nil.
	s = slotState{
		options: []option{{Path: ""}},
		cursor:  0,
	}
	if s.effective() != nil {
		t.Errorf("expected nil")
	}

	// Single-select out of range.
	s = slotState{cursor: 99}
	if s.effective() != nil {
		t.Errorf("expected nil for OOR")
	}

	// Single-select valid path.
	s = slotState{
		options: []option{{Path: "/a"}},
		cursor:  0,
	}
	if len(s.effective()) != 1 {
		t.Errorf("expected one")
	}
}

func TestSlotStateInitialEffective(t *testing.T) {
	// multi
	s := slotState{
		options:      []option{{Path: "/a"}, {Path: ""}, {Path: "/b"}},
		multi:        true,
		selectedInit: []int{0, 1, 2, 99},
	}
	got := s.initialEffective()
	if len(got) != 2 {
		t.Errorf("got %d", len(got))
	}

	// single
	s = slotState{options: []option{{Path: ""}}, initialCur: 0}
	if s.initialEffective() != nil {
		t.Errorf("expected nil empty path")
	}
	s = slotState{initialCur: 99}
	if s.initialEffective() != nil {
		t.Errorf("expected nil OOR")
	}
	s = slotState{options: []option{{Path: "/a"}}, initialCur: 0}
	if len(s.initialEffective()) != 1 {
		t.Errorf("expected one")
	}
}

func TestSelectionPosNotFound(t *testing.T) {
	s := slotState{selected: []int{1, 2, 3}}
	if got := s.selectionPos(99); got != -1 {
		t.Errorf("got %d", got)
	}
	if got := s.selectionPos(2); got != 1 {
		t.Errorf("got %d", got)
	}
}

// --- discoverShaders / New -----------------------------------------------

func TestDiscoverShadersWalkErrorPropagates(t *testing.T) {
	root := t.TempDir()
	cfg := filepath.Join(root, "config")
	sh := writeShaderDir(t, root, "shaders", map[string]string{
		"a.glsl":         "x",
		"sub/nested.glsl": "y",
	})
	// Make the sub directory unreadable; WalkDir will call walkfn with err.
	sub := filepath.Join(sh, "sub")
	if err := os.Chmod(sub, 0); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(sub, 0o755)
	_, _, err := discoverShaders(cfg, sh)
	if err == nil {
		t.Errorf("expected walk error")
	}
}

func TestDiscoverShadersMissingDir(t *testing.T) {
	_, _, err := discoverShaders("/cfg", "/no/such/dir")
	if err == nil {
		t.Errorf("expected error")
	}
}

func TestNewNoShaders(t *testing.T) {
	root := t.TempDir()
	cfg := filepath.Join(root, "config")
	_ = os.WriteFile(cfg, []byte(""), 0o644)
	sh := writeShaderDir(t, root, "shaders", map[string]string{})
	opts := ghostty.Options{ConfigPath: cfg, ShaderDir: sh, NoReload: true}
	_, err := New(opts)
	if err == nil {
		t.Errorf("expected error: no shaders")
	}
}

func TestNewBubblesDiscoverError(t *testing.T) {
	opts := ghostty.Options{ConfigPath: "/cfg", ShaderDir: "/no/such", NoReload: true}
	_, err := New(opts)
	if err == nil {
		t.Errorf("expected discover error")
	}
}

func TestNewLoadsCurrentShaders(t *testing.T) {
	root := t.TempDir()
	cfg := filepath.Join(root, "config")
	sh := writeShaderDir(t, root, "shaders", map[string]string{
		"crt.glsl":          "x",
		"cursor_blaze.glsl": "x",
	})
	_ = os.WriteFile(cfg, []byte(
		"custom-shader = shaders/crt.glsl\ncustom-shader = shaders/cursor_blaze.glsl\n",
	), 0o644)
	opts := ghostty.Options{ConfigPath: cfg, ShaderDir: sh, NoReload: true}
	m, err := New(opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.slots[0].selected) != 1 {
		t.Errorf("globals: %v", m.slots[0].selected)
	}
	if m.slots[1].cursor == 0 {
		t.Errorf("cursor should not point at 'None'")
	}
}

func TestNewOutsideShaderDirGlobalAndCursor(t *testing.T) {
	root := t.TempDir()
	cfg := filepath.Join(root, "ghostty", "config")
	_ = os.MkdirAll(filepath.Dir(cfg), 0o755)
	sh := writeShaderDir(t, root, "shaders", map[string]string{
		"crt.glsl":          "x",
		"cursor_blaze.glsl": "y",
	})
	external := filepath.Join(root, "external")
	_ = os.MkdirAll(external, 0o755)
	_ = os.WriteFile(filepath.Join(external, "outside.glsl"), []byte("z"), 0o644)
	_ = os.WriteFile(filepath.Join(external, "cursor_out.glsl"), []byte("z"), 0o644)
	_ = os.WriteFile(cfg, []byte(
		"custom-shader = "+filepath.Join(external, "outside.glsl")+"\n"+
			"custom-shader = "+filepath.Join(external, "cursor_out.glsl")+"\n"),
		0o644,
	)
	opts := ghostty.Options{ConfigPath: cfg, ShaderDir: sh, NoReload: true}
	m, err := New(opts)
	if err != nil {
		t.Fatal(err)
	}
	// One outside global appended.
	foundOutside := false
	for _, opt := range m.slots[0].options {
		if strings.Contains(opt.Name, "outside") {
			foundOutside = true
		}
	}
	if !foundOutside {
		t.Errorf("outside global not appended")
	}
	foundCursor := false
	for _, opt := range m.slots[1].options {
		if strings.Contains(opt.Name, "cursor_out") {
			foundCursor = true
		}
	}
	if !foundCursor {
		t.Errorf("outside cursor not appended")
	}
}

func TestNewActiveFallbackToCursorSlotWhenNoGlobals(t *testing.T) {
	root := t.TempDir()
	cfg := filepath.Join(root, "config")
	_ = os.WriteFile(cfg, []byte(""), 0o644)
	sh := writeShaderDir(t, root, "shaders", map[string]string{
		"cursor_blaze.glsl": "x",
	})
	opts := ghostty.Options{ConfigPath: cfg, ShaderDir: sh, NoReload: true}
	m, err := New(opts)
	if err != nil {
		t.Fatal(err)
	}
	if m.active != 1 {
		t.Errorf("active = %d", m.active)
	}
}

// --- SetSize / InitCmd ----------------------------------------------------

func TestSetSizeShader(t *testing.T) {
	m := newTestModel(t)
	m.SetSize(120, 40)
	if m.width != 120 || m.height != 40 {
		t.Errorf("size")
	}
}

func TestInitCmdShader(t *testing.T) {
	m := newTestModel(t)
	cmd := m.InitCmd()
	if cmd == nil {
		t.Fatal()
	}
	msg := cmd()
	if _, ok := msg.(previewDoneMsg); !ok {
		t.Fatalf("got %T", msg)
	}
}

// --- Update ---------------------------------------------------------------

func TestUpdatePreviewDoneShader(t *testing.T) {
	m := newTestModel(t)
	m2, _ := m.Update(previewDoneMsg{label: "x"})
	if !m2.statusGood {
		t.Errorf("status good")
	}
	m3, _ := m.Update(previewDoneMsg{label: "x", err: errSentinel{}})
	if m3.statusGood {
		t.Errorf("status bad expected")
	}
}

type errSentinel struct{}

func (errSentinel) Error() string { return "sentinel" }

func TestUpdateNavigationShader(t *testing.T) {
	m := newTestModel(t)
	// initially in global slot (multi).
	m, _ = m.Update(keyMsg("down"))
	m, _ = m.Update(keyMsg("j"))
	m, _ = m.Update(keyMsg("up"))
	m, _ = m.Update(keyMsg("k"))
	m, _ = m.Update(keyMsg("home"))
	m, _ = m.Update(keyMsg("end"))
	m, _ = m.Update(keyMsg("g"))
	m, _ = m.Update(keyMsg("G"))
	// tab to cursor slot
	m, _ = m.Update(keyMsg("tab"))
	if m.active != 1 {
		t.Errorf("tab")
	}
	m, _ = m.Update(keyMsg("shift+tab"))
	if m.active != 0 {
		t.Errorf("shift+tab")
	}
	m, _ = m.Update(keyMsg("right"))
	m, _ = m.Update(keyMsg("left"))
	m, _ = m.Update(keyMsg("l"))
	m, _ = m.Update(keyMsg("h"))
}

func TestToggleSelected(t *testing.T) {
	m := newTestModel(t)
	// In global slot.
	startSel := len(m.slots[0].selected)
	m, _ = m.Update(keyMsg("space"))
	if len(m.slots[0].selected) != startSel+1 {
		t.Errorf("expected +1 selected, got %d", len(m.slots[0].selected))
	}
	// Toggle again removes.
	m, _ = m.Update(keyMsg("space"))
	if len(m.slots[0].selected) != startSel {
		t.Errorf("expected back to %d, got %d", startSel, len(m.slots[0].selected))
	}
	// x is alias
	m, _ = m.Update(keyMsg("x"))
	if len(m.slots[0].selected) != startSel+1 {
		t.Errorf("x toggle")
	}
}

func TestToggleOnSingleSlotIsNoOp(t *testing.T) {
	m := newTestModel(t)
	m.active = 1 // cursor (single-select)
	beforeCursor := m.slots[1].cursor
	m, _ = m.Update(keyMsg("space"))
	if m.slots[1].cursor != beforeCursor {
		t.Errorf("cursor changed")
	}
}

func TestToggleWhenCursorNotInFiltered(t *testing.T) {
	m := newTestModel(t)
	// open search and type to filter such that cursor doesn't match.
	m, _ = m.Update(keyMsg("/"))
	m, _ = m.Update(keyMsg("d")) // matches "dither" but cursor likely 0 (bloom)
	m, _ = m.Update(keyMsg("enter"))
	// Force cursor to non-matching idx
	m.slots[0].cursor = 99 // out of any filtered set
	m, _ = m.Update(keyMsg("space"))
	if len(m.slots[0].selected) == 0 {
		t.Errorf("expected selection added")
	}
}

func TestSearchMode(t *testing.T) {
	m := newTestModel(t)
	m, _ = m.Update(keyMsg("/"))
	if !m.searching {
		t.Error()
	}
	m, _ = m.Update(keyMsg("c"))
	m, _ = m.Update(keyMsg("r"))
	m, _ = m.Update(keyMsg("backspace"))
	if m.query != "c" {
		t.Errorf("backspace got %q", m.query)
	}
	m, _ = m.Update(keyMsg("ctrl+u"))
	if m.query != "" {
		t.Errorf("ctrl+u: %q", m.query)
	}
	m, _ = m.Update(keyMsg("c"))
	m, _ = m.Update(keyMsg("enter"))
	if m.searching {
		t.Errorf("enter")
	}
	m, _ = m.Update(keyMsg("/"))
	m, _ = m.Update(keyMsg("esc"))
	if m.searching {
		t.Errorf("esc")
	}
	// down/up in search
	m, _ = m.Update(keyMsg("/"))
	m, _ = m.Update(keyMsg("down"))
	m, _ = m.Update(keyMsg("up"))
	// tab in search
	m, _ = m.Update(keyMsg("tab"))
	m, _ = m.Update(keyMsg("shift+tab"))
	// backspace empty no-op
	m.query = ""
	m, _ = m.Update(keyMsg("backspace"))
}

func TestSearchCtrlCShaderQuits(t *testing.T) {
	m := newTestModel(t)
	m, _ = m.Update(keyMsg("/"))
	_, cmd := m.Update(keyMsg("ctrl+c"))
	if cmd == nil {
		t.Fatal()
	}
	if _, ok := cmd().(ui.QuitAppMsg); !ok {
		t.Errorf("expected quit")
	}
}

func TestSearchUnknownKeyIgnoredShader(t *testing.T) {
	m := newTestModel(t)
	m, _ = m.Update(keyMsg("/"))
	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyF1})
	if cmd != nil {
		t.Error()
	}
	if m2.query != "" {
		t.Error()
	}
}

func TestUpdateNonKeyMsgShader(t *testing.T) {
	m := newTestModel(t)
	m2, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if cmd != nil {
		t.Error()
	}
	_ = m2
}

func TestUpdateUnknownKeyShader(t *testing.T) {
	m := newTestModel(t)
	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyF1})
	if cmd != nil {
		t.Error()
	}
	_ = m2
}

func TestUpdateCtrlCShader(t *testing.T) {
	m := newTestModel(t)
	_, cmd := m.Update(keyMsg("ctrl+c"))
	if cmd == nil {
		t.Fatal()
	}
	if _, ok := cmd().(ui.QuitAppMsg); !ok {
		t.Errorf("expected quit")
	}
}

func TestUpdateEscShader(t *testing.T) {
	m := newTestModel(t)
	m, _ = m.Update(keyMsg("space")) // dirty
	_, cmd := m.Update(keyMsg("esc"))
	if cmd == nil {
		t.Fatal()
	}
	_ = cmd()
}

func TestUpdateEnterShader(t *testing.T) {
	m := newTestModel(t)
	_, cmd := m.Update(keyMsg("enter"))
	if cmd == nil {
		t.Fatal()
	}
	_ = cmd()
}

func TestUpdateCtrlSShader(t *testing.T) {
	m := newTestModel(t)
	_, cmd := m.Update(keyMsg("ctrl+s"))
	if cmd == nil {
		t.Fatal()
	}
	_ = cmd()
}

// --- moveCursor / jumpEdge / ensureCursorMatches --------------------------

func TestMoveCursorEmptyFiltered(t *testing.T) {
	m := newTestModel(t)
	m.searching = true
	m.query = "zzz"
	prev := m.slots[m.active].cursor
	m, _ = m.moveCursor(1)
	if m.slots[m.active].cursor != prev {
		t.Errorf("changed")
	}
}

func TestJumpEdgeEmptyFiltered(t *testing.T) {
	m := newTestModel(t)
	m.searching = true
	m.query = "zzz"
	prev := m.slots[m.active].cursor
	m, _ = m.jumpEdge(true)
	if m.slots[m.active].cursor != prev {
		t.Errorf("changed")
	}
}

func TestEnsureCursorMatchesEmptyFiltered(t *testing.T) {
	m := newTestModel(t)
	m.searching = true
	m.query = "zzz"
	prev := m.slots[m.active].cursor
	m, _ = m.ensureCursorMatches()
	if m.slots[m.active].cursor != prev {
		t.Errorf("changed")
	}
}

func TestEnsureCursorMatchesAlreadyMatching(t *testing.T) {
	m := newTestModel(t)
	// No query → all match → cursor in filtered set → no-op.
	_, cmd := m.ensureCursorMatches()
	if cmd != nil {
		t.Errorf("expected nil cmd")
	}
}

func TestMoveCursorWrapsWhenNotInFilter(t *testing.T) {
	m := newTestModel(t)
	m.searching = true
	m.query = "crt" // only crt matches
	m.slots[0].cursor = 99
	m, _ = m.moveCursor(1)
	if m.slots[0].options[m.slots[0].cursor].Name != "crt.glsl" {
		t.Errorf("expected crt, got %s", m.slots[0].options[m.slots[0].cursor].Name)
	}
	m.slots[0].cursor = 99
	m, _ = m.moveCursor(-1)
	if m.slots[0].options[m.slots[0].cursor].Name != "crt.glsl" {
		t.Errorf("expected crt for -1")
	}
}

// --- isDirty --------------------------------------------------------------

func TestIsDirtyMulti(t *testing.T) {
	m := newTestModel(t)
	if m.isDirty() {
		t.Errorf("should start clean")
	}
	// Toggle global.
	m, _ = m.Update(keyMsg("space"))
	if !m.isDirty() {
		t.Errorf("should be dirty after toggle")
	}
}

func TestIsDirtySingleCursor(t *testing.T) {
	m := newTestModel(t)
	m.active = 1
	prev := m.slots[1].cursor
	m, _ = m.Update(keyMsg("down"))
	if m.slots[1].cursor == prev {
		t.Skip("only one cursor option")
	}
	if !m.isDirty() {
		t.Errorf("expected dirty")
	}
}

func TestSlotDirtyDifferentLengths(t *testing.T) {
	m := newTestModel(t)
	m.slots[0].selected = []int{0, 1}
	m.slots[0].selectedInit = []int{0}
	if !m.slotDirty(0) {
		t.Errorf("len diff should be dirty")
	}
	// Same length different content.
	m.slots[0].selected = []int{0, 2}
	m.slots[0].selectedInit = []int{0, 1}
	if !m.slotDirty(0) {
		t.Errorf("content diff dirty")
	}
}

// --- View ----------------------------------------------------------------

func TestViewShader(t *testing.T) {
	m := newTestModel(t)
	clean := m.View()
	if !strings.Contains(clean, "Saved state") {
		t.Errorf("clean banner: %s", clean)
	}
	// Force dirty.
	m, _ = m.Update(keyMsg("space"))
	m, _ = m.Update(previewDoneMsg{label: "x"})
	dirty := m.View()
	if !strings.Contains(dirty, "UNSAVED PREVIEW") {
		t.Errorf("dirty banner")
	}
}

func TestViewSearchHintsShader(t *testing.T) {
	m := newTestModel(t)
	m, _ = m.Update(keyMsg("/"))
	view := m.View()
	if !strings.Contains(view, "type to filter") {
		t.Errorf("missing hint: %s", view)
	}
}

func TestViewFilteredCountShader(t *testing.T) {
	m := newTestModel(t)
	m, _ = m.Update(keyMsg("/"))
	m, _ = m.Update(keyMsg("c"))
	m, _ = m.Update(keyMsg("enter"))
	view := m.View()
	if !strings.Contains(view, "of ") {
		t.Errorf("expected filter count: %s", view)
	}
}

func TestViewScrollWindowShader(t *testing.T) {
	// Create many shaders to overflow window.
	root := t.TempDir()
	cfg := filepath.Join(root, "config")
	_ = os.WriteFile(cfg, []byte(""), 0o644)
	many := map[string]string{}
	for i := 0; i < 50; i++ {
		many["sh"+string(rune('a'+i%26))+string(rune('a'+i/26))+".glsl"] = "x"
	}
	sh := writeShaderDir(t, root, "shaders", many)
	opts := ghostty.Options{ConfigPath: cfg, ShaderDir: sh, NoReload: true}
	m, err := New(opts)
	if err != nil {
		t.Fatal(err)
	}
	m.SetSize(80, 24)
	for i := 0; i < 40; i++ {
		m, _ = m.Update(keyMsg("down"))
	}
	view := m.View()
	if !strings.Contains(view, "showing ") {
		t.Errorf("expected scroll indicator: %s", view)
	}
}

func TestViewNoMatchesShader(t *testing.T) {
	m := newTestModel(t)
	m, _ = m.Update(keyMsg("/"))
	m, _ = m.Update(keyMsg("z"))
	m, _ = m.Update(keyMsg("z"))
	view := m.View()
	if !strings.Contains(view, "no matches") {
		t.Errorf("expected no matches: %s", view)
	}
}

func TestViewVerySmallHeightShader(t *testing.T) {
	m := newTestModel(t)
	m.SetSize(80, 1)
	_ = m.View()
}

func TestViewZeroWidthFooterShader(t *testing.T) {
	m := newTestModel(t)
	m.SetSize(0, 24)
	_ = m.View()
}

func TestViewStatusErrorShader(t *testing.T) {
	m := newTestModel(t)
	m, _ = m.Update(previewDoneMsg{err: errSentinel{}, label: "x"})
	view := m.View()
	if !strings.Contains(view, "sentinel") {
		t.Errorf("expected error: %s", view)
	}
}

func TestRenderMarker(t *testing.T) {
	// Single slot returns blank.
	if got := renderMarker(slotState{multi: false}, 0); got != " " {
		t.Errorf("got %q", got)
	}
	// Multi slot, not selected → [ ]
	s := slotState{multi: true, options: []option{{Name: "a"}}}
	if got := renderMarker(s, 0); !strings.Contains(got, "[ ]") {
		t.Errorf("got %q", got)
	}
	// Selected → [N]
	s.selected = []int{0}
	if got := renderMarker(s, 0); !strings.Contains(got, "[1]") {
		t.Errorf("got %q", got)
	}
}

// --- tea.Cmd helpers ------------------------------------------------------

func TestPreviewCmdShader(t *testing.T) {
	m := newTestModel(t)
	cmd := previewCmd(m.opts, m.currentSelection())
	msg := cmd().(previewDoneMsg)
	if msg.err != nil {
		t.Errorf("err: %v", msg.err)
	}
}

func TestPreviewCmdShaderError(t *testing.T) {
	root := t.TempDir()
	cfgDir := filepath.Join(root, "cfg")
	_ = os.MkdirAll(cfgDir, 0o755)
	opts := ghostty.Options{ConfigPath: cfgDir, NoReload: true}
	cmd := previewCmd(opts, selection{globals: []option{{Rel: "x"}}})
	msg := cmd().(previewDoneMsg)
	if msg.err == nil {
		t.Errorf("expected error")
	}
}

func TestCommitAndBackCmdShader(t *testing.T) {
	cmd := commitAndBackCmd(selection{}, true)
	batch := cmd().(tea.BatchMsg)
	sawSaved, sawSwitch := false, false
	for _, sub := range batch {
		switch m := sub().(type) {
		case ui.ShowToastMsg:
			if m.Kind == ui.ToastSaved {
				sawSaved = true
			}
		case ui.SwitchScreenMsg:
			if m.Target == ui.ScreenMenu {
				sawSwitch = true
			}
		}
	}
	if !sawSaved || !sawSwitch {
		t.Errorf("missing batch parts")
	}
}

func TestCommitAndBackCmdNotDirtyShader(t *testing.T) {
	cmd := commitAndBackCmd(selection{}, false)
	batch := cmd().(tea.BatchMsg)
	for _, sub := range batch {
		if toast, ok := sub().(ui.ShowToastMsg); ok {
			if toast.Kind != ui.ToastInfo {
				t.Errorf("info expected")
			}
		}
	}
}

func TestRestoreAndBackCmdShader(t *testing.T) {
	m := newTestModel(t)
	cmd := restoreAndBackCmd(m.opts, m.initialSelection(), true)
	batch := cmd().(tea.BatchMsg)
	for _, sub := range batch {
		if toast, ok := sub().(ui.ShowToastMsg); ok {
			if toast.Kind != ui.ToastReverted {
				t.Errorf("reverted expected")
			}
		}
	}
}

func TestRestoreAndBackCmdNotDirtyShader(t *testing.T) {
	m := newTestModel(t)
	cmd := restoreAndBackCmd(m.opts, m.initialSelection(), false)
	batch := cmd().(tea.BatchMsg)
	for _, sub := range batch {
		if toast, ok := sub().(ui.ShowToastMsg); ok {
			if toast.Kind != ui.ToastInfo {
				t.Errorf("info")
			}
		}
	}
}

func TestRestoreAndQuitCmdShader(t *testing.T) {
	m := newTestModel(t)
	cmd := restoreAndQuitCmd(m.opts, m.initialSelection())
	if _, ok := cmd().(ui.QuitAppMsg); !ok {
		t.Errorf("expected quit")
	}
}

func TestInitialSelectionCursor(t *testing.T) {
	// Load model with a current cursor shader set in config.
	root := t.TempDir()
	cfg := filepath.Join(root, "config")
	sh := writeShaderDir(t, root, "shaders", map[string]string{
		"crt.glsl":          "x",
		"cursor_blaze.glsl": "y",
	})
	_ = os.WriteFile(cfg, []byte("custom-shader = shaders/cursor_blaze.glsl\n"), 0o644)
	opts := ghostty.Options{ConfigPath: cfg, ShaderDir: sh, NoReload: true}
	m, err := New(opts)
	if err != nil {
		t.Fatal(err)
	}
	sel := m.initialSelection()
	if sel.cursor.Path == "" {
		t.Errorf("expected cursor in initial selection")
	}
}

func TestToggleSelectedNoMatches(t *testing.T) {
	m := newTestModel(t)
	m.searching = true
	m.query = "zzz"
	before := len(m.slots[0].selected)
	m, _ = m.toggleSelected()
	if len(m.slots[0].selected) != before {
		t.Errorf("should be no-op")
	}
}

func TestViewOriginalMarkerForSingleSlot(t *testing.T) {
	// Use cursor slot, ensure cursor moved to a non-initial idx so initial renders with "original".
	root := t.TempDir()
	cfg := filepath.Join(root, "config")
	sh := writeShaderDir(t, root, "shaders", map[string]string{
		"crt.glsl":          "x",
		"cursor_blaze.glsl": "y",
		"cursor_warp.glsl":  "y",
	})
	_ = os.WriteFile(cfg, []byte("custom-shader = shaders/cursor_blaze.glsl\n"), 0o644)
	opts := ghostty.Options{ConfigPath: cfg, ShaderDir: sh, NoReload: true}
	m, err := New(opts)
	if err != nil {
		t.Fatal(err)
	}
	m.SetSize(80, 24)
	m.active = 1
	// Move cursor down so initialCur (cursor_blaze) is no longer under cursor.
	m, _ = m.Update(keyMsg("down"))
	view := m.View()
	if !strings.Contains(view, "original") {
		t.Errorf("expected original marker: %s", view)
	}
}

func TestViewStatusGoodCleanShader(t *testing.T) {
	m := newTestModel(t)
	// Trigger a successful preview without making dirty.
	m, _ = m.Update(previewDoneMsg{label: "alpha"})
	if m.isDirty() {
		t.Fatalf("should be clean")
	}
	view := m.View()
	if !strings.Contains(view, "previewing alpha") {
		t.Errorf("expected good status: %s", view)
	}
}

func TestApplyPreviewWithCursor(t *testing.T) {
	m := newTestModel(t)
	// Tab to cursor slot and ensure currentSelection contains cursor.
	m.active = 1
	m.slots[1].cursor = 1 // skip "None"
	sel := m.currentSelection()
	if sel.cursor.Path == "" {
		t.Skip("no cursor available")
	}
	if err := applyPreview(m.opts, sel); err != nil {
		t.Errorf("apply: %v", err)
	}
}
