package app

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"ghostty-config/internal/ghostty"
	"ghostty-config/internal/shader"
	"ghostty-config/internal/theme"
	"ghostty-config/internal/ui"
)

const toastDuration = 1100 * time.Millisecond

type toastState struct {
	text  string
	kind  ui.ToastKind
	token int
}

type App struct {
	opts     ghostty.Options
	version  string
	current  ui.Screen
	width    int
	height   int
	quitting bool

	menu   menuModel
	shader *shader.Model
	theme  *theme.Model

	help       helpModel
	helpOpen   bool
	toast      *toastState
	toastNonce int
}

func New(opts ghostty.Options, version string) App {
	return App{
		opts:    opts,
		version: version,
		current: ui.ScreenMenu,
		menu:    newMenuModel(opts.ConfigPath, version),
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
		a.menu.width = m.Width
		a.help.width = m.Width
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
	case ui.OpenHelpMsg:
		a.helpOpen = true
		return a, nil
	case ui.CloseHelpMsg:
		a.helpOpen = false
		return a, nil
	case ui.ShowToastMsg:
		a.toastNonce++
		a.toast = &toastState{text: m.Text, kind: m.Kind, token: a.toastNonce}
		token := a.toastNonce
		return a, tea.Tick(toastDuration, func(time.Time) tea.Msg {
			return ui.ClearToastMsg{Token: token}
		})
	case ui.ClearToastMsg:
		if a.toast != nil && a.toast.token == m.Token {
			a.toast = nil
		}
		return a, nil
	}

	if a.helpOpen {
		nh, cmd := a.help.Update(msg)
		a.help = nh
		return a, cmd
	}

	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "?" {
		a.helpOpen = true
		return a, nil
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
		a.shader = nil
		a.theme = nil
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
	if a.helpOpen {
		return a.help.View()
	}

	var body string
	switch a.current {
	case ui.ScreenMenu:
		body = a.menu.View()
	case ui.ScreenShader:
		if a.shader != nil {
			body = a.shader.View()
		}
	case ui.ScreenTheme:
		if a.theme != nil {
			body = a.theme.View()
		}
	}

	if a.toast != nil {
		return ui.RenderToast(a.toast.text, a.toast.kind) + "\n" + body
	}
	return body
}
