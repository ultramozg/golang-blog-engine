package app

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ultramozg/golang-blog-engine/model"
)

func TestServeSitemap(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Create test posts with slugs
	testPosts := []model.Post{
		{
			Title: "First Test Post",
			Body:  "This is the first test post content.",
			Date:  "Mon Jan 2 15:04:05 2006",
			Slug:  "first-test-post",
		},
		{
			Title: "Second Test Post",
			Body:  "This is the second test post content.",
			Date:  "Tue Jan 3 15:04:05 2006",
			Slug:  "second-test-post",
		},
	}

	// Insert test posts with specific dates
	for _, post := range testPosts {
		_, err := app.DB.Exec(`
			INSERT INTO posts (title, body, datepost, slug, created_at, updated_at) 
			VALUES (?, ?, ?, ?, ?, ?)
		`, post.Title, post.Body, post.Date, post.Slug, post.Date, post.Date)
		if err != nil {
			t.Fatalf("Failed to insert test post: %v", err)
		}
	}

	// Test GET request
	req, err := http.NewRequest("GET", "/sitemap.xml", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.serveSitemap)
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Check content type
	expectedContentType := "application/xml; charset=utf-8"
	if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("Expected content type %s, got %s", expectedContentType, contentType)
	}

	// Check cache control header
	if cacheControl := rr.Header().Get("Cache-Control"); cacheControl != "public, max-age=3600" {
		t.Errorf("Expected cache control 'public, max-age=3600', got '%s'", cacheControl)
	}

	// Check XML content
	body := rr.Body.String()
	
	// Should contain XML declaration
	if !strings.Contains(body, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Error("Expected XML declaration in sitemap")
	}

	// Should contain urlset element
	if !strings.Contains(body, `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`) {
		t.Error("Expected urlset element in sitemap")
	}

	// Should contain homepage
	if !strings.Contains(body, "<loc>http://localhost:8080/</loc>") {
		t.Error("Expected homepage URL in sitemap")
	}

	// Should contain test posts
	if !strings.Contains(body, "<loc>http://localhost:8080/p/first-test-post</loc>") {
		t.Error("Expected first test post URL in sitemap")
	}

	if !strings.Contains(body, "<loc>http://localhost:8080/p/second-test-post</loc>") {
		t.Error("Expected second test post URL in sitemap")
	}

	// Should contain lastmod dates
	if !strings.Contains(body, "<lastmod>") {
		t.Errorf("Expected lastmod dates in sitemap. Body: %s", body)
	}
}

func TestServeSitemapHEAD(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Test HEAD request
	req, err := http.NewRequest("HEAD", "/sitemap.xml", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.serveSitemap)
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Body should be empty for HEAD request
	if body := rr.Body.String(); body != "" {
		t.Errorf("Expected empty body for HEAD request, got: %s", body)
	}
}

func TestServeSitemapMethodNotAllowed(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Test POST request (not allowed)
	req, err := http.NewRequest("POST", "/sitemap.xml", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.serveSitemap)
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("Expected status code %d, got %d", http.StatusMethodNotAllowed, status)
	}
}

func TestServeRobotsTxt(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Test GET request
	req, err := http.NewRequest("GET", "/robots.txt", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.serveRobotsTxt)
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Check content type
	expectedContentType := "text/plain; charset=utf-8"
	if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("Expected content type %s, got %s", expectedContentType, contentType)
	}

	// Check cache control header
	if cacheControl := rr.Header().Get("Cache-Control"); cacheControl != "public, max-age=86400" {
		t.Errorf("Expected cache control 'public, max-age=86400', got '%s'", cacheControl)
	}

	// Check robots.txt content
	body := rr.Body.String()
	
	// Should contain User-agent directive
	if !strings.Contains(body, "User-agent: *") {
		t.Error("Expected User-agent directive in robots.txt")
	}

	// Should contain Allow directive
	if !strings.Contains(body, "Allow: /") {
		t.Error("Expected Allow directive in robots.txt")
	}

	// Should contain Disallow directives for admin paths
	disallowedPaths := []string{"/login", "/logout", "/create", "/update", "/delete", "/auth-callback", "/api/", "/upload-file"}
	for _, path := range disallowedPaths {
		if !strings.Contains(body, "Disallow: "+path) {
			t.Errorf("Expected Disallow directive for %s in robots.txt", path)
		}
	}

	// Should contain sitemap reference
	if !strings.Contains(body, "Sitemap: http://localhost:8080/sitemap.xml") {
		t.Error("Expected sitemap reference in robots.txt")
	}
}

