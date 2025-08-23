package services

import (
	"database/sql"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/satori/go.uuid"
	"github.com/ultramozg/golang-blog-engine/model"
)

// FileService interface defines the contract for file operations
type FileService interface {
	UploadFile(file multipart.File, header *multipart.FileHeader) (*model.File, error)
	GetFile(fileUUID string) (*model.File, error)
	DeleteFile(fileUUID string) error
	ListFiles(limit, offset int) ([]model.File, error)
	GetFilePath(fileUUID string) (string, error)
	EnsureUploadDirectories() error
}

// FileServiceImpl implements the FileService interface
type FileServiceImpl struct {
	db          *sql.DB
	uploadDir   string
	maxFileSize int64
}

// NewFileService creates a new FileService instance
func NewFileService(db *sql.DB, uploadDir string, maxFileSize int64) FileService {
	return &FileServiceImpl{
		db:          db,
		uploadDir:   uploadDir,
		maxFileSize: maxFileSize,
	}
}

// EnsureUploadDirectories creates the necessary directory structure for file uploads
func (fs *FileServiceImpl) EnsureUploadDirectories() error {
	// Create base upload directory
	if err := os.MkdirAll(fs.uploadDir, 0755); err != nil {
		return fmt.Errorf("failed to create upload directory: %w", err)
	}

	// Create files subdirectory
	filesDir := filepath.Join(fs.uploadDir, "files")
	if err := os.MkdirAll(filesDir, 0755); err != nil {
		return fmt.Errorf("failed to create files directory: %w", err)
	}

	// Create year/month subdirectories for current date
	now := time.Now()
	yearMonth := fmt.Sprintf("%d/%02d", now.Year(), now.Month())
	monthDir := filepath.Join(filesDir, yearMonth)
	if err := os.MkdirAll(monthDir, 0755); err != nil {
		return fmt.Errorf("failed to create month directory: %w", err)
	}

	return nil
}

// UploadFile handles file upload with validation and secure storage
func (fs *FileServiceImpl) UploadFile(file multipart.File, header *multipart.FileHeader) (*model.File, error) {
	// Validate file size
	if header.Size > fs.maxFileSize {
		return nil, fmt.Errorf("file size %d exceeds maximum allowed size %d", header.Size, fs.maxFileSize)
	}

	// Validate file type (basic MIME type check)
	if !fs.isAllowedFileType(header.Header.Get("Content-Type")) {
		return nil, fmt.Errorf("file type %s is not allowed", header.Header.Get("Content-Type"))
	}

	// Generate UUID for secure file naming
	fileUUID := uuid.NewV4().String()
	
	// Generate secure stored filename
	storedName := fs.generateSecureFilename(fileUUID, header.Filename)
	
	// Create year/month directory structure
	now := time.Now()
	yearMonth := fmt.Sprintf("%d/%02d", now.Year(), now.Month())
	monthDir := filepath.Join(fs.uploadDir, "files", yearMonth)
	
	// Ensure directory exists
	if err := os.MkdirAll(monthDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Full file path
	filePath := filepath.Join(monthDir, storedName)
	relativePath := filepath.Join("files", yearMonth, storedName)

	// Create the file
	dst, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	// Copy file content
	_, err = io.Copy(dst, file)
	if err != nil {
		// Clean up the file if copy failed
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}

	// Create file record in database
	fileRecord := &model.File{
		UUID:          fileUUID,
		OriginalName:  header.Filename,
		StoredName:    storedName,
		Path:          relativePath,
		Size:          header.Size,
		MimeType:      header.Header.Get("Content-Type"),
		DownloadCount: 0,
	}

	if err := fileRecord.CreateFile(fs.db); err != nil {
		// Clean up the file if database insert failed
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to save file record: %w", err)
	}

	return fileRecord, nil
}

// GetFile retrieves file information by UUID
func (fs *FileServiceImpl) GetFile(fileUUID string) (*model.File, error) {
	file := &model.File{UUID: fileUUID}
	if err := file.GetFileByUUID(fs.db); err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}
	return file, nil
}

// DeleteFile removes a file from both filesystem and database
func (fs *FileServiceImpl) DeleteFile(fileUUID string) error {
	// Get file record first
	file, err := fs.GetFile(fileUUID)
	if err != nil {
		return err
	}

	// Delete from filesystem
	fullPath := filepath.Join(fs.uploadDir, file.Path)
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file from filesystem: %w", err)
	}

	// Delete from database
	if err := file.DeleteFile(fs.db); err != nil {
		return fmt.Errorf("failed to delete file record: %w", err)
	}

	return nil
}

// ListFiles returns a paginated list of files
func (fs *FileServiceImpl) ListFiles(limit, offset int) ([]model.File, error) {
	return model.GetFiles(fs.db, limit, offset)
}

// GetFilePath returns the full filesystem path for a file UUID
func (fs *FileServiceImpl) GetFilePath(fileUUID string) (string, error) {
	file, err := fs.GetFile(fileUUID)
	if err != nil {
		return "", err
	}
	return filepath.Join(fs.uploadDir, file.Path), nil
}

// generateSecureFilename creates a secure filename using UUID and original extension
func (fs *FileServiceImpl) generateSecureFilename(fileUUID, originalName string) string {
	ext := filepath.Ext(originalName)
	// Sanitize extension
	ext = strings.ToLower(ext)
	if ext == "" {
		ext = ".bin" // Default extension for files without extension
	}
	return fileUUID + ext
}

// isAllowedFileType checks if the MIME type is allowed for upload
func (fs *FileServiceImpl) isAllowedFileType(mimeType string) bool {
	allowedTypes := map[string]bool{
		"application/pdf":                          true,
		"application/msword":                       true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
		"application/vnd.ms-excel":                 true,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":       true,
		"application/vnd.ms-powerpoint":            true,
		"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
		"text/plain":                               true,
		"text/csv":                                 true,
		"application/zip":                          true,
		"application/x-zip-compressed":             true,
		"application/json":                         true,
		"application/xml":                          true,
		"text/xml":                                 true,
		"application/rtf":                          true,
		"application/x-tar":                        true,
		"application/gzip":                         true,
		"application/x-rar-compressed":             true,
		"application/x-7z-compressed":              true,
	}

	// If no content type is provided, we'll allow it but it will be treated as binary
	if mimeType == "" {
		return true
	}

	return allowedTypes[mimeType]
}