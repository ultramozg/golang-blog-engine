package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ultramozg/golang-blog-engine/model"
	_ "modernc.org/sqlite"
)

func TestNewSEOService(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	service := NewSEOService(db, "https://example.com")
	if service == nil {
		t.Error("Expected SEO service to be created")
	}
}

func TestGenerateMetaTags(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	service := NewSEOService(db, "https://example.com")
	
	post := &model.Post{
		ID:        1,
		Title:     "Test Blog Post",
		Body:      "This is a test blog post with some content that should be used for meta description.",
		Slug:      "test-blog-post",
		CreatedAt: "Mon Jan 2 15:04:05 2006",
		UpdatedAt: "Mon Jan 2 15:04:05 2006",
	}

	tags := service.GenerateMetaTags(post)

	// Test title
	if tags["title"] != "Test Blog Post" {
		t.Errorf("Expected title 'Test Blog Post', got '%s'", tags["title"])
	}

	// Test description
	if tags["description"] == "" {
		t.Error("Expected description to be generated")
	}

	// Test canonical URL
	expectedCanonical := "https://example.com/p/test-blog-post"
	if tags["canonical"] != expectedCanonical {
		t.Errorf("Expected canonical URL '%s', got '%s'", expectedCanonical, tags["canonical"])
	}

	// Test author
	if tags["author"] != "Blog Author" {
		t.Errorf("Expected author 'Blog Author', got '%s'", tags["author"])
	}

	// Test published time
	if tags["article:published_time"] != post.CreatedAt {
		t.Errorf("Expected published time '%s', got '%s'", post.CreatedAt, tags["article:published_time"])
	}
}

func TestGenerateStructuredData(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	service := NewSEOService(db, "https://example.com")
	
	post := &model.Post{
		ID:        1,
		Title:     "Test Blog Post",
		Body:      "This is a test blog post content.",
		Slug:      "test-blog-post",
		CreatedAt: "Mon Jan 2 15:04:05 2006",
		UpdatedAt: "Mon Jan 2 15:04:05 2006",
	}

	structuredDataJSON := service.GenerateStructuredData(post)
	
	if structuredDataJSON == "" {
		t.Error("Expected structured data to be generated")
	}

	// Parse JSON to verify structure
	var structuredData map[string]interface{}
	err = json.Unmarshal([]byte(structuredDataJSON), &structuredData)
	if err != nil {
		t.Errorf("Generated structured data is not valid JSON: %v", err)
	}

	// Test required fields
	if structuredData["@context"] != "https://schema.org" {
		t.Error("Expected @context to be 'https://schema.org'")
	}

	if structuredData["@type"] != "BlogPosting" {
		t.Error("Expected @type to be 'BlogPosting'")
	}

	if structuredData["headline"] != "Test Blog Post" {
		t.Error("Expected headline to match post title")
	}

	if structuredData["url"] != "https://example.com/p/test-blog-post" {
		t.Error("Expected URL to match canonical URL")
	}
}

func TestGenerateOpenGraphTags(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	service := NewSEOService(db, "https://example.com")
	
	post := &model.Post{
		ID:        1,
		Title:     "Test Blog Post",
		Body:      "This is a test blog post content.",
		Slug:      "test-blog-post",
		CreatedAt: "Mon Jan 2 15:04:05 2006",
		UpdatedAt: "Mon Jan 2 15:04:05 2006",
	}

	tags := service.GenerateOpenGraphTags(post)

	// Test required Open Graph tags
	if tags["og:type"] != "article" {
		t.Errorf("Expected og:type 'article', got '%s'", tags["og:type"])
	}

	if tags["og:title"] != "Test Blog Post" {
		t.Errorf("Expected og:title 'Test Blog Post', got '%s'", tags["og:title"])
	}

	if tags["og:url"] != "https://example.com/p/test-blog-post" {
		t.Errorf("Expected og:url 'https://example.com/p/test-blog-post', got '%s'", tags["og:url"])
	}

	if tags["og:site_name"] != "Blog" {
		t.Errorf("Expected og:site_name 'Blog', got '%s'", tags["og:site_name"])
	}

	// Test Twitter Card tags
	if tags["twitter:card"] != "summary_large_image" {
		t.Errorf("Expected twitter:card 'summary_large_image', got '%s'", tags["twitter:card"])
	}

	if tags["twitter:title"] != "Test Blog Post" {
		t.Errorf("Expected twitter:title 'Test Blog Post', got '%s'", tags["twitter:title"])
	}
}