func TestServeRobotsTxtHEAD(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Test HEAD request
	req, err := http.NewRequest("HEAD", "/robots.txt", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.serveRobotsTxt)
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Body should be empty for HEAD request
	if body := rr.Body.String(); body != "" {
		t.Errorf("Expected empty body for HEAD request, got: %s", body)
	}
}

func TestServeRobotsTxtMethodNotAllowed(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Test POST request (not allowed)
	req, err := http.NewRequest("POST", "/robots.txt", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.serveRobotsTxt)
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("Expected status code %d, got %d", http.StatusMethodNotAllowed, status)
	}
}

func TestGetAllPostsForSitemap(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Create test posts - some with slugs, some without
	testData := []struct {
		title string
		body  string
		slug  string
	}{
		{"Post with Slug", "Content 1", "post-with-slug"},
		{"Another Post", "Content 2", "another-post"},
		{"Post without Slug", "Content 3", ""}, // This should be excluded
	}

	for _, data := range testData {
		var query string
		var args []interface{}
		
		if data.slug != "" {
			query = `INSERT INTO posts (title, body, datepost, slug, created_at, updated_at) VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`
			args = []interface{}{data.title, data.body, "Mon Jan 2 15:04:05 2006", data.slug}
		} else {
			query = `INSERT INTO posts (title, body, datepost, created_at, updated_at) VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`
			args = []interface{}{data.title, data.body, "Mon Jan 2 15:04:05 2006"}
		}
		
		_, err := app.DB.Exec(query, args...)
		if err != nil {
			t.Fatalf("Failed to insert test post: %v", err)
		}
	}

	// Get posts for sitemap
	posts, err := app.getAllPostsForSitemap()
	if err != nil {
		t.Fatalf("Failed to get posts for sitemap: %v", err)
	}

	// Should only return posts with slugs
	if len(posts) != 2 {
		t.Errorf("Expected 2 posts with slugs, got %d", len(posts))
	}

	// Check that all returned posts have slugs
	for _, post := range posts {
		if post.Slug == "" {
			t.Error("Expected all posts to have slugs")
		}
	}

	// Check specific posts
	slugs := make(map[string]bool)
	for _, post := range posts {
		slugs[post.Slug] = true
	}

	if !slugs["post-with-slug"] {
		t.Error("Expected 'post-with-slug' to be in results")
	}

	if !slugs["another-post"] {
		t.Error("Expected 'another-post' to be in results")
	}
}

func TestSEOServiceIntegration(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Test that SEO service is properly initialized
	if app.SEOService == nil {
		t.Error("Expected SEO service to be initialized")
	}

	// Test SEO service functionality with a real post
	post := &model.Post{
		ID:        1,
		Title:     "Integration Test Post",
		Body:      "This is a test post for SEO integration testing.",
		Slug:      "integration-test-post",
		CreatedAt: "Mon Jan 2 15:04:05 2006",
		UpdatedAt: "Mon Jan 2 15:04:05 2006",
	}

	// Test meta tags generation
	metaTags := app.SEOService.GenerateMetaTags(post)
	if metaTags["title"] != "Integration Test Post" {
		t.Errorf("Expected title 'Integration Test Post', got '%s'", metaTags["title"])
	}

	// Test canonical URL generation
	canonicalURL := app.SEOService.GetCanonicalURL(post)
	expectedURL := "http://localhost:8080/p/integration-test-post"
	if canonicalURL != expectedURL {
		t.Errorf("Expected canonical URL '%s', got '%s'", expectedURL, canonicalURL)
	}

	// Test structured data generation
	structuredData := app.SEOService.GenerateStructuredData(post)
	if structuredData == "" {
		t.Error("Expected structured data to be generated")
	}

	// Test Open Graph tags generation
	ogTags := app.SEOService.GenerateOpenGraphTags(post)
	if ogTags["og:title"] != "Integration Test Post" {
		t.Errorf("Expected og:title 'Integration Test Post', got '%s'", ogTags["og:title"])
	}
}

