package ui

import (
	"strings"
	"testing"
)

// TestGetStatusIndicator tests the status indicator functionality
func TestGetStatusIndicator(t *testing.T) {
	tests := []struct {
		name           string
		status         string
		expectedSymbol string
	}{
		{"Running status", "Running", "●"},
		{"Failed status", "Failed", "✗"},
		{"Suspended status", "Suspended", "⏸"},
		{"Connecting status", "Connecting", "◐"},
		{"Reconnecting status", "Reconnecting", "◐"},
		{"Starting status", "Starting", "◯"},
		{"Degraded status", "Degraded", "⚠"},
		{"Cooldown status", "Cooldown", "◦"},
		{"Unknown status", "Unknown", "●"}, // Should default to ●
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			indicator := GetStatusIndicator(tt.status)

			// The indicator contains styling, so we check if it contains the expected symbol
			if !strings.Contains(indicator, tt.expectedSymbol) {
				t.Errorf("GetStatusIndicator(%s) = %s, expected to contain %s",
					tt.status, indicator, tt.expectedSymbol)
			}
		})
	}
}

// TestGetStatusStyle tests that each status has an appropriate style
func TestGetStatusStyle(t *testing.T) {
	statuses := []string{
		"Running",
		"Failed",
		"Suspended",
		"Connecting",
		"Reconnecting",
		"Starting",
		"Degraded",
		"Cooldown",
		"Unknown", // Should default to starting style
	}

	for _, status := range statuses {
		t.Run(status, func(t *testing.T) {
			style := GetStatusStyle(status)

			// Test that the style can render text (this is the real test)
			rendered := style.Render("test")
			if rendered == "" {
				t.Errorf("GetStatusStyle(%s) style failed to render text", status)
			}

			// Test that rendered output contains the input text
			if !strings.Contains(rendered, "test") {
				t.Errorf("GetStatusStyle(%s) rendered output should contain 'test', got: %s", status, rendered)
			}
		})
	}
}

// TestStatusStyleConsistency tests that status styles are consistent
func TestStatusStyleConsistency(t *testing.T) {
	// Test that the same status always returns the same style
	runningStyle1 := GetStatusStyle("Running")
	runningStyle2 := GetStatusStyle("Running")

	if runningStyle1.String() != runningStyle2.String() {
		t.Error("GetStatusStyle should return consistent styles for the same status")
	}

	// Test that different statuses return different visual representations
	runningIndicator := GetStatusIndicator("Running")
	failedIndicator := GetStatusIndicator("Failed")
	suspendedIndicator := GetStatusIndicator("Suspended")

	if runningIndicator == failedIndicator {
		t.Error("Running and Failed statuses should have different indicators")
	}

	if runningIndicator == suspendedIndicator {
		t.Error("Running and Suspended statuses should have different indicators")
	}

	if failedIndicator == suspendedIndicator {
		t.Error("Failed and Suspended statuses should have different indicators")
	}
}

// TestSuspendedStatusHandling tests specific handling of the new Suspended status
func TestSuspendedStatusHandling(t *testing.T) {
	// Test that Suspended status has proper styling
	suspendedStyle := GetStatusStyle("Suspended")
	suspendedIndicator := GetStatusIndicator("Suspended")

	// Should contain the pause symbol
	if !strings.Contains(suspendedIndicator, "⏸") {
		t.Errorf("Suspended status indicator should contain ⏸ symbol, got %s", suspendedIndicator)
	}

	// Should render properly
	rendered := suspendedStyle.Render("Suspended")
	if rendered == "" {
		t.Error("Suspended style should render non-empty text")
	}
}

// TestAllStatusSymbolsUnique tests that each status has a unique symbol
func TestAllStatusSymbolsUnique(t *testing.T) {
	statuses := []string{
		"Running",      // ●
		"Failed",       // ✗
		"Suspended",    // ⏸
		"Connecting",   // ◐
		"Reconnecting", // ◐ (same as Connecting intentionally)
		"Starting",     // ◯
		"Degraded",     // ⚠
		"Cooldown",     // ◦
	}

	symbolMap := make(map[string][]string)

	for _, status := range statuses {
		indicator := GetStatusIndicator(status)
		// Extract just the symbol by finding unicode characters
		for _, char := range indicator {
			if char > 127 { // Unicode character (symbol)
				symbol := string(char)
				symbolMap[symbol] = append(symbolMap[symbol], status)
				break
			}
		}
	}

	// Check that most symbols are unique (Connecting/Reconnecting intentionally share)
	uniqueSymbols := 0
	for symbol, statuses := range symbolMap {
		if len(statuses) == 1 {
			uniqueSymbols++
		} else if len(statuses) == 2 &&
			((statuses[0] == "Connecting" && statuses[1] == "Reconnecting") ||
				(statuses[0] == "Reconnecting" && statuses[1] == "Connecting")) {
			// This is expected - Connecting and Reconnecting should share a symbol
			uniqueSymbols++
		} else {
			t.Errorf("Symbol %s is used by unexpected statuses: %v", symbol, statuses)
		}
	}

	if uniqueSymbols < 6 { // Should have at least 6 unique symbols/symbol groups
		t.Errorf("Expected at least 6 unique status symbols, got %d", uniqueSymbols)
	}
}

// BenchmarkGetStatusIndicator benchmarks status indicator generation
func BenchmarkGetStatusIndicator(b *testing.B) {
	statuses := []string{
		"Running", "Failed", "Suspended", "Connecting",
		"Reconnecting", "Starting", "Degraded", "Cooldown",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		status := statuses[i%len(statuses)]
		GetStatusIndicator(status)
	}
}

// BenchmarkGetStatusStyle benchmarks status style generation
func BenchmarkGetStatusStyle(b *testing.B) {
	statuses := []string{
		"Running", "Failed", "Suspended", "Connecting",
		"Reconnecting", "Starting", "Degraded", "Cooldown",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		status := statuses[i%len(statuses)]
		GetStatusStyle(status)
	}
}
