package testutils

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"

	"github.com/ultramozg/golang-blog-engine/model"
	"github.com/ultramozg/golang-blog-engine/services"
)

// DatabaseTestHelper provides enhanced database testing utilities
type DatabaseTestHelper struct {
	DB      *sql.DB
	Config  *TestConfig
	TempDir string
	cleanup func()
}

// NewDatabaseTestHelper creates a new database test helper with enhanced features
func NewDatabaseTestHelper(t *testing.T) *DatabaseTestHelper {
	tempDir, err := os.MkdirTemp("", "db_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	config := &TestConfig{
		DBPath:      filepath.Join(tempDir, "test.db"),
		TempDir:     tempDir,
		Templates:   GetTemplatesPath(),
		AdminPass:   "testpass123",
		TestDataDir: filepath.Join(tempDir, "testdata"),
	}

	db, err := sql.Open("sqlite", config.DBPath)
	if err != nil {
		if rmErr := os.RemoveAll(tempDir); rmErr != nil {
			fmt.Printf("Warning: failed to cleanup temp directory after error: %v\n", rmErr)
		}
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Run migrations
	model.MigrateDatabase(db)

	cleanup := func() {
		if db != nil {
			if err := db.Close(); err != nil {
				fmt.Printf("Warning: failed to close database: %v\n", err)
			}
		}
		if err := os.RemoveAll(tempDir); err != nil {
			fmt.Printf("Warning: failed to remove temp directory: %v\n", err)
		}
	}

	return &DatabaseTestHelper{
		DB:      db,
		Config:  config,
		TempDir: tempDir,
		cleanup: cleanup,
	}
}

// Close cleans up the database and temporary files
func (dh *DatabaseTestHelper) Close() {
	if dh.cleanup != nil {
		dh.cleanup()
	}
}

// Transaction executes a function within a database transaction and rolls it back
func (dh *DatabaseTestHelper) Transaction(t *testing.T, fn func(*sql.Tx)) {
	tx, err := dh.DB.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback() // Always rollback for testing

	fn(tx)
}

// SeedTestPosts creates test posts with various configurations
func (dh *DatabaseTestHelper) SeedTestPosts(count int) error {
	slugService := services.NewSlugService(dh.DB)

	for i := 1; i <= count; i++ {
		title := fmt.Sprintf("Test Post %d", i)
		body := fmt.Sprintf("This is the body content for test post %d. It contains sample text for testing purposes.", i)
		date := time.Now().Add(-time.Duration(i) * time.Hour).Format("Mon Jan _2 15:04:05 2006")

		slug := slugService.GenerateSlug(title)
		uniqueSlug := slugService.EnsureUniqueSlug(slug, 0)

		_, err := dh.DB.Exec(`INSERT INTO posts (title, body, datepost, slug, created_at, updated_at, meta_description, keywords) 
			VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, ?, ?)`,
			title, body, date, uniqueSlug,
			fmt.Sprintf("Meta description for %s", title),
			fmt.Sprintf("test, post, %d", i))
		if err != nil {
			return fmt.Errorf("failed to seed post %d: %v", i, err)
		}
	}

	return nil
}

// SeedTestComments creates test comments for existing posts
func (dh *DatabaseTestHelper) SeedTestComments(postID int, count int) error {
	for i := 1; i <= count; i++ {
		name := fmt.Sprintf("Test User %d", i)
		date := time.Now().Add(-time.Duration(i) * time.Minute).Format("Mon Jan _2 15:04:05 2006")
		comment := fmt.Sprintf("This is test comment %d for post %d", i, postID)

		_, err := dh.DB.Exec(`INSERT INTO comments (postid, name, date, comment) VALUES (?, ?, ?, ?)`,
			postID, name, date, comment)
		if err != nil {
			return fmt.Errorf("failed to seed comment %d for post %d: %v", i, postID, err)
		}
	}

	return nil
}

// SeedTestUsers creates test users with different roles
func (dh *DatabaseTestHelper) SeedTestUsers() error {
	users := []struct {
		name     string
		password string
		userType int
	}{
		{"admin", "admin123", 1},
		{"testuser", "user123", 2},
		{"githubuser", "github123", 2},
	}

	for _, user := range users {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash password for %s: %v", user.name, err)
		}

		_, err = dh.DB.Exec(`INSERT INTO users (name, type, pass) VALUES (?, ?, ?)`,
			user.name, user.userType, string(hashedPassword))
		if err != nil {
			return fmt.Errorf("failed to create user %s: %v", user.name, err)
		}
	}

	return nil
}

// SeedTestFiles creates test file records in the database
func (dh *DatabaseTestHelper) SeedTestFiles(count int) error {
	for i := 1; i <= count; i++ {
		uuid := fmt.Sprintf("test-file-uuid-%d", i)
		originalName := fmt.Sprintf("test-file-%d.txt", i)
		storedName := fmt.Sprintf("stored-file-%d.txt", i)
		path := fmt.Sprintf("files/2024/01/documents/%s", storedName)
		size := int64(1024 * i) // Different sizes
		mimeType := "text/plain"

		_, err := dh.DB.Exec(`INSERT INTO files (uuid, original_name, stored_name, path, size, mime_type, download_count, created_at, is_image) 
			VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, ?)`,
			uuid, originalName, storedName, path, size, mimeType, 0, false)
		if err != nil {
			return fmt.Errorf("failed to seed file %d: %v", i, err)
		}
	}

	return nil
}

