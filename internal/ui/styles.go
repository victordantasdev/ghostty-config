package ui

import "github.com/charmbracelet/lipgloss"

var (
	TitleStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	SubtitleStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("141"))
	SelectedStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).Background(lipgloss.Color("62"))
	MutedStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	ErrorStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	SuccessStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	CheckedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	ActiveTabStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).Background(lipgloss.Color("62")).Padding(0, 1)
	InactiveTabStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Padding(0, 1)
	BadgeUserStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true)
	BadgeSysStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("110"))

	WarnStyle           = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))
	WarnBannerStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("232")).Background(lipgloss.Color("208")).Padding(0, 1)
	FooterDividerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	FooterKeyStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("240")).Padding(0, 1)
	FooterSaveKeyStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("28")).Padding(0, 1)
	FooterCancelKeyStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("124")).Padding(0, 1)
	FooterLabelStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	BreadcrumbHeadStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	BreadcrumbSegStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("141"))
	BreadcrumbSepStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	DirtyDotStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))
	CleanCheckStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("70"))
	ToastSavedStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("28")).Padding(0, 1)
	ToastRevertedStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("130")).Padding(0, 1)
	HelpKeyStyle        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229"))
	HelpHeaderStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("141"))
)