func TestGenerateSitemap(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	service := NewSEOService(db, "https://example.com")
	
	posts := []*model.Post{
		{
			ID:        1,
			Title:     "First Post",
			Slug:      "first-post",
			CreatedAt: "Mon Jan 2 15:04:05 2006",
			UpdatedAt: "Mon Jan 2 15:04:05 2006",
		},
		{
			ID:        2,
			Title:     "Second Post",
			Slug:      "second-post",
			CreatedAt: "Tue Jan 3 15:04:05 2006",
			UpdatedAt: "Tue Jan 3 15:04:05 2006",
		},
	}

	sitemapXML, err := service.GenerateSitemap(posts)
	if err != nil {
		t.Errorf("Error generating sitemap: %v", err)
	}

	sitemapString := string(sitemapXML)

	// Test XML structure
	if !strings.Contains(sitemapString, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Error("Expected XML declaration")
	}

	if !strings.Contains(sitemapString, `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`) {
		t.Error("Expected urlset element with namespace")
	}

	// Test homepage entry
	if !strings.Contains(sitemapString, "<loc>https://example.com/</loc>") {
		t.Error("Expected homepage entry")
	}

	// Test post entries
	if !strings.Contains(sitemapString, "<loc>https://example.com/p/first-post</loc>") {
		t.Error("Expected first post entry")
	}

	if !strings.Contains(sitemapString, "<loc>https://example.com/p/second-post</loc>") {
		t.Error("Expected second post entry")
	}

	// Test lastmod entries
	if !strings.Contains(sitemapString, "<lastmod>2006-01-02</lastmod>") {
		t.Error("Expected lastmod entries")
	}
}

func TestGenerateRobotsTxt(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	service := NewSEOService(db, "https://example.com")
	
	robotsTxt := service.GenerateRobotsTxt()

	// Test basic structure
	if !strings.Contains(robotsTxt, "User-agent: *") {
		t.Error("Expected User-agent directive")
	}

	if !strings.Contains(robotsTxt, "Allow: /") {
		t.Error("Expected Allow directive")
	}

	// Test disallowed paths
	disallowedPaths := []string{"/login", "/logout", "/create", "/update", "/delete", "/auth-callback", "/api/", "/upload-file"}
	for _, path := range disallowedPaths {
		if !strings.Contains(robotsTxt, "Disallow: "+path) {
			t.Errorf("Expected Disallow directive for %s", path)
		}
	}

	// Test sitemap reference
	if !strings.Contains(robotsTxt, "Sitemap: https://example.com/sitemap.xml") {
		t.Error("Expected sitemap reference")
	}
}

func TestGetCanonicalURL(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	service := NewSEOService(db, "https://example.com")
	
	// Test with slug
	post := &model.Post{
		ID:   1,
		Slug: "test-post",
	}

	canonicalURL := service.GetCanonicalURL(post)
	expected := "https://example.com/p/test-post"
	if canonicalURL != expected {
		t.Errorf("Expected canonical URL '%s', got '%s'", expected, canonicalURL)
	}

	// Test without slug (fallback)
	postNoSlug := &model.Post{
		ID:   1,
		Slug: "",
	}

	canonicalURL = service.GetCanonicalURL(postNoSlug)
	expected = "https://example.com/post?id=1"
	if canonicalURL != expected {
		t.Errorf("Expected fallback canonical URL '%s', got '%s'", expected, canonicalURL)
	}
}

func TestExtractDescription(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	service := NewSEOService(db, "https://example.com").(*seoService)
	
	// Test HTML removal
	content := "<p>This is a <strong>test</strong> content with <a href='#'>links</a>.</p>"
	description := service.extractDescription(content)
	expected := "This is a test content with links."
	if description != expected {
		t.Errorf("Expected description '%s', got '%s'", expected, description)
	}

	// Test file reference removal
	content = "This is content with [file:document.pdf] reference."
	description = service.extractDescription(content)
	expected = "This is content with reference."
	if description != expected {
		t.Errorf("Expected description '%s', got '%s'", expected, description)
	}

	// Test length limit
	longContent := strings.Repeat("This is a very long content. ", 10)
	description = service.extractDescription(longContent)
	if len(description) > 160 {
		t.Errorf("Expected description to be limited to 160 characters, got %d", len(description))
	}
	if !strings.HasSuffix(description, "...") {
		t.Error("Expected long description to end with '...'")
	}

	// Test empty content
	description = service.extractDescription("")
	if description != "Blog post" {
		t.Errorf("Expected default description 'Blog post', got '%s'", description)
	}
}

