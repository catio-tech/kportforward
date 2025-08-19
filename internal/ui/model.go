package ui

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/victorkazakov/kportforward/internal/config"
	"github.com/victorkazakov/kportforward/internal/updater"
	"github.com/victorkazakov/kportforward/internal/utils"
)

// UIURLProvider interface for accessing UI handler URLs
type UIURLProvider interface {
	GetGRPCUIURL(serviceName string) string
	GetSwaggerUIURL(serviceName string) string
}

// SortField represents different sorting options
type SortField int

const (
	SortByName SortField = iota
	SortByStatus
	SortByType
	SortByPort
	SortByUptime
)

var sortFieldNames = map[SortField]string{
	SortByName:   "Name",
	SortByStatus: "Status",
	SortByType:   "Type",
	SortByPort:   "Port",
	SortByUptime: "Uptime",
}

// ViewMode represents different view modes
type ViewMode int

const (
	ViewTable ViewMode = iota
	ViewDetail
)

// Model represents the main TUI model
type Model struct {
	// Data
	services        map[string]config.ServiceStatus
	serviceConfigs  map[string]config.Service
	serviceNames    []string
	kubeContext     string
	lastUpdate      time.Time
	updateAvailable bool
	UpdateInfo      *updater.UpdateInfo // Added for rich update info

	// UI Handler status
	grpcUIEnabled    bool
	swaggerUIEnabled bool

	// Manager reference for accessing UI handler URLs
	manager UIURLProvider

	// UI state
	selectedIndex int
	sortField     SortField
	sortReverse   bool
	viewMode      ViewMode

	// Display settings
	width       int
	height      int
	refreshRate time.Duration

	// Channels
	statusChan  <-chan map[string]config.ServiceStatus
	contextChan <-chan string
}

// StatusUpdateMsg represents a status update message
type StatusUpdateMsg map[string]config.ServiceStatus

// ContextUpdateMsg represents a context change message
type ContextUpdateMsg string

// UpdateAvailableMsg represents an update notification
type UpdateAvailableMsg bool

// UIHandlerStatusMsg represents UI handler status update
type UIHandlerStatusMsg struct {
	GRPCUIEnabled    bool
	SwaggerUIEnabled bool
}

// TickMsg represents a timer tick
type TickMsg time.Time

// NewModel creates a new TUI model
func NewModel(statusChan <-chan map[string]config.ServiceStatus, serviceConfigs map[string]config.Service, manager UIURLProvider) *Model {
	return &Model{
		services:       make(map[string]config.ServiceStatus),
		serviceConfigs: serviceConfigs,
		serviceNames:   make([]string, 0),
		selectedIndex:  0,
		sortField:      SortByName,
		sortReverse:    false,
		viewMode:       ViewTable,
		refreshRate:    250 * time.Millisecond,
		statusChan:     statusChan,
		manager:        manager,
	}
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.listenForStatusUpdates(),
		m.tickEvery(),
	)
}

// Update handles messages and updates the model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case StatusUpdateMsg:
		m.services = map[string]config.ServiceStatus(msg)
		m.updateServiceNames()
		m.lastUpdate = time.Now()
		return m, nil

	case ContextUpdateMsg:
		m.kubeContext = string(msg)
		return m, nil

	case UpdateAvailableMsg:
		m.updateAvailable = bool(msg)
		return m, nil

	case UIHandlerStatusMsg:
		m.grpcUIEnabled = msg.GRPCUIEnabled
		m.swaggerUIEnabled = msg.SwaggerUIEnabled
		return m, nil

	case struct {
		GRPCUIEnabled    bool
		SwaggerUIEnabled bool
	}:
		m.grpcUIEnabled = msg.GRPCUIEnabled
		m.swaggerUIEnabled = msg.SwaggerUIEnabled
		return m, nil

	case TickMsg:
		return m, tea.Batch(
			m.listenForStatusUpdates(),
			m.tickEvery(),
		)

	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	}

	return m, nil
}

