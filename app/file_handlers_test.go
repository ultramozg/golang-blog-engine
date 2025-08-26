package app

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"html/template"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/ultramozg/golang-blog-engine/model"
	"github.com/ultramozg/golang-blog-engine/services"
	"github.com/ultramozg/golang-blog-engine/session"
	_ "modernc.org/sqlite"
)

// setupTestApp creates a test app with in-memory database
func setupTestApp(t *testing.T) (*App, func()) {
	// Create in-memory database
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Run migrations
	model.MigrateDatabase(db)

	// Create temp directory for uploads
	tempDir, err := os.MkdirTemp("", "file_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	app := NewApp()
	app.DB = db
	app.Config = &Config{
		Templates: "../templates/*.gohtml",
	}
	app.Sessions = session.NewSessionDB()
	app.FileService = services.NewFileService(db, tempDir, 10*1024*1024)
	app.SlugService = services.NewSlugService(db)
	app.SEOService = services.NewSEOService(db, "http://localhost:8080")

	// Initialize templates
	funcMap := template.FuncMap{
		"processFileReferences": app.processFileReferences,
		"truncateHTML":          app.truncateHTML,
	}
	app.Temp = template.Must(template.New("").Funcs(funcMap).ParseGlob("../templates/*.gohtml"))

	// Ensure upload directories exist
	if err := app.FileService.EnsureUploadDirectories(); err != nil {
		t.Fatalf("Failed to create upload directories: %v", err)
	}

	app.initializeRoutes()

	cleanup := func() {
		db.Close()
		os.RemoveAll(tempDir)
	}

	return &app, cleanup
}

// setupAdminSession creates an admin session for testing
func setupAdminSession(req *http.Request, sessions *session.SessionDB) {
	user := model.User{Type: session.ADMIN, Name: "admin"}
	cookie := sessions.CreateSession(user)
	req.AddCookie(cookie)
}

func TestUploadFileHandler(t *testing.T) {
	tests := []struct {
		name           string
		setupAuth      bool
		filename       string
		content        string
		contentType    string
		expectedStatus int
		expectJSON     bool
	}{
		{
			name:           "Upload valid file as admin",
			setupAuth:      true,
			filename:       "test.txt",
			content:        "Hello, World!",
			contentType:    "text/plain",
			expectedStatus: http.StatusOK,
			expectJSON:     true,
		},
		{
			name:           "Upload without authentication",
			setupAuth:      false,
			filename:       "test.txt",
			content:        "Hello, World!",
			contentType:    "text/plain",
			expectedStatus: http.StatusUnauthorized,
			expectJSON:     false,
		},
		{
			name:           "Upload PDF file",
			setupAuth:      true,
			filename:       "document.pdf",
			content:        "%PDF-1.4 fake pdf content",
			contentType:    "application/pdf",
			expectedStatus: http.StatusOK,
			expectJSON:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test app
			app, cleanup := setupTestApp(t)
			defer cleanup()

			// Create multipart form
			var buf bytes.Buffer
			writer := multipart.NewWriter(&buf)

			// Create form file field with proper content type
			h := make(map[string][]string)
			h["Content-Disposition"] = []string{`form-data; name="file"; filename="` + tt.filename + `"`}
			h["Content-Type"] = []string{tt.contentType}
			part, err := writer.CreatePart(h)
			if err != nil {
				t.Fatalf("Failed to create form file: %v", err)
			}

			// Write content to the part
			_, err = part.Write([]byte(tt.content))
			if err != nil {
				t.Fatalf("Failed to write content: %v", err)
			}

			writer.Close()

			// Create request
			req := httptest.NewRequest(http.MethodPost, "/upload-file", &buf)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			// Setup authentication if needed
			if tt.setupAuth {
				setupAdminSession(req, app.Sessions)
			}

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call handler
			app.Router.ServeHTTP(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Response body: %s", tt.expectedStatus, rr.Code, rr.Body.String())
			}

			// Check JSON response if expected
			if tt.expectJSON && rr.Code == http.StatusOK {
				var response struct {
					Success      bool   `json:"success"`
					UUID         string `json:"uuid"`
					OriginalName string `json:"original_name"`
					Size         int64  `json:"size"`
					MimeType     string `json:"mime_type"`
					DownloadURL  string `json:"download_url"`
				}

				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to parse JSON response: %v", err)
				}

				if !response.Success {
					t.Error("Expected success to be true")
				}

				if response.UUID == "" {
					t.Error("Expected UUID to be set")
				}

				if response.OriginalName != tt.filename {
					t.Errorf("Expected original name '%s', got '%s'", tt.filename, response.OriginalName)
				}

				if response.Size != int64(len(tt.content)) {
					t.Errorf("Expected size %d, got %d", len(tt.content), response.Size)
				}

				if !strings.HasPrefix(response.DownloadURL, "/files/") {
					t.Errorf("Expected download URL to start with '/files/', got '%s'", response.DownloadURL)
				}
			}
		})
	}
}