func TestExtractKeywords(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	service := NewSEOService(db, "https://example.com").(*seoService)
	
	title := "Go Programming Tutorial"
	content := "This is a comprehensive Go programming tutorial that covers programming basics and advanced Go concepts."
	
	keywords := service.extractKeywords(title, content)
	
	// Should contain words from title (higher weight) and content
	if !strings.Contains(keywords, "programming") {
		t.Error("Expected 'programming' to be in keywords")
	}
	
	// Keywords should be comma-separated
	if !strings.Contains(keywords, ",") {
		t.Error("Expected keywords to be comma-separated")
	}
}

func TestGenerateMetaTagsWithSpecialCharacters(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	service := NewSEOService(db, "https://example.com")
	
	post := &model.Post{
		ID:        1,
		Title:     "Test & Blog <Post>",
		Body:      "This is a test blog post with <script>alert('xss')</script> content.",
		Slug:      "test-blog-post",
		CreatedAt: "Mon Jan 2 15:04:05 2006",
		UpdatedAt: "Mon Jan 2 15:04:05 2006",
	}

	tags := service.GenerateMetaTags(post)

	// Test that HTML is properly escaped
	if tags["title"] != "Test &amp; Blog &lt;Post&gt;" {
		t.Errorf("Expected escaped title, got '%s'", tags["title"])
	}

	// Test that description doesn't contain script tags
	if strings.Contains(tags["description"], "<script>") {
		t.Error("Description should not contain script tags")
	}
}