// View renders the TUI
func (m *Model) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	switch m.viewMode {
	case ViewDetail:
		return m.renderDetailView()
	default:
		return m.renderTableView()
	}
}

// handleKeyPress processes keyboard input
func (m *Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.viewMode {
	case ViewDetail:
		return m.handleDetailKeyPress(msg)
	default:
		return m.handleTableKeyPress(msg)
	}
}

// handleTableKeyPress handles keys in table view
func (m *Model) handleTableKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "up", "k":
		if m.selectedIndex > 0 {
			m.selectedIndex--
		}

	case "down", "j":
		if m.selectedIndex < len(m.serviceNames)-1 {
			m.selectedIndex++
		}

	case "enter", " ":
		m.viewMode = ViewDetail
		return m, nil

	case "n":
		m.sortField = SortByName
		m.updateServiceNames()

	case "s":
		m.sortField = SortByStatus
		m.updateServiceNames()

	case "t":
		m.sortField = SortByType
		m.updateServiceNames()

	case "p":
		m.sortField = SortByPort
		m.updateServiceNames()

	case "u":
		m.sortField = SortByUptime
		m.updateServiceNames()

	case "r":
		m.sortReverse = !m.sortReverse
		m.updateServiceNames()
	}

	return m, nil
}

// handleDetailKeyPress handles keys in detail view
func (m *Model) handleDetailKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "esc", "backspace":
		m.viewMode = ViewTable
		return m, nil
	}

	return m, nil
}

// renderTableView renders the main table view
func (m *Model) renderTableView() string {
	// Header
	header := m.renderHeader()

	// Table
	table := m.renderTable()

	// Footer
	footer := m.renderFooter()

	// Combine all parts
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		table,
		"",
		footer,
	)

	return containerStyle.
		Width(m.width - 4).
		Height(m.height - 2).
		Render(content)
}

// renderDetailView renders the detail view for selected service
func (m *Model) renderDetailView() string {
	if len(m.serviceNames) == 0 || m.selectedIndex >= len(m.serviceNames) {
		return "No service selected"
	}

	serviceName := m.serviceNames[m.selectedIndex]
	service, exists := m.services[serviceName]
	if !exists {
		return "Service not found"
	}

	// Service details
	details := []string{
		titleStyle.Render(fmt.Sprintf("Service Details: %s", serviceName)),
		"",
		fmt.Sprintf("Status: %s %s", GetStatusIndicator(service.Status), service.Status),
		fmt.Sprintf("Local Port: %d", service.LocalPort),
		fmt.Sprintf("Process ID: %d", service.PID),
		fmt.Sprintf("Restart Count: %d", service.RestartCount),
	}

	if !service.StartTime.IsZero() {
		uptime := time.Since(service.StartTime)
		details = append(details, fmt.Sprintf("Uptime: %s", utils.FormatUptime(uptime)))
	}

	// Add URL information if service is running
	if service.Status == "Running" {
		serviceType := m.getServiceType(serviceName)
		switch serviceType {
		case "web":
			details = append(details, fmt.Sprintf("🌐 Web URL: http://localhost:%d", service.LocalPort))
		case "rest":
			if m.swaggerUIEnabled && m.manager != nil {
				swaggerURL := m.manager.GetSwaggerUIURL(serviceName)
				if swaggerURL != "" {
					details = append(details, fmt.Sprintf("📋 Swagger UI: %s", swaggerURL))
				} else {
					details = append(details, fmt.Sprintf("🔗 REST API: http://localhost:%d", service.LocalPort))
				}
			}
		case "rpc":
			if m.grpcUIEnabled && m.manager != nil {
				grpcURL := m.manager.GetGRPCUIURL(serviceName)
				if grpcURL != "" {
					details = append(details, fmt.Sprintf("⚡ gRPC UI: %s", grpcURL))
				}
			}
		}
	}

	if service.LastError != "" {
		details = append(details,
			"",
			"Last Error:",
			errorMessageStyle.Render(service.LastError),
		)
	}

	if service.StatusMessage != "" {
		details = append(details,
			"",
			"Status:",
			service.StatusMessage,
		)
	}

	details = append(details,
		"",
		helpStyle.Render("[ESC] Back to table view  [q] Quit"),
	)

	content := strings.Join(details, "\n")

	return containerStyle.
		Width(m.width - 4).
		Height(m.height - 2).
		Render(content)
}

