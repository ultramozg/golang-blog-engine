package testutils

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// TestEnhancedDatabaseHelpers tests the enhanced database testing utilities
func TestEnhancedDatabaseHelpers(t *testing.T) {
	dh := NewDatabaseTestHelper(t)
	defer dh.Close()

	t.Run("Database Setup and Teardown", func(t *testing.T) {
		// Test that database is properly initialized
		AssertNotNil(t, dh.DB, "Database should be initialized")
		AssertNotEmpty(t, dh.Config.DBPath, "Database path should be set")

		// Test that tables exist
		dh.AssertTableExists(t, "posts")
		dh.AssertTableExists(t, "comments")
		dh.AssertTableExists(t, "users")
		dh.AssertTableExists(t, "files")
	})

	t.Run("Test Data Seeding", func(t *testing.T) {
		// Clear any existing data
		err := dh.ClearAllTables()
		AssertNil(t, err, "Should clear tables without error")

		// Test post seeding
		err = dh.SeedTestPosts(5)
		AssertNil(t, err, "Should seed posts without error")
		dh.AssertRecordCount(t, "posts", 5)

		// Test comment seeding
		err = dh.SeedTestComments(1, 3)
		AssertNil(t, err, "Should seed comments without error")
		dh.AssertRecordCount(t, "comments", 3)

		// Test user seeding
		err = dh.SeedTestUsers()
		AssertNil(t, err, "Should seed users without error")
		dh.AssertRecordCount(t, "users", 3)

		// Test file seeding
		err = dh.SeedTestFiles(2)
		AssertNil(t, err, "Should seed files without error")
		dh.AssertRecordCount(t, "files", 2)
	})

	t.Run("Transaction Testing", func(t *testing.T) {
		// Clear data first
		err := dh.ClearAllTables()
		AssertNil(t, err, "Should clear tables without error")

		// Test transaction rollback
		dh.Transaction(t, func(tx *sql.Tx) {
			_, err := tx.Exec("INSERT INTO posts (title, body, datepost, slug) VALUES (?, ?, ?, ?)",
				"Test Post", "Test Body", "2024-01-01", "test-post")
			AssertNil(t, err, "Should insert post in transaction")
		})

		// Verify rollback - post should not exist
		dh.AssertRecordCount(t, "posts", 0)
	})

	t.Run("Record Assertions", func(t *testing.T) {
		// Clear and seed data
		err := dh.ClearAllTables()
		AssertNil(t, err, "Should clear tables without error")

		err = dh.SeedTestPosts(1)
		AssertNil(t, err, "Should seed posts without error")

		// Test record existence assertion
		dh.AssertRecordExists(t, "posts", map[string]interface{}{
			"title": "Test Post 1",
		})
	})

	t.Run("Backup and Restore", func(t *testing.T) {
		// Clear and seed data
		err := dh.ClearAllTables()
		AssertNil(t, err, "Should clear tables without error")

		err = dh.SeedTestPosts(3)
		AssertNil(t, err, "Should seed posts without error")

		// Create backup
		backupPath, err := dh.BackupDatabase()
		AssertNil(t, err, "Should create backup without error")
		AssertTrue(t, len(backupPath) > 0, "Backup path should not be empty")

		// Verify backup file exists
		_, err = os.Stat(backupPath)
		AssertNil(t, err, "Backup file should exist")

		// Clear data
		err = dh.ClearAllTables()
		AssertNil(t, err, "Should clear tables without error")
		dh.AssertRecordCount(t, "posts", 0)

		// Restore from backup
		err = dh.RestoreDatabase(backupPath)
		AssertNil(t, err, "Should restore database without error")
		dh.AssertRecordCount(t, "posts", 3)
	})

	t.Run("Wait for Condition", func(t *testing.T) {
		// Test condition that becomes true
		counter := 0
		dh.WaitForCondition(t, func() bool {
			counter++
			return counter >= 3
		}, 1*time.Second, "Counter should reach 3")

		AssertEqual(t, 3, counter, "Counter should be 3")
	})
}

