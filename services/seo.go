package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/ultramozg/golang-blog-engine/model"
)

// SEOService interface defines methods for SEO optimization
type SEOService interface {
	GenerateMetaTags(post *model.Post) map[string]string
	GenerateStructuredData(post *model.Post) string
	GenerateOpenGraphTags(post *model.Post) map[string]string
	GenerateSitemap(posts []*model.Post) ([]byte, error)
	GenerateRobotsTxt() string
	GetCanonicalURL(post *model.Post) string
}

// seoService implements SEOService interface
type seoService struct {
	db      *sql.DB
	baseURL string
}

// NewSEOService creates a new SEO service instance
func NewSEOService(db *sql.DB, baseURL string) SEOService {
	return &seoService{
		db:      db,
		baseURL: strings.TrimSuffix(baseURL, "/"),
	}
}

// GenerateMetaTags generates meta tags for a blog post
func (s *seoService) GenerateMetaTags(post *model.Post) map[string]string {
	tags := make(map[string]string)

	// Title tag
	tags["title"] = html.EscapeString(post.Title)

	// Meta description - use post's meta_description field first, then extract from body
	var description string
	if post.MetaDescription != "" && post.MetaDescription != "Read this blog post to learn more about the topic." && len(strings.TrimSpace(post.MetaDescription)) > 10 {
		description = post.MetaDescription
	} else {
		description = s.extractDescription(post.Body)
	}
	// Ensure we always have a meaningful description
	if description == "" || description == "Read this blog post to learn more about the topic." || len(strings.TrimSpace(description)) < 10 {
		description = s.extractDescription(post.Body)
		if description == "" || len(strings.TrimSpace(description)) < 10 {
			if post.Title != "" {
				description = "Read this blog post about " + post.Title + " to learn more about the topic and get insights."
			} else {
				description = "Explore this blog post to discover interesting content and valuable insights."
			}
		}
	}
	tags["description"] = html.EscapeString(description)

	// Keywords - use post's keywords field first, then extract from content
	var keywords string
	if post.Keywords != "" {
		keywords = post.Keywords
	} else {
		keywords = s.extractKeywords(post.Title, post.Body)
	}
	if keywords != "" {
		tags["keywords"] = html.EscapeString(keywords)
	}

	// Canonical URL
	tags["canonical"] = s.GetCanonicalURL(post)

	// Author
	tags["author"] = "Blog Author"

	// Article published time
	if post.CreatedAt != "" {
		tags["article:published_time"] = post.CreatedAt
	}

	// Article modified time
	if post.UpdatedAt != "" {
		tags["article:modified_time"] = post.UpdatedAt
	}

	return tags
}

// GenerateStructuredData generates JSON-LD structured data for a blog post
func (s *seoService) GenerateStructuredData(post *model.Post) string {
	structuredData := map[string]interface{}{
		"@context": "https://schema.org",
		"@type":    "BlogPosting",
		"headline": post.Title,
		"url":      s.GetCanonicalURL(post),
		"author": map[string]interface{}{
			"@type": "Person",
			"name":  "Blog Author",
		},
		"publisher": map[string]interface{}{
			"@type": "Organization",
			"name":  "Blog",
			"url":   s.baseURL,
		},
		"description": s.getPostDescription(post),
	}

	// Add dates if available
	if post.CreatedAt != "" {
		structuredData["datePublished"] = post.CreatedAt
	}
	if post.UpdatedAt != "" {
		structuredData["dateModified"] = post.UpdatedAt
	}

	// Add main entity of page
	structuredData["mainEntityOfPage"] = map[string]interface{}{
		"@type": "WebPage",
		"@id":   s.GetCanonicalURL(post),
	}

	// Check for images in the post content
	images := s.extractImages(post.Body)
	if len(images) > 0 {
		if len(images) == 1 {
			structuredData["image"] = images[0]
		} else {
			structuredData["image"] = images
		}
	}

	jsonData, err := json.MarshalIndent(structuredData, "", "  ")
	if err != nil {
		return ""
	}

	return string(jsonData)
}

// GenerateOpenGraphTags generates Open Graph tags for social media sharing
func (s *seoService) GenerateOpenGraphTags(post *model.Post) map[string]string {
	tags := make(map[string]string)

	tags["og:type"] = "article"
	tags["og:title"] = html.EscapeString(post.Title)
	tags["og:description"] = html.EscapeString(s.getPostDescription(post))
	tags["og:url"] = s.GetCanonicalURL(post)
	tags["og:site_name"] = "Blog"

	// Add article specific tags
	if post.CreatedAt != "" {
		tags["article:published_time"] = post.CreatedAt
	}
	if post.UpdatedAt != "" {
		tags["article:modified_time"] = post.UpdatedAt
	}
	tags["article:author"] = "Blog Author"

	// Add image if available
	images := s.extractImages(post.Body)
	if len(images) > 0 {
		tags["og:image"] = images[0]
		tags["og:image:alt"] = post.Title
	}

	// Twitter Card tags
	tags["twitter:card"] = "summary_large_image"
	tags["twitter:title"] = html.EscapeString(post.Title)
	tags["twitter:description"] = html.EscapeString(s.getPostDescription(post))
	if len(images) > 0 {
		tags["twitter:image"] = images[0]
	}

	return tags
}