// renderHeader renders the header section
func (m *Model) renderHeader() string {
	title := titleStyle.Render("kportforward")

	context := ""
	if m.kubeContext != "" {
		// Make the context more prominent with bold styling
		context = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			Render(fmt.Sprintf("K8s: %s", m.kubeContext))
	}

	updateNotice := ""
	if m.updateAvailable {
		// Create basic update notice
		basicNotice := "Update Available!"

		// Check if this is a Homebrew installation
		execPath, _ := os.Executable()
		isHomebrew := strings.Contains(execPath, "/Cellar/kportforward") ||
			strings.Contains(execPath, "/opt/homebrew")

		// If we have detailed update info and it's a Homebrew installation,
		// include the version and update method
		if m.UpdateInfo != nil {
			if isHomebrew {
				basicNotice = fmt.Sprintf("Update Available! (%s → %s via brew upgrade)",
					m.UpdateInfo.CurrentVersion, m.UpdateInfo.LatestVersion)
			} else {
				basicNotice = fmt.Sprintf("Update Available! (%s → %s)",
					m.UpdateInfo.CurrentVersion, m.UpdateInfo.LatestVersion)
			}
		}

		updateNotice = lipgloss.NewStyle().Foreground(warningColor).Render(basicNotice)
	}

	// Calculate running/total services
	running := 0
	total := len(m.services)
	for _, service := range m.services {
		if service.Status == "Running" {
			running++
		}
	}

	status := fmt.Sprintf("Services (%d/%d running)", running, total)

	return headerStyle.Render(
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			title,
			"  ",
			context,
			"  ",
			updateNotice,
			"  ",
			status,
		),
	)
}

