package shader

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"ghostty-config/internal/ghostty"
	"ghostty-config/internal/ui"
)

type kind int

const (
	kindGlobal kind = iota
	kindCursor
)

const numSlots = 2

type option struct {
	Name string
	Path string
	Rel  string
	Kind kind
}

type slotState struct {
	options      []option
	cursor       int
	initialCur   int
	selected     []int
	selectedInit []int
	multi        bool
}

func (s slotState) effective() []option {
	if s.multi {
		out := make([]option, 0, len(s.selected))
		for _, i := range s.selected {
			if i >= 0 && i < len(s.options) && s.options[i].Path != "" {
				out = append(out, s.options[i])
			}
		}
		return out
	}
	if s.cursor < 0 || s.cursor >= len(s.options) {
		return nil
	}
	if s.options[s.cursor].Path == "" {
		return nil
	}
	return []option{s.options[s.cursor]}
}

func (s slotState) initialEffective() []option {
	if s.multi {
		out := make([]option, 0, len(s.selectedInit))
		for _, i := range s.selectedInit {
			if i >= 0 && i < len(s.options) && s.options[i].Path != "" {
				out = append(out, s.options[i])
			}
		}
		return out
	}
	if s.initialCur < 0 || s.initialCur >= len(s.options) {
		return nil
	}
	if s.options[s.initialCur].Path == "" {
		return nil
	}
	return []option{s.options[s.initialCur]}
}

func (s slotState) selectionPos(idx int) int {
	for pos, i := range s.selected {
		if i == idx {
			return pos
		}
	}
	return -1
}

type selection struct {
	globals []option
	cursor  option
}

type previewDoneMsg struct {
	label string
	err   error
}

type Model struct {
	opts       ghostty.Options
	slots      [numSlots]slotState
	active     int
	status     string
	statusGood bool
	lastErr    error
	width      int
	height     int
	searching  bool
	query      string
}

func New(opts ghostty.Options) (Model, error) {
	global, cursor, err := discoverShaders(opts.ConfigPath, opts.ShaderDir)
	if err != nil {
		return Model{}, err
	}
	if len(global) == 0 && len(cursor) <= 1 {
		return Model{}, fmt.Errorf("no .glsl shaders found in %s", opts.ShaderDir)
	}

	curGlobals, curCursor := readCurrentShaders(opts.ConfigPath)

	selected := make([]int, 0, len(curGlobals))
	for _, p := range curGlobals {
		idx := indexForShader(global, p)
		if idx < 0 {
			global = append(global, option{
				Name: "Current outside shader-dir: " + filepath.Base(p),
				Path: p,
				Rel:  pathForConfig(opts.ConfigPath, p),
				Kind: kindGlobal,
			})
			idx = len(global) - 1
		}
		selected = append(selected, idx)
	}

	cIdx := indexForShader(cursor, curCursor)
	if curCursor != "" && cIdx < 0 {
		cursor = append(cursor, option{
			Name: "Current outside shader-dir: " + filepath.Base(curCursor),
			Path: curCursor,
			Rel:  pathForConfig(opts.ConfigPath, curCursor),
			Kind: kindCursor,
		})
		cIdx = len(cursor) - 1
	}
	if cIdx < 0 {
		cIdx = 0
	}

	selectedInit := append([]int(nil), selected...)

	m := Model{
		opts: opts,
		slots: [numSlots]slotState{
			{
				options:      global,
				cursor:       0,
				initialCur:   0,
				selected:     selected,
				selectedInit: selectedInit,
				multi:        true,
			},
			{
				options:    cursor,
				cursor:     cIdx,
				initialCur: cIdx,
				multi:      false,
			},
		},
	}

	if len(m.slots[0].options) == 0 {
		m.active = 1
	}

	return m, nil
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *Model) InitCmd() tea.Cmd {
	return previewCmd(m.opts, m.currentSelection())
}

func (m Model) currentSelection() selection {
	sel := selection{globals: m.slots[0].effective()}
	cursorEff := m.slots[1].effective()
	if len(cursorEff) > 0 {
		sel.cursor = cursorEff[0]
	}
	return sel
}

