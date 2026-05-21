package theme

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"ghostty-config/internal/ghostty"
	"ghostty-config/internal/ui"
)

const toastDelay = 850 * time.Millisecond

const numSlots = 2

const (
	slotLight = 0
	slotDark  = 1
)

type option struct {
	Name   string
	Path   string
	Source string
}

type selection struct {
	light string
	dark  string
}

type previewDoneMsg struct {
	label string
	err   error
}

type Model struct {
	opts             ghostty.Options
	options          []option
	active           int
	cursors          [numSlots]int
	initialCursors   [numSlots]int
	initialRawValues []string
	status           string
	statusGood       bool
	lastErr          error
	width            int
	height           int
	searching        bool
	query            string
}

func New(opts ghostty.Options) (Model, error) {
	themes, err := discoverThemes(opts.UserThemeDir, opts.SystemThemeDir)
	if err != nil {
		return Model{}, err
	}
	if len(themes) == 0 {
		return Model{}, fmt.Errorf("no themes found in %s or %s", opts.UserThemeDir, opts.SystemThemeDir)
	}

	rawValues := ghostty.ReadActiveValues(opts.ConfigPath, "theme")
	curLight, curDark := parseThemeValues(rawValues)

	lightIdx := indexForTheme(themes, curLight)
	darkIdx := indexForTheme(themes, curDark)
	if lightIdx < 0 {
		lightIdx = 0
	}
	if darkIdx < 0 {
		darkIdx = 0
	}

	m := Model{
		opts:             opts,
		options:          themes,
		active:           slotLight,
		cursors:          [numSlots]int{lightIdx, darkIdx},
		initialCursors:   [numSlots]int{lightIdx, darkIdx},
		initialRawValues: rawValues,
	}
	return m, nil
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *Model) InitCmd() tea.Cmd {
	return previewCmd(m.opts, m.previewName())
}

func (m Model) previewName() string {
	idx := m.cursors[m.active]
	if idx < 0 || idx >= len(m.options) {
		return ""
	}
	return m.options[idx].Name
}

func (m Model) isDirty() bool {
	for i := 0; i < numSlots; i++ {
		if m.cursors[i] != m.initialCursors[i] {
			return true
		}
	}
	return false
}

func (m Model) currentSelection() selection {
	light := m.cursors[slotLight]
	dark := m.cursors[slotDark]
	sel := selection{}
	if light >= 0 && light < len(m.options) {
		sel.light = m.options[light].Name
	}
	if dark >= 0 && dark < len(m.options) {
		sel.dark = m.options[dark].Name
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
			return m, restoreAndQuitCmd(m.opts, m.initialRawValues)
		case "esc", "q":
			return m, restoreAndBackCmd(m.opts, m.initialRawValues, m.isDirty())
		case "enter", "ctrl+s":
			return m, commitAndBackCmd(m.opts, m.currentSelection(), m.isDirty())
		case "tab", "shift+tab", "right", "left", "l", "h":
			m.active = (m.active + 1) % numSlots
			m.query = ""
			return m, previewCmd(m.opts, m.previewName())
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
		return m, restoreAndQuitCmd(m.opts, m.initialRawValues)
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
		return m, previewCmd(m.opts, m.previewName())
	}
	if len(msg.Runes) > 0 {
		m.query += string(msg.Runes)
		return m.ensureCursorMatches()
	}
	return m, nil
}

func (m Model) filteredIndices() []int {
	q := strings.ToLower(m.query)
	out := make([]int, 0, len(m.options))
	for i, t := range m.options {
		if q == "" || strings.Contains(strings.ToLower(t.Name), q) {
			out = append(out, i)
		}
	}
	return out
}

func (m Model) moveCursor(delta int) (Model, tea.Cmd) {
	idxs := m.filteredIndices()
	if len(idxs) == 0 {
		return m, nil
	}
	cursor := m.cursors[m.active]
	pos := -1
	for i, idx := range idxs {
		if idx == cursor {
			pos = i
			break
		}
	}
	if pos == -1 {
		if delta >= 0 {
			cursor = idxs[0]
		} else {
			cursor = idxs[len(idxs)-1]
		}
	} else {
		pos = (pos + delta + len(idxs)) % len(idxs)
		cursor = idxs[pos]
	}
	m.cursors[m.active] = cursor
	return m, previewCmd(m.opts, m.previewName())
}

func (m Model) jumpEdge(first bool) (Model, tea.Cmd) {
	idxs := m.filteredIndices()
	if len(idxs) == 0 {
		return m, nil
	}
	if first {
		m.cursors[m.active] = idxs[0]
	} else {
		m.cursors[m.active] = idxs[len(idxs)-1]
	}
	return m, previewCmd(m.opts, m.previewName())
}

