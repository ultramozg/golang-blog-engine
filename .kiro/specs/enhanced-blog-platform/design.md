# Design Document

## Overview

This design enhances the existing Go blog platform with comprehensive testing, CI/CD automation, security updates, SEO-friendly URLs, and media management capabilities. The solution maintains the current SQLite-based architecture while adding new features through modular components.

## Architecture

### Current Architecture Analysis
- **Framework**: Native Go net/http with custom routing
- **Database**: SQLite with manual SQL queries
- **Authentication**: Session-based with GitHub OAuth integration
- **Templates**: Go html/template system
- **Static Files**: Direct file serving from public/ directory

### Enhanced Architecture Components

```mermaid
graph TB
    A[HTTP Router] --> B[Middleware Stack]
    B --> C[Handler Layer]
    C --> D[Service Layer]
    D --> E[Repository Layer]
    E --> F[SQLite Database]
    
    C --> G[File Storage Service]
    C --> H[Image Processing Service]
    C --> I[URL Slug Service]
    
    G --> J[File System Storage]
    H --> K[Image Optimization]
    I --> L[Slug Generation & Redirects]
    
    M[CI/CD Pipeline] --> N[Test Runner]
    M --> O[Security Scanner]
    M --> P[Deployment]
```

## Components and Interfaces

### 1. Testing Infrastructure

**Test Coverage Enhancement**
- Unit tests for all handlers, models, and utility functions
- Integration tests for database operations
- HTTP endpoint testing with proper mocking
- Test utilities for database setup/teardown

**Test Structure**
```go
// Test categories
- Handler tests (app/*_test.go)
- Model tests (model/*_test.go) 
- Service tests (services/*_test.go)
- Integration tests (tests/integration/*_test.go)
```

### 2. CI/CD Pipeline

**GitHub Actions Workflow**
- Automated testing on pull requests
- Go version matrix testing
- Code coverage reporting
- Security vulnerability scanning
- Automated dependency updates

### 3. URL Slug System

**Slug Generation Service**
```go
type SlugService interface {
    GenerateSlug(title string) string
    EnsureUniqueSlug(slug string, postID int) string
    GetPostBySlug(slug string) (*Post, error)
}
```

**URL Migration Strategy**
- Add `slug` column to posts table
- Generate slugs for existing posts
- Implement redirect middleware for old URLs
- Update all internal links to use slugs

### 4. File Management System

**File Storage Service**
```go
type FileService interface {
    UploadFile(file multipart.File, filename string) (*FileInfo, error)
    GetFile(fileID string) (*FileInfo, error)
    DeleteFile(fileID string) error
    ListFiles(userID string) ([]*FileInfo, error)
}

type FileInfo struct {
    ID          string
    OriginalName string
    StoredName   string
    Size        int64
    MimeType    string
    UploadedAt  time.Time
    DownloadCount int
}
```

**Storage Structure**
```
uploads/
├── files/
│   ├── 2024/01/
│   └── 2024/02/
└── images/
    ├── 2024/01/
    │   ├── originals/
    │   └── thumbnails/
    └── 2024/02/
```

### 5. Image Management System

**Image Processing Service**
```go
type ImageService interface {
    UploadImage(file multipart.File, filename string) (*ImageInfo, error)
    GenerateThumbnail(imageID string, width, height int) (*ImageInfo, error)
    OptimizeImage(imageID string) error
    GetImageURL(imageID string, size string) string
}

type ImageInfo struct {
    ID          string
    OriginalName string
    StoredName   string
    Width       int
    Height      int
    Size        int64
    MimeType    string
    AltText     string
    UploadedAt  time.Time
}
```

**Image Processing Features**
- Automatic WebP conversion for modern browsers
- Thumbnail generation (small: 150x150, medium: 300x300, large: 800x600)
- Image optimization without quality loss
- Responsive image serving

## Data Models

### Enhanced Post Model
```go
type Post struct {
    ID          int       `json:"id"`
    Title       string    `json:"title"`
    Slug        string    `json:"slug"`        // New field
    Body        string    `json:"body"`
    Date        string    `json:"date"`
    CreatedAt   time.Time `json:"created_at"`  // New field
    UpdatedAt   time.Time `json:"updated_at"`  // New field
}
```

