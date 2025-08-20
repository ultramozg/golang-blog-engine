package model

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestMigrateExistingDatabase(t *testing.T) {
	// Create a temporary database file
	tmpfile, err := os.CreateTemp("", "test_migration_*.db")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	// Open database connection
	db, err := sql.Open("sqlite3", tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Create old schema without slug columns
	oldSchema := `
	CREATE TABLE posts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		body TEXT NOT NULL,
		datepost TEXT NOT NULL
	);
	
	INSERT INTO posts (title, body, datepost) VALUES 
		('Test Post 1', 'Body 1', '2024-01-01'),
		('Test Post 2', 'Body 2', '2024-01-02'),
		('Another Test!', 'Body 3', '2024-01-03');
	`

	_, err = db.Exec(oldSchema)
	if err != nil {
		t.Fatal(err)
	}

	// Run migration
	MigrateExistingDatabase(db)

	// Verify slug column was added
	var columnExists int
	err = db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('posts') WHERE name='slug'").Scan(&columnExists)
	if err != nil {
		t.Fatal(err)
	}
	if columnExists == 0 {
		t.Error("Slug column was not added")
	}

	// Verify created_at column was added
	err = db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('posts') WHERE name='created_at'").Scan(&columnExists)
	if err != nil {
		t.Fatal(err)
	}
	if columnExists == 0 {
		t.Error("created_at column was not added")
	}

	// Verify updated_at column was added
	err = db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('posts') WHERE name='updated_at'").Scan(&columnExists)
	if err != nil {
		t.Fatal(err)
	}
	if columnExists == 0 {
		t.Error("updated_at column was not added")
	}

	// Verify slugs were generated for existing posts
	rows, err := db.Query("SELECT id, title, COALESCE(slug, '') FROM posts")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	slugCount := 0
	for rows.Next() {
		var id int
		var title, slug string
		if err := rows.Scan(&id, &title, &slug); err != nil {
			t.Fatal(err)
		}
		
		if slug == "" {
			t.Errorf("Post %d ('%s') does not have a slug", id, title)
		} else {
			slugCount++
			t.Logf("Post %d: '%s' -> slug: '%s'", id, title, slug)
		}
	}

	if slugCount == 0 {
		t.Error("No slugs were generated for existing posts")
	}

	// Verify index was created
	var indexExists int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_posts_slug'").Scan(&indexExists)
	if err != nil {
		t.Fatal(err)
	}
	if indexExists == 0 {
		t.Error("Slug index was not created")
	}
}

func TestGenerateSlugsForExistingPosts(t *testing.T) {
	// Create a temporary database file
	tmpfile, err := os.CreateTemp("", "test_slug_generation_*.db")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	// Open database connection
	db, err := sql.Open("sqlite3", tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Create schema with slug column
	schema := `
	CREATE TABLE posts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		body TEXT NOT NULL,
		datepost TEXT NOT NULL,
		slug TEXT
	);
	
	INSERT INTO posts (title, body, datepost, slug) VALUES 
		('Hello World', 'Body 1', '2024-01-01', NULL),
		('Test Post!', 'Body 2', '2024-01-02', ''),
		('Another Test', 'Body 3', '2024-01-03', 'existing-slug'),
		('Café & Restaurant', 'Body 4', '2024-01-04', NULL);
	`

	_, err = db.Exec(schema)
	if err != nil {
		t.Fatal(err)
	}

	// Run slug generation
	GenerateSlugsForExistingPosts(db)

	// Verify slugs were generated correctly
	tests := []struct {
		id           int
		expectedSlug string
	}{
		{1, "hello-world"},
		{2, "test-post"},
		{3, "existing-slug"}, // Should remain unchanged
		{4, "cafe-restaurant"},
	}

	for _, test := range tests {
		var slug string
		err = db.QueryRow("SELECT slug FROM posts WHERE id = ?", test.id).Scan(&slug)
		if err != nil {
			t.Fatal(err)
		}

		if slug != test.expectedSlug {
			t.Errorf("Post %d: expected slug '%s', got '%s'", test.id, test.expectedSlug, slug)
		}
	}
}

func TestSlugUniqueness(t *testing.T) {
	// Create a temporary database file
	tmpfile, err := os.CreateTemp("", "test_slug_uniqueness_*.db")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	// Open database connection
	db, err := sql.Open("sqlite3", tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Create schema with slug column
	schema := `
	CREATE TABLE posts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		body TEXT NOT NULL,
		datepost TEXT NOT NULL,
		slug TEXT
	);
	
	INSERT INTO posts (title, body, datepost, slug) VALUES 
		('Test Post', 'Body 1', '2024-01-01', NULL),
		('Test Post', 'Body 2', '2024-01-02', NULL),
		('Test Post', 'Body 3', '2024-01-03', NULL);
	`

	_, err = db.Exec(schema)
	if err != nil {
		t.Fatal(err)
	}

	// Run slug generation
	GenerateSlugsForExistingPosts(db)

	// Verify unique slugs were generated
	var slugs []string
	rows, err := db.Query("SELECT slug FROM posts ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			t.Fatal(err)
		}
		slugs = append(slugs, slug)
	}

	// Check that all slugs are unique
	slugMap := make(map[string]bool)
	for _, slug := range slugs {
		if slugMap[slug] {
			t.Errorf("Duplicate slug found: %s", slug)
		}
		slugMap[slug] = true
	}

	// Verify expected pattern
	expectedSlugs := []string{"test-post", "test-post-1", "test-post-2"}
	if len(slugs) != len(expectedSlugs) {
		t.Errorf("Expected %d slugs, got %d", len(expectedSlugs), len(slugs))
	}

	for i, expected := range expectedSlugs {
		if i < len(slugs) && slugs[i] != expected {
			t.Errorf("Expected slug %d to be '%s', got '%s'", i, expected, slugs[i])
		}
	}
}

func TestMigrationIdempotency(t *testing.T) {
	// Create a temporary database file
	tmpfile, err := os.CreateTemp("", "test_idempotency_*.db")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	// Open database connection
	db, err := sql.Open("sqlite3", tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Create old schema
	oldSchema := `
	CREATE TABLE posts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		body TEXT NOT NULL,
		datepost TEXT NOT NULL
	);
	
	INSERT INTO posts (title, body, datepost) VALUES 
		('Test Post', 'Body 1', '2024-01-01');
	`

	_, err = db.Exec(oldSchema)
	if err != nil {
		t.Fatal(err)
	}

	// Run migration multiple times
	MigrateExistingDatabase(db)
	MigrateExistingDatabase(db)
	MigrateExistingDatabase(db)

	// Verify columns exist and data is intact
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM posts").Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("Expected 1 post, got %d", count)
	}

	// Verify slug was generated
	var slug string
	err = db.QueryRow("SELECT slug FROM posts WHERE id = 1").Scan(&slug)
	if err != nil {
		t.Fatal(err)
	}
	if slug == "" {
		t.Error("Slug was not generated")
	}
}