func TestPostHandlerSEOIntegration(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Create a test post
	_, err := app.DB.Exec(`
		INSERT INTO posts (title, body, datepost, slug, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?)
	`, "SEO Test Post", "This is a test post for SEO integration.", "Mon Jan 2 15:04:05 2006", "seo-test-post", "Mon Jan 2 15:04:05 2006", "Mon Jan 2 15:04:05 2006")
	if err != nil {
		t.Fatalf("Failed to insert test post: %v", err)
	}

	// Test slug-based URL
	req, err := http.NewRequest("GET", "/p/seo-test-post", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.getPostBySlug)
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Check canonical URL header
	linkHeader := rr.Header().Get("Link")
	expectedLink := "<http://localhost:8080/p/seo-test-post>; rel=\"canonical\""
	if linkHeader != expectedLink {
		t.Errorf("Expected Link header '%s', got '%s'", expectedLink, linkHeader)
	}

	// Check response body contains SEO elements
	body := rr.Body.String()
	
	// Should contain title
	if !strings.Contains(body, "<title>SEO Test Post</title>") {
		t.Error("Expected title tag in response")
	}

	// Should contain meta description
	if !strings.Contains(body, `<meta name="description"`) {
		t.Error("Expected meta description in response")
	}

	// Should contain canonical link
	if !strings.Contains(body, `<link rel="canonical" href="http://localhost:8080/p/seo-test-post">`) {
		t.Error("Expected canonical link in response")
	}

	// Should contain Open Graph tags
	if !strings.Contains(body, `<meta property="og:title" content="SEO Test Post">`) {
		t.Error("Expected Open Graph title in response")
	}

	if !strings.Contains(body, `<meta property="og:url" content="http://localhost:8080/p/seo-test-post">`) {
		t.Error("Expected Open Graph URL in response")
	}

	// Should contain structured data
	if !strings.Contains(body, `<script type="application/ld+json">`) {
		t.Error("Expected structured data script in response")
	}

	if !strings.Contains(body, `\"@type\": \"BlogPosting\"`) {
		t.Error("Expected BlogPosting structured data in response")
	}
}

func TestPostHandlerIDBasedSEOIntegration(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Create a test post
	result, err := app.DB.Exec(`
		INSERT INTO posts (title, body, datepost, slug, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?)
	`, "ID-based SEO Test", "This is a test post for ID-based SEO integration.", "Mon Jan 2 15:04:05 2006", "id-based-seo-test", "Mon Jan 2 15:04:05 2006", "Mon Jan 2 15:04:05 2006")
	if err != nil {
		t.Fatalf("Failed to insert test post: %v", err)
	}

	// Get the ID of the inserted post
	postID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get post ID: %v", err)
	}

	// Test ID-based URL
	req, err := http.NewRequest("GET", fmt.Sprintf("/post?id=%d", postID), nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.getPost)
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Check canonical URL header (should point to slug-based URL)
	linkHeader := rr.Header().Get("Link")
	expectedLink := "<http://localhost:8080/p/id-based-seo-test>; rel=\"canonical\""
	if linkHeader != expectedLink {
		t.Errorf("Expected Link header '%s', got '%s'", expectedLink, linkHeader)
	}

	// Check response body contains SEO elements
	body := rr.Body.String()
	
	// Should contain title
	if !strings.Contains(body, "<title>ID-based SEO Test</title>") {
		t.Error("Expected title tag in response")
	}

	// Should contain canonical link pointing to slug URL
	if !strings.Contains(body, `<link rel="canonical" href="http://localhost:8080/p/id-based-seo-test">`) {
		t.Error("Expected canonical link pointing to slug URL in response")
	}
}

func TestCoursesAndLinksRemoval(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Test that courses URLs return 404
	coursesURLs := []string{
		"/courses",
		"/courses/",
		"/courses/programming",
		"/api/courses",
	}

	for _, url := range coursesURLs {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := app.Router
		handler.ServeHTTP(rr, req)

		// Should return 404 for all courses-related URLs
		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Expected status code %d for URL %s, got %d", http.StatusNotFound, url, status)
		}
	}

	// Test that links URLs return 404
	linksURLs := []string{
		"/links",
		"/links/",
		"/links/useful",
		"/api/links",
	}

	for _, url := range linksURLs {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := app.Router
		handler.ServeHTTP(rr, req)

		// Should return 404 for all links-related URLs
		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Expected status code %d for URL %s, got %d", http.StatusNotFound, url, status)
		}
	}
}

func TestNavigationDoesNotContainCoursesOrLinks(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Test homepage to check navigation
	req, err := http.NewRequest("GET", "/page?p=0", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.getPage)
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	body := rr.Body.String()

	// Navigation should not contain courses or links
	if strings.Contains(strings.ToLower(body), "courses") {
		t.Error("Navigation should not contain 'courses' links")
	}

	if strings.Contains(strings.ToLower(body), "links") && !strings.Contains(strings.ToLower(body), "canonical") {
		// Allow "links" in context of canonical links, but not navigation links
		if strings.Contains(body, `href="/links"`) || strings.Contains(body, `href="/courses"`) {
			t.Error("Navigation should not contain 'links' or 'courses' navigation items")
		}
	}
}