// TestEnhancedHTTPHelpers tests the enhanced HTTP testing utilities
func TestEnhancedHTTPHelpers(t *testing.T) {
	// Setup test app
	runner := NewTestRunner(t)
	defer runner.Close()

	err := runner.SetupTest()
	AssertNil(t, err, "Should setup test without error")

	// Create enhanced HTTP client
	client := NewHTTPTestClient(t, runner.HTTP.App)
	defer client.Close()

	t.Run("Request Builder - GET", func(t *testing.T) {
		resp, err := client.NewRequest().
			GET("/page").
			Query("p", "0").
			Execute()

		AssertNil(t, err, "Should execute GET request without error")
		defer resp.Body.Close()

		AssertHTTPStatusCode(t, resp, http.StatusOK)
	})

	t.Run("Request Builder - POST with Form Data", func(t *testing.T) {
		// Login first to get session
		sessionHelper := client.NewSessionHelper()
		sessionCookie, err := sessionHelper.LoginAsAdmin("admin", runner.HTTP.App.Config.AdminPass)
		AssertNil(t, err, "Should login without error")
		AssertNotNil(t, sessionCookie, "Should get session cookie")

		// Create post
		resp, err := client.NewRequest().
			POST("/create").
			Form("title", "Test Post").
			Form("body", "Test Body").
			Cookie(sessionCookie).
			Execute()

		AssertNil(t, err, "Should execute POST request without error")
		defer resp.Body.Close()

		AssertHTTPRedirect(t, resp, "/")
	})

	t.Run("Request Builder - Multipart Form", func(t *testing.T) {
		// Login first
		sessionHelper := client.NewSessionHelper()
		sessionCookie, err := sessionHelper.LoginAsAdmin("admin", runner.HTTP.App.Config.AdminPass)
		AssertNil(t, err, "Should login without error")

		// Upload file
		fileContent := []byte("test file content")
		resp, err := client.NewRequest().
			POST("/upload-file").
			Cookie(sessionCookie).
			Multipart().
			Field("description", "Test file").
			File("file", "test.txt", fileContent, "text/plain").
			Execute()

		AssertNil(t, err, "Should execute multipart request without error")
		defer resp.Body.Close()

		// Note: This might fail if authentication is required for file upload
		// The test demonstrates the multipart functionality
	})

	t.Run("Response Helper", func(t *testing.T) {
		resp, err := client.NewRequest().
			GET("/page").
			Query("p", "0").
			Execute()

		AssertNil(t, err, "Should execute request without error")

		helper, err := NewResponseHelper(resp)
		AssertNil(t, err, "Should create response helper without error")

		AssertEqual(t, http.StatusOK, helper.StatusCode())
		AssertTrue(t, len(helper.BodyString()) > 0, "Response body should not be empty")
		AssertTrue(t, helper.ContainsString("Test Post"), "Response should contain test data")
	})

	t.Run("Session Helper", func(t *testing.T) {
		sessionHelper := client.NewSessionHelper()

		// Test login
		sessionCookie, err := sessionHelper.LoginAsAdmin("admin", runner.HTTP.App.Config.AdminPass)
		AssertNil(t, err, "Should login without error")
		AssertNotNil(t, sessionCookie, "Should get session cookie")
		AssertEqual(t, "session", sessionCookie.Name)

		// Test logout
		err = sessionHelper.Logout(sessionCookie)
		AssertNil(t, err, "Should logout without error")
	})

	t.Run("Performance Measurement", func(t *testing.T) {
		resp, duration, err := client.MeasureResponseTime(func() (*http.Response, error) {
			return client.NewRequest().
				GET("/page").
				Query("p", "0").
				Execute()
		})

		AssertNil(t, err, "Should execute request without error")
		defer resp.Body.Close()

		AssertHTTPResponseTime(t, duration, 5*time.Second)
		AssertHTTPStatusCode(t, resp, http.StatusOK)
	})

	t.Run("Concurrent Requests", func(t *testing.T) {
		//nolint:bodyclose // Response bodies are closed in the loop below
		responses, errors := client.ConcurrentRequests(5, func(index int) (*http.Response, error) {
			return client.NewRequest().
				GET("/page").
				Query("p", "0").
				Execute()
		})

		AssertLen(t, responses, 5, "Should have 5 responses")
		AssertLen(t, errors, 5, "Should have 5 error slots")

		for i, err := range errors {
			AssertNil(t, err, fmt.Sprintf("Request %d should not have error", i))
		}

		for i, resp := range responses {
			if resp != nil {
				AssertHTTPStatusCode(t, resp, http.StatusOK, fmt.Sprintf("Request %d should return 200", i))
				resp.Body.Close()
			}
		}
	})
}

