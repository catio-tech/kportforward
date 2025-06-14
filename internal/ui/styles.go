package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Color palette for dark terminal theme
var (
	// Primary colors
	primaryColor   = lipgloss.Color("#00D4AA") // Bright teal
	secondaryColor = lipgloss.Color("#FF6B6B") // Coral red
	accentColor    = lipgloss.Color("#4ECDC4") // Light teal

	// Status colors
	successColor = lipgloss.Color("#55FF55") // Bright green
	warningColor = lipgloss.Color("#FFAA00") // Orange
	errorColor   = lipgloss.Color("#FF5555") // Bright red
	mutedColor   = lipgloss.Color("#888888") // Gray

	// Background colors
	selectedBg  = lipgloss.Color("#2A2A2A") // Dark gray
	borderColor = lipgloss.Color("#444444") // Medium gray
)

// Base styles
var (
	// Main container style
	containerStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(0, 1)

	// Header style
	headerStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			Padding(0, 1)

	// Title style
	titleStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	// Context info style
	contextStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Italic(true)

	// Status indicator styles
	statusRunningStyle = lipgloss.NewStyle().
				Foreground(successColor).
				Bold(true)

	statusFailedStyle = lipgloss.NewStyle().
				Foreground(errorColor).
				Bold(true)

	statusStartingStyle = lipgloss.NewStyle().
				Foreground(warningColor).
				Bold(true)

	statusCooldownStyle = lipgloss.NewStyle().
				Foreground(mutedColor).
				Bold(true)

	// Table styles
	tableHeaderStyle = lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true).
				Underline(true)

	tableRowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))

	tableSelectedRowStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(selectedBg).
				Bold(true)

	// URL link style
	urlStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(false)

	// Help text style
	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true)

	// Error message style
	errorMessageStyle = lipgloss.NewStyle().
				Foreground(errorColor).
				Italic(true)

	// Footer style
	footerStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true).
			Padding(0, 1)
)

// GetStatusStyle returns the appropriate style for a service status
func GetStatusStyle(status string) lipgloss.Style {
	switch status {
	case "Running":
		return statusRunningStyle
	case "Failed":
		return statusFailedStyle
	case "Starting":
		return statusStartingStyle
	case "Cooldown":
		return statusCooldownStyle
	default:
		return statusStartingStyle
	}
}

// GetStatusIndicator returns a colored status indicator
func GetStatusIndicator(status string) string {
	style := GetStatusStyle(status)
	return style.Render("●")
}

// FormatURL formats a URL with styling
func FormatURL(url string) string {
	return urlStyle.Render(url)
}

// FormatTableHeader formats table headers
func FormatTableHeader(text string) string {
	return tableHeaderStyle.Render(text)
}

// FormatTableRow formats a table row (selected or normal)
func FormatTableRow(text string, selected bool) string {
	if selected {
		return tableSelectedRowStyle.Render(text)
	}
	return tableRowStyle.Render(text)
}