func (m Model) initialSelection() selection {
	sel := selection{globals: m.slots[0].initialEffective()}
	cursorEff := m.slots[1].initialEffective()
	if len(cursorEff) > 0 {
		sel.cursor = cursorEff[0]
	}
	return sel
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case previewDoneMsg:
		if msg.err != nil {
			m.lastErr = msg.err
			m.statusGood = false
			m.status = fmt.Sprintf("%s: %v", msg.label, msg.err)
		} else {
			m.lastErr = nil
			m.statusGood = true
			m.status = "previewing " + msg.label
		}
		return m, nil
	case tea.KeyMsg:
		if m.searching {
			return m.updateSearching(msg)
		}
		switch msg.String() {
		case "ctrl+c":
			return m, restoreAndQuitCmd(m.opts, m.initialSelection())
		case "esc", "q":
			return m, restoreAndBackCmd(m.opts, m.initialSelection())
		case "enter":
			return m, func() tea.Msg { return ui.SwitchScreenMsg{Target: ui.ScreenMenu} }
		case "tab", "shift+tab", "right", "left", "l", "h":
			m.active = (m.active + 1) % numSlots
			m.query = ""
			return m, nil
		case " ", "space", "x":
			return m.toggleSelected()
		case "up", "k":
			return m.moveCursor(-1)
		case "down", "j":
			return m.moveCursor(1)
		case "home", "g":
			return m.jumpEdge(true)
		case "end", "G":
			return m.jumpEdge(false)
		case "/":
			m.searching = true
			return m, nil
		}
	}
	return m, nil
}

func (m Model) updateSearching(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searching = false
		m.query = ""
		return m.ensureCursorMatches()
	case "enter":
		m.searching = false
		return m, nil
	case "ctrl+c":
		return m, restoreAndQuitCmd(m.opts, m.initialSelection())
	case "ctrl+u":
		m.query = ""
		return m.ensureCursorMatches()
	case "backspace":
		if len(m.query) > 0 {
			r := []rune(m.query)
			m.query = string(r[:len(r)-1])
			return m.ensureCursorMatches()
		}
		return m, nil
	case "up":
		return m.moveCursor(-1)
	case "down":
		return m.moveCursor(1)
	case "tab", "shift+tab":
		m.active = (m.active + 1) % numSlots
		m.query = ""
		return m, nil
	}
	if len(msg.Runes) > 0 {
		m.query += string(msg.Runes)
		return m.ensureCursorMatches()
	}
	return m, nil
}

func (m Model) filteredIndices() []int {
	slot := m.slots[m.active]
	q := strings.ToLower(m.query)
	out := make([]int, 0, len(slot.options))
	for i, sh := range slot.options {
		if q == "" || strings.Contains(strings.ToLower(sh.Name), q) {
			out = append(out, i)
		}
	}
	return out
}

func (m Model) previewIfSingle() tea.Cmd {
	if m.slots[m.active].multi {
		return nil
	}
	return previewCmd(m.opts, m.currentSelection())
}

func (m Model) toggleSelected() (Model, tea.Cmd) {
	slot := &m.slots[m.active]
	if !slot.multi {
		return m, nil
	}
	idxs := m.filteredIndices()
	if len(idxs) == 0 {
		return m, nil
	}
	idx := slot.cursor
	found := false
	for _, i := range idxs {
		if i == idx {
			found = true
			break
		}
	}
	if !found {
		idx = idxs[0]
		slot.cursor = idx
	}
	pos := slot.selectionPos(idx)
	if pos >= 0 {
		slot.selected = append(slot.selected[:pos], slot.selected[pos+1:]...)
	} else {
		slot.selected = append(slot.selected, idx)
	}
	return m, previewCmd(m.opts, m.currentSelection())
}

func (m Model) moveCursor(delta int) (Model, tea.Cmd) {
	idxs := m.filteredIndices()
	if len(idxs) == 0 {
		return m, nil
	}
	slot := &m.slots[m.active]
	pos := -1
	for i, idx := range idxs {
		if idx == slot.cursor {
			pos = i
			break
		}
	}
	if pos == -1 {
		if delta >= 0 {
			slot.cursor = idxs[0]
		} else {
			slot.cursor = idxs[len(idxs)-1]
		}
	} else {
		pos = (pos + delta + len(idxs)) % len(idxs)
		slot.cursor = idxs[pos]
	}
	return m, m.previewIfSingle()
}

func (m Model) jumpEdge(first bool) (Model, tea.Cmd) {
	idxs := m.filteredIndices()
	if len(idxs) == 0 {
		return m, nil
	}
	slot := &m.slots[m.active]
	if first {
		slot.cursor = idxs[0]
	} else {
		slot.cursor = idxs[len(idxs)-1]
	}
	return m, m.previewIfSingle()
}

func (m Model) ensureCursorMatches() (Model, tea.Cmd) {
	idxs := m.filteredIndices()
	if len(idxs) == 0 {
		return m, nil
	}
	slot := &m.slots[m.active]
	for _, idx := range idxs {
		if idx == slot.cursor {
			return m, nil
		}
	}
	prev := slot.cursor
	slot.cursor = idxs[0]
	if slot.cursor != prev {
		return m, m.previewIfSingle()
	}
	return m, nil
}

