package app

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ultramozg/golang-blog-engine/model"
	"github.com/ultramozg/golang-blog-engine/session"
	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

// Test helper functions for handler tests

// createTestApp creates a test app with isolated database
func createTestApp(t *testing.T) (*App, func()) {
	tempDir, err := os.MkdirTemp("", "handler_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tempDir, "test.db")

	// Create data directory and files in current directory
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

	if err := os.WriteFile("data/courses.yml", []byte(coursesContent), 0600); err != nil {
		t.Fatalf("Failed to create courses.yml: %v", err)
	}

	if err := os.WriteFile("data/links.yml", []byte(linksContent), 0600); err != nil {
		t.Fatalf("Failed to create links.yml: %v", err)
	}

	// Set environment variables for testing
	originalDBURI := os.Getenv("DBURI")
	originalTemplates := os.Getenv("TEMPLATES")
	originalAdminPass := os.Getenv("ADMIN_PASSWORD")
	originalProduction := os.Getenv("PRODUCTION")

	os.Setenv("DBURI", dbPath)
	os.Setenv("TEMPLATES", "../templates/*.gohtml")
	os.Setenv("ADMIN_PASSWORD", "testpass123")
	os.Setenv("PRODUCTION", "false")

	// Create and initialize app
	app := NewApp()
	app.Initialize()

	cleanup := func() {
		if app.DB != nil {
			app.DB.Close()
		}
		os.RemoveAll(tempDir)

		// Restore original environment variables
		if originalDBURI != "" {
			os.Setenv("DBURI", originalDBURI)
		} else {
			os.Unsetenv("DBURI")
		}
		if originalTemplates != "" {
			os.Setenv("TEMPLATES", originalTemplates)
		} else {
			os.Unsetenv("TEMPLATES")
		}
		if originalAdminPass != "" {
			os.Setenv("ADMIN_PASSWORD", originalAdminPass)
		} else {
			os.Unsetenv("ADMIN_PASSWORD")
		}
		if originalProduction != "" {
			os.Setenv("PRODUCTION", originalProduction)
		} else {
			os.Unsetenv("PRODUCTION")
		}
	}

	return &app, cleanup
}

// seedTestPosts adds test posts to the database
func seedTestPosts(db *sql.DB) error {
	testPosts := []struct {
		title, body, date string
	}{
		{"Test Post 1", "This is the body of test post 1", "Mon Jan 1 12:00:00 2024"},
		{"Test Post 2", "This is the body of test post 2", "Mon Jan 2 12:00:00 2024"},
		{"Test Post 3", "This is the body of test post 3", "Mon Jan 3 12:00:00 2024"},
	}

	for _, post := range testPosts {
		_, err := db.Exec(`INSERT INTO posts (title, body, datepost) VALUES (?, ?, ?)`,
			post.title, post.body, post.date)
		if err != nil {
			return err
		}
	}
	return nil
}

// createAdminSession creates an admin session cookie
func createAdminSession(app *App) *http.Cookie {
	user := model.User{Type: session.ADMIN, Name: "admin"}
	return app.Sessions.CreateSession(user)
}

// createGitHubSession creates a GitHub user session cookie
func createGitHubSession(app *App) *http.Cookie {
	user := model.User{Type: session.GITHUB, Name: "testuser"}
	return app.Sessions.CreateSession(user)
}

// makeRequest creates and executes an HTTP request
func makeRequest(method, path, body string, cookies []*http.Cookie, headers map[string]string) (*http.Request, *httptest.ResponseRecorder) {
	var bodyReader *strings.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	var req *http.Request
	if bodyReader != nil {
		req = httptest.NewRequest(method, path, bodyReader)
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	// Set headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Set cookies
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	rr := httptest.NewRecorder()
	return req, rr
}

// Test cases for root handler
func TestRootHandler(t *testing.T) {
	app, cleanup := createTestApp(t)
	defer cleanup()

	tests := []struct {
		name             string
		path             string
		method           string
		expectedStatus   int
		expectedLocation string
	}{
		{
			name:             "Root path redirects to page 0",
			path:             "/",
			method:           http.MethodGet,
			expectedStatus:   http.StatusFound,
			expectedLocation: "/page?p=0",
		},
		{
			name:           "Non-root path returns 404",
			path:           "/nonexistent",
			method:         http.MethodGet,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, rr := makeRequest(tt.method, tt.path, "", nil, nil)
			app.root(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedLocation != "" {
				location := rr.Header().Get("Location")
				if location != tt.expectedLocation {
					t.Errorf("Expected location %s, got %s", tt.expectedLocation, location)
				}
			}
		})
	}
}

// Test cases for getPage handler
func TestGetPageHandler(t *testing.T) {
	app, cleanup := createTestApp(t)
	defer cleanup()

	// Seed test data
	if err := seedTestPosts(app.DB); err != nil {
		t.Fatalf("Failed to seed test data: %v", err)
	}

	tests := []struct {
		name            string
		path            string
		method          string
		expectedStatus  int
		checkContent    bool
		expectedContent string
	}{
		{
			name:            "Get first page",
			path:            "/page?p=0",
			method:          http.MethodGet,
			expectedStatus:  http.StatusOK,
			checkContent:    true,
			expectedContent: "Test Post",
		},
		{
			name:           "Get page with invalid parameter",
			path:           "/page?p=invalid",
			method:         http.MethodGet,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "HEAD request returns OK",
			path:           "/page?p=0",
			method:         http.MethodHead,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST request returns method not allowed",
			path:           "/page?p=0",
			method:         http.MethodPost,
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, rr := makeRequest(tt.method, tt.path, "", nil, nil)
			app.getPage(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.checkContent && !strings.Contains(rr.Body.String(), tt.expectedContent) {
				t.Errorf("Expected content to contain %s", tt.expectedContent)
			}
		})
	}
}

// Test cases for getPost handler
func TestGetPostHandler(t *testing.T) {
	app, cleanup := createTestApp(t)
	defer cleanup()

	// Seed test data
	if err := seedTestPosts(app.DB); err != nil {
		t.Fatalf("Failed to seed test data: %v", err)
	}

	tests := []struct {
		name            string
		path            string
		method          string
		expectedStatus  int
		checkContent    bool
		expectedContent string
	}{
		{
			name:            "Get existing post",
			path:            "/post?id=1",
			method:          http.MethodGet,
			expectedStatus:  http.StatusOK,
			checkContent:    true,
			expectedContent: "Test Post 1",
		},
		{
			name:           "Get non-existing post",
			path:           "/post?id=999",
			method:         http.MethodGet,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Get post with invalid ID",
			path:           "/post?id=invalid",
			method:         http.MethodGet,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "HEAD request returns OK",
			path:           "/post?id=1",
			method:         http.MethodHead,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST request returns method not allowed",
			path:           "/post?id=1",
			method:         http.MethodPost,
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, rr := makeRequest(tt.method, tt.path, "", nil, nil)
			app.getPost(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.checkContent && !strings.Contains(rr.Body.String(), tt.expectedContent) {
				t.Errorf("Expected content to contain %s", tt.expectedContent)
			}
		})
	}
}

// Test cases for login handler
func TestLoginHandler(t *testing.T) {
	app, cleanup := createTestApp(t)
	defer cleanup()

	tests := []struct {
		name             string
		method           string
		formData         string
		expectedStatus   int
		checkCookie      bool
		checkLocation    bool
		expectedLocation string
	}{
		{
			name:           "GET request shows login form",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
		},
		{
			name:             "Valid admin login",
			method:           http.MethodPost,
			formData:         "login=admin&password=testpass123",
			expectedStatus:   http.StatusSeeOther,
			checkCookie:      true,
			checkLocation:    true,
			expectedLocation: "/",
		},
		{
			name:           "Invalid password",
			method:         http.MethodPost,
			formData:       "login=admin&password=wrongpass",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Empty credentials",
			method:         http.MethodPost,
			formData:       "login=&password=",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Non-existing user",
			method:         http.MethodPost,
			formData:       "login=nonexistent&password=anypass",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "HEAD request returns OK",
			method:         http.MethodHead,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "PUT request returns method not allowed",
			method:         http.MethodPut,
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := map[string]string{}
			if tt.formData != "" {
				headers["Content-Type"] = "application/x-www-form-urlencoded"
			}

			req, rr := makeRequest(tt.method, "/login", tt.formData, nil, headers)
			app.login(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.checkCookie {
				cookies := rr.Result().Cookies()
				found := false
				for _, cookie := range cookies {
					if cookie.Name == "session" {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected session cookie to be set")
				}
			}

			if tt.checkLocation {
				location := rr.Header().Get("Location")
				if location != tt.expectedLocation {
					t.Errorf("Expected location %s, got %s", tt.expectedLocation, location)
				}
			}
		})
	}
}

// Test cases for logout handler
func TestLogoutHandler(t *testing.T) {
	app, cleanup := createTestApp(t)
	defer cleanup()

	tests := []struct {
		name             string
		method           string
		withSession      bool
		expectedStatus   int
		checkLocation    bool
		expectedLocation string
	}{
		{
			name:             "Logout with admin session",
			method:           http.MethodGet,
			withSession:      true,
			expectedStatus:   http.StatusSeeOther,
			checkLocation:    true,
			expectedLocation: "/",
		},
		{
			name:           "Logout without session",
			method:         http.MethodGet,
			withSession:    false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "HEAD request returns OK",
			method:         http.MethodHead,
			withSession:    true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST request returns method not allowed",
			method:         http.MethodPost,
			withSession:    true,
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cookies []*http.Cookie
			if tt.withSession {
				cookies = []*http.Cookie{createAdminSession(app)}
			}

			req, rr := makeRequest(tt.method, "/logout", "", cookies, nil)
			app.logout(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.checkLocation {
				location := rr.Header().Get("Location")
				if location != tt.expectedLocation {
					t.Errorf("Expected location %s, got %s", tt.expectedLocation, location)
				}
			}
		})
	}
}

// Test cases for createPost handler
func TestCreatePostHandler(t *testing.T) {
	app, cleanup := createTestApp(t)
	defer cleanup()

	tests := []struct {
		name             string
		method           string
		withSession      bool
		formData         string
		expectedStatus   int
		checkLocation    bool
		expectedLocation string
	}{
		{
			name:           "GET request shows create form",
			method:         http.MethodGet,
			withSession:    true,
			expectedStatus: http.StatusOK,
		},
		{
			name:             "Create valid post",
			method:           http.MethodPost,
			withSession:      true,
			formData:         "title=New Test Post&body=This is a new test post body",
			expectedStatus:   http.StatusSeeOther,
			checkLocation:    true,
			expectedLocation: "/",
		},
		{
			name:           "Create post with empty title",
			method:         http.MethodPost,
			withSession:    true,
			formData:       "title=&body=This is a test post body",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Create post with empty body",
			method:         http.MethodPost,
			withSession:    true,
			formData:       "title=Test Post&body=",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:             "Create post without session (handler only, no middleware)",
			method:           http.MethodPost,
			withSession:      false,
			formData:         "title=Test Post&body=Test body",
			expectedStatus:   http.StatusSeeOther, // Handler doesn't check auth, middleware does
			checkLocation:    true,
			expectedLocation: "/",
		},
		{
			name:           "PUT request returns method not allowed",
			method:         http.MethodPut,
			withSession:    true,
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cookies []*http.Cookie
			if tt.withSession {
				cookies = []*http.Cookie{createAdminSession(app)}
			}

			headers := map[string]string{}
			if tt.formData != "" {
				headers["Content-Type"] = "application/x-www-form-urlencoded"
			}

			req, rr := makeRequest(tt.method, "/create", tt.formData, cookies, headers)
			app.createPost(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.checkLocation {
				location := rr.Header().Get("Location")
				if location != tt.expectedLocation {
					t.Errorf("Expected location %s, got %s", tt.expectedLocation, location)
				}
			}
		})
	}
}

// Test cases for updatePost handler
func TestUpdatePostHandler(t *testing.T) {
	app, cleanup := createTestApp(t)
	defer cleanup()

	// Seed test data
	if err := seedTestPosts(app.DB); err != nil {
		t.Fatalf("Failed to seed test data: %v", err)
	}

	tests := []struct {
		name             string
		method           string
		path             string
		withSession      bool
		formData         string
		expectedStatus   int
		checkLocation    bool
		expectedLocation string
	}{
		{
			name:           "GET request shows update form",
			method:         http.MethodGet,
			path:           "/update?id=1",
			withSession:    true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "GET request with invalid ID",
			method:         http.MethodGet,
			path:           "/update?id=invalid",
			withSession:    true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "GET request with non-existing post",
			method:         http.MethodGet,
			path:           "/update?id=999",
			withSession:    true,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:             "Update existing post",
			method:           http.MethodPost,
			path:             "/update",
			withSession:      true,
			formData:         "id=1&title=Updated Post&body=Updated body",
			expectedStatus:   http.StatusSeeOther,
			checkLocation:    true,
			expectedLocation: "/",
		},
		{
			name:           "Update with invalid ID",
			method:         http.MethodPost,
			path:           "/update",
			withSession:    true,
			formData:       "id=invalid&title=Updated Post&body=Updated body",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Update with empty title",
			method:         http.MethodPost,
			path:           "/update",
			withSession:    true,
			formData:       "id=1&title=&body=Updated body",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:             "Update without session (handler only, no middleware)",
			method:           http.MethodPost,
			path:             "/update",
			withSession:      false,
			formData:         "id=1&title=Updated Post&body=Updated body",
			expectedStatus:   http.StatusSeeOther, // Handler doesn't check auth, middleware does
			checkLocation:    true,
			expectedLocation: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cookies []*http.Cookie
			if tt.withSession {
				cookies = []*http.Cookie{createAdminSession(app)}
			}

			headers := map[string]string{}
			if tt.formData != "" {
				headers["Content-Type"] = "application/x-www-form-urlencoded"
			}

			req, rr := makeRequest(tt.method, tt.path, tt.formData, cookies, headers)
			app.updatePost(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.checkLocation {
				location := rr.Header().Get("Location")
				if location != tt.expectedLocation {
					t.Errorf("Expected location %s, got %s", tt.expectedLocation, location)
				}
			}
		})
	}
}

// Test cases for deletePost handler
func TestDeletePostHandler(t *testing.T) {
	app, cleanup := createTestApp(t)
	defer cleanup()

	// Seed test data
	if err := seedTestPosts(app.DB); err != nil {
		t.Fatalf("Failed to seed test data: %v", err)
	}

	tests := []struct {
		name             string
		path             string
		withSession      bool
		expectedStatus   int
		checkLocation    bool
		expectedLocation string
	}{
		{
			name:             "Delete existing post",
			path:             "/delete?id=1",
			withSession:      true,
			expectedStatus:   http.StatusSeeOther,
			checkLocation:    true,
			expectedLocation: "/",
		},
		{
			name:           "Delete with invalid ID",
			path:           "/delete?id=invalid",
			withSession:    true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Delete non-existing post",
			path:           "/delete?id=999",
			withSession:    true,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:             "Delete without session (handler only, no middleware)",
			path:             "/delete?id=2",
			withSession:      false,
			expectedStatus:   http.StatusSeeOther, // Handler doesn't check auth, middleware does
			checkLocation:    true,
			expectedLocation: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cookies []*http.Cookie
			if tt.withSession {
				cookies = []*http.Cookie{createAdminSession(app)}
			}

			req, rr := makeRequest(http.MethodGet, tt.path, "", cookies, nil)
			app.deletePost(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.checkLocation {
				location := rr.Header().Get("Location")
				if location != tt.expectedLocation {
					t.Errorf("Expected location %s, got %s", tt.expectedLocation, location)
				}
			}
		})
	}
}

// Test cases for about handler
func TestAboutHandler(t *testing.T) {
	app, cleanup := createTestApp(t)
	defer cleanup()

	tests := []struct {
		name           string
		method         string
		expectedStatus int
	}{
		{
			name:           "GET request returns OK",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "HEAD request returns OK",
			method:         http.MethodHead,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST request returns method not allowed",
			method:         http.MethodPost,
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, rr := makeRequest(tt.method, "/about", "", nil, nil)
			app.about(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

// Test cases for links handler
func TestLinksHandler(t *testing.T) {
	app, cleanup := createTestApp(t)
	defer cleanup()

	tests := []struct {
		name            string
		method          string
		expectedStatus  int
		checkContent    bool
		expectedContent string
	}{
		{
			name:            "GET request returns OK",
			method:          http.MethodGet,
			expectedStatus:  http.StatusOK,
			checkContent:    true,
			expectedContent: "Test Link",
		},
		{
			name:           "HEAD request returns OK",
			method:         http.MethodHead,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST request returns method not allowed",
			method:         http.MethodPost,
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, rr := makeRequest(tt.method, "/links", "", nil, nil)
			app.links(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.checkContent && !strings.Contains(rr.Body.String(), tt.expectedContent) {
				t.Errorf("Expected content to contain %s", tt.expectedContent)
			}
		})
	}
}

// Test cases for courses handler
func TestCoursesHandler(t *testing.T) {
	app, cleanup := createTestApp(t)
	defer cleanup()

	tests := []struct {
		name            string
		method          string
		expectedStatus  int
		checkContent    bool
		expectedContent string
	}{
		{
			name:            "GET request returns OK",
			method:          http.MethodGet,
			expectedStatus:  http.StatusOK,
			checkContent:    true,
			expectedContent: "Test Course",
		},
		{
			name:           "HEAD request returns OK",
			method:         http.MethodHead,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST request returns method not allowed",
			method:         http.MethodPost,
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, rr := makeRequest(tt.method, "/courses", "", nil, nil)
			app.courses(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.checkContent && !strings.Contains(rr.Body.String(), tt.expectedContent) {
				t.Errorf("Expected content to contain %s", tt.expectedContent)
			}
		})
	}
}

// Test cases for createComment handler
func TestCreateCommentHandler(t *testing.T) {
	app, cleanup := createTestApp(t)
	defer cleanup()

	// Seed test data
	if err := seedTestPosts(app.DB); err != nil {
		t.Fatalf("Failed to seed test data: %v", err)
	}

	tests := []struct {
		name           string
		method         string
		withSession    bool
		sessionType    string
		formData       string
		expectedStatus int
		checkLocation  bool
	}{
		{
			name:           "Create comment as admin",
			method:         http.MethodPost,
			withSession:    true,
			sessionType:    "admin",
			formData:       "id=1&name=Test User&comment=This is a test comment",
			expectedStatus: http.StatusSeeOther,
			checkLocation:  true,
		},
		{
			name:           "Create comment as GitHub user",
			method:         http.MethodPost,
			withSession:    true,
			sessionType:    "github",
			formData:       "id=1&name=GitHub User&comment=This is a GitHub comment",
			expectedStatus: http.StatusSeeOther,
			checkLocation:  true,
		},
		{
			name:           "Create comment without session",
			method:         http.MethodPost,
			withSession:    false,
			formData:       "id=1&name=Test User&comment=This is a test comment",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Create comment with invalid post ID",
			method:         http.MethodPost,
			withSession:    true,
			sessionType:    "admin",
			formData:       "id=invalid&name=Test User&comment=This is a test comment",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Create comment with empty name",
			method:         http.MethodPost,
			withSession:    true,
			sessionType:    "admin",
			formData:       "id=1&name=&comment=This is a test comment",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Create comment with empty comment",
			method:         http.MethodPost,
			withSession:    true,
			sessionType:    "admin",
			formData:       "id=1&name=Test User&comment=",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "GET request returns method not allowed",
			method:         http.MethodGet,
			withSession:    true,
			sessionType:    "admin",
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cookies []*http.Cookie
			if tt.withSession {
				if tt.sessionType == "admin" {
					cookies = []*http.Cookie{createAdminSession(app)}
				} else {
					cookies = []*http.Cookie{createGitHubSession(app)}
				}
			}

			headers := map[string]string{}
			if tt.formData != "" {
				headers["Content-Type"] = "application/x-www-form-urlencoded"
				headers["Referer"] = "/post?id=1" // Set referer for redirect
			}

			req, rr := makeRequest(tt.method, "/create-comment", tt.formData, cookies, headers)
			app.createComment(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.checkLocation && rr.Code == http.StatusSeeOther {
				location := rr.Header().Get("Location")
				if location == "" {
					t.Errorf("Expected redirect location to be set")
				}
			}
		})
	}
}

// Test cases for deleteComment handler
func TestDeleteCommentHandler(t *testing.T) {
	app, cleanup := createTestApp(t)
	defer cleanup()

	// Seed test data
	if err := seedTestPosts(app.DB); err != nil {
		t.Fatalf("Failed to seed test data: %v", err)
	}

	// Add a test comment
	_, err := app.DB.Exec(`INSERT INTO comments (postid, name, date, comment) VALUES (?, ?, ?, ?)`,
		1, "Test User", time.Now().Format("Mon Jan _2 15:04:05 2006"), "Test comment")
	if err != nil {
		t.Fatalf("Failed to create test comment: %v", err)
	}

	tests := []struct {
		name           string
		path           string
		withSession    bool
		expectedStatus int
		checkLocation  bool
	}{
		{
			name:           "Delete comment as admin",
			path:           "/delete-comment?id=1",
			withSession:    true,
			expectedStatus: http.StatusSeeOther,
			checkLocation:  true,
		},
		{
			name:           "Delete comment without admin session",
			path:           "/delete-comment?id=1",
			withSession:    false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Delete comment with invalid ID",
			path:           "/delete-comment?id=invalid",
			withSession:    true,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cookies []*http.Cookie
			if tt.withSession {
				cookies = []*http.Cookie{createAdminSession(app)}
			}

			headers := map[string]string{
				"Referer": "/post?id=1", // Set referer for redirect
			}

			req, rr := makeRequest(http.MethodGet, tt.path, "", cookies, headers)
			app.deleteComment(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.checkLocation && rr.Code == http.StatusSeeOther {
				location := rr.Header().Get("Location")
				if location == "" {
					t.Errorf("Expected redirect location to be set")
				}
			}
		})
	}
}

// Test cases for security middleware
func TestSecurityMiddleware(t *testing.T) {
	app, cleanup := createTestApp(t)
	defer cleanup()

	// Create a test handler that just returns OK
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name           string
		path           string
		withSession    bool
		sessionType    string
		expectedStatus int
	}{
		{
			name:           "Admin endpoint with admin session",
			path:           "/create",
			withSession:    true,
			sessionType:    "admin",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Admin endpoint without session",
			path:           "/create",
			withSession:    false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Admin endpoint with GitHub session",
			path:           "/update",
			withSession:    true,
			sessionType:    "github",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Comment endpoint with admin session",
			path:           "/create-comment",
			withSession:    true,
			sessionType:    "admin",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Comment endpoint with GitHub session",
			path:           "/delete-comment",
			withSession:    true,
			sessionType:    "github",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Comment endpoint without session",
			path:           "/create-comment",
			withSession:    false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Public endpoint without session",
			path:           "/about",
			withSession:    false,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cookies []*http.Cookie
			if tt.withSession {
				if tt.sessionType == "admin" {
					cookies = []*http.Cookie{createAdminSession(app)}
				} else {
					cookies = []*http.Cookie{createGitHubSession(app)}
				}
			}

			req, rr := makeRequest(http.MethodGet, tt.path, "", cookies, nil)

			// Apply security middleware
			securedHandler := app.securityMiddleware(testHandler)
			securedHandler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

// Test utility functions
func TestUtilityFunctions(t *testing.T) {
	t.Run("absolute function", func(t *testing.T) {
		tests := []struct {
			input    int
			expected int
		}{
			{5, 5},
			{0, 0},
			{-1, 0},
			{-10, 0},
		}

		for _, tt := range tests {
			result := absolute(tt.input)
			if result != tt.expected {
				t.Errorf("absolute(%d) = %d, expected %d", tt.input, result, tt.expected)
			}
		}
	})

	t.Run("isNextPage function", func(t *testing.T) {
		tests := []struct {
			nextPage   int
			totalPosts int
			expected   bool
		}{
			{0, 10, true},  // 10 posts, page 0, has next page
			{1, 10, false}, // 10 posts, page 1, no next page (8 posts per page)
			{0, 5, false},  // 5 posts, page 0, no next page
			{0, 20, true},  // 20 posts, page 0, has next page
		}

		for _, tt := range tests {
			result := isNextPage(tt.nextPage, tt.totalPosts)
			if result != tt.expected {
				t.Errorf("isNextPage(%d, %d) = %v, expected %v", tt.nextPage, tt.totalPosts, result, tt.expected)
			}
		}
	})

	t.Run("HashPassword function", func(t *testing.T) {
		password := "testpassword123"
		success, hashedPassword := HashPassword(password)

		if !success {
			t.Errorf("HashPassword should return true for valid password")
		}

		if hashedPassword == password {
			t.Errorf("Password should be hashed, not returned as plain text")
		}

		if len(hashedPassword) < 20 {
			t.Errorf("Hashed password should be longer than plain text")
		}

		// Test that the same password produces different hashes (due to salt)
		success2, hashedPassword2 := HashPassword(password)
		if !success2 {
			t.Errorf("HashPassword should return true for valid password")
		}

		if hashedPassword == hashedPassword2 {
			t.Errorf("Same password should produce different hashes due to salt")
		}

		// Verify the hash can be used to verify the original password
		err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
		if err != nil {
			t.Errorf("Hashed password should verify against original password")
		}
	})
}