// renderTable renders the services table
func (m *Model) renderTable() string {
	if len(m.serviceNames) == 0 {
		return "No services configured"
	}

	// Calculate column widths based on terminal width
	nameWidth := 25
	statusWidth := 15 // Increased to fit "Reconnecting" status
	urlWidth := 35    // Increased to account for emoji icons
	typeWidth := 8
	portWidth := 6 // Width for port number
	uptimeWidth := 10
	errorWidth := m.width - nameWidth - statusWidth - urlWidth - typeWidth - portWidth - uptimeWidth - 24

	// Ensure minimum widths to prevent negative values
	if errorWidth < 10 {
		errorWidth = 10
		urlWidth = m.width - nameWidth - statusWidth - typeWidth - portWidth - uptimeWidth - errorWidth - 24
	}

	// Ensure urlWidth is never negative or too small
	if urlWidth < 5 {
		urlWidth = 5
	}

	// Table header
	headers := []string{
		FormatTableHeader(fmt.Sprintf("%-*s", nameWidth, "Name")),
		FormatTableHeader(fmt.Sprintf("%-*s", statusWidth, "Status")),
		FormatTableHeader(fmt.Sprintf("%-*s", urlWidth, "URL")),
		FormatTableHeader(fmt.Sprintf("%-*s", typeWidth, "Type")),
		FormatTableHeader(fmt.Sprintf("%-*s", portWidth, "Port")),
		FormatTableHeader(fmt.Sprintf("%-*s", uptimeWidth, "Uptime")),
		FormatTableHeader(fmt.Sprintf("%-*s", errorWidth, "Error/Status")),
	}

	headerRow := strings.Join(headers, " ")

	// Table rows
	rows := []string{headerRow}

	for i, serviceName := range m.serviceNames {
		service := m.services[serviceName]
		selected := (i == m.selectedIndex)

		// Get raw content for each column
		nameContent := truncateString(serviceName, nameWidth)
		statusContent := service.Status
		urlContent := m.formatServiceURL(service, serviceName, urlWidth)
		typeContent := truncateString(m.getServiceType(serviceName), typeWidth)

		// Port column content
		portContent := fmt.Sprintf("%d", service.LocalPort)
		if service.LocalPort == 0 {
			portContent = "-"
		}

		uptimeContent := "-"
		if !service.StartTime.IsZero() {
			uptime := time.Since(service.StartTime)
			uptimeContent = utils.FormatUptime(uptime)
		}

		// Show status message if no error, otherwise show error
		errorContent := service.LastError
		if errorContent == "" && service.StatusMessage != "" {
			errorContent = service.StatusMessage
		}
		errorContent = truncateString(errorContent, errorWidth)

		// Create columns with exact width (pad first, then style)
		nameCol := fmt.Sprintf("%-*s", nameWidth, nameContent)
		statusCol := fmt.Sprintf("%s %-*s", GetStatusIndicator(service.Status), statusWidth-2, statusContent)

		// Handle URL with proper width - style only the actual URL part
		var urlCol string
		if service.Status == "Running" || service.Status == "Degraded" ||
			service.Status == "Connecting" || service.Status == "Reconnecting" {
			// Only style if it's an actual URL, then pad to correct width using visual width
			styledURL := FormatURL(urlContent)
			visualWidthOfContent := visualWidth(urlContent)
			padding := urlWidth - visualWidthOfContent
			if padding < 0 {
				padding = 0
			}
			urlCol = styledURL + strings.Repeat(" ", padding)
		} else {
			urlCol = fmt.Sprintf("%-*s", urlWidth, urlContent)
		}

		typeCol := fmt.Sprintf("%-*s", typeWidth, typeContent)
		portCol := fmt.Sprintf("%-*s", portWidth, portContent)
		uptimeCol := fmt.Sprintf("%-*s", uptimeWidth, uptimeContent)
		errorCol := fmt.Sprintf("%-*s", errorWidth, errorContent)

		// Combine row with single spaces between columns
		rowContent := nameCol + " " + statusCol + " " + urlCol + " " + typeCol + " " + portCol + " " + uptimeCol + " " + errorCol

		rows = append(rows, FormatTableRow(rowContent, selected))
	}

	return strings.Join(rows, "\n")
}

// renderFooter renders the footer with help text
func (m *Model) renderFooter() string {
	sortInfo := fmt.Sprintf("Sort: %s", sortFieldNames[m.sortField])
	if m.sortReverse {
		sortInfo += " (desc)"
	}

	help := []string{
		"[↑↓] Navigate",
		"[Enter] Details",
		"[n/s/t/p/u] Sort by Name/Status/Type/Port/Uptime",
		"[r] Reverse",
		"[q] Quit",
	}

	return footerStyle.Render(
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			sortInfo,
			"  •  ",
			strings.Join(help, "  "),
		),
	)
}

