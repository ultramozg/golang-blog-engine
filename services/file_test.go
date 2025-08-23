package services

import (
	"bytes"
	"database/sql"
	"fmt"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ultramozg/golang-blog-engine/model"
	_ "modernc.org/sqlite"
)

func setupFileTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Run migrations
	model.MigrateDatabase(db)

	return db
}

func setupTestFileService(t *testing.T) (FileService, string, func()) {
	db := setupFileTestDB(t)
	
	// Create temporary directory for test uploads
	tempDir, err := os.MkdirTemp("", "file_service_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	fs := NewFileService(db, tempDir, 10*1024*1024) // 10MB max file size

	cleanup := func() {
		db.Close()
		os.RemoveAll(tempDir)
	}

	return fs, tempDir, cleanup
}

func createTestMultipartFile(t *testing.T, filename, content, mimeType string) (multipart.File, *multipart.FileHeader) {
	// Create a buffer to write our multipart form
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)

	// Create form file field
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}

	// Write content to the part
	_, err = part.Write([]byte(content))
	if err != nil {
		t.Fatalf("Failed to write content: %v", err)
	}

	writer.Close()

	// Parse the multipart form
	reader := multipart.NewReader(&b, writer.Boundary())
	form, err := reader.ReadForm(10 << 20) // 10MB max
	if err != nil {
		t.Fatalf("Failed to read form: %v", err)
	}

	files := form.File["file"]
	if len(files) == 0 {
		t.Fatal("No files found in form")
	}

	fileHeader := files[0]
	
	// Override the content type if specified
	if mimeType != "" {
		if fileHeader.Header == nil {
			fileHeader.Header = make(textproto.MIMEHeader)
		}
		fileHeader.Header.Set("Content-Type", mimeType)
	}

	file, err := fileHeader.Open()
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}

	return file, fileHeader
}

func TestFileService_EnsureUploadDirectories(t *testing.T) {
	fs, tempDir, cleanup := setupTestFileService(t)
	defer cleanup()

	err := fs.EnsureUploadDirectories()
	if err != nil {
		t.Fatalf("EnsureUploadDirectories failed: %v", err)
	}

	// Check if directories were created
	filesDir := filepath.Join(tempDir, "files")
	if _, err := os.Stat(filesDir); os.IsNotExist(err) {
		t.Error("Files directory was not created")
	}

	// Check if current year/month directory was created
	now := time.Now()
	monthDir := filepath.Join(filesDir, fmt.Sprintf("%d/%02d", now.Year(), now.Month()))
	if _, err := os.Stat(monthDir); os.IsNotExist(err) {
		t.Error("Month directory was not created")
	}
}

func TestFileService_UploadFile_Success(t *testing.T) {
	fs, tempDir, cleanup := setupTestFileService(t)
	defer cleanup()

	// Ensure directories exist
	err := fs.EnsureUploadDirectories()
	if err != nil {
		t.Fatalf("Failed to ensure directories: %v", err)
	}

	// Create test file
	file, header := createTestMultipartFile(t, "test.txt", "Hello, World!", "text/plain")
	defer file.Close()

	// Upload file
	fileRecord, err := fs.UploadFile(file, header)
	if err != nil {
		t.Fatalf("UploadFile failed: %v", err)
	}

	// Verify file record
	if fileRecord.OriginalName != "test.txt" {
		t.Errorf("Expected original name 'test.txt', got '%s'", fileRecord.OriginalName)
	}

	if fileRecord.MimeType != "text/plain" {
		t.Errorf("Expected MIME type 'text/plain', got '%s'", fileRecord.MimeType)
	}

	if fileRecord.Size != 13 { // "Hello, World!" is 13 bytes
		t.Errorf("Expected size 13, got %d", fileRecord.Size)
	}

	// Verify file exists on filesystem
	fullPath := filepath.Join(tempDir, fileRecord.Path)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Error("Uploaded file does not exist on filesystem")
	}

	// Verify file content
	content, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("Failed to read uploaded file: %v", err)
	}

	if string(content) != "Hello, World!" {
		t.Errorf("Expected content 'Hello, World!', got '%s'", string(content))
	}
}