// ClearAllTables removes all data from all tables
func (dh *DatabaseTestHelper) ClearAllTables() error {
	tables := []string{"comments", "posts", "users", "files"}
	for _, table := range tables {
		_, err := dh.DB.Exec(fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			return fmt.Errorf("failed to clear table %s: %v", table, err)
		}
	}
	return nil
}

// GetTableCount returns the number of records in a table
func (dh *DatabaseTestHelper) GetTableCount(table string) (int, error) {
	var count int
	err := dh.DB.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
	return count, err
}

// ExecuteSQL executes arbitrary SQL and returns the result
func (dh *DatabaseTestHelper) ExecuteSQL(query string, args ...interface{}) (sql.Result, error) {
	return dh.DB.Exec(query, args...)
}

// QuerySQL executes a query and returns rows
func (dh *DatabaseTestHelper) QuerySQL(query string, args ...interface{}) (*sql.Rows, error) {
	return dh.DB.Query(query, args...)
}

// QueryRowSQL executes a query that returns a single row
func (dh *DatabaseTestHelper) QueryRowSQL(query string, args ...interface{}) *sql.Row {
	return dh.DB.QueryRow(query, args...)
}

// CreateTestSchema creates additional test tables if needed
func (dh *DatabaseTestHelper) CreateTestSchema() error {
	// Add any additional test-specific tables here
	testTables := []string{
		`CREATE TABLE IF NOT EXISTS test_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			message TEXT NOT NULL,
			level TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, query := range testTables {
		_, err := dh.DB.Exec(query)
		if err != nil {
			return fmt.Errorf("failed to create test schema: %v", err)
		}
	}

	return nil
}

// BackupDatabase creates a backup of the current database state
func (dh *DatabaseTestHelper) BackupDatabase() (string, error) {
	backupPath := filepath.Join(dh.TempDir, fmt.Sprintf("backup_%d.db", time.Now().UnixNano()))

	// Simple file copy for SQLite
	sourceFile, err := os.Open(dh.Config.DBPath)
	if err != nil {
		return "", fmt.Errorf("failed to open source database: %v", err)
	}
	defer sourceFile.Close()

	// #nosec G304 - backupPath is controlled by test code
	destFile, err := os.Create(backupPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file: %v", err)
	}
	defer destFile.Close()

	_, err = destFile.ReadFrom(sourceFile)
	if err != nil {
		return "", fmt.Errorf("failed to copy database: %v", err)
	}

	return backupPath, nil
}

// RestoreDatabase restores the database from a backup
func (dh *DatabaseTestHelper) RestoreDatabase(backupPath string) error {
	// Close current connection
	if dh.DB != nil {
		if err := dh.DB.Close(); err != nil {
			fmt.Printf("Warning: failed to close database: %v\n", err)
		}
	}

	// Copy backup over current database
	// #nosec G304 - backupPath is controlled by test code
	backupFile, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %v", err)
	}
	defer backupFile.Close()

	destFile, err := os.Create(dh.Config.DBPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %v", err)
	}
	defer destFile.Close()

	_, err = destFile.ReadFrom(backupFile)
	if err != nil {
		return fmt.Errorf("failed to restore database: %v", err)
	}

	// Reopen database connection
	dh.DB, err = sql.Open("sqlite", dh.Config.DBPath)
	if err != nil {
		return fmt.Errorf("failed to reopen database: %v", err)
	}

	return nil
}

// AssertTableExists checks if a table exists in the database
func (dh *DatabaseTestHelper) AssertTableExists(t *testing.T, tableName string) {
	var count int
	query := `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`
	err := dh.DB.QueryRow(query, tableName).Scan(&count)
	if err != nil {
		t.Fatalf("Error checking if table %s exists: %v", tableName, err)
	}
	if count != 1 {
		t.Errorf("Expected table %s to exist, but it doesn't", tableName)
	}
}

// AssertRecordCount checks if a table has the expected number of records
func (dh *DatabaseTestHelper) AssertRecordCount(t *testing.T, tableName string, expected int) {
	count, err := dh.GetTableCount(tableName)
	if err != nil {
		t.Fatalf("Error counting records in table %s: %v", tableName, err)
	}
	if count != expected {
		t.Errorf("Expected %d records in table %s, got %d", expected, tableName, count)
	}
}

// AssertRecordExists checks if a record exists with the given conditions
func (dh *DatabaseTestHelper) AssertRecordExists(t *testing.T, tableName string, conditions map[string]interface{}) {
	whereClause := ""
	args := make([]interface{}, 0, len(conditions))

	i := 0
	for column, value := range conditions {
		if i > 0 {
			whereClause += " AND "
		}
		whereClause += fmt.Sprintf("%s = ?", column)
		args = append(args, value)
		i++
	}

	// Use parameterized query to prevent SQL injection
	// Note: This is a simplified fix - in production, you'd want to validate table/column names against a whitelist
	query := "SELECT COUNT(*) FROM " + tableName + " WHERE " + whereClause
	var count int
	err := dh.DB.QueryRow(query, args...).Scan(&count)
	if err != nil {
		t.Fatalf("Error checking if record exists in table %s: %v", tableName, err)
	}
	if count == 0 {
		t.Errorf("Expected record to exist in table %s with conditions %v, but it doesn't", tableName, conditions)
	}
}

// WaitForCondition waits for a database condition to be met
func (dh *DatabaseTestHelper) WaitForCondition(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("Timeout waiting for condition: %s", message)
}
