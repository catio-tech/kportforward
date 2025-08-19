package ui

import (
	"testing"

	"github.com/victorkazakov/kportforward/internal/config"
)

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		width    int
		expected string
	}{
		{
			name:     "normal truncation",
			input:    "hello world",
			width:    8,
			expected: "hello...",
		},
		{
			name:     "no truncation needed",
			input:    "hello",
			width:    10,
			expected: "hello",
		},
		{
			name:     "width equal to string length",
			input:    "hello",
			width:    5,
			expected: "hello",
		},
		{
			name:     "very small width",
			input:    "hello",
			width:    3,
			expected: "hel",
		},
		{
			name:     "width of 1",
			input:    "hello",
			width:    1,
			expected: "h",
		},
		{
			name:     "zero width - should not panic",
			input:    "hello",
			width:    0,
			expected: "",
		},
		{
			name:     "negative width - should not panic",
			input:    "hello",
			width:    -5,
			expected: "",
		},
		{
			name:     "empty string with negative width",
			input:    "",
			width:    -1,
			expected: "",
		},
		{
			name:     "edge case - width smaller than string",
			input:    "hello world this is a very long string",
			width:    2,
			expected: "he",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This should not panic
			result := truncateString(tt.input, tt.width)
			if result != tt.expected {
				t.Errorf("truncateString(%q, %d) = %q; want %q", tt.input, tt.width, result, tt.expected)
			}
		})
	}
}

func TestTruncateStringNoPanic(t *testing.T) {
	// Test the specific scenario that was causing panics
	testCases := []int{-10, -5, -1, 0}

	for _, width := range testCases {
		t.Run("no panic with negative width", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("truncateString panicked with width %d: %v", width, r)
				}
			}()

			// This should not panic even with negative width
			result := truncateString("http://localhost:8080", width)
			if result != "" {
				t.Errorf("Expected empty string for width %d, got %q", width, result)
			}
		})
	}
}

func TestFormatServiceURLWithSmallWidth(t *testing.T) {
	// Create a model with very small width to simulate the panic scenario
	m := &Model{
		width:    50, // Very small terminal width
		services: make(map[string]config.ServiceStatus),
	}

	// Add a test service
	service := config.ServiceStatus{
		Status:    "Running",
		LocalPort: 8080,
	}

	// Test with various small widths that could cause negative urlWidth
	testWidths := []int{-5, -1, 0, 1, 2, 3, 4, 5}

	for _, width := range testWidths {
		t.Run("no panic with small width", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("formatServiceURL panicked with width %d: %v", width, r)
				}
			}()

			// This should not panic even with very small width
			result := m.formatServiceURL(service, "test-service", width)

			// Should return a valid result (could be empty or truncated)
			if len(result) > width && width > 0 {
				t.Errorf("Result length %d exceeds width %d: %q", len(result), width, result)
			}
		})
	}
}
