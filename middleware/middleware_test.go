package middleware

import (
	"strings"
	"testing"
)

func TestSanitizeSlug(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Valid slug with hyphens",
			input:    "valid-slug-here",
			expected: "valid-slug-here",
		},
		{
			name:     "Valid slug with underscores",
			input:    "valid_slug_here",
			expected: "valid_slug_here",
		},
		{
			name:     "Valid slug with numbers",
			input:    "slug123",
			expected: "slug123",
		},
		{
			name:     "Mixed valid characters",
			input:    "test-post_2024",
			expected: "test-post_2024",
		},
		{
			name:     "Directory traversal attempt",
			input:    "../dangerous-slug",
			expected: "",
		},
		{
			name:     "Forward slash",
			input:    "invalid/slug",
			expected: "",
		},
		{
			name:     "Backward slash",
			input:    "invalid\\slug",
			expected: "",
		},
		{
			name:     "Double dots",
			input:    "invalid..slug",
			expected: "",
		},
		{
			name:     "Empty slug",
			input:    "",
			expected: "",
		},
		{
			name:     "Too long slug",
			input:    strings.Repeat("a", 201),
			expected: "",
		},
		{
			name:     "Special characters",
			input:    "slug@with#special$chars",
			expected: "",
		},
		{
			name:     "Spaces",
			input:    "slug with spaces",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeSlug(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeSlug(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}