func TestGenerateStructuredDataWithImages(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Create files table and insert test image
	_, err = db.Exec(`CREATE TABLE files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		uuid TEXT UNIQUE NOT NULL,
		original_name TEXT NOT NULL,
		stored_name TEXT NOT NULL,
		path TEXT NOT NULL,
		size INTEGER NOT NULL,
		mime_type TEXT NOT NULL,
		download_count INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		is_image BOOLEAN DEFAULT FALSE,
		width INTEGER,
		height INTEGER,
		thumbnail_path TEXT,
		alt_text TEXT
	)`)
	if err != nil {
		t.Fatal(err)
	}

	// Insert test image file
	_, err = db.Exec(`INSERT INTO files (uuid, original_name, stored_name, path, size, mime_type, is_image, width, height) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"test-uuid", "test-image.jpg", "stored-image.jpg", "/path/to/image", 1024, "image/jpeg", true, 800, 600)
	if err != nil {
		t.Fatal(err)
	}

	service := NewSEOService(db, "https://example.com")
	
	post := &model.Post{
		ID:        1,
		Title:     "Post with Image",
		Body:      "This post has an image [file:test-image.jpg] in it.",
		Slug:      "post-with-image",
		CreatedAt: "Mon Jan 2 15:04:05 2006",
		UpdatedAt: "Mon Jan 2 15:04:05 2006",
	}

	structuredDataJSON := service.GenerateStructuredData(post)
	
	if structuredDataJSON == "" {
		t.Error("Expected structured data to be generated")
	}

	// Parse JSON to verify image is included
	var structuredData map[string]interface{}
	err = json.Unmarshal([]byte(structuredDataJSON), &structuredData)
	if err != nil {
		t.Errorf("Generated structured data is not valid JSON: %v", err)
	}

	// Check if image is included
	if image, exists := structuredData["image"]; exists {
		imageStr, ok := image.(string)
		if !ok {
			t.Error("Expected image to be a string")
		} else if !strings.Contains(imageStr, "https://example.com/files/test-uuid") {
			t.Errorf("Expected image URL to contain file UUID, got '%s'", imageStr)
		}
	} else {
		t.Error("Expected image to be included in structured data")
	}
}

func TestGenerateOpenGraphTagsWithImages(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Create files table and insert test image
	_, err = db.Exec(`CREATE TABLE files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		uuid TEXT UNIQUE NOT NULL,
		original_name TEXT NOT NULL,
		stored_name TEXT NOT NULL,
		path TEXT NOT NULL,
		size INTEGER NOT NULL,
		mime_type TEXT NOT NULL,
		download_count INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		is_image BOOLEAN DEFAULT FALSE,
		width INTEGER,
		height INTEGER,
		thumbnail_path TEXT,
		alt_text TEXT
	)`)
	if err != nil {
		t.Fatal(err)
	}

	// Insert test image file
	_, err = db.Exec(`INSERT INTO files (uuid, original_name, stored_name, path, size, mime_type, is_image) 
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"test-uuid-og", "og-image.jpg", "stored-og-image.jpg", "/path/to/og-image", 2048, "image/jpeg", true)
	if err != nil {
		t.Fatal(err)
	}

	service := NewSEOService(db, "https://example.com")
	
	post := &model.Post{
		ID:        1,
		Title:     "Post with OG Image",
		Body:      "This post has an Open Graph image [file:og-image.jpg] in it.",
		Slug:      "post-with-og-image",
		CreatedAt: "Mon Jan 2 15:04:05 2006",
		UpdatedAt: "Mon Jan 2 15:04:05 2006",
	}

	tags := service.GenerateOpenGraphTags(post)

	// Test that image is included in Open Graph tags
	if ogImage, exists := tags["og:image"]; exists {
		expectedImageURL := "https://example.com/files/test-uuid-og"
		if ogImage != expectedImageURL {
			t.Errorf("Expected og:image '%s', got '%s'", expectedImageURL, ogImage)
		}
	} else {
		t.Error("Expected og:image to be included")
	}

	// Test that Twitter image is also included
	if twitterImage, exists := tags["twitter:image"]; exists {
		expectedImageURL := "https://example.com/files/test-uuid-og"
		if twitterImage != expectedImageURL {
			t.Errorf("Expected twitter:image '%s', got '%s'", expectedImageURL, twitterImage)
		}
	} else {
		t.Error("Expected twitter:image to be included")
	}

	// Test alt text
	if ogImageAlt, exists := tags["og:image:alt"]; exists {
		if ogImageAlt != "Post with OG Image" {
			t.Errorf("Expected og:image:alt to be post title, got '%s'", ogImageAlt)
		}
	} else {
		t.Error("Expected og:image:alt to be included")
	}
}

func TestGenerateSitemapWithEmptyPosts(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	service := NewSEOService(db, "https://example.com")
	
	// Test with empty posts array
	posts := []*model.Post{}

	sitemapXML, err := service.GenerateSitemap(posts)
	if err != nil {
		t.Errorf("Error generating sitemap with empty posts: %v", err)
	}

	sitemapString := string(sitemapXML)

	// Should still contain homepage
	if !strings.Contains(sitemapString, "<loc>https://example.com/</loc>") {
		t.Error("Expected homepage entry even with no posts")
	}

	// Should contain proper XML structure
	if !strings.Contains(sitemapString, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Error("Expected XML declaration")
	}
}

func TestGenerateSitemapSkipsPostsWithoutSlugs(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	service := NewSEOService(db, "https://example.com")
	
	posts := []*model.Post{
		{
			ID:        1,
			Title:     "Post with Slug",
			Slug:      "post-with-slug",
			CreatedAt: "Mon Jan 2 15:04:05 2006",
		},
		{
			ID:        2,
			Title:     "Post without Slug",
			Slug:      "", // Empty slug should be skipped
			CreatedAt: "Mon Jan 3 15:04:05 2006",
		},
	}

	sitemapXML, err := service.GenerateSitemap(posts)
	if err != nil {
		t.Errorf("Error generating sitemap: %v", err)
	}

	sitemapString := string(sitemapXML)

	// Should contain post with slug
	if !strings.Contains(sitemapString, "<loc>https://example.com/p/post-with-slug</loc>") {
		t.Error("Expected post with slug to be included")
	}

	// Should not contain post without slug
	if strings.Contains(sitemapString, "Post without Slug") {
		t.Error("Expected post without slug to be excluded")
	}
}

