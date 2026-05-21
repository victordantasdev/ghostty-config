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

type ToastKind int

const (
	ToastSaved ToastKind = iota
	ToastReverted
	ToastInfo
)

type ShowToastMsg struct {
	Text string
	Kind ToastKind
}

type ClearToastMsg struct {
	Token int
}

type OpenHelpMsg struct{}

type CloseHelpMsg struct{}
