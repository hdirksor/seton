package styles

import "github.com/charmbracelet/lipgloss"

// Theme defines the visual contract for the application.
// Implement this interface to provide a custom theme.
type Theme interface {
	Banner() string
	View() lipgloss.Style
	Header() lipgloss.Style
	Dim() lipgloss.Style
	Success() lipgloss.Style
	Warn() lipgloss.Style
	Err() lipgloss.Style
	Selected() lipgloss.Style
	FocusedRow() lipgloss.Style
	Card() lipgloss.Style
	Block() lipgloss.Style
}

// Active is the theme used throughout the application.
var Active Theme = DefaultTheme{}

// Convenience functions delegating to Active.

func Banner() string          { return Active.Banner() }
func View() lipgloss.Style    { return Active.View() }
func Header() lipgloss.Style  { return Active.Header() }
func Dim() lipgloss.Style     { return Active.Dim() }
func Success() lipgloss.Style { return Active.Success() }
func Warn() lipgloss.Style    { return Active.Warn() }
func Err() lipgloss.Style     { return Active.Err() }
func Selected() lipgloss.Style  { return Active.Selected() }
func FocusedRow() lipgloss.Style { return Active.FocusedRow() }
func Card() lipgloss.Style    { return Active.Card() }
func Block() lipgloss.Style   { return Active.Block() }

// DefaultTheme is the built-in ANSI 16-color theme.
type DefaultTheme struct{}

// ANSI 16-color palette — implementation detail of DefaultTheme.
const (
	red         = lipgloss.ANSIColor(1)
	green       = lipgloss.ANSIColor(2)
	yellow      = lipgloss.ANSIColor(3)
	blue        = lipgloss.ANSIColor(4)
	magenta     = lipgloss.ANSIColor(5)
	cyan        = lipgloss.ANSIColor(6)
	white       = lipgloss.ANSIColor(7)
	brightBlack = lipgloss.ANSIColor(8)
	brightWhite = lipgloss.ANSIColor(15)
)

func (DefaultTheme) Banner() string {
	return lipgloss.NewStyle().Foreground(magenta).
		Render("\n░█▀█░█▀█░▀█▀░█▀▀░█▀▀\n░█░█░█░█░░█░░█▀▀░▀▀█\n░▀░▀░▀▀▀░░▀░░▀▀▀░▀▀▀") + "\n\n"
}

func (DefaultTheme) View() lipgloss.Style {
	return lipgloss.NewStyle().Padding(1, 2)
}

func (DefaultTheme) Header() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true)
}

func (DefaultTheme) Dim() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(brightBlack)
}

func (DefaultTheme) Success() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(green).Bold(true)
}

func (DefaultTheme) Warn() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(yellow).Bold(true)
}

func (DefaultTheme) Err() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(red).Bold(true)
}

func (DefaultTheme) Selected() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(green)
}

func (DefaultTheme) FocusedRow() lipgloss.Style {
	return lipgloss.NewStyle().Background(blue).Foreground(brightWhite).Bold(true)
}

func (DefaultTheme) Card() lipgloss.Style {
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(cyan).Padding(0, 1).MarginBottom(1)
}

func (DefaultTheme) Block() lipgloss.Style {
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).Foreground(white)
}