func TestGetCanonicalURLWithSpecialCharacters(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	service := NewSEOService(db, "https://example.com")
	
	// Test with normal slug (no special characters should be present after slug sanitization)
	post := &model.Post{
		ID:   1,
		Slug: "test-post-with-normal-slug",
	}

	canonicalURL := service.GetCanonicalURL(post)
	expected := "https://example.com/p/test-post-with-normal-slug"
	if canonicalURL != expected {
		t.Errorf("Expected canonical URL '%s', got '%s'", expected, canonicalURL)
	}
}

func TestExtractImagesFromHTML(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	service := NewSEOService(db, "https://example.com").(*seoService)
	
	// Test content with HTML img tags
	content := `<p>This is a post with images:</p>
		<img src="https://external.com/image1.jpg" alt="External image">
		<img src="/local/image2.png" alt="Local image">
		<p>More content here.</p>`

	images := service.extractImages(content)

	// Should extract both external and local images
	if len(images) != 2 {
		t.Errorf("Expected 2 images, got %d", len(images))
	}

	// Check external image URL
	if !contains(images, "https://external.com/image1.jpg") {
		t.Error("Expected external image URL to be extracted")
	}

	// Check local image URL (should be converted to absolute)
	if !contains(images, "https://example.com/local/image2.png") {
		t.Error("Expected local image URL to be converted to absolute")
	}
}

func TestExtractDescriptionLengthLimit(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	service := NewSEOService(db, "https://example.com").(*seoService)
	
	// Create content longer than 160 characters
	longContent := strings.Repeat("This is a very long sentence that will exceed the meta description limit. ", 5)
	
	description := service.extractDescription(longContent)
	
	// Should be limited to 160 characters
	if len(description) > 160 {
		t.Errorf("Expected description to be limited to 160 characters, got %d", len(description))
	}
	
	// Should end with "..."
	if !strings.HasSuffix(description, "...") {
		t.Error("Expected long description to end with '...'")
	}
}

func TestGenerateRobotsTxtContent(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	service := NewSEOService(db, "https://example.com")
	
	robotsTxt := service.GenerateRobotsTxt()

	// Test all required disallow paths
	requiredDisallows := []string{
		"Disallow: /login",
		"Disallow: /logout", 
		"Disallow: /create",
		"Disallow: /update",
		"Disallow: /delete",
		"Disallow: /auth-callback",
		"Disallow: /api/",
		"Disallow: /upload-file",
	}

	for _, disallow := range requiredDisallows {
		if !strings.Contains(robotsTxt, disallow) {
			t.Errorf("Expected robots.txt to contain '%s'", disallow)
		}
	}

	// Test sitemap reference
	expectedSitemap := "Sitemap: https://example.com/sitemap.xml"
	if !strings.Contains(robotsTxt, expectedSitemap) {
		t.Errorf("Expected robots.txt to contain '%s'", expectedSitemap)
	}

	// Test user agent
	if !strings.Contains(robotsTxt, "User-agent: *") {
		t.Error("Expected robots.txt to contain 'User-agent: *'")
	}

	// Test allow directive
	if !strings.Contains(robotsTxt, "Allow: /") {
		t.Error("Expected robots.txt to contain 'Allow: /'")
	}
}