// GenerateSitemap generates XML sitemap with canonical URLs only
func (s *seoService) GenerateSitemap(posts []*model.Post) ([]byte, error) {
	var sitemap strings.Builder

	sitemap.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	sitemap.WriteString("\n")
	sitemap.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	sitemap.WriteString("\n")

	// Add homepage
	sitemap.WriteString("  <url>\n")
	homeURL := s.baseURL
	if homeURL == "" || homeURL == "http://localhost" {
		homeURL = "http://localhost:8080" // Default for development
	}
	if !strings.HasSuffix(homeURL, "/") {
		homeURL += "/"
	}
	sitemap.WriteString(fmt.Sprintf("    <loc>%s</loc>\n", html.EscapeString(homeURL)))
	sitemap.WriteString("    <changefreq>daily</changefreq>\n")
	sitemap.WriteString("    <priority>1.0</priority>\n")
	sitemap.WriteString("  </url>\n")

	// Add blog posts with canonical URLs only
	for _, post := range posts {
		if post.Slug == "" {
			continue // Skip posts without slugs
		}

		sitemap.WriteString("  <url>\n")
		sitemap.WriteString(fmt.Sprintf("    <loc>%s</loc>\n", html.EscapeString(s.GetCanonicalURL(post))))

		// Add lastmod if available - try multiple date formats
		var lastModDate string
		if post.UpdatedAt != "" {
			lastModDate = s.parseAndFormatDate(post.UpdatedAt)
		} else if post.CreatedAt != "" {
			lastModDate = s.parseAndFormatDate(post.CreatedAt)
		} else if post.Date != "" {
			lastModDate = s.parseAndFormatDate(post.Date)
		}

		if lastModDate != "" {
			sitemap.WriteString(fmt.Sprintf("    <lastmod>%s</lastmod>\n", lastModDate))
		}

		sitemap.WriteString("    <changefreq>weekly</changefreq>\n")
		sitemap.WriteString("    <priority>0.8</priority>\n")
		sitemap.WriteString("  </url>\n")
	}

	sitemap.WriteString("</urlset>\n")

	return []byte(sitemap.String()), nil
}

// GenerateRobotsTxt generates robots.txt content with proper crawling instructions
func (s *seoService) GenerateRobotsTxt() string {
	var robots strings.Builder

	// Standard robots.txt format with proper line endings
	robots.WriteString("User-agent: *\n")
	robots.WriteString("Allow: /\n")
	robots.WriteString("Allow: /p/\n")
	robots.WriteString("Allow: /about\n")
	robots.WriteString("Allow: /public/\n")
	robots.WriteString("Disallow: /login\n")
	robots.WriteString("Disallow: /logout\n")
	robots.WriteString("Disallow: /create\n")
	robots.WriteString("Disallow: /update\n")
	robots.WriteString("Disallow: /delete\n")
	robots.WriteString("Disallow: /auth-callback\n")
	robots.WriteString("Disallow: /api/\n")
	robots.WriteString("Disallow: /upload-file\n")
	robots.WriteString("Disallow: /files/\n")
	robots.WriteString("\n")

	// Always include sitemap URL with proper domain configuration
	sitemapURL := s.baseURL
	if sitemapURL == "" || sitemapURL == "http://localhost" {
		sitemapURL = "http://localhost:8080" // Default for development
	}
	// Ensure sitemap URL is properly formatted
	if !strings.HasSuffix(sitemapURL, "/") {
		sitemapURL += "/"
	}
	robots.WriteString(fmt.Sprintf("Sitemap: %ssitemap.xml\n", sitemapURL))

	return robots.String()
}

// GetCanonicalURL returns the canonical URL for a post (slug-based)
func (s *seoService) GetCanonicalURL(post *model.Post) string {
	baseURL := s.baseURL
	if baseURL == "" || baseURL == "http://localhost" {
		baseURL = "http://localhost:8080" // Default for development
	}

	if post.Slug != "" {
		return fmt.Sprintf("%s/p/%s", baseURL, url.PathEscape(post.Slug))
	}
	// Fallback to ID-based URL if no slug (shouldn't happen in normal operation)
	return fmt.Sprintf("%s/post?id=%d", baseURL, post.ID)
}