### File Model
```go
type File struct {
    ID            int       `json:"id"`
    UUID          string    `json:"uuid"`
    OriginalName  string    `json:"original_name"`
    StoredName    string    `json:"stored_name"`
    Path          string    `json:"path"`
    Size          int64     `json:"size"`
    MimeType      string    `json:"mime_type"`
    DownloadCount int       `json:"download_count"`
    CreatedAt     time.Time `json:"created_at"`
}
```

### Image Model
```go
type Image struct {
    ID           int       `json:"id"`
    UUID         string    `json:"uuid"`
    OriginalName string    `json:"original_name"`
    StoredName   string    `json:"stored_name"`
    Path         string    `json:"path"`
    Width        int       `json:"width"`
    Height       int       `json:"height"`
    Size         int64     `json:"size"`
    MimeType     string    `json:"mime_type"`
    AltText      string    `json:"alt_text"`
    CreatedAt    time.Time `json:"created_at"`
}
```

### Database Schema Updates
```sql
-- Add slug column to existing posts table
ALTER TABLE posts ADD COLUMN slug TEXT UNIQUE;
ALTER TABLE posts ADD COLUMN created_at DATETIME DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE posts ADD COLUMN updated_at DATETIME DEFAULT CURRENT_TIMESTAMP;

-- Create files table
CREATE TABLE files (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT UNIQUE NOT NULL,
    original_name TEXT NOT NULL,
    stored_name TEXT NOT NULL,
    path TEXT NOT NULL,
    size INTEGER NOT NULL,
    mime_type TEXT NOT NULL,
    download_count INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create images table
CREATE TABLE images (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT UNIQUE NOT NULL,
    original_name TEXT NOT NULL,
    stored_name TEXT NOT NULL,
    path TEXT NOT NULL,
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,
    size INTEGER NOT NULL,
    mime_type TEXT NOT NULL,
    alt_text TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## Error Handling

### Structured Error Response
```go
type APIError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
}
```

### Error Categories
- **Validation Errors**: Invalid input data (400)
- **Authentication Errors**: Unauthorized access (401)
- **Authorization Errors**: Insufficient permissions (403)
- **Not Found Errors**: Resource not found (404)
- **File Errors**: Upload/processing failures (422)
- **Server Errors**: Internal processing errors (500)

### File Upload Error Handling
- File size limits (configurable, default 10MB for files, 5MB for images)
- MIME type validation
- Storage space checks
- Image processing failures
- Malicious file detection

## Testing Strategy

### Unit Testing
- **Coverage Target**: 80% minimum
- **Test Categories**:
  - Handler functions with mocked dependencies
  - Model methods with test database
  - Service layer business logic
  - Utility functions and helpers

### Integration Testing
- Database operations with real SQLite instance
- File upload and processing workflows
- Authentication and authorization flows
- URL slug generation and redirect logic

### End-to-End Testing
- Complete user workflows (create post, upload image, etc.)
- Cross-browser compatibility for file uploads
- Performance testing for image processing

### Test Database Strategy
- Separate test database for each test suite
- Database migrations testing
- Data seeding for consistent test scenarios
- Cleanup procedures for test isolation

## Security Considerations

### Dependency Updates
- Update all vulnerable packages identified in go.mod
- Implement automated dependency scanning
- Regular security audit schedule

### File Upload Security
- MIME type validation and file signature checking
- File size limits and storage quotas
- Virus scanning for uploaded files
- Secure file naming to prevent path traversal
- Content-Security-Policy headers for image serving

### URL Security
- Slug validation to prevent XSS
- Proper URL encoding and sanitization
- Rate limiting for slug generation
- Redirect validation to prevent open redirects

## Performance Considerations

### Image Optimization
- Lazy loading for images in blog posts
- WebP format support with fallbacks
- CDN-ready file serving headers
- Thumbnail caching strategy

### Database Performance
- Indexes on slug column for fast lookups
- Query optimization for file listings
- Connection pooling configuration
- Database backup and recovery procedures

### Caching Strategy
- HTTP caching headers for static assets
- In-memory caching for frequently accessed slugs
- File metadata caching
- Template caching optimization