func TestSitemapAutomaticUpdates(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Create initial post
	_, err := app.DB.Exec(`
		INSERT INTO posts (title, body, datepost, slug, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?)
	`, "Initial Post", "Initial content", "Mon Jan 1 15:04:05 2006", "initial-post", "Mon Jan 1 15:04:05 2006", "Mon Jan 1 15:04:05 2006")
	if err != nil {
		t.Fatalf("Failed to insert initial post: %v", err)
	}

	// Get initial sitemap
	req, err := http.NewRequest("GET", "/sitemap.xml", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.serveSitemap)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	initialSitemap := rr.Body.String()

	// Should contain initial post
	if !strings.Contains(initialSitemap, "<loc>http://localhost:8080/p/initial-post</loc>") {
		t.Error("Initial sitemap should contain initial post")
	}

	// Add another post
	_, err = app.DB.Exec(`
		INSERT INTO posts (title, body, datepost, slug, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?)
	`, "New Post", "New content", "Mon Jan 2 15:04:05 2006", "new-post", "Mon Jan 2 15:04:05 2006", "Mon Jan 2 15:04:05 2006")
	if err != nil {
		t.Fatalf("Failed to insert new post: %v", err)
	}

	// Get updated sitemap
	req, err = http.NewRequest("GET", "/sitemap.xml", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	updatedSitemap := rr.Body.String()

	// Should contain both posts
	if !strings.Contains(updatedSitemap, "<loc>http://localhost:8080/p/initial-post</loc>") {
		t.Error("Updated sitemap should still contain initial post")
	}

	if !strings.Contains(updatedSitemap, "<loc>http://localhost:8080/p/new-post</loc>") {
		t.Error("Updated sitemap should contain new post")
	}

	// Delete a post
	_, err = app.DB.Exec("DELETE FROM posts WHERE slug = ?", "initial-post")
	if err != nil {
		t.Fatalf("Failed to delete post: %v", err)
	}

	// Get sitemap after deletion
	req, err = http.NewRequest("GET", "/sitemap.xml", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	finalSitemap := rr.Body.String()

	// Should not contain deleted post
	if strings.Contains(finalSitemap, "<loc>http://localhost:8080/p/initial-post</loc>") {
		t.Error("Final sitemap should not contain deleted post")
	}

	// Should still contain remaining post
	if !strings.Contains(finalSitemap, "<loc>http://localhost:8080/p/new-post</loc>") {
		t.Error("Final sitemap should still contain remaining post")
	}
}

func TestCanonicalURLRedirects(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Create a test post
	result, err := app.DB.Exec(`
		INSERT INTO posts (title, body, datepost, slug, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?)
	`, "Redirect Test Post", "Content for redirect testing", "Mon Jan 2 15:04:05 2006", "redirect-test-post", "Mon Jan 2 15:04:05 2006", "Mon Jan 2 15:04:05 2006")
	if err != nil {
		t.Fatalf("Failed to insert test post: %v", err)
	}

	postID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get post ID: %v", err)
	}

	// Test that ID-based URL redirects to slug-based URL
	req, err := http.NewRequest("GET", fmt.Sprintf("/post?id=%d", postID), nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := app.Router
	handler.ServeHTTP(rr, req)

	// Should get a 301 redirect (if redirect middleware is implemented)
	// For now, we test that canonical headers are set correctly
	if status := rr.Code; status == http.StatusMovedPermanently {
		// Check redirect location
		location := rr.Header().Get("Location")
		expectedLocation := "/p/redirect-test-post"
		if location != expectedLocation {
			t.Errorf("Expected redirect to '%s', got '%s'", expectedLocation, location)
		}
	} else if status == http.StatusOK {
		// If not redirecting, at least check canonical URL header
		linkHeader := rr.Header().Get("Link")
		expectedLink := "<http://localhost:8080/p/redirect-test-post>; rel=\"canonical\""
		if linkHeader != expectedLink {
			t.Errorf("Expected Link header '%s', got '%s'", expectedLink, linkHeader)
		}
	} else {
		t.Errorf("Expected either redirect (301) or OK (200), got %d", status)
	}
}

func TestSEOHeadersInResponses(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Create a test post
	_, err := app.DB.Exec(`
		INSERT INTO posts (title, body, datepost, slug, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?)
	`, "SEO Headers Test", "Content for SEO headers testing", "Mon Jan 2 15:04:05 2006", "seo-headers-test", "Mon Jan 2 15:04:05 2006", "Mon Jan 2 15:04:05 2006")
	if err != nil {
		t.Fatalf("Failed to insert test post: %v", err)
	}

	// Test slug-based URL
	req, err := http.NewRequest("GET", "/p/seo-headers-test", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.getPostBySlug)
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	body := rr.Body.String()

	// Test all required SEO elements are present
	seoElements := []string{
		`<title>SEO Headers Test</title>`,
		`<meta name="description"`,
		`<link rel="canonical" href="http://localhost:8080/p/seo-headers-test">`,
		`<meta property="og:title" content="SEO Headers Test">`,
		`<meta property="og:type" content="article">`,
		`<meta property="og:url" content="http://localhost:8080/p/seo-headers-test">`,
		`<meta property="twitter:card" content="summary_large_image">`,
		`<script type="application/ld+json">`,
		`\"@type\": \"BlogPosting\"`,
	}

	for _, element := range seoElements {
		if !strings.Contains(body, element) {
			t.Errorf("Expected response to contain SEO element: %s", element)
		}
	}

	// Test canonical URL header
	linkHeader := rr.Header().Get("Link")
	expectedLink := "<http://localhost:8080/p/seo-headers-test>; rel=\"canonical\""
	if linkHeader != expectedLink {
		t.Errorf("Expected Link header '%s', got '%s'", expectedLink, linkHeader)
	}
}

func TestSitemapCacheHeaders(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Test sitemap cache headers
	req, err := http.NewRequest("GET", "/sitemap.xml", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.serveSitemap)
	handler.ServeHTTP(rr, req)

	// Check cache control header
	cacheControl := rr.Header().Get("Cache-Control")
	expectedCacheControl := "public, max-age=3600"
	if cacheControl != expectedCacheControl {
		t.Errorf("Expected Cache-Control header '%s', got '%s'", expectedCacheControl, cacheControl)
	}

	// Check content type
	contentType := rr.Header().Get("Content-Type")
	expectedContentType := "application/xml; charset=utf-8"
	if contentType != expectedContentType {
		t.Errorf("Expected Content-Type header '%s', got '%s'", expectedContentType, contentType)
	}
}

func TestRobotsTxtCacheHeaders(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Test robots.txt cache headers
	req, err := http.NewRequest("GET", "/robots.txt", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.serveRobotsTxt)
	handler.ServeHTTP(rr, req)

	// Check cache control header
	cacheControl := rr.Header().Get("Cache-Control")
	expectedCacheControl := "public, max-age=86400"
	if cacheControl != expectedCacheControl {
		t.Errorf("Expected Cache-Control header '%s', got '%s'", expectedCacheControl, cacheControl)
	}

	// Check content type
	contentType := rr.Header().Get("Content-Type")
	expectedContentType := "text/plain; charset=utf-8"
	if contentType != expectedContentType {
		t.Errorf("Expected Content-Type header '%s', got '%s'", expectedContentType, contentType)
	}
}

// TestComprehensiveSEOFunctionality tests comprehensive SEO features
func TestComprehensiveSEOFunctionality(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Create test posts with various content types
	testPosts := []struct {
		title, body, slug string
	}{
		{
			"SEO Test Post with Images",
			"This post contains <img src='/test-image.jpg' alt='Test Image'> and [file:document.pdf] references.",
			"seo-test-post-images",
		},
		{
			"Post with Special Characters & Symbols",
			"This post has special characters: <>&\"' and unicode: 你好",
			"post-special-characters-symbols",
		},
		{
			"Very Long Title That Should Be Handled Properly in Meta Tags and Structured Data Without Breaking",
			strings.Repeat("This is a very long content that should be truncated properly in meta descriptions. ", 10),
			"very-long-title-handled-properly",
		},
	}

	// Insert test posts
	for _, post := range testPosts {
		_, err := app.DB.Exec(`
			INSERT INTO posts (title, body, datepost, slug, created_at, updated_at) 
			VALUES (?, ?, ?, ?, ?, ?)
		`, post.title, post.body, "Mon Jan 2 15:04:05 2006", post.slug, "Mon Jan 2 15:04:05 2006", "Mon Jan 2 15:04:05 2006")
		if err != nil {
			t.Fatalf("Failed to insert test post: %v", err)
		}
	}

	// Test each post's SEO implementation
	for _, post := range testPosts {
		t.Run("SEO_"+post.slug, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/p/"+post.slug, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(app.getPostBySlug)
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != http.StatusOK {
				t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
			}

			body := rr.Body.String()

			// Test canonical URL header
			linkHeader := rr.Header().Get("Link")
			expectedLink := fmt.Sprintf("<http://localhost:8080/p/%s>; rel=\"canonical\"", post.slug)
			if linkHeader != expectedLink {
				t.Errorf("Expected Link header '%s', got '%s'", expectedLink, linkHeader)
			}

			// Test HTML meta tags
			requiredMetaTags := []string{
				`<title>`, // Just check for title tag presence
				`<meta name="description"`,
				`<meta name="author" content="Blog Author">`,
				fmt.Sprintf(`<link rel="canonical" href="http://localhost:8080/p/%s">`, post.slug),
			}

			for _, tag := range requiredMetaTags {
				if !strings.Contains(body, tag) {
					t.Errorf("Expected response to contain meta tag: %s", tag)
				}
			}

			// Test Open Graph tags
			requiredOGTags := []string{
				`<meta property="og:type" content="article">`,
				`<meta property="og:title"`, // Just check for presence
				fmt.Sprintf(`<meta property="og:url" content="http://localhost:8080/p/%s">`, post.slug),
				`<meta property="og:site_name" content="Blog">`,
				`<meta property="og:description"`,
			}

			for _, tag := range requiredOGTags {
				if !strings.Contains(body, tag) {
					t.Errorf("Expected response to contain OG tag: %s", tag)
				}
			}

			// Test Twitter Card tags
			requiredTwitterTags := []string{
				`<meta property="twitter:card" content="summary_large_image">`,
				`<meta property="twitter:title"`, // Just check for presence
				`<meta property="twitter:description"`,
			}

			for _, tag := range requiredTwitterTags {
				if !strings.Contains(body, tag) {
					t.Errorf("Expected response to contain Twitter tag: %s", tag)
				}
			}

			// Test structured data
			if !strings.Contains(body, `<script type="application/ld+json">`) {
				t.Error("Expected structured data script tag")
			}

			if !strings.Contains(body, `\"@type\": \"BlogPosting\"`) {
				t.Error("Expected BlogPosting structured data")
			}

			// Test that content is properly escaped
			if strings.Contains(body, `content="<script>`) {
				t.Error("Meta tag content should not contain unescaped script tags")
			}
		})
	}
}

// TestSEOWithFileReferences tests SEO functionality with file references
func TestSEOWithFileReferences(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Create files table and insert test files
	_, err := app.DB.Exec(`CREATE TABLE IF NOT EXISTS files (
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
		t.Fatalf("Failed to create files table: %v", err)
	}

	// Insert test image file
	_, err = app.DB.Exec(`INSERT INTO files (uuid, original_name, stored_name, path, size, mime_type, is_image, width, height, alt_text) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"test-image-uuid", "test-image.jpg", "stored-test-image.jpg", "/path/to/image", 1024, "image/jpeg", true, 800, 600, "Test image alt text")
	if err != nil {
		t.Fatalf("Failed to insert test image: %v", err)
	}

	// Insert test document file
	_, err = app.DB.Exec(`INSERT INTO files (uuid, original_name, stored_name, path, size, mime_type, is_image) 
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"test-doc-uuid", "document.pdf", "stored-document.pdf", "/path/to/doc", 2048, "application/pdf", false)
	if err != nil {
		t.Fatalf("Failed to insert test document: %v", err)
	}

	// Create post with file references
	_, err = app.DB.Exec(`
		INSERT INTO posts (title, body, datepost, slug, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?)
	`, "Post with Files", "This post has an image [file:test-image.jpg] and a document [file:document.pdf].", "Mon Jan 2 15:04:05 2006", "post-with-files", "Mon Jan 2 15:04:05 2006", "Mon Jan 2 15:04:05 2006")
	if err != nil {
		t.Fatalf("Failed to insert test post: %v", err)
	}

	// Test post with file references
	req, err := http.NewRequest("GET", "/p/post-with-files", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.getPostBySlug)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	body := rr.Body.String()

	// Test that structured data includes image (check for escaped JSON)
	if !strings.Contains(body, `\"image\"`) && !strings.Contains(body, `"image"`) {
		t.Error("Expected structured data to include image information")
	}

	// Test that Open Graph includes image
	if !strings.Contains(body, `<meta property="og:image"`) {
		t.Error("Expected Open Graph image tag")
	}

	// Test that Twitter Card includes image
	if !strings.Contains(body, `<meta property="twitter:image"`) {
		t.Error("Expected Twitter Card image tag")
	}
}

// TestSEOErrorHandling tests SEO error handling scenarios
func TestSEOErrorHandling(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	t.Run("PostWithoutSlug", func(t *testing.T) {
		// Insert post without slug
		result, err := app.DB.Exec(`
			INSERT INTO posts (title, body, datepost, created_at, updated_at) 
			VALUES (?, ?, ?, ?, ?)
		`, "Post Without Slug", "This post has no slug", "Mon Jan 2 15:04:05 2006", "Mon Jan 2 15:04:05 2006", "Mon Jan 2 15:04:05 2006")
		if err != nil {
			t.Fatalf("Failed to insert test post: %v", err)
		}

		postID, err := result.LastInsertId()
		if err != nil {
			t.Fatalf("Failed to get post ID: %v", err)
		}

		// Test ID-based access
		req, err := http.NewRequest("GET", fmt.Sprintf("/post?id=%d", postID), nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(app.getPost)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
		}

		// Should still have canonical URL (fallback to ID-based)
		linkHeader := rr.Header().Get("Link")
		expectedLink := fmt.Sprintf("<http://localhost:8080/post?id=%d>; rel=\"canonical\"", postID)
		if linkHeader != expectedLink {
			t.Errorf("Expected fallback canonical Link header '%s', got '%s'", expectedLink, linkHeader)
		}
	})

	t.Run("SitemapWithDatabaseError", func(t *testing.T) {
		// Close database to simulate error
		app.DB.Close()

		req, err := http.NewRequest("GET", "/sitemap.xml", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(app.serveSitemap)
		handler.ServeHTTP(rr, req)

		// Should return 500 error
		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, status)
		}
	})
}