// TestEnhancedTestConfig tests the enhanced test configuration management
func TestEnhancedTestConfig(t *testing.T) {
	t.Run("Default Configuration", func(t *testing.T) {
		tcm := NewTestConfigManager()
		AssertNotNil(t, tcm, "Should create test config manager")

		config := tcm.GetDatabaseConfig()
		AssertEqual(t, "sqlite", config.Driver)
		AssertEqual(t, ":memory:", config.DSN)
		AssertTrue(t, config.MaxConnections > 0, "Max connections should be positive")

		authConfig := tcm.GetAuthConfig()
		AssertNotEmpty(t, authConfig.AdminUsername, "Admin username should not be empty")
		AssertNotEmpty(t, authConfig.AdminPassword, "Admin password should not be empty")
	})

	t.Run("Environment Variable Loading", func(t *testing.T) {
		// Set test environment variables
		originalDBDSN := os.Getenv("TEST_DB_DSN")
		originalAdminPass := os.Getenv("TEST_ADMIN_PASSWORD")

		defer func() {
			// Restore original values
			if originalDBDSN == "" {
				os.Unsetenv("TEST_DB_DSN")
			} else {
				os.Setenv("TEST_DB_DSN", originalDBDSN)
			}
			if originalAdminPass == "" {
				os.Unsetenv("TEST_ADMIN_PASSWORD")
			} else {
				os.Setenv("TEST_ADMIN_PASSWORD", originalAdminPass)
			}
		}()

		os.Setenv("TEST_DB_DSN", "test.db")
		os.Setenv("TEST_ADMIN_PASSWORD", "testpass456")

		tcm := NewTestConfigManager()
		tcm.LoadFromEnvironment()

		dbConfig := tcm.GetDatabaseConfig()
		AssertEqual(t, "test.db", dbConfig.DSN)

		authConfig := tcm.GetAuthConfig()
		AssertEqual(t, "testpass456", authConfig.AdminPassword)
	})

	t.Run("Configuration File Operations", func(t *testing.T) {
		tcm := NewTestConfigManager()

		// Create temporary config file
		tempDir, err := os.MkdirTemp("", "config_test_")
		AssertNil(t, err, "Should create temp dir without error")
		defer os.RemoveAll(tempDir)

		configFile := filepath.Join(tempDir, "test_config.json")

		// Save configuration
		err = tcm.SaveToFile(configFile)
		AssertNil(t, err, "Should save config file without error")

		// Verify file exists
		_, err = os.Stat(configFile)
		AssertNil(t, err, "Config file should exist")

		// Load configuration
		newTcm := NewTestConfigManager()
		err = newTcm.LoadFromFile(configFile)
		AssertNil(t, err, "Should load config file without error")

		// Verify loaded configuration matches
		originalAuth := tcm.GetAuthConfig()
		loadedAuth := newTcm.GetAuthConfig()
		AssertEqual(t, originalAuth.AdminUsername, loadedAuth.AdminUsername)
		AssertEqual(t, originalAuth.AdminPassword, loadedAuth.AdminPassword)
	})

	t.Run("Configuration Validation", func(t *testing.T) {
		tcm := NewTestConfigManager()

		// Valid configuration should pass
		err := tcm.Validate()
		AssertNil(t, err, "Valid configuration should pass validation")

		// Invalid configuration should fail
		tcm.config.Database.Driver = ""
		err = tcm.Validate()
		AssertNotNil(t, err, "Invalid configuration should fail validation")
		AssertContainsSubstring(t, err.Error(), "database driver")
	})

	t.Run("Environment Variable Management", func(t *testing.T) {
		tcm := NewTestConfigManager()
		defer tcm.Cleanup()

		// Set custom environment variable
		tcm.SetCustomEnvironmentVariable("TEST_CUSTOM_VAR", "custom_value")

		// Apply environment variables
		err := tcm.SetEnvironmentVariables()
		AssertNil(t, err, "Should set environment variables without error")

		// Verify custom variable is set
		value := os.Getenv("TEST_CUSTOM_VAR")
		AssertEqual(t, "custom_value", value)

		// Cleanup should restore original values
		tcm.Cleanup()

		// Custom variable should be unset after cleanup
		value = os.Getenv("TEST_CUSTOM_VAR")
		AssertEmpty(t, value, "Custom variable should be unset after cleanup")
	})

	t.Run("Temporary Directory Management", func(t *testing.T) {
		tcm := NewTestConfigManager()
		defer tcm.Cleanup()

		// Create temporary directory
		tempDir, err := tcm.CreateTempDirectory("test_")
		AssertNil(t, err, "Should create temp directory without error")
		AssertNotEmpty(t, tempDir, "Temp directory path should not be empty")

		// Verify directory exists
		_, err = os.Stat(tempDir)
		AssertNil(t, err, "Temp directory should exist")

		// Cleanup should remove directory
		tcm.Cleanup()

		// Directory should be removed after cleanup
		_, err = os.Stat(tempDir)
		AssertTrue(t, os.IsNotExist(err), "Temp directory should be removed after cleanup")
	})

	t.Run("Configuration Cloning", func(t *testing.T) {
		tcm := NewTestConfigManager()

		// Modify original configuration
		tcm.config.Auth.AdminPassword = "modified_password"

		// Clone configuration
		clonedTcm, err := tcm.Clone()
		AssertNil(t, err, "Should clone configuration without error")
		AssertNotNil(t, clonedTcm, "Cloned config manager should not be nil")

		// Verify clone has same values
		originalAuth := tcm.GetAuthConfig()
		clonedAuth := clonedTcm.GetAuthConfig()
		AssertEqual(t, originalAuth.AdminPassword, clonedAuth.AdminPassword)

		// Modify clone and verify original is unchanged
		clonedTcm.config.Auth.AdminPassword = "cloned_password"

		originalAuth = tcm.GetAuthConfig()
		clonedAuth = clonedTcm.GetAuthConfig()
		AssertNotEqual(t, originalAuth.AdminPassword, clonedAuth.AdminPassword)
	})
}

