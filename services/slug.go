package services

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// SlugService interface defines the contract for slug generation and management
type SlugService interface {
	GenerateSlug(title string) string
	EnsureUniqueSlug(slug string, postID int) string
	IsSlugUnique(slug string, excludePostID int) bool
	SanitizeTitle(title string) string
}

// slugService implements the SlugService interface
type slugService struct {
	db *sql.DB
}

// NewSlugService creates a new instance of SlugService
func NewSlugService(db *sql.DB) SlugService {
	return &slugService{
		db: db,
	}
}

// GenerateSlug creates a URL-safe slug from a title
func (s *slugService) GenerateSlug(title string) string {
	if title == "" {
		return ""
	}

	// Sanitize the title
	slug := s.SanitizeTitle(title)

	// Ensure it's not empty after sanitization
	if slug == "" {
		return "untitled"
	}

	return slug
}

// SanitizeTitle converts a title into a URL-safe slug format
func (s *slugService) SanitizeTitle(title string) string {
	// Convert to lowercase
	slug := strings.ToLower(title)

	// Remove accents and normalize unicode characters
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	slug, _, _ = transform.String(t, slug)

	// Replace spaces and underscores with hyphens
	slug = regexp.MustCompile(`[\s_]+`).ReplaceAllString(slug, "-")

	// Remove all non-alphanumeric characters except hyphens
	slug = regexp.MustCompile(`[^a-z0-9\-]`).ReplaceAllString(slug, "")

	// Replace multiple consecutive hyphens with single hyphen
	slug = regexp.MustCompile(`-+`).ReplaceAllString(slug, "-")

	// Remove leading and trailing hyphens
	slug = strings.Trim(slug, "-")

	// Limit length to 100 characters
	if len(slug) > 100 {
		slug = slug[:100]
		// Remove trailing hyphen if truncation created one
		slug = strings.TrimRight(slug, "-")
	}

	return slug
}

// EnsureUniqueSlug ensures the slug is unique by appending a number if necessary
func (s *slugService) EnsureUniqueSlug(slug string, postID int) string {
	if slug == "" {
		slug = "untitled"
	}

	originalSlug := slug
	counter := 1

	// Check if the slug is unique
	for !s.IsSlugUnique(slug, postID) {
		slug = fmt.Sprintf("%s-%d", originalSlug, counter)
		counter++

		// Prevent infinite loop by limiting attempts
		if counter > 1000 {
			break
		}
	}

	return slug
}

// IsSlugUnique checks if a slug is unique in the database
func (s *slugService) IsSlugUnique(slug string, excludePostID int) bool {
	var count int
	var err error

	if excludePostID > 0 {
		// When updating a post, exclude the current post from uniqueness check
		err = s.db.QueryRow("SELECT COUNT(*) FROM posts WHERE slug = ? AND id != ?", slug, excludePostID).Scan(&count)
	} else {
		// When creating a new post, check all posts
		err = s.db.QueryRow("SELECT COUNT(*) FROM posts WHERE slug = ?", slug).Scan(&count)
	}

	if err != nil {
		// If there's an error (like column doesn't exist yet), assume not unique for safety
		return false
	}

	return count == 0
}