// formatServiceURL formats the URL for a service based on type and UI handler status
func (m *Model) formatServiceURL(service config.ServiceStatus, serviceName string, maxWidth int) string {
	if service.Status != "Running" && service.Status != "Degraded" &&
		service.Status != "Connecting" && service.Status != "Reconnecting" {
		return "-"
	}

	serviceType := m.getServiceType(serviceName)

	// Determine URL and icon based on service type and UI handler status
	var url, icon string
	switch serviceType {
	case "web":
		// Always show URL for web services (direct port-forward)
		icon = "🌐" // Globe icon for web pages
		url = fmt.Sprintf("%s http://localhost:%d", icon, service.LocalPort)
	case "rest":
		// Show Swagger UI URL if enabled, otherwise show direct port-forward
		if m.swaggerUIEnabled && m.manager != nil {
			swaggerURL := m.manager.GetSwaggerUIURL(serviceName)
			if swaggerURL != "" {
				icon = "📋" // Clipboard icon for Swagger UI documentation
				url = fmt.Sprintf("%s %s", icon, swaggerURL)
			} else {
				icon = "🔗" // Link icon for direct REST API access
				url = fmt.Sprintf("%s http://localhost:%d", icon, service.LocalPort)
			}
		} else {
			return "-"
		}
	case "rpc":
		// Show gRPC UI URL if enabled, otherwise don't show URL
		if m.grpcUIEnabled && m.manager != nil {
			grpcURL := m.manager.GetGRPCUIURL(serviceName)
			if grpcURL != "" {
				icon = "⚡" // Lightning bolt icon for gRPC UI (fast RPC calls)
				url = fmt.Sprintf("%s %s", icon, grpcURL)
			} else {
				return "-"
			}
		} else {
			return "-"
		}
	default:
		// For other service types, don't show URL
		return "-"
	}

	if len(url) > maxWidth {
		url = truncateString(url, maxWidth)
	}

	return url
}

// updateServiceNames updates and sorts the service names list
func (m *Model) updateServiceNames() {
	m.serviceNames = make([]string, 0, len(m.services))
	for name := range m.services {
		m.serviceNames = append(m.serviceNames, name)
	}

	// Sort based on current field
	sort.Slice(m.serviceNames, func(i, j int) bool {
		a, b := m.services[m.serviceNames[i]], m.services[m.serviceNames[j]]

		var less bool
		switch m.sortField {
		case SortByStatus:
			less = a.Status < b.Status
		case SortByType:
			less = m.getServiceType(m.serviceNames[i]) < m.getServiceType(m.serviceNames[j])
		case SortByPort:
			less = a.LocalPort < b.LocalPort
		case SortByUptime:
			less = a.StartTime.Before(b.StartTime)
		default: // SortByName
			less = m.serviceNames[i] < m.serviceNames[j]
		}

		if m.sortReverse {
			return !less
		}
		return less
	})

	// Ensure selected index is still valid
	if m.selectedIndex >= len(m.serviceNames) {
		m.selectedIndex = len(m.serviceNames) - 1
	}
	if m.selectedIndex < 0 {
		m.selectedIndex = 0
	}
}

// getServiceType returns the type of a service from the service configs
func (m *Model) getServiceType(serviceName string) string {
	if serviceConfig, exists := m.serviceConfigs[serviceName]; exists {
		return serviceConfig.Type
	}
	return "unknown"
}

// visualWidth calculates the visual width of a string accounting for Unicode emojis
func visualWidth(s string) int {
	// Simple approximation: count emojis as 2 characters, regular chars as 1
	width := 0
	for _, r := range s {
		if r > 0x1F600 && r < 0x1F64F || // Emoticons
			r > 0x1F300 && r < 0x1F5FF || // Misc symbols
			r > 0x1F680 && r < 0x1F6FF || // Transport symbols
			r > 0x2600 && r < 0x26FF || // Misc symbols
			r > 0x2700 && r < 0x27BF { // Dingbats
			width += 2
		} else {
			width += 1
		}
	}
	return width
}

// truncateString truncates a string to fit within the specified width
func truncateString(s string, width int) string {
	// Handle invalid width values
	if width <= 0 {
		return ""
	}
	if len(s) <= width {
		return s
	}
	if width <= 3 {
		// Ensure we don't exceed string length
		if len(s) < width {
			return s
		}
		return s[:width]
	}
	return s[:width-3] + "..."
}

// listenForStatusUpdates listens for status updates
func (m *Model) listenForStatusUpdates() tea.Cmd {
	return func() tea.Msg {
		select {
		case status := <-m.statusChan:
			return StatusUpdateMsg(status)
		default:
			return nil
		}
	}
}

// tickEvery returns a command that ticks at the refresh rate
func (m *Model) tickEvery() tea.Cmd {
	return tea.Tick(m.refreshRate, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}