// TestCoursesAndLinksCompleteRemoval tests comprehensive removal of courses and links
func TestCoursesAndLinksCompleteRemoval(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Test comprehensive list of courses and links URLs
	testURLs := []struct {
		url          string
		description  string
	}{
		{"/courses", "courses root"},
		{"/courses/", "courses root with slash"},
		{"/courses/programming", "courses subcategory"},
		{"/courses/web-development", "courses subcategory"},
		{"/courses/index.html", "courses index file"},
		{"/courses.html", "courses HTML file"},
		{"/courses.json", "courses JSON API"},
		{"/api/courses", "courses API endpoint"},
		{"/api/courses/", "courses API endpoint with slash"},
		{"/api/courses/1", "specific course API"},
		{"/links", "links root"},
		{"/links/", "links root with slash"},
		{"/links/useful", "links subcategory"},
		{"/links/resources", "links subcategory"},
		{"/links/index.html", "links index file"},
		{"/links.html", "links HTML file"},
		{"/links.json", "links JSON API"},
		{"/api/links", "links API endpoint"},
		{"/api/links/", "links API endpoint with slash"},
		{"/api/links/1", "specific link API"},
	}

	for _, testURL := range testURLs {
		t.Run("URL_"+testURL.url, func(t *testing.T) {
			// Test GET request
			req, err := http.NewRequest("GET", testURL.url, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler := app.Router
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != http.StatusNotFound {
				t.Errorf("Expected status code %d for %s (%s), got %d", http.StatusNotFound, testURL.url, testURL.description, status)
			}

			// Test other HTTP methods
			for _, method := range []string{"POST", "PUT", "DELETE", "PATCH"} {
				req, err := http.NewRequest(method, testURL.url, nil)
				if err != nil {
					t.Fatal(err)
				}

				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)

				// Should return 404 or 405, but not 200
				if status := rr.Code; status == http.StatusOK {
					t.Errorf("Expected non-200 status for %s %s, got %d", method, testURL.url, status)
				}
			}
		})
	}
}