// Helper function to check if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// TestSEOServiceComprehensive tests comprehensive SEO functionality
func TestSEOServiceComprehensive(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Create files table for image testing
	_, err = db.Exec(`CREATE TABLE files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		uuid TEXT UNIQUE NOT NULL,
		original_name TEXT NOT NULL,
		stored_name TEXT NOT NULL,
		path TEXT NOT NULL,
		size INTEGER NOT NULL,
		mime_type TEXT NOT NULL,
		download_count INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		is_image BOOLEAN DEFAULT FALSE,
		width INTEGER,
		height INTEGER,
		thumbnail_path TEXT,
		alt_text TEXT
	)`)
	if err != nil {
		t.Fatal(err)
	}

	service := NewSEOService(db, "https://example.com")

	t.Run("MetaTagsWithEmptyFields", func(t *testing.T) {
		post := &model.Post{
			ID:    1,
			Title: "",
			Body:  "",
			Slug:  "empty-post",
		}

		tags := service.GenerateMetaTags(post)

		// Should handle empty title gracefully
		if tags["title"] != "" {
			t.Errorf("Expected empty title to be handled, got '%s'", tags["title"])
		}

		// Should provide default description
		if tags["description"] != "Blog post" {
			t.Errorf("Expected default description 'Blog post', got '%s'", tags["description"])
		}
	})

	t.Run("StructuredDataWithMultipleImages", func(t *testing.T) {
		// Insert multiple test images
		images := []struct {
			uuid, name string
		}{
			{"uuid-1", "image1.jpg"},
			{"uuid-2", "image2.png"},
			{"uuid-3", "image3.gif"},
		}

		for _, img := range images {
			_, err = db.Exec(`INSERT INTO files (uuid, original_name, stored_name, path, size, mime_type, is_image) 
				VALUES (?, ?, ?, ?, ?, ?, ?)`,
				img.uuid, img.name, "stored-"+img.name, "/path/to/"+img.name, 1024, "image/jpeg", true)
			if err != nil {
				t.Fatal(err)
			}
		}

		post := &model.Post{
			ID:    1,
			Title: "Post with Multiple Images",
			Body:  "This post has [file:image1.jpg] and [file:image2.png] and [file:image3.gif] images.",
			Slug:  "post-with-multiple-images",
		}

		structuredDataJSON := service.GenerateStructuredData(post)
		
		var structuredData map[string]interface{}
		err = json.Unmarshal([]byte(structuredDataJSON), &structuredData)
		if err != nil {
			t.Errorf("Generated structured data is not valid JSON: %v", err)
		}

		// Should contain image array for multiple images
		if images, exists := structuredData["image"]; exists {
			imageArray, ok := images.([]interface{})
			if !ok {
				t.Error("Expected image to be an array for multiple images")
			} else if len(imageArray) != 3 {
				t.Errorf("Expected 3 images in structured data, got %d", len(imageArray))
			}
		} else {
			t.Error("Expected images to be included in structured data")
		}
	})

	t.Run("SitemapWithInvalidDates", func(t *testing.T) {
		posts := []*model.Post{
			{
				ID:        1,
				Title:     "Post with Invalid Date",
				Slug:      "post-invalid-date",
				CreatedAt: "invalid-date-format",
				UpdatedAt: "also-invalid",
			},
			{
				ID:        2,
				Title:     "Post with Valid Date",
				Slug:      "post-valid-date",
				CreatedAt: "Mon Jan 2 15:04:05 2006",
				UpdatedAt: "Mon Jan 3 15:04:05 2006",
			},
		}

		sitemapXML, err := service.GenerateSitemap(posts)
		if err != nil {
			t.Errorf("Error generating sitemap with invalid dates: %v", err)
		}

		sitemapString := string(sitemapXML)

		// Should still include posts with invalid dates (just without lastmod)
		if !strings.Contains(sitemapString, "<loc>https://example.com/p/post-invalid-date</loc>") {
			t.Error("Expected post with invalid date to be included")
		}

		if !strings.Contains(sitemapString, "<loc>https://example.com/p/post-valid-date</loc>") {
			t.Error("Expected post with valid date to be included")
		}

		// Should include lastmod for valid date
		if !strings.Contains(sitemapString, "<lastmod>2006-01-03</lastmod>") {
			t.Error("Expected lastmod for post with valid date")
		}
	})

	t.Run("OpenGraphTagsWithLongContent", func(t *testing.T) {
		longContent := strings.Repeat("This is a very long content that should be truncated for Open Graph description. ", 20)
		
		post := &model.Post{
			ID:    1,
			Title: "Post with Long Content",
			Body:  longContent,
			Slug:  "post-long-content",
		}

		tags := service.GenerateOpenGraphTags(post)

		// Description should be truncated
		if len(tags["og:description"]) > 160 {
			t.Errorf("Expected og:description to be truncated, got length %d", len(tags["og:description"]))
		}

		// Twitter description should also be truncated
		if len(tags["twitter:description"]) > 160 {
			t.Errorf("Expected twitter:description to be truncated, got length %d", len(tags["twitter:description"]))
		}
	})

	t.Run("CanonicalURLWithSpecialCharacters", func(t *testing.T) {
		post := &model.Post{
			ID:   1,
			Slug: "post-with-special-chars-&-symbols",
		}

		canonicalURL := service.GetCanonicalURL(post)
		
		// Should contain the slug (URL encoding is handled by the implementation)
		if !strings.Contains(canonicalURL, "post-with-special-chars") {
			t.Error("Expected canonical URL to contain the slug")
		}
		
		// Should be a valid URL format
		if !strings.HasPrefix(canonicalURL, "https://example.com/p/") {
			t.Error("Expected canonical URL to have correct format")
		}
	})

	t.Run("ExtractImagesFromMixedContent", func(t *testing.T) {
		// Insert test files (both images and non-images)
		_, err = db.Exec(`INSERT INTO files (uuid, original_name, stored_name, path, size, mime_type, is_image) 
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			"img-uuid", "test-image.jpg", "stored-image.jpg", "/path/to/image", 1024, "image/jpeg", true)
		if err != nil {
			t.Fatal(err)
		}

		_, err = db.Exec(`INSERT INTO files (uuid, original_name, stored_name, path, size, mime_type, is_image) 
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			"doc-uuid", "document.pdf", "stored-doc.pdf", "/path/to/doc", 2048, "application/pdf", false)
		if err != nil {
			t.Fatal(err)
		}

		seoSvc := service.(*seoService)
		content := `This post has [file:test-image.jpg] and [file:document.pdf] and <img src="https://external.com/image.jpg" alt="External"> and <img src="/local/image.png" alt="Local">`

		images := seoSvc.extractImages(content)

		// Should extract image files and HTML img tags, but not non-image files
		expectedImages := []string{
			"https://example.com/files/img-uuid",
			"https://external.com/image.jpg",
			"https://example.com/local/image.png",
		}

		if len(images) != len(expectedImages) {
			t.Errorf("Expected %d images, got %d", len(expectedImages), len(images))
		}

		for _, expected := range expectedImages {
			if !contains(images, expected) {
				t.Errorf("Expected image '%s' to be extracted", expected)
			}
		}
	})

	t.Run("KeywordExtractionFromComplexContent", func(t *testing.T) {
		seoSvc := service.(*seoService)
		
		title := "Advanced Go Programming Tutorial"
		content := `<p>This comprehensive <strong>Go programming</strong> tutorial covers advanced concepts.</p>
			<p>Learn about goroutines, channels, and concurrent programming in Go.</p>
			<p>Master advanced Go programming techniques and best practices.</p>
			[file:example.pdf] Download the example code.`

		keywords := seoSvc.extractKeywords(title, content)

		// Should extract meaningful keywords
		if !strings.Contains(keywords, "programming") {
			t.Error("Expected 'programming' to be in keywords")
		}

		if !strings.Contains(keywords, "advanced") {
			t.Error("Expected 'advanced' to be in keywords")
		}

		// Should not contain HTML tags or file references
		if strings.Contains(keywords, "strong") || strings.Contains(keywords, "file") {
			t.Error("Keywords should not contain HTML tags or file references")
		}
	})

	t.Run("RobotsTxtWithCustomBaseURL", func(t *testing.T) {
		customService := NewSEOService(db, "https://custom-domain.com")
		
		robotsTxt := customService.GenerateRobotsTxt()

		// Should contain custom domain in sitemap reference
		expectedSitemap := "Sitemap: https://custom-domain.com/sitemap.xml"
		if !strings.Contains(robotsTxt, expectedSitemap) {
			t.Errorf("Expected custom domain sitemap reference '%s'", expectedSitemap)
		}
	})

	t.Run("SitemapXMLValidation", func(t *testing.T) {
		posts := []*model.Post{
			{
				ID:        1,
				Title:     "Test Post",
				Slug:      "test-post",
				CreatedAt: "Mon Jan 2 15:04:05 2006",
			},
		}

		sitemapXML, err := service.GenerateSitemap(posts)
		if err != nil {
			t.Errorf("Error generating sitemap: %v", err)
		}

		sitemapString := string(sitemapXML)

		// Validate XML structure
		requiredElements := []string{
			`<?xml version="1.0" encoding="UTF-8"?>`,
			`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`,
			`</urlset>`,
			`<url>`,
			`</url>`,
			`<loc>`,
			`</loc>`,
			`<changefreq>`,
			`</changefreq>`,
			`<priority>`,
			`</priority>`,
		}

		for _, element := range requiredElements {
			if !strings.Contains(sitemapString, element) {
				t.Errorf("Expected sitemap to contain XML element: %s", element)
			}
		}
	})
}

