package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"ghostty-config/internal/ghostty"
	"ghostty-config/internal/shader"
	"ghostty-config/internal/theme"
	"ghostty-config/internal/ui"
)

type App struct {
	opts     ghostty.Options
	current  ui.Screen
	width    int
	height   int
	quitting bool

	menu   menuModel
	shader *shader.Model
	theme  *theme.Model
}

func New(opts ghostty.Options) App {
	return App{
		opts:    opts,
		current: ui.ScreenMenu,
		menu:    newMenuModel(),
	}
}

func (a App) Init() tea.Cmd {
	return nil
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = m.Width
		a.height = m.Height
		if a.shader != nil {
			a.shader.SetSize(m.Width, m.Height)
		}
		if a.theme != nil {
			a.theme.SetSize(m.Width, m.Height)
		}
	case ui.SwitchScreenMsg:
		return a.switchTo(m.Target)
	case ui.QuitAppMsg:
		a.quitting = true
		return a, tea.Quit
	}

	switch a.current {
	case ui.ScreenMenu:
		nm, cmd := a.menu.Update(msg)
		a.menu = nm
		return a, cmd
	case ui.ScreenShader:
		if a.shader == nil {
			return a, nil
		}
		ns, cmd := a.shader.Update(msg)
		a.shader = &ns
		return a, cmd
	case ui.ScreenTheme:
		if a.theme == nil {
			return a, nil
		}
		nt, cmd := a.theme.Update(msg)
		a.theme = &nt
		return a, cmd
	}
	return a, nil
}

func (a App) switchTo(target ui.Screen) (tea.Model, tea.Cmd) {
	switch target {
	case ui.ScreenMenu:
		a.current = ui.ScreenMenu
		a.menu.errorMsg = ""
		return a, nil
	case ui.ScreenShader:
		if a.shader == nil {
			sh, err := shader.New(a.opts)
			if err != nil {
				a.menu.errorMsg = err.Error()
				a.current = ui.ScreenMenu
				return a, nil
			}
			sh.SetSize(a.width, a.height)
			a.shader = &sh
		}
		a.current = ui.ScreenShader
		return a, a.shader.InitCmd()
	case ui.ScreenTheme:
		if a.theme == nil {
			tm, err := theme.New(a.opts)
			if err != nil {
				a.menu.errorMsg = err.Error()
				a.current = ui.ScreenMenu
				return a, nil
			}
			tm.SetSize(a.width, a.height)
			a.theme = &tm
		}
		a.current = ui.ScreenTheme
		return a, a.theme.InitCmd()
	}
	return a, nil
}

func (a App) View() string {
	if a.quitting {
		return ""
	}
	switch a.current {
	case ui.ScreenMenu:
		return a.menu.View()
	case ui.ScreenShader:
		if a.shader == nil {
			return ""
		}
		return a.shader.View()
	case ui.ScreenTheme:
		if a.theme == nil {
			return ""
		}
		return a.theme.View()
	}
	return ""
}
