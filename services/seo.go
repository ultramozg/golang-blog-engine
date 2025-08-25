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
	
	// Meta description - extract from post body if not set
	description := s.extractDescription(post.Body)
	tags["description"] = html.EscapeString(description)
	
	// Keywords - basic extraction from title and content
	keywords := s.extractKeywords(post.Title, post.Body)
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
		"description": s.extractDescription(post.Body),
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
	tags["og:description"] = html.EscapeString(s.extractDescription(post.Body))
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
	tags["twitter:description"] = html.EscapeString(s.extractDescription(post.Body))
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
	sitemap.WriteString(fmt.Sprintf("    <loc>%s/</loc>\n", s.baseURL))
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
		
		// Add lastmod if available
		if post.UpdatedAt != "" {
			// Parse and format the date properly for sitemap
			if parsedTime, err := time.Parse("Mon Jan _2 15:04:05 2006", post.UpdatedAt); err == nil {
				sitemap.WriteString(fmt.Sprintf("    <lastmod>%s</lastmod>\n", parsedTime.Format("2006-01-02")))
			}
		} else if post.CreatedAt != "" {
			if parsedTime, err := time.Parse("Mon Jan _2 15:04:05 2006", post.CreatedAt); err == nil {
				sitemap.WriteString(fmt.Sprintf("    <lastmod>%s</lastmod>\n", parsedTime.Format("2006-01-02")))
			}
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
	
	robots.WriteString("User-agent: *\n")
	robots.WriteString("Allow: /\n")
	robots.WriteString("Disallow: /login\n")
	robots.WriteString("Disallow: /logout\n")
	robots.WriteString("Disallow: /create\n")
	robots.WriteString("Disallow: /update\n")
	robots.WriteString("Disallow: /delete\n")
	robots.WriteString("Disallow: /auth-callback\n")
	robots.WriteString("Disallow: /api/\n")
	robots.WriteString("Disallow: /upload-file\n")
	robots.WriteString("\n")
	robots.WriteString(fmt.Sprintf("Sitemap: %s/sitemap.xml\n", s.baseURL))
	
	return robots.String()
}

// GetCanonicalURL returns the canonical URL for a post (slug-based)
func (s *seoService) GetCanonicalURL(post *model.Post) string {
	if post.Slug != "" {
		return fmt.Sprintf("%s/p/%s", s.baseURL, url.PathEscape(post.Slug))
	}
	// Fallback to ID-based URL if no slug (shouldn't happen in normal operation)
	return fmt.Sprintf("%s/post?id=%d", s.baseURL, post.ID)
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
	
	// Limit to 160 characters for meta description
	if len(plainText) > 160 {
		plainText = plainText[:157] + "..."
	}
	
	if plainText == "" {
		return "Blog post"
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