// TestEnhancedAssertions tests the enhanced assertion helpers
func TestEnhancedAssertions(t *testing.T) {
	t.Run("Basic Assertions", func(t *testing.T) {
		// Test AssertEqual
		AssertEqual(t, 42, 42, "Equal integers should pass")
		AssertEqual(t, "hello", "hello", "Equal strings should pass")

		// Test AssertNotEqual
		AssertNotEqual(t, 42, 43, "Different integers should pass")
		AssertNotEqual(t, "hello", "world", "Different strings should pass")

		// Test AssertTrue/AssertFalse
		AssertTrue(t, true, "True condition should pass")
		AssertFalse(t, false, "False condition should pass")

		// Test AssertNil/AssertNotNil
		AssertNil(t, nil, "Nil value should pass")
		AssertNotNil(t, "not nil", "Non-nil value should pass")
	})

	t.Run("Collection Assertions", func(t *testing.T) {
		// Test AssertEmpty/AssertNotEmpty
		AssertEmpty(t, "", "Empty string should pass")
		AssertEmpty(t, []int{}, "Empty slice should pass")
		AssertNotEmpty(t, "not empty", "Non-empty string should pass")
		AssertNotEmpty(t, []int{1, 2, 3}, "Non-empty slice should pass")

		// Test AssertLen
		AssertLen(t, []int{1, 2, 3}, 3, "Slice length should match")
		AssertLen(t, "hello", 5, "String length should match")
		AssertLen(t, map[string]int{"a": 1, "b": 2}, 2, "Map length should match")
	})

	t.Run("String Assertions", func(t *testing.T) {
		text := "Hello, World!"

		// Test substring assertions
		AssertContainsSubstring(t, text, "Hello", "Should contain substring")
		AssertContainsSubstring(t, text, "World", "Should contain substring")
		AssertNotContainsSubstring(t, text, "Goodbye", "Should not contain substring")

		// Test regex assertions
		AssertMatchesRegex(t, text, `Hello.*World`, "Should match regex pattern")
		AssertNotMatchesRegex(t, text, `^Goodbye`, "Should not match regex pattern")
	})

	t.Run("Panic Assertions", func(t *testing.T) {
		// Test AssertPanics
		AssertPanics(t, func() {
			panic("test panic")
		}, "Function should panic")

		// Test AssertNotPanics
		AssertNotPanics(t, func() {
			// This function should not panic
		}, "Function should not panic")
	})

	t.Run("Time-based Assertions", func(t *testing.T) {
		counter := 0

		// Test AssertEventually
		AssertEventually(t, func() bool {
			counter++
			return counter >= 3
		}, 1*time.Second, 10*time.Millisecond, "Counter should eventually reach 3")

		// Test AssertNever
		stable := true
		AssertNever(t, func() bool {
			return !stable
		}, 100*time.Millisecond, 10*time.Millisecond, "Stable condition should never change")
	})

	t.Run("Performance Assertions", func(t *testing.T) {
		// Test response time assertion
		start := time.Now()
		time.Sleep(10 * time.Millisecond)
		duration := time.Since(start)

		AssertResponseTime(t, duration, 100*time.Millisecond, "Response time should be acceptable")

		// Test memory usage assertion (simplified)
		var m1, m2 runtime.MemStats
		runtime.ReadMemStats(&m1)

		// Allocate some memory
		_ = make([]byte, 1024)

		runtime.ReadMemStats(&m2)
		AssertMemoryUsage(t, m1.Alloc, m2.Alloc, 10*1024, "Memory usage should be within limits")
	})
}

// BenchmarkEnhancedTestInfrastructure benchmarks the enhanced test infrastructure
func BenchmarkEnhancedTestInfrastructure(b *testing.B) {
	b.Run("DatabaseHelper", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			dh := NewDatabaseTestHelper(&testing.T{})
			dh.SeedTestPosts(10)
			dh.Close()
		}
	})

	b.Run("HTTPClient", func(b *testing.B) {
		runner := NewTestRunner(&testing.T{})
		defer runner.Close()
		runner.SetupTest()

		client := NewHTTPTestClient(&testing.T{}, runner.HTTP.App)
		defer client.Close()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			resp, err := client.NewRequest().
				GET("/page").
				Query("p", "0").
				Execute()
			if err == nil && resp != nil {
				resp.Body.Close()
			}
		}
	})

	b.Run("ConfigManager", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			tcm := NewTestConfigManager()
			tcm.LoadFromEnvironment()
			tcm.Validate()
		}
	})
}
