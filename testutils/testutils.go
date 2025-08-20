package testutils

import (
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/ultramozg/golang-blog-engine/app"
	"github.com/ultramozg/golang-blog-engine/model"
	"golang.org/x/crypto/bcrypt"
)

// TestConfig holds configuration for testing
type TestConfig struct {
	DBPath      string
	TempDir     string
	Templates   string
	AdminPass   string
	TestDataDir string
}

// NewTestConfig creates a new test configuration
func NewTestConfig() *TestConfig {
	tempDir, _ := ioutil.TempDir("", "blog_test_")
	return &TestConfig{
		DBPath:      filepath.Join(tempDir, "test.db"),
		TempDir:     tempDir,
		Templates:   "templates/*.gohtml",
		AdminPass:   "testpass123",
		TestDataDir: filepath.Join(tempDir, "testdata"),
	}
}

// TestDatabase provides utilities for database testing
type TestDatabase struct {
	DB     *sql.DB
	Config *TestConfig
}

// NewTestDatabase creates a new test database instance
func NewTestDatabase(t *testing.T) *TestDatabase {
	config := NewTestConfig()
	
	// Create test database
	db, err := sql.Open("sqlite3", config.DBPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Run migrations
	model.MigrateDatabase(db)

	return &TestDatabase{
		DB:     db,
		Config: config,
	}
}

// Close cleans up the test database and temporary files
func (td *TestDatabase) Close() error {
	if td.DB != nil {
		td.DB.Close()
	}
	return os.RemoveAll(td.Config.TempDir)
}

// SeedTestData inserts test data into the database
func (td *TestDatabase) SeedTestData() error {
	// Insert test posts
	testPosts := []struct {
		title, body, date string
	}{
		{"Test Post 1", "This is the body of test post 1", "Mon Jan 1 12:00:00 2024"},
		{"Test Post 2", "This is the body of test post 2", "Mon Jan 2 12:00:00 2024"},
		{"Test Post 3", "This is the body of test post 3", "Mon Jan 3 12:00:00 2024"},
	}

	for _, post := range testPosts {
		_, err := td.DB.Exec(`INSERT INTO posts (title, body, datepost) VALUES (?, ?, ?)`,
			post.title, post.body, post.date)
		if err != nil {
			return fmt.Errorf("failed to seed post data: %v", err)
		}
	}

	// Insert test comments
	testComments := []struct {
		postID      int
		name, date, comment string
	}{
		{1, "Test User", "Mon Jan 1 13:00:00 2024", "This is a test comment"},
		{1, "Another User", "Mon Jan 1 14:00:00 2024", "Another test comment"},
		{2, "Test User", "Mon Jan 2 13:00:00 2024", "Comment on second post"},
	}

	for _, comment := range testComments {
		_, err := td.DB.Exec(`INSERT INTO comments (postid, name, date, comment) VALUES (?, ?, ?, ?)`,
			comment.postID, comment.name, comment.date, comment.comment)
		if err != nil {
			return fmt.Errorf("failed to seed comment data: %v", err)
		}
	}

	return nil
}

// ClearTestData removes all test data from the database
func (td *TestDatabase) ClearTestData() error {
	tables := []string{"comments", "posts", "users"}
	for _, table := range tables {
		_, err := td.DB.Exec(fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			return fmt.Errorf("failed to clear table %s: %v", table, err)
		}
	}
	return nil
}

// CreateTestUser creates a test user in the database
func (td *TestDatabase) CreateTestUser(name, password string, userType int) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %v", err)
	}

	_, err = td.DB.Exec(`INSERT INTO users (name, type, pass) VALUES (?, ?, ?)`,
		name, userType, string(hashedPassword))
	if err != nil {
		return fmt.Errorf("failed to create test user: %v", err)
	}

	return nil
}

// HTTPTestHelper provides utilities for HTTP testing
type HTTPTestHelper struct {
	App    *app.App
	Server *httptest.Server
	Client *http.Client
}

// NewHTTPTestHelper creates a new HTTP test helper
func NewHTTPTestHelper(t *testing.T, testDB *TestDatabase) *HTTPTestHelper {
	// Create data directory and files if they don't exist
	if err := os.MkdirAll("data", 0755); err != nil {
		t.Fatalf("Failed to create data directory: %v", err)
	}

	// Create minimal test data files
	coursesContent := `infos:
  - title: "Test Course"
    link: "https://example.com/course"
    description: "Test course description"`
	
	linksContent := `infos:
  - title: "Test Link"
    link: "https://example.com/link"
    description: "Test link description"`

	if err := ioutil.WriteFile("data/courses.yml", []byte(coursesContent), 0644); err != nil {
		t.Fatalf("Failed to create courses.yml: %v", err)
	}

	if err := ioutil.WriteFile("data/links.yml", []byte(linksContent), 0644); err != nil {
		t.Fatalf("Failed to create links.yml: %v", err)
	}

	// Set environment variables for testing
	os.Setenv("DBURI", testDB.Config.DBPath)
	os.Setenv("TEMPLATES", GetTemplatesPath())
	os.Setenv("ADMIN_PASSWORD", testDB.Config.AdminPass)
	os.Setenv("PRODUCTION", "false")

	// Create and initialize app
	testApp := app.NewApp()
	testApp.Initialize()

	// Create test server
	server := httptest.NewServer(testApp.Router)

	// Create HTTP client that doesn't follow redirects
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return &HTTPTestHelper{
		App:    &testApp,
		Server: server,
		Client: client,
	}
}

