package ui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/victorkazakov/kportforward/internal/config"
	"github.com/victorkazakov/kportforward/internal/updater"
)

// TUI represents the terminal user interface
type TUI struct {
	program    *tea.Program
	model      *Model
	statusChan <-chan map[string]config.ServiceStatus
	ctx        context.Context
	cancel     context.CancelFunc
	quitChan   chan bool
}

// NewTUI creates a new terminal user interface
func NewTUI(statusChan <-chan map[string]config.ServiceStatus, serviceConfigs map[string]config.Service, manager UIURLProvider) *TUI {
	ctx, cancel := context.WithCancel(context.Background())

	model := NewModel(statusChan, serviceConfigs, manager)
	program := tea.NewProgram(
		model,
		tea.WithAltScreen(), // Use alternate screen buffer
		// Mouse support removed to enable text selection in terminal
	)

	return &TUI{
		program:    program,
		model:      model,
		statusChan: statusChan,
		ctx:        ctx,
		cancel:     cancel,
		quitChan:   make(chan bool, 1),
	}
}

// Start begins the TUI event loop
func (t *TUI) Start() error {
	// Start the program in a goroutine and track it
	go func() {
		if _, err := t.program.Run(); err != nil {
			// Don't log error - it might be a normal quit
		}
		// Signal that TUI has quit
		select {
		case t.quitChan <- true:
		default:
		}
	}()

	// Give the TUI a moment to initialize
	time.Sleep(50 * time.Millisecond)

	return nil
}

// Stop gracefully shuts down the TUI
func (t *TUI) Stop() error {
	t.cancel()
	if t.program != nil {
		t.program.Quit()

		// Give the program a moment to quit gracefully
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}

// UpdateKubernetesContext sends a context update to the TUI
func (t *TUI) UpdateKubernetesContext(context string) {
	if t.program != nil {
		t.program.Send(ContextUpdateMsg(context))
	}
}

// NotifyUpdateAvailable sends an update notification to the TUI
func (t *TUI) NotifyUpdateAvailable(updateInfo *updater.UpdateInfo) {
	if t.program != nil {
		// Store the updateInfo in the model for detailed information
		t.model.UpdateInfo = updateInfo

		// Send update availability message
		t.program.Send(UpdateAvailableMsg(updateInfo != nil && updateInfo.Available))
	}
}

// GetQuitChannel returns a channel that signals when the TUI quits
func (t *TUI) GetQuitChannel() <-chan bool {
	return t.quitChan
}

// UpdateUIHandlerStatus sends UI handler status update to the TUI
func (t *TUI) UpdateUIHandlerStatus(grpcUIEnabled, swaggerUIEnabled bool) {
	if t.program != nil {
		msg := struct {
			GRPCUIEnabled    bool
			SwaggerUIEnabled bool
		}{
			GRPCUIEnabled:    grpcUIEnabled,
			SwaggerUIEnabled: swaggerUIEnabled,
		}
		t.program.Send(msg)
	}
}