func TestServeFileHandler(t *testing.T) {
	// Setup test app
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// First upload a file
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	h := make(map[string][]string)
	h["Content-Disposition"] = []string{`form-data; name="file"; filename="test.txt"`}
	h["Content-Type"] = []string{"text/plain"}
	part, err := writer.CreatePart(h)
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	part.Write([]byte("Hello, World!"))
	writer.Close()

	// Upload request
	uploadReq := httptest.NewRequest(http.MethodPost, "/upload-file", &buf)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	setupAdminSession(uploadReq, app.Sessions)

	uploadRR := httptest.NewRecorder()
	app.Router.ServeHTTP(uploadRR, uploadReq)

	if uploadRR.Code != http.StatusOK {
		t.Fatalf("Upload failed with status %d", uploadRR.Code)
	}

	// Parse upload response to get UUID
	var uploadResponse struct {
		UUID string `json:"uuid"`
	}
	if err := json.Unmarshal(uploadRR.Body.Bytes(), &uploadResponse); err != nil {
		t.Fatalf("Failed to parse upload response: %v", err)
	}

	tests := []struct {
		name           string
		uuid           string
		expectedStatus int
		expectContent  bool
	}{
		{
			name:           "Download existing file",
			uuid:           uploadResponse.UUID,
			expectedStatus: http.StatusOK,
			expectContent:  true,
		},
		{
			name:           "Download non-existent file",
			uuid:           "non-existent-uuid",
			expectedStatus: http.StatusNotFound,
			expectContent:  false,
		},
		{
			name:           "Invalid UUID format",
			uuid:           "",
			expectedStatus: http.StatusBadRequest,
			expectContent:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create download request
			req := httptest.NewRequest(http.MethodGet, "/files/"+tt.uuid, nil)
			rr := httptest.NewRecorder()

			// Call handler
			app.Router.ServeHTTP(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Check content if expected
			if tt.expectContent && rr.Code == http.StatusOK {
				body := rr.Body.String()
				if body != "Hello, World!" {
					t.Errorf("Expected content 'Hello, World!', got '%s'", body)
				}

				// Check headers
				contentType := rr.Header().Get("Content-Type")
				if contentType == "" {
					t.Error("Expected Content-Type header to be set")
				}

				contentDisposition := rr.Header().Get("Content-Disposition")
				if !strings.Contains(contentDisposition, "attachment") {
					t.Error("Expected Content-Disposition to contain 'attachment'")
				}
			}
		})
	}
}

func TestListFilesHandler(t *testing.T) {
	// Setup test app
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Upload a few test files first
	testFiles := []string{"file1.txt", "file2.txt", "file3.txt"}
	for _, filename := range testFiles {
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		h := make(map[string][]string)
		h["Content-Disposition"] = []string{`form-data; name="file"; filename="` + filename + `"`}
		h["Content-Type"] = []string{"text/plain"}
		part, err := writer.CreatePart(h)
		if err != nil {
			t.Fatalf("Failed to create form file: %v", err)
		}
		part.Write([]byte("test content for " + filename))
		writer.Close()

		uploadReq := httptest.NewRequest(http.MethodPost, "/upload-file", &buf)
		uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
		setupAdminSession(uploadReq, app.Sessions)

		uploadRR := httptest.NewRecorder()
		app.Router.ServeHTTP(uploadRR, uploadReq)

		if uploadRR.Code != http.StatusOK {
			t.Fatalf("Upload of %s failed with status %d", filename, uploadRR.Code)
		}
	}

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedCount  int
	}{
		{
			name:           "List all files",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			expectedCount:  3,
		},
		{
			name:           "List with limit",
			queryParams:    "?limit=2",
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:           "List with offset",
			queryParams:    "?limit=2&offset=1",
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest(http.MethodGet, "/api/files"+tt.queryParams, nil)
			rr := httptest.NewRecorder()

			// Call handler
			app.Router.ServeHTTP(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if rr.Code == http.StatusOK {
				// Parse JSON response
				var response struct {
					Files []struct {
						UUID         string `json:"uuid"`
						OriginalName string `json:"original_name"`
						Size         int64  `json:"size"`
						MimeType     string `json:"mime_type"`
						DownloadURL  string `json:"download_url"`
					} `json:"files"`
				}

				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to parse JSON response: %v", err)
				}

				if len(response.Files) != tt.expectedCount {
					t.Errorf("Expected %d files, got %d", tt.expectedCount, len(response.Files))
				}

				// Verify file structure
				for _, file := range response.Files {
					if file.UUID == "" {
						t.Error("Expected UUID to be set")
					}
					if file.OriginalName == "" {
						t.Error("Expected OriginalName to be set")
					}
					if file.Size <= 0 {
						t.Error("Expected Size to be positive")
					}
					if !strings.HasPrefix(file.DownloadURL, "/files/") {
						t.Error("Expected DownloadURL to start with '/files/'")
					}
				}
			}
		})
	}
}

func TestFileUploadSecurity(t *testing.T) {
	// Setup test app
	app, cleanup := setupTestApp(t)
	defer cleanup()

	tests := []struct {
		name           string
		filename       string
		content        string
		contentType    string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "Upload executable file",
			filename:       "malicious.exe",
			content:        "fake executable content",
			contentType:    "application/x-executable",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "Upload script file",
			filename:       "script.sh",
			content:        "#!/bin/bash\necho 'hello'",
			contentType:    "application/x-sh",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "Upload allowed file type",
			filename:       "document.pdf",
			content:        "%PDF-1.4 fake pdf",
			contentType:    "application/pdf",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create multipart form
			var buf bytes.Buffer
			writer := multipart.NewWriter(&buf)
			h := make(map[string][]string)
			h["Content-Disposition"] = []string{`form-data; name="file"; filename="` + tt.filename + `"`}
			h["Content-Type"] = []string{tt.contentType}
			part, err := writer.CreatePart(h)
			if err != nil {
				t.Fatalf("Failed to create form file: %v", err)
			}
			part.Write([]byte(tt.content))
			writer.Close()

			// Create request
			req := httptest.NewRequest(http.MethodPost, "/upload-file", &buf)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			setupAdminSession(req, app.Sessions)

			rr := httptest.NewRecorder()
			app.Router.ServeHTTP(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Check for error message if expected
			if tt.expectError && rr.Code != http.StatusOK {
				body := rr.Body.String()
				if !strings.Contains(body, "not allowed") && !strings.Contains(body, "Failed to upload") {
					t.Error("Expected error message about file type not being allowed")
				}
			}
		})
	}
}
