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
)
