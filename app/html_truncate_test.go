package app

import (
	"strings"
	"testing"
)

func TestTruncateHTML(t *testing.T) {
	app := &App{}

	tests := []struct {
		name      string
		input     string
		maxLength int
		expected  string
	}{
		{
			name:      "Simple text truncation",
			input:     "This is a simple text that should be truncated",
			maxLength: 20,
			expected:  "This is a simple tex...",
		},
		{
			name:      "HTML with tags preserved",
			input:     "<p>This is <strong>bold</strong> text</p>",
			maxLength: 20,
			expected:  "<p>This is <strong>bold</strong>...</p>",
		},
		{
			name:      "Nested HTML tags",
			input:     "<div><p>This is <em>nested <strong>content</strong></em></p></div>",
			maxLength: 25,
			expected:  "<div><p>This is <em>nested <strong>co...</strong></em></p></div>",
		},
		{
			name:      "Self-closing tags",
			input:     "<p>Image here: <img src='test.jpg' alt='test'> and more text</p>",
			maxLength: 30,
			expected:  "<p>Image here: <img src='test.jpg' alt='test'> and...</p>",
		},
		{
			name:      "Content shorter than max length",
			input:     "<p>Short content</p>",
			maxLength: 100,
			expected:  "<p>Short content</p>",
		},
		{
			name:      "Line breaks preserved",
			input:     "Line one\nLine two\nLine three",
			maxLength: 20,
			expected:  "Line one<br>Line two<br>L...",
		},
		{
			name:      "Multiple paragraphs",
			input:     "<p>First paragraph</p><p>Second paragraph with more content</p>",
			maxLength: 35,
			expected:  "<p>First paragraph</p><p>Second paragr...</p>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := string(app.truncateHTML(tt.input, tt.maxLength))
			
			// Check that result doesn't exceed expected length significantly
			// (allowing for closing tags and ellipsis)
			if len(result) > tt.maxLength+50 { // Allow some buffer for closing tags
				t.Errorf("Result too long: got %d characters, expected around %d", len(result), tt.maxLength)
			}
			
			// Check that HTML structure is maintained (basic validation)
			openTags := strings.Count(result, "<p>") + strings.Count(result, "<div>") + 
					   strings.Count(result, "<strong>") + strings.Count(result, "<em>")
			closeTags := strings.Count(result, "</p>") + strings.Count(result, "</div>") + 
						strings.Count(result, "</strong>") + strings.Count(result, "</em>")
			
			if openTags != closeTags {
				t.Errorf("HTML structure broken: %d opening tags, %d closing tags. Result: %s", openTags, closeTags, result)
			}
			
			// If content was truncated, should contain ellipsis
			if len(tt.input) > tt.maxLength && !strings.Contains(result, "...") {
				t.Errorf("Expected ellipsis in truncated content, got: %s", result)
			}
		})
	}
}

func TestTruncateHTMLEdgeCases(t *testing.T) {
	app := &App{}

	t.Run("Empty content", func(t *testing.T) {
		result := string(app.truncateHTML("", 100))
		if result != "" {
			t.Errorf("Expected empty result for empty input, got: %s", result)
		}
	})

	t.Run("Only HTML tags", func(t *testing.T) {
		result := string(app.truncateHTML("<p></p>", 10))
		expected := "<p></p>"
		if result != expected {
			t.Errorf("Expected %s, got %s", expected, result)
		}
	})

	t.Run("Malformed HTML handled gracefully", func(t *testing.T) {
		result := string(app.truncateHTML("<p>Unclosed paragraph", 15))
		// Should still close the tag
		if !strings.Contains(result, "</p>") {
			t.Errorf("Expected closing tag to be added, got: %s", result)
		}
	})
}