// TestNavigationCoursesLinksRemoval tests that navigation doesn't contain removed sections
func TestNavigationCoursesLinksRemoval(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Test various pages that might contain navigation
	pagesToTest := []string{
		"/page?p=0",
		"/about",
		"/login",
	}

	for _, page := range pagesToTest {
		t.Run("Navigation_"+page, func(t *testing.T) {
			req, err := http.NewRequest("GET", page, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler := app.Router
			handler.ServeHTTP(rr, req)

			// Skip if page returns error (some pages might require auth)
			if rr.Code >= 400 {
				return
			}

			body := strings.ToLower(rr.Body.String())

			// Check for navigation links to removed sections
			prohibitedLinks := []string{
				`href="/courses"`,
				`href="/courses/"`,
				`href="/links"`,
				`href="/links/"`,
			}

			for _, link := range prohibitedLinks {
				if strings.Contains(body, link) {
					t.Errorf("Page %s should not contain navigation link: %s", page, link)
				}
			}

			// Check for menu text (but allow legitimate uses)
			if strings.Contains(body, ">courses<") || strings.Contains(body, ">links<") {
				// Only fail if it's clearly navigation context
				if strings.Contains(body, "nav") || strings.Contains(body, "menu") {
					t.Errorf("Page %s should not contain courses/links in navigation", page)
				}
			}
		})
	}
}

// TestSitemapAutomaticUpdatesComprehensive tests comprehensive sitemap update scenarios
func TestSitemapAutomaticUpdatesComprehensive(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Helper function to get sitemap content
	getSitemap := func() string {
		req, err := http.NewRequest("GET", "/sitemap.xml", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(app.serveSitemap)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("Expected status code %d, got %d", http.StatusOK, status)
		}

		return rr.Body.String()
	}

	// Test initial state
	initialSitemap := getSitemap()
	if !strings.Contains(initialSitemap, "<loc>http://localhost:8080/</loc>") {
		t.Error("Initial sitemap should contain homepage")
	}

	// Test adding posts with different characteristics
	testPosts := []struct {
		title, body, slug string
	}{
		{"Post with Normal Slug", "Normal content", "post-normal-slug"},
		{"Post with Special Characters", "Content with special chars", "post-special-characters"},
		{"Post with Very Long Title That Should Be Handled Properly", "Long content", "post-very-long-title"},
	}

	for i, post := range testPosts {
		// Add post
		_, err := app.DB.Exec(`
			INSERT INTO posts (title, body, datepost, slug, created_at, updated_at) 
			VALUES (?, ?, ?, ?, ?, ?)
		`, post.title, post.body, "Mon Jan 2 15:04:05 2006", post.slug, "Mon Jan 2 15:04:05 2006", "Mon Jan 2 15:04:05 2006")
		if err != nil {
			t.Fatalf("Failed to insert test post %d: %v", i, err)
		}

		// Check sitemap includes new post
		sitemap := getSitemap()
		expectedURL := fmt.Sprintf("<loc>http://localhost:8080/p/%s</loc>", post.slug)
		if !strings.Contains(sitemap, expectedURL) {
			t.Errorf("Sitemap should contain new post URL: %s", expectedURL)
		}
	}

	// Test updating post (changing slug)
	_, err := app.DB.Exec(`UPDATE posts SET title = ?, slug = ?, updated_at = ? WHERE slug = ?`,
		"Updated Post Title", "updated-post-slug", "Mon Jan 3 15:04:05 2006", "post-normal-slug")
	if err != nil {
		t.Fatalf("Failed to update post: %v", err)
	}

	updatedSitemap := getSitemap()
	if strings.Contains(updatedSitemap, "<loc>http://localhost:8080/p/post-normal-slug</loc>") {
		t.Error("Sitemap should not contain old slug after update")
	}
	if !strings.Contains(updatedSitemap, "<loc>http://localhost:8080/p/updated-post-slug</loc>") {
		t.Error("Sitemap should contain new slug after update")
	}

	// Test deleting posts
	_, err = app.DB.Exec(`DELETE FROM posts WHERE slug = ?`, "post-special-characters")
	if err != nil {
		t.Fatalf("Failed to delete post: %v", err)
	}

	finalSitemap := getSitemap()
	if strings.Contains(finalSitemap, "<loc>http://localhost:8080/p/post-special-characters</loc>") {
		t.Error("Sitemap should not contain deleted post")
	}

	// Should still contain remaining posts
	if !strings.Contains(finalSitemap, "<loc>http://localhost:8080/p/updated-post-slug</loc>") {
		t.Error("Sitemap should still contain remaining posts")
	}
	if !strings.Contains(finalSitemap, "<loc>http://localhost:8080/p/post-very-long-title</loc>") {
		t.Error("Sitemap should still contain remaining posts")
	}

	// Test post without slug (should be excluded)
	_, err = app.DB.Exec(`
		INSERT INTO posts (title, body, datepost, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?)
	`, "Post Without Slug", "This post has no slug", "Mon Jan 4 15:04:05 2006", "Mon Jan 4 15:04:05 2006", "Mon Jan 4 15:04:05 2006")
	if err != nil {
		t.Fatalf("Failed to insert post without slug: %v", err)
	}

	noSlugSitemap := getSitemap()
	if strings.Contains(noSlugSitemap, "Post Without Slug") {
		t.Error("Sitemap should not contain posts without slugs")
	}
}