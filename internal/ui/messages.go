package ui

type Screen int

const (
	ScreenMenu Screen = iota
	ScreenShader
	ScreenTheme
)

type SwitchScreenMsg struct {
	Target Screen
}

type QuitAppMsg struct{}
