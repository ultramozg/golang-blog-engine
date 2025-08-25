package services

import (
	"database/sql"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/satori/go.uuid"
	"github.com/ultramozg/golang-blog-engine/model"
)

// validatePath ensures the path is safe and within the expected directory
func validatePath(basePath, targetPath string) error {
	// Clean and resolve the paths
	cleanBase := filepath.Clean(basePath)
	cleanTarget := filepath.Clean(targetPath)

	// Get absolute paths
	absBase, err := filepath.Abs(cleanBase)
	if err != nil {
		return fmt.Errorf("failed to get absolute base path: %w", err)
	}

	absTarget, err := filepath.Abs(cleanTarget)
	if err != nil {
		return fmt.Errorf("failed to get absolute target path: %w", err)
	}

	// Check if target is within base directory
	relPath, err := filepath.Rel(absBase, absTarget)
	if err != nil {
		return fmt.Errorf("failed to get relative path: %w", err)
	}

	// Ensure the relative path doesn't contain ".." (directory traversal)
	if strings.Contains(relPath, "..") {
		return errors.New("path traversal detected")
	}

	return nil
}

// FileService interface defines the contract for file operations
type FileService interface {
	UploadFile(file multipart.File, header *multipart.FileHeader) (*model.File, error)
	GetFile(fileUUID string) (*model.File, error)
	DeleteFile(fileUUID string) error
	ListFiles(limit, offset int) ([]model.File, error)
	GetFilePath(fileUUID string) (string, error)
	EnsureUploadDirectories() error
	IsImageFile(mimeType string) bool
	ProcessImage(fileRecord *model.File) error
	GenerateThumbnail(fileRecord *model.File) error
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
	if err := os.MkdirAll(fs.uploadDir, 0750); err != nil {
		return fmt.Errorf("failed to create upload directory: %w", err)
	}

	// Create files subdirectory
	filesDir := filepath.Join(fs.uploadDir, "files")
	if err := os.MkdirAll(filesDir, 0750); err != nil {
		return fmt.Errorf("failed to create files directory: %w", err)
	}

	// Create year/month subdirectories for current date
	now := time.Now()
	yearMonth := fmt.Sprintf("%d/%02d", now.Year(), now.Month())

	// Create documents subdirectory
	documentsDir := filepath.Join(filesDir, yearMonth, "documents")
	if err := os.MkdirAll(documentsDir, 0750); err != nil {
		return fmt.Errorf("failed to create documents directory: %w", err)
	}

	// Create images subdirectory
	imagesDir := filepath.Join(filesDir, yearMonth, "images")
	if err := os.MkdirAll(imagesDir, 0750); err != nil {
		return fmt.Errorf("failed to create images directory: %w", err)
	}

	// Create thumbnails subdirectory
	thumbnailsDir := filepath.Join(filesDir, yearMonth, "thumbnails")
	if err := os.MkdirAll(thumbnailsDir, 0750); err != nil {
		return fmt.Errorf("failed to create thumbnails directory: %w", err)
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

	// Determine if this is an image file
	isImage := fs.IsImageFile(header.Header.Get("Content-Type"))

	// Create year/month directory structure
	now := time.Now()
	yearMonth := fmt.Sprintf("%d/%02d", now.Year(), now.Month())

	// Choose subdirectory based on file type
	var subDir string
	if isImage {
		subDir = "images"
	} else {
		subDir = "documents"
	}

	targetDir := filepath.Join(fs.uploadDir, "files", yearMonth, subDir)

	// Ensure directory exists
	if err := os.MkdirAll(targetDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Full file path
	filePath := filepath.Join(targetDir, storedName)
	relativePath := filepath.Join("files", yearMonth, subDir, storedName)

	// Validate file path to prevent directory traversal
	if err := validatePath(fs.uploadDir, filePath); err != nil {
		return nil, fmt.Errorf("invalid file path: %w", err)
	}

	// Create the file
	dst, err := os.Create(filePath) // #nosec G304 - Path is validated above
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	// Copy file content
	_, err = io.Copy(dst, file)
	if err != nil {
		// Clean up the file if copy failed
		if removeErr := os.Remove(filePath); removeErr != nil {
			log.Printf("Failed to remove file after copy error: %v", removeErr)
		}
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
		IsImage:       isImage,
	}

	// Process image if it's an image file
	if isImage {
		if err := fs.ProcessImage(fileRecord); err != nil {
			// Clean up the file if image processing failed
			if removeErr := os.Remove(filePath); removeErr != nil {
				log.Printf("Failed to remove file after image processing error: %v", removeErr)
			}
			return nil, fmt.Errorf("failed to process image: %w", err)
		}
	}

	if err := fileRecord.CreateFile(fs.db); err != nil {
		// Clean up the file and thumbnail if database insert failed
		if removeErr := os.Remove(filePath); removeErr != nil {
			log.Printf("Failed to remove file after database error: %v", removeErr)
		}
		if fileRecord.ThumbnailPath != nil {
			if removeErr := os.Remove(filepath.Join(fs.uploadDir, *fileRecord.ThumbnailPath)); removeErr != nil {
				log.Printf("Failed to remove thumbnail after database error: %v", removeErr)
			}
		}
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

	// Delete thumbnail if it exists
	if file.ThumbnailPath != nil && *file.ThumbnailPath != "" {
		thumbnailPath := filepath.Join(fs.uploadDir, *file.ThumbnailPath)
		if err := os.Remove(thumbnailPath); err != nil && !os.IsNotExist(err) {
			// Log warning but don't fail the operation
			fmt.Printf("Warning: failed to delete thumbnail %s: %v\n", thumbnailPath, err)
		}
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
		// Document types
		"application/pdf":    true,
		"application/msword": true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
		"application/vnd.ms-excel": true,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         true,
		"application/vnd.ms-powerpoint":                                             true,
		"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
		"text/plain":                   true,
		"text/csv":                     true,
		"application/zip":              true,
		"application/x-zip-compressed": true,
		"application/json":             true,
		"application/xml":              true,
		"text/xml":                     true,
		"application/rtf":              true,
		"application/x-tar":            true,
		"application/gzip":             true,
		"application/x-rar-compressed": true,
		"application/x-7z-compressed":  true,
		// Image types
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
		"image/bmp":  true,
		"image/tiff": true,
	}

	// If no content type is provided, we'll allow it but it will be treated as binary
	if mimeType == "" {
		return true
	}

	return allowedTypes[mimeType]
}

// IsImageFile checks if the MIME type represents an image
func (fs *FileServiceImpl) IsImageFile(mimeType string) bool {
	imageTypes := map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
		"image/bmp":  true,
		"image/tiff": true,
	}
	return imageTypes[mimeType]
}

// ProcessImage extracts image metadata and generates thumbnail
func (fs *FileServiceImpl) ProcessImage(fileRecord *model.File) error {
	if !fs.IsImageFile(fileRecord.MimeType) {
		return fmt.Errorf("file is not an image")
	}

	// Get full file path
	fullPath := filepath.Join(fs.uploadDir, fileRecord.Path)

	// Validate file path to prevent directory traversal
	if err := validatePath(fs.uploadDir, fullPath); err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}

	// Open the image file
	file, err := os.Open(fullPath) // #nosec G304 - Path is validated above
	if err != nil {
		return fmt.Errorf("failed to open image file: %w", err)
	}
	defer file.Close()

	// Decode image to get dimensions
	img, _, err := image.Decode(file)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	// Get image dimensions
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Update file record with image metadata
	fileRecord.IsImage = true
	fileRecord.Width = &width
	fileRecord.Height = &height

	// Generate thumbnail
	if err := fs.GenerateThumbnail(fileRecord); err != nil {
		return fmt.Errorf("failed to generate thumbnail: %w", err)
	}

	return nil
}

// GenerateThumbnail creates a thumbnail for the image
func (fs *FileServiceImpl) GenerateThumbnail(fileRecord *model.File) error {
	if !fileRecord.IsImage {
		return fmt.Errorf("file is not an image")
	}

	// Get full file path
	fullPath := filepath.Join(fs.uploadDir, fileRecord.Path)

	// Validate file path to prevent directory traversal
	if err := validatePath(fs.uploadDir, fullPath); err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}

	// Open the original image
	file, err := os.Open(fullPath) // #nosec G304 - Path is validated above
	if err != nil {
		return fmt.Errorf("failed to open image file: %w", err)
	}
	defer file.Close()

	// Decode the image
	img, format, err := image.Decode(file)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	// Calculate thumbnail dimensions (300x300 max, maintaining aspect ratio)
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	thumbnailSize := 300
	var newWidth, newHeight int

	if width > height {
		newWidth = thumbnailSize
		newHeight = (height * thumbnailSize) / width
	} else {
		newHeight = thumbnailSize
		newWidth = (width * thumbnailSize) / height
	}

	// Create thumbnail using simple nearest neighbor scaling
	thumbnail := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	// Simple scaling algorithm
	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			srcX := (x * width) / newWidth
			srcY := (y * height) / newHeight
			thumbnail.Set(x, y, img.At(srcX, srcY))
		}
	}

	// Generate thumbnail filename and path
	ext := filepath.Ext(fileRecord.StoredName)
	thumbnailName := strings.TrimSuffix(fileRecord.StoredName, ext) + "_thumb" + ext

	// Extract year/month from original path
	pathParts := strings.Split(fileRecord.Path, string(filepath.Separator))
	if len(pathParts) < 4 { // files/YYYY/MM/subdir/filename
		return fmt.Errorf("invalid file path structure")
	}

	yearMonth := filepath.Join(pathParts[1], pathParts[2]) // YYYY/MM
	thumbnailRelativePath := filepath.Join("files", yearMonth, "thumbnails", thumbnailName)
	thumbnailFullPath := filepath.Join(fs.uploadDir, thumbnailRelativePath)

	// Ensure thumbnails directory exists
	thumbnailDir := filepath.Dir(thumbnailFullPath)
	if err := os.MkdirAll(thumbnailDir, 0750); err != nil {
		return fmt.Errorf("failed to create thumbnail directory: %w", err)
	}

	// Validate thumbnail path to prevent directory traversal
	if err := validatePath(fs.uploadDir, thumbnailFullPath); err != nil {
		return fmt.Errorf("invalid thumbnail path: %w", err)
	}

	// Create thumbnail file
	thumbnailFile, err := os.Create(thumbnailFullPath) // #nosec G304 - Path is validated above
	if err != nil {
		return fmt.Errorf("failed to create thumbnail file: %w", err)
	}
	defer thumbnailFile.Close()

	// Encode thumbnail based on original format
	switch format {
	case "jpeg", "jpg":
		err = jpeg.Encode(thumbnailFile, thumbnail, &jpeg.Options{Quality: 85})
	case "png":
		err = png.Encode(thumbnailFile, thumbnail)
	default:
		// Default to JPEG for other formats
		err = jpeg.Encode(thumbnailFile, thumbnail, &jpeg.Options{Quality: 85})
	}

	if err != nil {
		return fmt.Errorf("failed to encode thumbnail: %w", err)
	}

	// Update file record with thumbnail path
	fileRecord.ThumbnailPath = &thumbnailRelativePath

	return nil
}
