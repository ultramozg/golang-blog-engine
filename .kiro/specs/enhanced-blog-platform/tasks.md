# Implementation Plan

- [ ] 1. Set up comprehensive testing infrastructure
  - Create test utilities for database setup and teardown
  - Implement test helpers for HTTP request/response testing
  - Add test configuration management
  - _Requirements: 1.1, 1.2, 1.3, 1.4_

- [ ] 2. Expand unit test coverage for existing functionality
- [ ] 2.1 Create comprehensive model tests
  - Write unit tests for all Post model methods (GetPost, UpdatePost, DeletePost, CreatePost)
  - Write unit tests for Comment model methods (CreateComment, DeleteComment)
  - Write unit tests for User model methods (IsUserExist, CreateUser, IsAdmin, CheckCredentials)
  - Test edge cases and error conditions for all model operations
  - _Requirements: 1.1, 1.2, 1.3_

- [x] 2.2 Create comprehensive handler tests
  - Write unit tests for all HTTP handlers with proper mocking
  - Test all HTTP methods and response codes for each endpoint
  - Test authentication and authorization middleware
  - Test error handling and edge cases for all handlers
  - _Requirements: 1.1, 1.2, 1.3_

- [x] 2.3 Create utility and helper function tests
  - Write tests for HashPassword function
  - Write tests for configuration loading and environment variable handling
  - Write tests for template parsing and rendering
  - Write tests for session management functions
  - _Requirements: 1.1, 1.2, 1.3_

- [x] 3. Implement GitHub Actions CI/CD workflow
  - Create .github/workflows/test.yml with Go testing pipeline
  - Configure test matrix for multiple Go versions
  - Add code coverage reporting and badge generation
  - Set up automated security vulnerability scanning
  - Configure branch protection rules requiring tests to pass
  - _Requirements: 2.1, 2.2, 2.3, 2.4_

- [x] 4. Update vulnerable dependencies
  - Update golang.org/x/crypto to latest secure version
  - Update golang.org/x/oauth2 to latest version
  - Update github.com/mattn/go-sqlite3 to non-retracted version
  - Update all other dependencies to latest stable versions
  - Test application functionality after dependency updates
  - _Requirements: 3.1, 3.2, 3.3, 3.4_

- [x] 5. Implement URL slug system foundation
- [x] 5.1 Create slug generation service
  - Implement SlugService interface with slug generation logic
  - Create function to sanitize titles and generate URL-safe slugs
  - Implement unique slug generation with conflict resolution
  - Write comprehensive unit tests for slug generation
  - _Requirements: 4.1, 4.4_

- [x] 5.2 Update database schema for slugs
  - Create database migration to add slug column to posts table
  - Add created_at and updated_at columns to posts table
  - Generate slugs for all existing posts in database
  - Create database indexes for slug column performance
  - _Requirements: 4.1, 4.2_

- [x] 5.3 Implement slug-based routing
  - Update Post model to include slug field and methods
  - Create GetPostBySlug method in Post model
  - Implement slug-based URL handlers alongside existing ID-based handlers
  - Update post creation and update logic to generate/update slugs
  - _Requirements: 4.1, 4.5_

- [x] 5.4 Implement URL redirect system
  - Create middleware to handle redirects from old ID-based URLs to slug URLs
  - Implement 301 permanent redirects for SEO preservation
  - Update all internal links in templates to use slug URLs
  - Test redirect functionality for existing and new posts
  - _Requirements: 4.3, 4.5_

- [x] 6. Implement direct file attachment system for posts
- [x] 6.1 Create file storage infrastructure
  - Create File model with database schema for file metadata
  - Implement file storage directory structure creation
  - Create FileService interface and implementation for file operations
  - Implement secure file naming and path generation with UUID
  - _Requirements: 5.1, 5.2_

- [x] 6.2 Implement file upload handlers for post integration
  - Create file upload endpoint with multipart form handling for post editor
  - Implement file validation (size, type, security checks)
  - Create file metadata storage in database
  - Implement file serving endpoint with proper headers and content types
  - _Requirements: 5.1, 5.3, 5.4_

- [x] 6.3 Integrate file upload directly into post creation/editing interface
  - Add drag-and-drop file upload zone to post creation and editing forms
  - Implement real-time file upload with progress feedback
  - Create file reference system using [file:filename] syntax in post content
  - Add buttons to insert file references and direct links into post content
  - Process file references in post display to show download links
  - _Requirements: 5.5, 5.6_

- [x] 7. Extend file upload system to support automatic image processing
- [x] 7.1 Extend file storage infrastructure for images
  - Add image-specific fields to existing File model (is_image, width, height, thumbnail_path, alt_text)
  - Update database schema to extend files table with image metadata columns
  - Modify file storage directory structure to include images and thumbnails subdirectories
  - Add image processing libraries (image/jpeg, image/png) for thumbnail generation
  - _Requirements: 6.1, 6.2_

