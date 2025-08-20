package services

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	// Create a temporary database file
	tmpfile, err := os.CreateTemp("", "test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	// Open database connection
	db, err := sql.Open("sqlite3", tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	// Create posts table with slug column
	_, err = db.Exec(`
		CREATE TABLE posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			body TEXT NOT NULL,
			datepost TEXT NOT NULL,
			slug TEXT UNIQUE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatal(err)
	}

	// Clean up function
	t.Cleanup(func() {
		db.Close()
		os.Remove(tmpfile.Name())
	})

	return db
}

func TestNewSlugService(t *testing.T) {
	db := setupTestDB(t)
	service := NewSlugService(db)

	if service == nil {
		t.Error("NewSlugService should return a non-nil service")
	}
}

func TestSanitizeTitle(t *testing.T) {
	db := setupTestDB(t)
	service := NewSlugService(db)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple title",
			input:    "Hello World",
			expected: "hello-world",
		},
		{
			name:     "Title with special characters",
			input:    "Hello, World! How are you?",
			expected: "hello-world-how-are-you",
		},
		{
			name:     "Title with numbers",
			input:    "Top 10 Programming Languages in 2024",
			expected: "top-10-programming-languages-in-2024",
		},
		{
			name:     "Title with accented characters",
			input:    "Café and Naïve Programming",
			expected: "cafe-and-naive-programming",
		},
		{
			name:     "Title with underscores",
			input:    "hello_world_example",
			expected: "hello-world-example",
		},
		{
			name:     "Title with multiple spaces",
			input:    "Hello    World    Test",
			expected: "hello-world-test",
		},
		{
			name:     "Title with leading/trailing spaces",
			input:    "  Hello World  ",
			expected: "hello-world",
		},
		{
			name:     "Title with hyphens",
			input:    "Hello-World-Test",
			expected: "hello-world-test",
		},
		{
			name:     "Title with multiple consecutive hyphens",
			input:    "Hello---World",
			expected: "hello-world",
		},
		{
			name:     "Empty title",
			input:    "",
			expected: "",
		},
		{
			name:     "Title with only special characters",
			input:    "!@#$%^&*()",
			expected: "",
		},
		{
			name:     "Very long title",
			input:    "This is a very long title that should be truncated to ensure it does not exceed the maximum length limit of one hundred characters which is the standard",
			expected: "this-is-a-very-long-title-that-should-be-truncated-to-ensure-it-does-not-exceed-the-maximum-length-l",
		},
		{
			name:     "Title ending with hyphen after truncation",
			input:    "This is a very long title that should be truncated and the truncation point has a hyphen-",
			expected: "this-is-a-very-long-title-that-should-be-truncated-and-the-truncation-point-has-a-hyphen",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.SanitizeTitle(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeTitle(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerateSlug(t *testing.T) {
	db := setupTestDB(t)
	service := NewSlugService(db)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Normal title",
			input:    "My First Blog Post",
			expected: "my-first-blog-post",
		},
		{
			name:     "Empty title",
			input:    "",
			expected: "",
		},
		{
			name:     "Title that becomes empty after sanitization",
			input:    "!@#$%",
			expected: "untitled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.GenerateSlug(tt.input)
			if result != tt.expected {
				t.Errorf("GenerateSlug(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsSlugUnique(t *testing.T) {
	db := setupTestDB(t)
	service := NewSlugService(db)

	// Insert test data
	_, err := db.Exec("INSERT INTO posts (id, title, body, datepost, slug) VALUES (1, 'Test Post', 'Body', '2024-01-01', 'test-post')")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		slug          string
		excludePostID int
		expected      bool
	}{
		{
			name:          "Unique slug",
			slug:          "unique-slug",
			excludePostID: 0,
			expected:      true,
		},
		{
			name:          "Existing slug",
			slug:          "test-post",
			excludePostID: 0,
			expected:      false,
		},
		{
			name:          "Existing slug but excluded",
			slug:          "test-post",
			excludePostID: 1,
			expected:      true,
		},
		{
			name:          "Existing slug with different exclusion",
			slug:          "test-post",
			excludePostID: 2,
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.IsSlugUnique(tt.slug, tt.excludePostID)
			if result != tt.expected {
				t.Errorf("IsSlugUnique(%q, %d) = %v, want %v", tt.slug, tt.excludePostID, result, tt.expected)
			}
		})
	}
}

func TestEnsureUniqueSlug(t *testing.T) {
	db := setupTestDB(t)
	service := NewSlugService(db)

	// Insert test data
	_, err := db.Exec("INSERT INTO posts (id, title, body, datepost, slug) VALUES (1, 'Test Post', 'Body', '2024-01-01', 'test-post')")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec("INSERT INTO posts (id, title, body, datepost, slug) VALUES (2, 'Test Post 2', 'Body', '2024-01-01', 'test-post-1')")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		slug     string
		postID   int
		expected string
	}{
		{
			name:     "Unique slug remains unchanged",
			slug:     "unique-slug",
			postID:   0,
			expected: "unique-slug",
		},
		{
			name:     "Duplicate slug gets number suffix",
			slug:     "test-post",
			postID:   0,
			expected: "test-post-2",
		},
		{
			name:     "Empty slug becomes untitled",
			slug:     "",
			postID:   0,
			expected: "untitled",
		},
		{
			name:     "Existing slug for same post remains unchanged",
			slug:     "test-post",
			postID:   1,
			expected: "test-post",
		},
		{
			name:     "Multiple conflicts resolved with incrementing number",
			slug:     "test-post",
			postID:   3,
			expected: "test-post-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.EnsureUniqueSlug(tt.slug, tt.postID)
			if result != tt.expected {
				t.Errorf("EnsureUniqueSlug(%q, %d) = %q, want %q", tt.slug, tt.postID, result, tt.expected)
			}
		})
	}
}

func TestSlugServiceWithoutSlugColumn(t *testing.T) {
	// Create a database without slug column to test error handling
	tmpfile, err := os.CreateTemp("", "test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	db, err := sql.Open("sqlite3", tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Create posts table WITHOUT slug column
	_, err = db.Exec(`
		CREATE TABLE posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			body TEXT NOT NULL,
			datepost TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatal(err)
	}

	service := NewSlugService(db)

	// Test that IsSlugUnique returns false when column doesn't exist
	result := service.IsSlugUnique("test-slug", 0)
	if result != false {
		t.Errorf("IsSlugUnique should return false when slug column doesn't exist, got %v", result)
	}
}

func TestSlugServiceEdgeCases(t *testing.T) {
	db := setupTestDB(t)
	service := NewSlugService(db)

	// Test with various edge cases
	edgeCases := []struct {
		name   string
		input  string
		minLen int
		maxLen int
	}{
		{
			name:   "Single character",
			input:  "a",
			minLen: 1,
			maxLen: 1,
		},
		{
			name:   "Unicode characters",
			input:  "测试博客文章",
			minLen: 0,
			maxLen: 100,
		},
		{
			name:   "Mixed languages",
			input:  "Hello 世界 Test",
			minLen: 1,
			maxLen: 100,
		},
		{
			name:   "Numbers only",
			input:  "12345",
			minLen: 5,
			maxLen: 5,
		},
	}

	for _, tc := range edgeCases {
		t.Run(tc.name, func(t *testing.T) {
			result := service.GenerateSlug(tc.input)
			if len(result) < tc.minLen || len(result) > tc.maxLen {
				t.Errorf("GenerateSlug(%q) length = %d, want between %d and %d", tc.input, len(result), tc.minLen, tc.maxLen)
			}
		})
	}
}

// Benchmark tests
func BenchmarkGenerateSlug(b *testing.B) {
	db := setupTestDB(&testing.T{})
	service := NewSlugService(db)

	title := "This is a Sample Blog Post Title for Benchmarking"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.GenerateSlug(title)
	}
}

func BenchmarkSanitizeTitle(b *testing.B) {
	db := setupTestDB(&testing.T{})
	service := NewSlugService(db)

	title := "This is a Sample Blog Post Title with Special Characters!@#$%^&*() and Accented Letters like café"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.SanitizeTitle(title)
	}
}