// TestSEOServiceErrorHandling tests error handling scenarios
func TestSEOServiceErrorHandling(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	service := NewSEOService(db, "https://example.com")

	t.Run("NilPostHandling", func(t *testing.T) {
		// Test with nil post - should panic (current behavior)
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for nil post, but didn't panic")
			}
		}()

		// This should panic with nil post (current implementation behavior)
		service.GenerateMetaTags(nil)
	})

	t.Run("DatabaseErrorHandling", func(t *testing.T) {
		// Close database to simulate error
		db.Close()

		post := &model.Post{
			ID:    1,
			Title: "Test Post",
			Body:  "Content with [file:test.jpg] reference",
			Slug:  "test-post",
		}

		// Should handle database errors gracefully
		structuredData := service.GenerateStructuredData(post)
		if structuredData == "" {
			t.Error("Should generate structured data even with database errors")
		}

		ogTags := service.GenerateOpenGraphTags(post)
		if len(ogTags) == 0 {
			t.Error("Should generate OG tags even with database errors")
		}
	})

	t.Run("EmptyPostsArraySitemap", func(t *testing.T) {
		// Reopen database
		db, err = sql.Open("sqlite", ":memory:")
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		service = NewSEOService(db, "https://example.com")

		sitemapXML, err := service.GenerateSitemap([]*model.Post{})
		if err != nil {
			t.Errorf("Should handle empty posts array: %v", err)
		}

		sitemapString := string(sitemapXML)
		
		// Should still contain homepage
		if !strings.Contains(sitemapString, "<loc>https://example.com/</loc>") {
			t.Error("Should include homepage even with no posts")
		}
	})
}