func (m Model) ensureCursorMatches() (Model, tea.Cmd) {
	idxs := m.filteredIndices()
	if len(idxs) == 0 {
		return m, nil
	}
	for _, idx := range idxs {
		if idx == m.cursors[m.active] {
			return m, nil
		}
	}
	prev := m.cursors[m.active]
	m.cursors[m.active] = idxs[0]
	if m.cursors[m.active] != prev {
		return m, previewCmd(m.opts, m.previewName())
	}
	return m, nil
}

func slotLabelFor(i int) string {
	if i == slotLight {
		return "Light"
	}
	return "Dark"
}

func (m Model) View() string {
	var b strings.Builder
	b.WriteString(ui.RenderBreadcrumb("Ghostty configurator", "Themes"))
	b.WriteByte('\n')

	dirty := m.isDirty()
	if dirty {
		b.WriteString(ui.RenderWarnBanner("UNSAVED PREVIEW — press Enter to keep, Esc to revert"))
	} else {
		b.WriteString(ui.CleanCheckStyle.Render("✓ Saved state · navigate to preview new themes"))
	}
	b.WriteByte('\n')

	var tabs []string
	for i := 0; i < numSlots; i++ {
		name := "—"
		if m.cursors[i] >= 0 && m.cursors[i] < len(m.options) {
			name = m.options[m.cursors[i]].Name
		}
		marker := ui.CleanCheckStyle.Render(" ✓")
		if m.cursors[i] != m.initialCursors[i] {
			marker = ui.DirtyDotStyle.Render(" ●")
		}
		label := slotLabelFor(i) + ": " + name + marker
		if i == m.active {
			tabs = append(tabs, ui.ActiveTabStyle.Render("▸ "+label))
		} else {
			tabs = append(tabs, ui.InactiveTabStyle.Render("  "+label))
		}
	}
	b.WriteString(strings.Join(tabs, " "))
	b.WriteByte('\n')

	if m.searching {
		b.WriteString(ui.MutedStyle.Render("type to filter  •  ↑/↓: navigate  •  Enter: apply filter  •  Backspace: delete  •  Ctrl+U: clear  •  Esc: exit search"))
		b.WriteByte('\n')
	}
	if m.searching || m.query != "" {
		cursor := ""
		if m.searching {
			cursor = "_"
		}
		b.WriteString(ui.MutedStyle.Render("/") + m.query + ui.MutedStyle.Render(cursor))
		b.WriteByte('\n')
	}
	b.WriteByte('\n')

	idxs := m.filteredIndices()
	cursor := m.cursors[m.active]

	visibleHeight := m.height - 11
	if visibleHeight < 8 {
		visibleHeight = 8
	}

	cursorPos := 0
	for i, idx := range idxs {
		if idx == cursor {
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
		t := m.options[idx]
		body := t.Name + "  " + renderBadge(t)
		var line string
		switch {
		case idx == cursor:
			line = ui.SelectedStyle.Render("▸ " + body)
		case idx == m.initialCursors[m.active]:
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
		b.WriteString(ui.MutedStyle.Render(fmt.Sprintf("%d of %d", len(idxs), len(m.options))))
		b.WriteByte('\n')
	}

	b.WriteByte('\n')
	status := m.status
	if status == "" {
		status = "ready"
	}
	switch {
	case m.lastErr != nil:
		b.WriteString(ui.ErrorStyle.Render(status))
	case dirty && m.statusGood:
		b.WriteString(ui.WarnStyle.Render("► " + status + " — not saved yet (press Enter to keep)"))
	case m.statusGood:
		b.WriteString(ui.SuccessStyle.Render(status))
	default:
		b.WriteString(ui.MutedStyle.Render(status))
	}
	b.WriteByte('\n')

	if cursor >= 0 && cursor < len(m.options) {
		t := m.options[cursor]
		b.WriteString(ui.MutedStyle.Render(t.Path))
		b.WriteByte('\n')
	}

	b.WriteByte('\n')
	footerWidth := m.width
	if footerWidth < 1 {
		footerWidth = 80
	}
	b.WriteString(ui.RenderFooter(footerWidth, []ui.FooterAction{
		{Key: "ENTER", Label: "Save", Variant: ui.FooterSave},
		{Key: "ESC", Label: "Cancel & revert", Variant: ui.FooterCancel},
		{Key: "TAB", Label: "Switch slot", Variant: ui.FooterDefault},
		{Key: "/", Label: "Search", Variant: ui.FooterDefault},
		{Key: "?", Label: "Help", Variant: ui.FooterDefault},
	}))
	b.WriteByte('\n')

	return b.String()
}

func renderBadge(t option) string {
	if t.Source == "user" {
		return ui.BadgeUserStyle.Render("[user]")
	}
	return ui.BadgeSysStyle.Render("[builtin]")
}

func previewCmd(opts ghostty.Options, name string) tea.Cmd {
	return func() tea.Msg {
		if name == "" {
			return previewDoneMsg{label: "none"}
		}
		if err := applyPreview(opts, name); err != nil {
			return previewDoneMsg{label: name, err: err}
		}
		return previewDoneMsg{label: name}
	}
}

func commitAndBackCmd(opts ghostty.Options, sel selection, dirty bool) tea.Cmd {
	write := func() tea.Msg {
		if err := writeSelection(opts, sel); err != nil {
			return previewDoneMsg{label: selectionLabel(sel), err: err}
		}
		_ = ghostty.Reload(opts)
		toastText := "No changes"
		kind := ui.ToastInfo
		if dirty {
			toastText = "Saved · " + selectionLabel(sel)
			kind = ui.ToastSaved
		}
		return ui.ShowToastMsg{Text: toastText, Kind: kind}
	}
	back := tea.Tick(toastDelay, func(time.Time) tea.Msg {
		return ui.SwitchScreenMsg{Target: ui.ScreenMenu}
	})
	return tea.Batch(write, back)
}

func restoreAndBackCmd(opts ghostty.Options, raw []string, dirty bool) tea.Cmd {
	restore := func() tea.Msg {
		_ = restoreValues(opts, raw)
		if dirty {
			return ui.ShowToastMsg{Text: "Reverted to previous theme", Kind: ui.ToastReverted}
		}
		return ui.ShowToastMsg{Text: "No changes", Kind: ui.ToastInfo}
	}
	back := tea.Tick(toastDelay, func(time.Time) tea.Msg {
		return ui.SwitchScreenMsg{Target: ui.ScreenMenu}
	})
	return tea.Batch(restore, back)
}

func restoreAndQuitCmd(opts ghostty.Options, raw []string) tea.Cmd {
	return func() tea.Msg {
		_ = restoreValues(opts, raw)
		return ui.QuitAppMsg{}
	}
}

func applyPreview(opts ghostty.Options, name string) error {
	if err := ghostty.WriteConfigKey(opts.ConfigPath, "theme", []string{name}); err != nil {
		return err
	}
	return ghostty.Reload(opts)
}

func writeSelection(opts ghostty.Options, sel selection) error {
	if sel.light == "" && sel.dark == "" {
		return ghostty.WriteConfigKey(opts.ConfigPath, "theme", nil)
	}
	if sel.light == sel.dark {
		return ghostty.WriteConfigKey(opts.ConfigPath, "theme", []string{sel.light})
	}
	parts := []string{}
	if sel.light != "" {
		parts = append(parts, "light:"+sel.light)
	}
	if sel.dark != "" {
		parts = append(parts, "dark:"+sel.dark)
	}
	return ghostty.WriteConfigKey(opts.ConfigPath, "theme", []string{strings.Join(parts, ",")})
}

func restoreValues(opts ghostty.Options, raw []string) error {
	if err := ghostty.WriteConfigKey(opts.ConfigPath, "theme", raw); err != nil {
		return err
	}
	return ghostty.Reload(opts)
}

func selectionLabel(sel selection) string {
	if sel.light == sel.dark {
		if sel.light == "" {
			return "none"
		}
		return sel.light
	}
	return fmt.Sprintf("light=%s · dark=%s", sel.light, sel.dark)
}

func parseThemeValues(values []string) (string, string) {
	var light, dark string
	for _, v := range values {
		parts := strings.Split(v, ",")
		simple := ""
		for _, p := range parts {
			p = strings.TrimSpace(p)
			switch {
			case strings.HasPrefix(p, "light:"):
				light = strings.TrimSpace(strings.TrimPrefix(p, "light:"))
			case strings.HasPrefix(p, "dark:"):
				dark = strings.TrimSpace(strings.TrimPrefix(p, "dark:"))
			case p != "":
				simple = p
			}
		}
		if simple != "" {
			if light == "" {
				light = simple
			}
			if dark == "" {
				dark = simple
			}
		}
	}
	return light, dark
}

func indexForTheme(themes []option, name string) int {
	if name == "" {
		return -1
	}
	for i, t := range themes {
		if t.Name == name {
			return i
		}
	}
	lower := strings.ToLower(name)
	for i, t := range themes {
		if strings.ToLower(t.Name) == lower {
			return i
		}
	}
	return -1
}

func discoverThemes(userDir, systemDir string) ([]option, error) {
	byName := map[string]option{}

	if err := collectThemes(systemDir, "builtin", byName); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if err := collectThemes(userDir, "user", byName); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	out := make([]option, 0, len(byName))
	for _, t := range byName {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

func collectThemes(dir, source string, byName map[string]option) error {
	if dir == "" {
		return nil
	}
	if _, err := os.Stat(dir); err != nil {
		return err
	}
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path == dir {
				return nil
			}
			return fs.SkipDir
		}
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		name := d.Name()
		if existing, ok := byName[name]; ok && existing.Source == "user" {
			return nil
		}
		byName[name] = option{Name: name, Path: abs, Source: source}
		return nil
	})
}
