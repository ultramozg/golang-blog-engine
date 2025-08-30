package app

import (
	"testing"
)

func TestExtractExcerpt(t *testing.T) {
	app := &App{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple text",
			input:    "This is a simple test post with some content.",
			expected: "This is a simple test post with some content.",
		},
		{
			name:     "Text with HTML tags",
			input:    "<p>This is a <strong>bold</strong> text with <em>emphasis</em>.</p>",
			expected: "This is a bold text with emphasis.",
		},
		{
			name:     "Text with code blocks and newlines",
			input:    "Here is some code:\n<code>\nfunction test() {\n    return true;\n}\n</code>\nEnd of post.",
			expected: "Here is some code: function test() { return true; } End of post.",
		},
		{
			name:     "Text with file references",
			input:    "Check out this image: [file:example.jpg] and this document: [file:doc.pdf]",
			expected: "Check out this image: and this document:",
		},
		{
			name:     "Long text that should be truncated",
			input:    "This is a very long text that should be truncated when it exceeds the 500 character limit. Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.",
			expected: "This is a very long text that should be truncated when it exceeds the 500 character limit. Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui offic...",
		},
		{
			name:     "Empty content",
			input:    "",
			expected: "No content available",
		},
		{
			name:     "Only HTML tags",
			input:    "<div><p></p></div>",
			expected: "No content available",
		},
		{
			name:     "Multiple newlines and spaces",
			input:    "Line 1\n\n\nLine 2    with    spaces\n\nLine 3",
			expected: "Line 1 Line 2 with spaces Line 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := app.extractExcerpt(tt.input)
			if result != tt.expected {
				t.Errorf("extractExcerpt() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExtractExcerptLength(t *testing.T) {
	app := &App{}

	// Test that the function respects the 500 character limit
	longText := "a"
	for i := 0; i < 600; i++ {
		longText += "a"
	}

	result := app.extractExcerpt(longText)
	if len(result) > 500 {
		t.Errorf("extractExcerpt() returned text longer than 500 characters: %d", len(result))
	}

	if result[len(result)-3:] != "..." {
		t.Errorf("extractExcerpt() should end with '...' when truncated, got: %s", result[len(result)-10:])
	}
}