func TestFileService_UploadFile_FileSizeExceeded(t *testing.T) {
	db := setupFileTestDB(t)
	defer db.Close()

	tempDir, err := os.MkdirTemp("", "file_service_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create file service with very small max file size
	fs := NewFileService(db, tempDir, 5) // 5 bytes max

	// Create test file larger than limit
	file, header := createTestMultipartFile(t, "large.txt", "This is a large file", "text/plain")
	defer file.Close()

	// Upload should fail
	_, err = fs.UploadFile(file, header)
	if err == nil {
		t.Error("Expected error for file size exceeded, got nil")
	}

	if !strings.Contains(err.Error(), "exceeds maximum allowed size") {
		t.Errorf("Expected size error, got: %v", err)
	}
}

func TestFileService_UploadFile_InvalidFileType(t *testing.T) {
	fs, _, cleanup := setupTestFileService(t)
	defer cleanup()

	// Create test file with disallowed MIME type
	file, header := createTestMultipartFile(t, "script.exe", "malicious content", "application/x-executable")
	defer file.Close()

	// Upload should fail
	_, err := fs.UploadFile(file, header)
	if err == nil {
		t.Error("Expected error for invalid file type, got nil")
	}

	if !strings.Contains(err.Error(), "is not allowed") {
		t.Errorf("Expected file type error, got: %v", err)
	}
}

func TestFileService_GetFile(t *testing.T) {
	fs, _, cleanup := setupTestFileService(t)
	defer cleanup()

	// Ensure directories exist
	err := fs.EnsureUploadDirectories()
	if err != nil {
		t.Fatalf("Failed to ensure directories: %v", err)
	}

	// Upload a test file first
	file, header := createTestMultipartFile(t, "test.txt", "Hello, World!", "text/plain")
	defer file.Close()

	uploadedFile, err := fs.UploadFile(file, header)
	if err != nil {
		t.Fatalf("Failed to upload test file: %v", err)
	}

	// Get the file by UUID
	retrievedFile, err := fs.GetFile(uploadedFile.UUID)
	if err != nil {
		t.Fatalf("GetFile failed: %v", err)
	}

	// Verify file details
	if retrievedFile.UUID != uploadedFile.UUID {
		t.Errorf("Expected UUID '%s', got '%s'", uploadedFile.UUID, retrievedFile.UUID)
	}

	if retrievedFile.OriginalName != uploadedFile.OriginalName {
		t.Errorf("Expected original name '%s', got '%s'", uploadedFile.OriginalName, retrievedFile.OriginalName)
	}
}

func TestFileService_GetFile_NotFound(t *testing.T) {
	fs, _, cleanup := setupTestFileService(t)
	defer cleanup()

	// Try to get non-existent file
	_, err := fs.GetFile("non-existent-uuid")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}

	if !strings.Contains(err.Error(), "file not found") {
		t.Errorf("Expected 'file not found' error, got: %v", err)
	}
}

func TestFileService_DeleteFile(t *testing.T) {
	fs, tempDir, cleanup := setupTestFileService(t)
	defer cleanup()

	// Ensure directories exist
	err := fs.EnsureUploadDirectories()
	if err != nil {
		t.Fatalf("Failed to ensure directories: %v", err)
	}

	// Upload a test file first
	file, header := createTestMultipartFile(t, "test.txt", "Hello, World!", "text/plain")
	defer file.Close()

	uploadedFile, err := fs.UploadFile(file, header)
	if err != nil {
		t.Fatalf("Failed to upload test file: %v", err)
	}

	// Verify file exists
	fullPath := filepath.Join(tempDir, uploadedFile.Path)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Error("File should exist before deletion")
	}

	// Delete the file
	err = fs.DeleteFile(uploadedFile.UUID)
	if err != nil {
		t.Fatalf("DeleteFile failed: %v", err)
	}

	// Verify file no longer exists on filesystem
	if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
		t.Error("File should not exist after deletion")
	}

	// Verify file record no longer exists in database
	_, err = fs.GetFile(uploadedFile.UUID)
	if err == nil {
		t.Error("File record should not exist after deletion")
	}
}

func TestFileService_ListFiles(t *testing.T) {
	fs, _, cleanup := setupTestFileService(t)
	defer cleanup()

	// Ensure directories exist
	err := fs.EnsureUploadDirectories()
	if err != nil {
		t.Fatalf("Failed to ensure directories: %v", err)
	}

	// Upload multiple test files
	testFiles := []string{"file1.txt", "file2.txt", "file3.txt"}
	for _, filename := range testFiles {
		file, header := createTestMultipartFile(t, filename, "test content", "text/plain")
		_, err := fs.UploadFile(file, header)
		file.Close()
		if err != nil {
			t.Fatalf("Failed to upload %s: %v", filename, err)
		}
	}

	// List files
	files, err := fs.ListFiles(10, 0)
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}

	if len(files) != 3 {
		t.Errorf("Expected 3 files, got %d", len(files))
	}

	// Test pagination
	files, err = fs.ListFiles(2, 0)
	if err != nil {
		t.Fatalf("ListFiles with limit failed: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("Expected 2 files with limit, got %d", len(files))
	}
}

func TestFileService_GetFilePath(t *testing.T) {
	fs, tempDir, cleanup := setupTestFileService(t)
	defer cleanup()

	// Ensure directories exist
	err := fs.EnsureUploadDirectories()
	if err != nil {
		t.Fatalf("Failed to ensure directories: %v", err)
	}

	// Upload a test file
	file, header := createTestMultipartFile(t, "test.txt", "Hello, World!", "text/plain")
	defer file.Close()

	uploadedFile, err := fs.UploadFile(file, header)
	if err != nil {
		t.Fatalf("Failed to upload test file: %v", err)
	}

	// Get file path
	filePath, err := fs.GetFilePath(uploadedFile.UUID)
	if err != nil {
		t.Fatalf("GetFilePath failed: %v", err)
	}

	expectedPath := filepath.Join(tempDir, uploadedFile.Path)
	if filePath != expectedPath {
		t.Errorf("Expected path '%s', got '%s'", expectedPath, filePath)
	}

	// Verify file exists at the path
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("File does not exist at returned path")
	}
}