// Close shuts down the test server
func (h *HTTPTestHelper) Close() {
	if h.Server != nil {
		h.Server.Close()
	}
}

// MakeRequest makes an HTTP request and returns the response
func (h *HTTPTestHelper) MakeRequest(method, path string, body string, headers map[string]string) (*http.Response, error) {
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	req, err := http.NewRequest(method, h.Server.URL+path, bodyReader)
	if err != nil {
		return nil, err
	}

	// Set headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return h.Client.Do(req)
}

// MakeRequestWithCookies makes an HTTP request with cookies
func (h *HTTPTestHelper) MakeRequestWithCookies(method, path string, body string, headers map[string]string, cookies []*http.Cookie) (*http.Response, error) {
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	req, err := http.NewRequest(method, h.Server.URL+path, bodyReader)
	if err != nil {
		return nil, err
	}

	// Set headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Set cookies
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	return h.Client.Do(req)
}

// LoginAsAdmin performs admin login and returns session cookie
func (h *HTTPTestHelper) LoginAsAdmin() (*http.Cookie, error) {
	loginData := "login=admin&password=" + h.App.Config.AdminPass
	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}

	resp, err := h.MakeRequest("POST", "/login", loginData, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check for redirect status codes (302, 303, etc.)
	if resp.StatusCode < 300 || resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("login failed with status: %d, body: %s", resp.StatusCode, string(body))
	}

	// Extract session cookie
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "session" {
			return cookie, nil
		}
	}

	return nil, fmt.Errorf("session cookie not found")
}

// TestRunner provides utilities for running tests with setup and teardown
type TestRunner struct {
	DB   *TestDatabase
	HTTP *HTTPTestHelper
}

// NewTestRunner creates a new test runner with database and HTTP helpers
func NewTestRunner(t *testing.T) *TestRunner {
	db := NewTestDatabase(t)
	http := NewHTTPTestHelper(t, db)

	return &TestRunner{
		DB:   db,
		HTTP: http,
	}
}

// Close cleans up all test resources
func (tr *TestRunner) Close() {
	if tr.HTTP != nil {
		tr.HTTP.Close()
	}
	if tr.DB != nil {
		tr.DB.Close()
	}
}

// SetupTest performs common test setup
func (tr *TestRunner) SetupTest() error {
	// Clear any existing data
	if err := tr.DB.ClearTestData(); err != nil {
		return err
	}

	// Create admin user
	if err := tr.DB.CreateTestUser("admin", tr.HTTP.App.Config.AdminPass, 1); err != nil {
		return err
	}

	// Seed test data
	if err := tr.DB.SeedTestData(); err != nil {
		return err
	}

	return nil
}

// AssertStatusCode checks if the response has the expected status code
func AssertStatusCode(t *testing.T, resp *http.Response, expected int) {
	if resp.StatusCode != expected {
		t.Errorf("Expected status code %d, got %d", expected, resp.StatusCode)
	}
}

// AssertContains checks if the response body contains the expected string
func AssertContains(t *testing.T, body, expected string) {
	if !strings.Contains(body, expected) {
		t.Errorf("Expected response to contain '%s', but it didn't. Body: %s", expected, body)
	}
}

// AssertNotContains checks if the response body does not contain the string
func AssertNotContains(t *testing.T, body, unexpected string) {
	if strings.Contains(body, unexpected) {
		t.Errorf("Expected response to not contain '%s', but it did. Body: %s", unexpected, body)
	}
}

// AssertRedirect checks if the response is a redirect to the expected location
func AssertRedirect(t *testing.T, resp *http.Response, expectedLocation string) {
	if resp.StatusCode < 300 || resp.StatusCode >= 400 {
		t.Errorf("Expected redirect status code (3xx), got %d", resp.StatusCode)
		return
	}

	location := resp.Header.Get("Location")
	if location != expectedLocation {
		t.Errorf("Expected redirect to '%s', got '%s'", expectedLocation, location)
	}
}

// AssertCookieExists checks if a cookie with the given name exists
func AssertCookieExists(t *testing.T, resp *http.Response, cookieName string) *http.Cookie {
	for _, cookie := range resp.Cookies() {
		if cookie.Name == cookieName {
			return cookie
		}
	}
	t.Errorf("Expected cookie '%s' to exist, but it didn't", cookieName)
	return nil
}



// WaitForCondition waits for a condition to be true with timeout
func WaitForCondition(condition func() bool, timeout time.Duration, interval time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(interval)
	}
	return false
}

// CreateTempFile creates a temporary file with given content
func CreateTempFile(t *testing.T, content string) string {
	tmpFile, err := ioutil.TempFile("", "test_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	return tmpFile.Name()
}

// RemoveTempFile removes a temporary file
func RemoveTempFile(path string) error {
	return os.Remove(path)
}