func slotLabel(i int) string {
	if i == 0 {
		return "Global"
	}
	return "Cursor"
}

func (m Model) View() string {
	var b strings.Builder
	b.WriteString(ui.TitleStyle.Render("Shaders"))
	b.WriteString("  ")
	b.WriteString(ui.MutedStyle.Render("(Enter: keep • Esc/q: cancel and back to menu)"))
	b.WriteString("\n")

	var tabs []string
	for i := 0; i < numSlots; i++ {
		slot := m.slots[i]
		label := slotLabel(i) + ": " + slotSummary(slot)
		if i == m.active {
			tabs = append(tabs, ui.ActiveTabStyle.Render("▸ "+label))
		} else {
			tabs = append(tabs, ui.InactiveTabStyle.Render("  "+label))
		}
	}
	b.WriteString(strings.Join(tabs, " "))
	b.WriteString("\n")

	if m.searching {
		b.WriteString(ui.MutedStyle.Render("type to filter  •  ↑/↓: navigate  •  Enter: apply filter  •  Backspace: delete  •  Ctrl+U: clear  •  Esc: exit search"))
	} else {
		hint := "↑/↓ or j/k: navigate"
		if m.slots[m.active].multi {
			hint += "  •  Space: toggle"
		} else {
			hint += " & preview"
		}
		hint += "  •  Tab: switch slot  •  /: search"
		if m.query != "" {
			hint += " (filter active)"
		}
		b.WriteString(ui.MutedStyle.Render(hint))
	}
	b.WriteString("\n")
	if m.searching || m.query != "" {
		cursor := ""
		if m.searching {
			cursor = "_"
		}
		b.WriteString(ui.MutedStyle.Render("/") + m.query + ui.MutedStyle.Render(cursor))
		b.WriteByte('\n')
	}
	b.WriteByte('\n')

	slot := m.slots[m.active]
	idxs := m.filteredIndices()

	visibleHeight := m.height - 11
	if visibleHeight < 8 {
		visibleHeight = 8
	}

	cursorPos := 0
	for i, idx := range idxs {
		if idx == slot.cursor {
			cursorPos = i
			break
		}
	}

	start := 0
	if cursorPos >= visibleHeight {
		start = cursorPos - visibleHeight + 1
	}
	end := start + visibleHeight
	if end > len(idxs) {
		end = len(idxs)
	}

	if len(idxs) == 0 {
		b.WriteString(ui.MutedStyle.Render("  no matches"))
		b.WriteByte('\n')
	}

	for i := start; i < end; i++ {
		idx := idxs[i]
		sh := slot.options[idx]
		marker := renderMarker(slot, idx)
		body := marker + " " + sh.Name
		var line string
		switch {
		case idx == slot.cursor:
			line = ui.SelectedStyle.Render("▸ " + body)
		case !slot.multi && idx == slot.initialCur:
			line = ui.SuccessStyle.Render("• "+body) + ui.MutedStyle.Render("  original")
		default:
			line = "  " + body
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}

	if start > 0 || end < len(idxs) {
		b.WriteString(ui.MutedStyle.Render(fmt.Sprintf("showing %d-%d of %d", start+1, end, len(idxs))))
		b.WriteByte('\n')
	} else if m.query != "" && len(idxs) > 0 {
		b.WriteString(ui.MutedStyle.Render(fmt.Sprintf("%d of %d", len(idxs), len(slot.options))))
		b.WriteByte('\n')
	}

	b.WriteByte('\n')
	status := m.status
	if status == "" {
		status = "ready"
	}
	if m.statusGood {
		b.WriteString(ui.SuccessStyle.Render(status))
	} else if m.lastErr != nil {
		b.WriteString(ui.ErrorStyle.Render(status))
	} else {
		b.WriteString(ui.MutedStyle.Render(status))
	}
	b.WriteByte('\n')

	if slot.cursor >= 0 && slot.cursor < len(slot.options) {
		selected := slot.options[slot.cursor]
		if selected.Rel != "" {
			b.WriteString(ui.MutedStyle.Render(selected.Rel))
			b.WriteByte('\n')
		}
	}

	return b.String()
}

func renderMarker(slot slotState, idx int) string {
	if !slot.multi {
		return " "
	}
	pos := slot.selectionPos(idx)
	if pos < 0 {
		return ui.MutedStyle.Render("[ ]")
	}
	return ui.CheckedStyle.Render(fmt.Sprintf("[%d]", pos+1))
}

func slotSummary(slot slotState) string {
	eff := slot.effective()
	if len(eff) == 0 {
		return "none"
	}
	names := make([]string, len(eff))
	for i, sh := range eff {
		names[i] = sh.Name
	}
	return strings.Join(names, " → ")
}

func selectionLabel(sel selection) string {
	g := "none"
	if len(sel.globals) > 0 {
		names := make([]string, len(sel.globals))
		for i, x := range sel.globals {
			names[i] = x.Name
		}
		g = strings.Join(names, " → ")
	}
	c := "none"
	if sel.cursor.Path != "" {
		c = sel.cursor.Name
	}
	return fmt.Sprintf("global=%s · cursor=%s", g, c)
}

func previewCmd(opts ghostty.Options, sel selection) tea.Cmd {
	return func() tea.Msg {
		label := selectionLabel(sel)
		if err := applyPreview(opts, sel); err != nil {
			return previewDoneMsg{label: label, err: err}
		}
		return previewDoneMsg{label: label}
	}
}

func restoreAndBackCmd(opts ghostty.Options, sel selection) tea.Cmd {
	return func() tea.Msg {
		_ = applyPreview(opts, sel)
		return ui.SwitchScreenMsg{Target: ui.ScreenMenu}
	}
}

func restoreAndQuitCmd(opts ghostty.Options, sel selection) tea.Cmd {
	return func() tea.Msg {
		_ = applyPreview(opts, sel)
		return ui.QuitAppMsg{}
	}
}

func applyPreview(opts ghostty.Options, sel selection) error {
	desired := make([]string, 0, len(sel.globals)+1)
	for _, g := range sel.globals {
		desired = append(desired, g.Rel)
	}
	if sel.cursor.Path != "" {
		desired = append(desired, sel.cursor.Rel)
	}
	if err := ghostty.WriteConfigKey(opts.ConfigPath, "custom-shader", desired); err != nil {
		return err
	}
	return ghostty.Reload(opts)
}

func isCursorShader(name string) bool {
	return strings.Contains(strings.ToLower(name), "cursor")
}

func discoverShaders(configPath, shaderDir string) ([]option, []option, error) {
	if _, err := os.Stat(shaderDir); err != nil {
		return nil, nil, err
	}

	global := []option{}
	cursor := []option{{Name: "None (no cursor shader)", Path: "", Rel: "", Kind: kindCursor}}

	err := filepath.WalkDir(shaderDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.ToLower(filepath.Ext(path)) != ".glsl" {
			return nil
		}

		abs, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		rel := pathForConfig(configPath, abs)

		name, err := filepath.Rel(shaderDir, abs)
		if err != nil {
			name = filepath.Base(abs)
		}
		opt := option{Name: filepath.ToSlash(name), Path: abs, Rel: rel}
		if isCursorShader(filepath.Base(abs)) {
			opt.Kind = kindCursor
			cursor = append(cursor, opt)
		} else {
			opt.Kind = kindGlobal
			global = append(global, opt)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	sort.Slice(global, func(i, j int) bool {
		return strings.ToLower(global[i].Name) < strings.ToLower(global[j].Name)
	})
	sort.Slice(cursor[1:], func(i, j int) bool {
		return strings.ToLower(cursor[1+i].Name) < strings.ToLower(cursor[1+j].Name)
	})
	return global, cursor, nil
}

func pathForConfig(configPath, shaderPath string) string {
	rel, err := filepath.Rel(filepath.Dir(configPath), shaderPath)
	if err != nil || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || rel == ".." {
		rel = shaderPath
	}
	return filepath.ToSlash(rel)
}

func readCurrentShaders(configPath string) ([]string, string) {
	values := ghostty.ReadActiveValues(configPath, "custom-shader")
	configDir := filepath.Dir(configPath)

	var globals []string
	var cursorPath string
	for _, value := range values {
		raw := value
		if !filepath.IsAbs(raw) {
			raw = filepath.Join(configDir, raw)
		}
		abs, err := filepath.Abs(raw)
		if err != nil {
			abs = raw
		}
		abs = filepath.Clean(abs)
		if isCursorShader(filepath.Base(abs)) {
			if cursorPath == "" {
				cursorPath = abs
			}
		} else {
			globals = append(globals, abs)
		}
	}
	return globals, cursorPath
}

func indexForShader(opts []option, current string) int {
	if current == "" {
		return -1
	}
	current = filepath.Clean(current)
	for i, sh := range opts {
		if sh.Path != "" && filepath.Clean(sh.Path) == current {
			return i
		}
	}
	return -1
}
