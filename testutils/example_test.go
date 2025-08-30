package testutils

import (
	"io"
	"net/http"
	"os"
	"testing"
)

// TestTestingInfrastructure demonstrates how to use the testing utilities
func TestTestingInfrastructure(t *testing.T) {
	// Create test runner
	runner := NewTestRunner(t)
	defer runner.Close()

	// Setup test data
	if err := runner.SetupTest(); err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}

	t.Run("Database Operations", func(t *testing.T) {
		// Test database seeding
		var count int
		err := runner.DB.DB.QueryRow("SELECT COUNT(*) FROM posts").Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query posts: %v", err)
		}
		if count != 3 {
			t.Errorf("Expected 3 posts, got %d", count)
		}

		// Test comment seeding
		err = runner.DB.DB.QueryRow("SELECT COUNT(*) FROM comments").Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query comments: %v", err)
		}
		if count != 3 {
			t.Errorf("Expected 3 comments, got %d", count)
		}
	})

	t.Run("HTTP Testing", func(t *testing.T) {
		// Test GET request
		resp, err := runner.HTTP.MakeRequest("GET", "/page?p=0", "", nil)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		AssertStatusCode(t, resp, http.StatusOK)

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		AssertContains(t, string(body), "Test Post")
	})

	t.Run("Authentication Testing", func(t *testing.T) {
		// Test admin login
		sessionCookie, err := runner.HTTP.LoginAsAdmin()
		if err != nil {
			t.Fatalf("Failed to login as admin: %v", err)
		}

		if sessionCookie == nil {
			t.Fatal("Expected session cookie, got nil")
		}

		// Test authenticated request
		resp, err := runner.HTTP.MakeRequestWithCookies("GET", "/create", "", nil, []*http.Cookie{sessionCookie})
		if err != nil {
			t.Fatalf("Failed to make authenticated request: %v", err)
		}
		defer resp.Body.Close()

		AssertStatusCode(t, resp, http.StatusOK)
	})

	t.Run("Form Submission Testing", func(t *testing.T) {
		// Login first
		sessionCookie, err := runner.HTTP.LoginAsAdmin()
		if err != nil {
			t.Fatalf("Failed to login as admin: %v", err)
		}

		// Test post creation
		formData := "title=Test+New+Post&body=This+is+a+test+post+body"
		headers := map[string]string{
			"Content-Type": ContentTypeFormURLEncoded,
		}

		resp, err := runner.HTTP.MakeRequestWithCookies("POST", "/create", formData, headers, []*http.Cookie{sessionCookie})
		if err != nil {
			t.Fatalf("Failed to create post: %v", err)
		}
		defer resp.Body.Close()

		AssertRedirect(t, resp, "/")

		// Verify post was created
		var count int
		err = runner.DB.DB.QueryRow("SELECT COUNT(*) FROM posts WHERE title = ?", "Test New Post").Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query new post: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected 1 new post, got %d", count)
		}
	})
}

// TestTestEnvironment demonstrates environment management
func TestTestEnvironment(t *testing.T) {
	env := NewTestEnvironment()
	defer env.Cleanup()

	// Test environment variable management
	originalValue := "original"
	testKey := "TEST_KEY"

	// Set original value
	env.SetEnv(testKey, originalValue)
	if value := os.Getenv(testKey); value != originalValue {
		t.Errorf("Expected %s, got %s", originalValue, value)
	}

	// Change value
	newValue := "new"
	env.SetEnv(testKey, newValue)
	if value := os.Getenv(testKey); value != newValue {
		t.Errorf("Expected %s, got %s", newValue, value)
	}

	// Test temp directory creation
	tempDir, err := env.CreateTempDir("test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Errorf("Temp directory was not created: %s", tempDir)
	}
}

// TestFileManagerUtilities demonstrates file management utilities
func TestFileManagerUtilities(t *testing.T) {
	env := NewTestEnvironment()
	defer env.Cleanup()

	tempDir, err := env.CreateTempDir("filemanager_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	fm := NewTestFileManager(tempDir)
	defer fm.Cleanup()

	// Test file creation
	content := "test file content"
	filePath, err := fm.CreateFile("test.txt", content)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Verify file exists and has correct content
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("File was not created: %s", filePath)
	}

	readContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(readContent) != content {
		t.Errorf("Expected %s, got %s", content, string(readContent))
	}

	// Test directory creation
	dirPath, err := fm.CreateDir("testdir")
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		t.Errorf("Directory was not created: %s", dirPath)
	}
}

// TestAssertionHelpers demonstrates assertion helper functions
func TestAssertionHelpers(t *testing.T) {
	// Create a mock response for testing assertions
	runner := NewTestRunner(t)
	defer runner.Close()

	if err := runner.SetupTest(); err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}

	// Test status code assertion
	resp, err := runner.HTTP.MakeRequest("GET", "/page?p=0", "", nil)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// This should pass
	AssertStatusCode(t, resp, http.StatusOK)

	// Test content assertions
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	bodyStr := string(body)
	AssertContains(t, bodyStr, "Test Post")             // Should contain test data
	AssertNotContains(t, bodyStr, "NonExistentContent") // Should not contain this

	// Test redirect assertion
	redirectResp, err := runner.HTTP.MakeRequest("GET", "/", "", nil)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer redirectResp.Body.Close()

	// The root handler redirects with 302 Found status
	if redirectResp.StatusCode >= 300 && redirectResp.StatusCode < 400 {
		AssertRedirect(t, redirectResp, "/page?p=0")
	}
}

// BenchmarkTestInfrastructure benchmarks the test infrastructure setup
func BenchmarkTestInfrastructure(b *testing.B) {
	for i := 0; i < b.N; i++ {
		runner := NewTestRunner(&testing.T{})
		runner.SetupTest()
		runner.Close()
	}
}