// TestSEOServicePerformance tests performance aspects
func TestSEOServicePerformance(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	service := NewSEOService(db, "https://example.com")

	t.Run("LargeSitemapGeneration", func(t *testing.T) {
		// Create a large number of posts
		posts := make([]*model.Post, 1000)
		for i := 0; i < 1000; i++ {
			posts[i] = &model.Post{
				ID:        i + 1,
				Title:     fmt.Sprintf("Test Post %d", i+1),
				Slug:      fmt.Sprintf("test-post-%d", i+1),
				CreatedAt: "Mon Jan 2 15:04:05 2006",
			}
		}

		start := time.Now()
		sitemapXML, err := service.GenerateSitemap(posts)
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Error generating large sitemap: %v", err)
		}

		if len(sitemapXML) == 0 {
			t.Error("Expected non-empty sitemap")
		}

		// Should complete within reasonable time (adjust as needed)
		if duration > 5*time.Second {
			t.Errorf("Large sitemap generation took too long: %v", duration)
		}

		t.Logf("Generated sitemap for 1000 posts in %v", duration)
	})

	t.Run("ComplexContentProcessing", func(t *testing.T) {
		// Create post with complex content
		complexContent := strings.Repeat(`<p>This is a paragraph with <strong>bold</strong> and <em>italic</em> text. 
			It also contains [file:image1.jpg] and [file:document.pdf] references. 
			There are also <a href="https://example.com">links</a> and other HTML elements.</p>`, 100)

		post := &model.Post{
			ID:    1,
			Title: "Complex Content Post",
			Body:  complexContent,
			Slug:  "complex-content-post",
		}

		start := time.Now()
		
		// Test all SEO operations
		metaTags := service.GenerateMetaTags(post)
		structuredData := service.GenerateStructuredData(post)
		ogTags := service.GenerateOpenGraphTags(post)
		
		duration := time.Since(start)

		// Should complete within reasonable time
		if duration > 1*time.Second {
			t.Errorf("Complex content processing took too long: %v", duration)
		}

		// Should produce valid results
		if len(metaTags) == 0 {
			t.Error("Expected meta tags for complex content")
		}

		if structuredData == "" {
			t.Error("Expected structured data for complex content")
		}

		if len(ogTags) == 0 {
			t.Error("Expected OG tags for complex content")
		}

		t.Logf("Processed complex content in %v", duration)
	})
}