- [x] 7.2 Implement automatic image detection and processing in file upload
  - Extend existing file upload handler to detect image MIME types automatically
  - Implement automatic image processing (thumbnail generation) when image files are uploaded
  - Add image metadata extraction (width, height) during upload process
  - Update FileService to handle image-specific operations within existing file upload flow
  - _Requirements: 6.1, 6.2, 6.6_

- [x] 7.3 Implement automatic image insertion into post content
  - Modify file upload endpoint to automatically insert image references into post content during upload
  - Create image embedding syntax that works with existing file reference system
  - Update post content processing to render image references as responsive images
  - Ensure multiple images can be attached and automatically inserted into single post
  - _Requirements: 6.3, 6.4, 6.5_

- [x] 7.4 Implement image rendering and accessibility in blog posts
  - Update blog post templates to render image references as responsive images with thumbnails
  - Add alt text support for images within the existing file management system
  - Implement responsive image display that works with existing blog post styling
  - Test image rendering across different screen sizes and devices
  - _Requirements: 6.5, 6.7_

- [x] 8. Create integration tests for extended file and image functionality
  - Write integration tests for file upload workflows including automatic image processing
  - Create integration tests for automatic image insertion into post content
  - Test complete user workflows from image upload to display in blog posts
  - Test multiple image attachments to single post functionality
  - _Requirements: 1.1, 1.2, 1.3, 1.4_

- [x] 9. Update templates and UI for enhanced file and image features
  - Update post creation/editing forms to support automatic image insertion during upload
  - Enhance existing file upload interface to handle image processing seamlessly
  - Update blog post templates to render both file downloads and embedded images responsively
  - Add visual feedback for automatic image insertion into post content
  - _Requirements: 5.5, 6.3, 6.4, 6.5_

- [x] 10. Implement error handling and validation for extended functionality
  - Extend existing file operation error handling to cover image processing failures
  - Add validation for automatic image insertion and thumbnail generation
  - Implement user-friendly error messages for image processing and insertion failures
  - Add logging for image processing operations and automatic content insertion
  - _Requirements: 5.4, 6.7_

- [x] 11. Performance optimization and security hardening for extended file system
  - Implement file size limits with specific limits for image files
  - Add rate limiting for upload endpoints including image processing operations
  - Create security headers for serving both files and images
  - Optimize database queries with proper indexing for extended files table with image fields
  - _Requirements: 3.4, 5.4, 6.7_

- [ ] 12. Remove courses and links sections from the application
  - Remove courses and links handlers from app.go routing
  - Delete courses.gohtml and links.gohtml templates
  - Remove courses.yml and links.yml data files from all directories
  - Update header.gohtml navigation to remove courses and links menu items
  - Update any references to courses or links in other templates
  - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5_

- [ ] 13. Implement comprehensive SEO optimization system
- [ ] 13.1 Create SEO service for meta tags and structured data
  - Implement SEOService interface with meta tag generation methods
  - Create functions to generate JSON-LD structured data for blog posts
  - Implement Open Graph tag generation for social media sharing
  - Add canonical URL generation logic to prevent duplicate content issues
  - _Requirements: 7.1, 7.3, 7.4, 7.10_

- [ ] 13.2 Implement sitemap and robots.txt generation
  - Create sitemap.xml generation endpoint with canonical URLs only
  - Implement robots.txt serving with proper crawling instructions
  - Add automatic sitemap updates when posts are created/updated/deleted
  - Ensure sitemap includes proper lastmod dates for posts
  - _Requirements: 7.6, 7.7, 7.9_

- [ ] 13.3 Integrate SEO components into post templates and handlers
  - Update post templates to include meta tags, canonical URLs, and structured data
  - Modify post handlers to generate and serve SEO metadata
  - Implement canonical URL headers in all post responses
  - Add image alt text and structured data for posts with images
  - _Requirements: 7.1, 7.2, 7.8, 7.10_

- [ ] 13.4 Enhance URL redirect system for SEO compliance
  - Update existing redirect middleware to include canonical URL headers
  - Ensure all redirects from ID-based URLs to slug URLs use 301 status codes
  - Implement canonical URL validation and sanitization
  - Test that only canonical URLs appear in sitemap and search results
  - _Requirements: 7.2, 7.10_

- [ ] 14. Update database schema for SEO fields
  - Add meta_description and keywords columns to posts table
  - Create database migration for SEO field additions
  - Update Post model to include SEO fields
  - Implement SEO field validation and sanitization
  - _Requirements: 7.3, 7.8_

- [ ] 15. Create comprehensive tests for SEO and content removal features
  - Write unit tests for SEO service methods (meta tags, structured data, sitemap)
  - Create integration tests for canonical URL redirects and SEO headers
  - Test that courses and links sections are completely removed and return 404
  - Write tests for sitemap generation and automatic updates
  - _Requirements: 1.1, 1.2, 1.3, 1.4_

- [ ] 16. Final integration and testing
  - Run complete test suite and ensure all tests pass
  - Perform manual testing of all new features including SEO and content removal
  - Test backward compatibility with existing functionality
  - Verify CI/CD pipeline works correctly with all changes
  - Test Google Search Console compatibility and canonical URL resolution
  - _Requirements: 1.4, 2.4_