// getPostDescription gets the description for a post, preferring the meta_description field
func (s *seoService) getPostDescription(post *model.Post) string {
	if post.MetaDescription != "" && post.MetaDescription != "Read this blog post to learn more about the topic." && len(strings.TrimSpace(post.MetaDescription)) > 10 {
		return post.MetaDescription
	}
	description := s.extractDescription(post.Body)
	if description == "" || description == "Read this blog post to learn more about the topic." || len(strings.TrimSpace(description)) < 10 {
		if post.Title != "" {
			return "Read this blog post about " + post.Title + " to learn more about the topic and get insights."
		} else {
			return "Explore this blog post to discover interesting content and valuable insights."
		}
	}
	return description
}

// extractDescription extracts a description from post content
func (s *seoService) extractDescription(content string) string {
	// Remove HTML tags
	htmlTagRegex := regexp.MustCompile(`<[^>]*>`)
	plainText := htmlTagRegex.ReplaceAllString(content, "")

	// Remove file references
	fileRefRegex := regexp.MustCompile(`\[file:[^\]]+\]`)
	plainText = fileRefRegex.ReplaceAllString(plainText, "")

	// Clean up whitespace
	spaceRegex := regexp.MustCompile(`\s+`)
	plainText = spaceRegex.ReplaceAllString(plainText, " ")
	plainText = strings.TrimSpace(plainText)

	// If content is too short, return empty string for further processing
	if len(plainText) < 30 {
		return ""
	}

	// Limit to 155 characters for meta description (optimal SEO length), but try to break at word boundaries
	if len(plainText) > 155 {
		// Find the last space before 152 characters to leave room for "..."
		truncated := plainText[:152]
		lastSpace := strings.LastIndex(truncated, " ")
		if lastSpace > 100 { // Only break at word boundary if it's not too short
			plainText = plainText[:lastSpace] + "..."
		} else {
			plainText = truncated + "..."
		}
	}

	return plainText
}

// extractKeywords extracts keywords from title and content
func (s *seoService) extractKeywords(title, content string) string {
	// Simple keyword extraction - in a real implementation, you might use more sophisticated NLP
	words := make(map[string]int)

	// Extract from title (higher weight)
	titleWords := strings.Fields(strings.ToLower(title))
	for _, word := range titleWords {
		word = regexp.MustCompile(`[^a-z0-9]`).ReplaceAllString(word, "")
		if len(word) > 3 {
			words[word] += 3
		}
	}

	// Extract from content
	htmlTagRegex := regexp.MustCompile(`<[^>]*>`)
	plainContent := htmlTagRegex.ReplaceAllString(content, " ")
	contentWords := strings.Fields(strings.ToLower(plainContent))

	for _, word := range contentWords {
		word = regexp.MustCompile(`[^a-z0-9]`).ReplaceAllString(word, "")
		if len(word) > 4 {
			words[word]++
		}
	}

	// Get top keywords
	var keywords []string
	for word, count := range words {
		if count >= 2 && len(keywords) < 10 {
			keywords = append(keywords, word)
		}
	}

	return strings.Join(keywords, ", ")
}

// extractImages extracts image URLs from post content
func (s *seoService) extractImages(content string) []string {
	var images []string

	// Look for file references that might be images
	fileRefRegex := regexp.MustCompile(`\[file:([^\]]+)\]`)
	matches := fileRefRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		filename := match[1]

		// Query database to check if this file is an image
		var uuid string
		var isImage bool
		err := s.db.QueryRow("SELECT uuid, is_image FROM files WHERE original_name = ? AND is_image = 1 ORDER BY created_at DESC LIMIT 1", filename).Scan(&uuid, &isImage)
		if err == nil && isImage {
			imageURL := fmt.Sprintf("%s/files/%s", s.baseURL, uuid)
			images = append(images, imageURL)
		}
	}

	// Also look for regular HTML img tags
	imgTagRegex := regexp.MustCompile(`<img[^>]+src="([^"]+)"`)
	imgMatches := imgTagRegex.FindAllStringSubmatch(content, -1)
	for _, match := range imgMatches {
		imgSrc := match[1]
		if strings.HasPrefix(imgSrc, "http") {
			images = append(images, imgSrc)
		} else if strings.HasPrefix(imgSrc, "/") {
			images = append(images, s.baseURL+imgSrc)
		}
	}

	return images
}

// parseAndFormatDate tries to parse various date formats and return ISO 8601 date for sitemap
func (s *seoService) parseAndFormatDate(dateStr string) string {
	if dateStr == "" {
		return ""
	}

	// Try different date formats
	formats := []string{
		"2006-01-02 15:04:05",       // SQLite datetime format
		"Mon Jan _2 15:04:05 2006",  // Go default format
		"2006-01-02T15:04:05Z",      // ISO 8601
		"2006-01-02T15:04:05-07:00", // ISO 8601 with timezone
		"2006-01-02",                // Date only
	}

	for _, format := range formats {
		if parsedTime, err := time.Parse(format, dateStr); err == nil {
			return parsedTime.Format("2006-01-02")
		}
	}

	// If all parsing fails, return empty string
	return ""
}
