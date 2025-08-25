package integration

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/ultramozg/golang-blog-engine/testutils"
)

// TestSEOIntegration tests comprehensive SEO functionality
func TestSEOIntegration(t *testing.T) {
	runner := testutils.NewTestRunner(t)
	defer runner.Close()

	if err := runner.SetupTest(); err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}

	t.Run("SitemapGeneration", func(t *testing.T) {
		testSitemapGeneration(t, runner)
	})

	t.Run("RobotsTxtGeneration", func(t *testing.T) {
		testRobotsTxtGeneration(t, runner)
	})

	t.Run("CanonicalURLs", func(t *testing.T) {
		testCanonicalURLs(t, runner)
	})

	t.Run("SEOMetaTags", func(t *testing.T) {
		testSEOMetaTags(t, runner)
	})

	t.Run("StructuredData", func(t *testing.T) {
		testStructuredData(t, runner)
	})

	t.Run("OpenGraphTags", func(t *testing.T) {
		testOpenGraphTags(t, runner)
	})
}

// TestContentRemovalIntegration tests that courses and links sections are completely removed
func TestContentRemovalIntegration(t *testing.T) {
	runner := testutils.NewTestRunner(t)
	defer runner.Close()

	if err := runner.SetupTest(); err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}

	t.Run("CoursesURLsReturn404", func(t *testing.T) {
		testCoursesURLsReturn404(t, runner)
	})

	t.Run("LinksURLsReturn404", func(t *testing.T) {
		testLinksURLsReturn404(t, runner)
	})

	t.Run("NavigationDoesNotContainRemovedSections", func(t *testing.T) {
		testNavigationDoesNotContainRemovedSections(t, runner)
	})

	t.Run("TemplatesDoNotRenderRemovedContent", func(t *testing.T) {
		testTemplatesDoNotRenderRemovedContent(t, runner)
	})
}

func testSitemapGeneration(t *testing.T, runner *testutils.TestRunner) {
	// Test sitemap.xml endpoint
	resp, err := runner.HTTP.MakeRequest("GET", "/sitemap.xml", "", nil)
	if err != nil {
		t.Fatalf("Failed to request sitemap: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertStatusCode(t, resp, http.StatusOK)

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/xml; charset=utf-8" {
		t.Errorf("Expected content type 'application/xml; charset=utf-8', got '%s'", contentType)
	}

	// Check cache headers
	cacheControl := resp.Header.Get("Cache-Control")
	if cacheControl != "public, max-age=3600" {
		t.Errorf("Expected cache control 'public, max-age=3600', got '%s'", cacheControl)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read sitemap body: %v", err)
	}

	sitemapContent := string(body)

	// Test XML structure
	testutils.AssertContains(t, sitemapContent, `<?xml version="1.0" encoding="UTF-8"?>`)
	testutils.AssertContains(t, sitemapContent, `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	testutils.AssertContains(t, sitemapContent, "</urlset>")

	// Test homepage entry
	testutils.AssertContains(t, sitemapContent, "<loc>http://localhost:8080/</loc>")
	testutils.AssertContains(t, sitemapContent, "<priority>1.0</priority>")

	// Test that test posts are included with slug-based URLs
	testutils.AssertContains(t, sitemapContent, "<loc>http://localhost:8080/p/test-post-1</loc>")
	testutils.AssertContains(t, sitemapContent, "<loc>http://localhost:8080/p/test-post-2</loc>")
	testutils.AssertContains(t, sitemapContent, "<loc>http://localhost:8080/p/test-post-3</loc>")

	// Test that changefreq and priority are included
	testutils.AssertContains(t, sitemapContent, "<changefreq>weekly</changefreq>")
	testutils.AssertContains(t, sitemapContent, "<priority>0.8</priority>")

	// Note: lastmod may not be present if dates aren't in the expected format

	// Test HEAD request
	resp, err = runner.HTTP.MakeRequest("HEAD", "/sitemap.xml", "", nil)
	if err != nil {
		t.Fatalf("Failed to make HEAD request to sitemap: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertStatusCode(t, resp, http.StatusOK)

	// Body should be empty for HEAD request
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read HEAD response body: %v", err)
	}
	if len(body) != 0 {
		t.Error("Expected empty body for HEAD request")
	}
}

func testRobotsTxtGeneration(t *testing.T, runner *testutils.TestRunner) {
	// Test robots.txt endpoint
	resp, err := runner.HTTP.MakeRequest("GET", "/robots.txt", "", nil)
	if err != nil {
		t.Fatalf("Failed to request robots.txt: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertStatusCode(t, resp, http.StatusOK)

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/plain; charset=utf-8" {
		t.Errorf("Expected content type 'text/plain; charset=utf-8', got '%s'", contentType)
	}

	// Check cache headers
	cacheControl := resp.Header.Get("Cache-Control")
	if cacheControl != "public, max-age=86400" {
		t.Errorf("Expected cache control 'public, max-age=86400', got '%s'", cacheControl)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read robots.txt body: %v", err)
	}

	robotsContent := string(body)

	// Test required directives
	testutils.AssertContains(t, robotsContent, "User-agent: *")
	testutils.AssertContains(t, robotsContent, "Allow: /")

	// Test disallowed paths
	disallowedPaths := []string{
		"Disallow: /login",
		"Disallow: /logout",
		"Disallow: /create",
		"Disallow: /update",
		"Disallow: /delete",
		"Disallow: /auth-callback",
		"Disallow: /api/",
		"Disallow: /upload-file",
	}

	for _, disallow := range disallowedPaths {
		testutils.AssertContains(t, robotsContent, disallow)
	}

	// Test sitemap reference
	expectedSitemap := "Sitemap: http://localhost:8080/sitemap.xml"
	testutils.AssertContains(t, robotsContent, expectedSitemap)

	// Test HEAD request
	resp, err = runner.HTTP.MakeRequest("HEAD", "/robots.txt", "", nil)
	if err != nil {
		t.Fatalf("Failed to make HEAD request to robots.txt: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertStatusCode(t, resp, http.StatusOK)
}

func testCanonicalURLs(t *testing.T, runner *testutils.TestRunner) {
	// Test canonical URLs in slug-based post access
	resp, err := runner.HTTP.MakeRequest("GET", "/p/test-post-1", "", nil)
	if err != nil {
		t.Fatalf("Failed to request post by slug: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertStatusCode(t, resp, http.StatusOK)

	// Check canonical URL header
	linkHeader := resp.Header.Get("Link")
	expectedLink := "<http://localhost:8080/p/test-post-1>; rel=\"canonical\""
	if linkHeader != expectedLink {
		t.Errorf("Expected Link header '%s', got '%s'", expectedLink, linkHeader)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read post body: %v", err)
	}

	bodyStr := string(body)

	// Check canonical link in HTML
	expectedCanonicalLink := `<link rel="canonical" href="http://localhost:8080/p/test-post-1">`
	testutils.AssertContains(t, bodyStr, expectedCanonicalLink)

	// Test ID-based URL canonical headers
	// First, get the post ID for test-post-1
	var postID int
	err = runner.DB.DB.QueryRow("SELECT id FROM posts WHERE slug = ?", "test-post-1").Scan(&postID)
	if err != nil {
		t.Fatalf("Failed to get post ID: %v", err)
	}

	resp, err = runner.HTTP.MakeRequest("GET", fmt.Sprintf("/post?id=%d", postID), "", nil)
	if err != nil {
		t.Fatalf("Failed to request post by ID: %v", err)
	}
	defer resp.Body.Close()

	// Should still return OK (or redirect if middleware is implemented)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusMovedPermanently {
		t.Errorf("Expected status 200 or 301, got %d", resp.StatusCode)
	}

	if resp.StatusCode == http.StatusOK {
		// Check that canonical URL still points to slug-based URL
		linkHeader = resp.Header.Get("Link")
		if linkHeader != expectedLink {
			t.Errorf("Expected canonical Link header '%s' for ID-based URL, got '%s'", expectedLink, linkHeader)
		}
	}
}

func testSEOMetaTags(t *testing.T, runner *testutils.TestRunner) {
	// Test SEO meta tags in post response
	resp, err := runner.HTTP.MakeRequest("GET", "/p/test-post-1", "", nil)
	if err != nil {
		t.Fatalf("Failed to request post: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertStatusCode(t, resp, http.StatusOK)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read post body: %v", err)
	}

	bodyStr := string(body)

	// Test required meta tags
	testutils.AssertContains(t, bodyStr, `<title>Test Post 1</title>`)
	testutils.AssertContains(t, bodyStr, `<meta name="description"`)
	testutils.AssertContains(t, bodyStr, `<meta name="author" content="Blog Author">`)

	// Test that meta description is properly extracted and escaped
	if strings.Contains(bodyStr, `content=""`) {
		t.Error("Meta description should not be empty")
	}

	// Test that meta tags don't contain dangerous content (but allow structured data script)
	// Check that meta tag content doesn't contain unescaped script tags
	metaTagRegex := `<meta[^>]+content="[^"]*<script[^"]*"`
	if matched, _ := regexp.MatchString(metaTagRegex, bodyStr); matched {
		t.Error("Meta tags should not contain unescaped script tags in content")
	}
}

func testStructuredData(t *testing.T, runner *testutils.TestRunner) {
	// Test structured data in post response
	resp, err := runner.HTTP.MakeRequest("GET", "/p/test-post-1", "", nil)
	if err != nil {
		t.Fatalf("Failed to request post: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertStatusCode(t, resp, http.StatusOK)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read post body: %v", err)
	}

	bodyStr := string(body)

	// Test structured data script tag
	testutils.AssertContains(t, bodyStr, `<script type="application/ld+json">`)

	// Test structured data content (it's JSON-encoded in the script tag)
	testutils.AssertContains(t, bodyStr, `\"@context\": \"https://schema.org\"`)
	testutils.AssertContains(t, bodyStr, `\"@type\": \"BlogPosting\"`)
	testutils.AssertContains(t, bodyStr, `\"headline\": \"Test Post 1\"`)
	testutils.AssertContains(t, bodyStr, `\"author\"`)
	testutils.AssertContains(t, bodyStr, `\"publisher\"`)
	testutils.AssertContains(t, bodyStr, `\"mainEntityOfPage\"`)

	// Test that structured data contains date information
	if !strings.Contains(bodyStr, `\"datePublished\"`) && !strings.Contains(bodyStr, `\"dateModified\"`) {
		t.Error("Structured data should contain date information")
	}
}

func testOpenGraphTags(t *testing.T, runner *testutils.TestRunner) {
	// Test Open Graph tags in post response
	resp, err := runner.HTTP.MakeRequest("GET", "/p/test-post-1", "", nil)
	if err != nil {
		t.Fatalf("Failed to request post: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertStatusCode(t, resp, http.StatusOK)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read post body: %v", err)
	}

	bodyStr := string(body)

	// Test required Open Graph tags
	testutils.AssertContains(t, bodyStr, `<meta property="og:type" content="article">`)
	testutils.AssertContains(t, bodyStr, `<meta property="og:title" content="Test Post 1">`)
	testutils.AssertContains(t, bodyStr, `<meta property="og:url" content="http://localhost:8080/p/test-post-1">`)
	testutils.AssertContains(t, bodyStr, `<meta property="og:site_name" content="Blog">`)
	testutils.AssertContains(t, bodyStr, `<meta property="og:description"`)

	// Test article-specific tags
	testutils.AssertContains(t, bodyStr, `<meta property="article:author" content="Blog Author">`)

	// Test Twitter Card tags (they use property attribute, not name)
	testutils.AssertContains(t, bodyStr, `<meta property="twitter:card" content="summary_large_image">`)
	testutils.AssertContains(t, bodyStr, `<meta property="twitter:title" content="Test Post 1">`)
	testutils.AssertContains(t, bodyStr, `<meta property="twitter:description"`)
}

func testCoursesURLsReturn404(t *testing.T, runner *testutils.TestRunner) {
	coursesURLs := []string{
		"/courses",
		"/courses/",
		"/courses/programming",
		"/courses/web-development",
		"/api/courses",
		"/courses.html",
		"/courses.json",
	}

	for _, url := range coursesURLs {
		resp, err := runner.HTTP.MakeRequest("GET", url, "", nil)
		if err != nil {
			t.Fatalf("Failed to request %s: %v", url, err)
		}
		defer resp.Body.Close()

		testutils.AssertStatusCode(t, resp, http.StatusNotFound)

		// Test different HTTP methods
		for _, method := range []string{"POST", "PUT", "DELETE"} {
			resp, err := runner.HTTP.MakeRequest(method, url, "", nil)
			if err != nil {
				t.Fatalf("Failed to make %s request to %s: %v", method, url, err)
			}
			defer resp.Body.Close()

			// Should return 404 or 405 (Method Not Allowed), but not 200
			if resp.StatusCode == http.StatusOK {
				t.Errorf("Expected non-200 status for %s %s, got %d", method, url, resp.StatusCode)
			}
		}
	}
}

func testLinksURLsReturn404(t *testing.T, runner *testutils.TestRunner) {
	linksURLs := []string{
		"/links",
		"/links/",
		"/links/useful",
		"/links/resources",
		"/api/links",
		"/links.html",
		"/links.json",
	}

	for _, url := range linksURLs {
		resp, err := runner.HTTP.MakeRequest("GET", url, "", nil)
		if err != nil {
			t.Fatalf("Failed to request %s: %v", url, err)
		}
		defer resp.Body.Close()

		testutils.AssertStatusCode(t, resp, http.StatusNotFound)

		// Test different HTTP methods
		for _, method := range []string{"POST", "PUT", "DELETE"} {
			resp, err := runner.HTTP.MakeRequest(method, url, "", nil)
			if err != nil {
				t.Fatalf("Failed to make %s request to %s: %v", method, url, err)
			}
			defer resp.Body.Close()

			// Should return 404 or 405 (Method Not Allowed), but not 200
			if resp.StatusCode == http.StatusOK {
				t.Errorf("Expected non-200 status for %s %s, got %d", method, url, resp.StatusCode)
			}
		}
	}
}

func testNavigationDoesNotContainRemovedSections(t *testing.T, runner *testutils.TestRunner) {
	// Test various pages to ensure navigation doesn't contain courses or links
	pagesToTest := []string{
		"/page?p=0",
		"/p/test-post-1",
		"/about",
		"/login",
	}

	for _, page := range pagesToTest {
		resp, err := runner.HTTP.MakeRequest("GET", page, "", nil)
		if err != nil {
			t.Fatalf("Failed to request %s: %v", page, err)
		}
		defer resp.Body.Close()

		// Skip if page returns error (like login might require auth)
		if resp.StatusCode >= 400 {
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read body for %s: %v", page, err)
		}

		bodyStr := strings.ToLower(string(body))

		// Check that navigation doesn't contain courses or links
		// Allow "links" in context of canonical links or other legitimate uses
		if strings.Contains(bodyStr, `href="/courses"`) {
			t.Errorf("Page %s should not contain navigation link to courses", page)
		}

		if strings.Contains(bodyStr, `href="/links"`) {
			t.Errorf("Page %s should not contain navigation link to links section", page)
		}

		// Check for menu items or navigation text
		if strings.Contains(bodyStr, ">courses<") || strings.Contains(bodyStr, ">course<") {
			// Allow if it's part of legitimate content, but not navigation
			if strings.Contains(bodyStr, "nav") || strings.Contains(bodyStr, "menu") {
				t.Errorf("Page %s should not contain courses in navigation", page)
			}
		}
	}
}

func testTemplatesDoNotRenderRemovedContent(t *testing.T, runner *testutils.TestRunner) {
	// Test that templates don't try to render courses or links data
	resp, err := runner.HTTP.MakeRequest("GET", "/page?p=0", "", nil)
	if err != nil {
		t.Fatalf("Failed to request homepage: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertStatusCode(t, resp, http.StatusOK)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read homepage body: %v", err)
	}

	bodyStr := string(body)

	// Should not contain template errors or references to removed data
	errorIndicators := []string{
		"courses.yml",
		"links.yml",
		"template: no such file",
		"undefined variable",
		"can't evaluate field courses",
		"can't evaluate field links",
	}

	for _, indicator := range errorIndicators {
		testutils.AssertNotContains(t, bodyStr, indicator)
	}

	// Test that the page renders successfully without courses/links data
	testutils.AssertContains(t, bodyStr, "<html")
	testutils.AssertContains(t, bodyStr, "</html>")
	testutils.AssertContains(t, bodyStr, "Test Post 1") // Should contain blog posts
}

// TestSitemapUpdatesOnPostChanges tests that sitemap reflects post changes
func TestSitemapUpdatesOnPostChanges(t *testing.T) {
	runner := testutils.NewTestRunner(t)
	defer runner.Close()

	if err := runner.SetupTest(); err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}

	// Login as admin
	sessionCookie, err := runner.HTTP.LoginAsAdmin()
	if err != nil {
		t.Fatalf("Failed to login as admin: %v", err)
	}

	// Get initial sitemap
	resp, err := runner.HTTP.MakeRequest("GET", "/sitemap.xml", "", nil)
	if err != nil {
		t.Fatalf("Failed to get initial sitemap: %v", err)
	}
	defer resp.Body.Close()

	initialBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read initial sitemap: %v", err)
	}

	initialSitemap := string(initialBody)

	// Should contain existing test posts
	testutils.AssertContains(t, initialSitemap, "/p/test-post-1")
	testutils.AssertContains(t, initialSitemap, "/p/test-post-2")
	testutils.AssertContains(t, initialSitemap, "/p/test-post-3")

	// Create a new post
	formData := "title=New+Sitemap+Test+Post&body=This+post+should+appear+in+sitemap"
	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}

	resp, err = runner.HTTP.MakeRequestWithCookies("POST", "/create", formData, headers, []*http.Cookie{sessionCookie})
	if err != nil {
		t.Fatalf("Failed to create post: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertRedirect(t, resp, "/")

	// Get updated sitemap
	resp, err = runner.HTTP.MakeRequest("GET", "/sitemap.xml", "", nil)
	if err != nil {
		t.Fatalf("Failed to get updated sitemap: %v", err)
	}
	defer resp.Body.Close()

	updatedBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read updated sitemap: %v", err)
	}

	updatedSitemap := string(updatedBody)

	// Should contain the new post
	testutils.AssertContains(t, updatedSitemap, "/p/new-sitemap-test-post")

	// Should still contain existing posts
	testutils.AssertContains(t, updatedSitemap, "/p/test-post-1")
	testutils.AssertContains(t, updatedSitemap, "/p/test-post-2")
	testutils.AssertContains(t, updatedSitemap, "/p/test-post-3")

	// Delete a post
	resp, err = runner.HTTP.MakeRequestWithCookies("GET", "/delete?slug=test-post-1", "", nil, []*http.Cookie{sessionCookie})
	if err != nil {
		t.Fatalf("Failed to delete post: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertRedirect(t, resp, "/")

	// Get sitemap after deletion
	resp, err = runner.HTTP.MakeRequest("GET", "/sitemap.xml", "", nil)
	if err != nil {
		t.Fatalf("Failed to get sitemap after deletion: %v", err)
	}
	defer resp.Body.Close()

	finalBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read final sitemap: %v", err)
	}

	finalSitemap := string(finalBody)

	// Should not contain deleted post
	testutils.AssertNotContains(t, finalSitemap, "/p/test-post-1")

	// Should still contain other posts
	testutils.AssertContains(t, finalSitemap, "/p/test-post-2")
	testutils.AssertContains(t, finalSitemap, "/p/test-post-3")
	testutils.AssertContains(t, finalSitemap, "/p/new-sitemap-test-post")
}

// TestSEOComprehensiveIntegration tests comprehensive SEO integration scenarios
func TestSEOComprehensiveIntegration(t *testing.T) {
	runner := testutils.NewTestRunner(t)
	defer runner.Close()

	if err := runner.SetupTest(); err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}

	t.Run("SEOWithSpecialCharacters", func(t *testing.T) {
		testSEOWithSpecialCharacters(t, runner)
	})

	t.Run("SEOWithLongContent", func(t *testing.T) {
		testSEOWithLongContent(t, runner)
	})

	t.Run("SEOWithFileReferences", func(t *testing.T) {
		testSEOWithFileReferences(t, runner)
	})

	t.Run("SEOErrorHandling", func(t *testing.T) {
		testSEOErrorHandling(t, runner)
	})

	t.Run("SitemapPerformance", func(t *testing.T) {
		testSitemapPerformance(t, runner)
	})
}

func testSEOWithSpecialCharacters(t *testing.T, runner *testutils.TestRunner) {
	// Create post with special characters
	_, err := runner.DB.DB.Exec(`
		INSERT INTO posts (title, body, datepost, slug, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?)
	`, "Post with Special Characters & Symbols <script>", "Content with <>&\"' and unicode: 你好 <script>alert('xss')</script>", "Mon Jan 2 15:04:05 2006", "post-special-characters", "Mon Jan 2 15:04:05 2006", "Mon Jan 2 15:04:05 2006")
	if err != nil {
		t.Fatalf("Failed to insert test post: %v", err)
	}

	resp, err := runner.HTTP.MakeRequest("GET", "/p/post-special-characters", "", nil)
	if err != nil {
		t.Fatalf("Failed to request post: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertStatusCode(t, resp, http.StatusOK)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	bodyStr := string(body)

	// Test that special characters are properly escaped in meta tags
	testutils.AssertContains(t, bodyStr, `&amp;`)
	testutils.AssertContains(t, bodyStr, `&lt;`)
	testutils.AssertContains(t, bodyStr, `&gt;`)

	// Test that script tags are not present in meta content
	testutils.AssertNotContains(t, bodyStr, `content="<script>`)
	testutils.AssertNotContains(t, bodyStr, `content="alert('xss')`)

	// Test that structured data is properly escaped
	if strings.Contains(bodyStr, `"<script>`) {
		t.Error("Structured data should not contain unescaped script tags")
	}
}

func testSEOWithLongContent(t *testing.T, runner *testutils.TestRunner) {
	longTitle := strings.Repeat("Very Long Title That Should Be Handled Properly ", 10)
	longContent := strings.Repeat("This is a very long content that should be truncated properly in meta descriptions and other SEO elements. ", 20)

	// Create post with long content
	_, err := runner.DB.DB.Exec(`
		INSERT INTO posts (title, body, datepost, slug, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?)
	`, longTitle, longContent, "Mon Jan 2 15:04:05 2006", "post-long-content", "Mon Jan 2 15:04:05 2006", "Mon Jan 2 15:04:05 2006")
	if err != nil {
		t.Fatalf("Failed to insert test post: %v", err)
	}

	resp, err := runner.HTTP.MakeRequest("GET", "/p/post-long-content", "", nil)
	if err != nil {
		t.Fatalf("Failed to request post: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertStatusCode(t, resp, http.StatusOK)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	bodyStr := string(body)

	// Test that meta description is truncated
	descRegex := regexp.MustCompile(`<meta name="description" content="([^"]*)"`)
	matches := descRegex.FindStringSubmatch(bodyStr)
	if len(matches) > 1 {
		description := matches[1]
		if len(description) > 160 {
			t.Errorf("Meta description should be truncated to 160 chars, got %d", len(description))
		}
		if !strings.HasSuffix(description, "...") {
			t.Error("Long meta description should end with '...'")
		}
	}

	// Test that Open Graph description is also truncated
	ogDescRegex := regexp.MustCompile(`<meta property="og:description" content="([^"]*)"`)
	ogMatches := ogDescRegex.FindStringSubmatch(bodyStr)
	if len(ogMatches) > 1 {
		ogDescription := ogMatches[1]
		if len(ogDescription) > 160 {
			t.Errorf("OG description should be truncated to 160 chars, got %d", len(ogDescription))
		}
	}
}

func testSEOWithFileReferences(t *testing.T, runner *testutils.TestRunner) {
	// Create files table if not exists
	_, err := runner.DB.DB.Exec(`CREATE TABLE IF NOT EXISTS files (
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

	// Insert test image
	_, err = runner.DB.DB.Exec(`INSERT INTO files (uuid, original_name, stored_name, path, size, mime_type, is_image, width, height, alt_text) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"test-seo-image-uuid", "seo-test.jpg", "stored-seo-test.jpg", "/path/to/seo-image", 1024, "image/jpeg", true, 800, 600, "SEO test image")
	if err != nil {
		t.Fatalf("Failed to insert test image: %v", err)
	}

	// Create post with image reference
	_, err = runner.DB.DB.Exec(`
		INSERT INTO posts (title, body, datepost, slug, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?)
	`, "Post with SEO Image", "This post has an image [file:seo-test.jpg] for SEO testing.", "Mon Jan 2 15:04:05 2006", "post-seo-image", "Mon Jan 2 15:04:05 2006", "Mon Jan 2 15:04:05 2006")
	if err != nil {
		t.Fatalf("Failed to insert test post: %v", err)
	}

	resp, err := runner.HTTP.MakeRequest("GET", "/p/post-seo-image", "", nil)
	if err != nil {
		t.Fatalf("Failed to request post: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertStatusCode(t, resp, http.StatusOK)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	bodyStr := string(body)

	// Test that image is included in structured data
	testutils.AssertContains(t, bodyStr, `\"image\"`)
	testutils.AssertContains(t, bodyStr, "test-seo-image-uuid")

	// Test that Open Graph image is included
	testutils.AssertContains(t, bodyStr, `<meta property="og:image"`)
	testutils.AssertContains(t, bodyStr, "test-seo-image-uuid")

	// Test that Twitter Card image is included
	testutils.AssertContains(t, bodyStr, `<meta property="twitter:image"`)
	testutils.AssertContains(t, bodyStr, "test-seo-image-uuid")

	// Test that image alt text is used
	testutils.AssertContains(t, bodyStr, `<meta property="og:image:alt" content="Post with SEO Image">`)
}

func testSEOErrorHandling(t *testing.T, runner *testutils.TestRunner) {
	// Test non-existent post
	resp, err := runner.HTTP.MakeRequest("GET", "/p/non-existent-post", "", nil)
	if err != nil {
		t.Fatalf("Failed to request non-existent post: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertStatusCode(t, resp, http.StatusNotFound)

	// Test malformed slug
	resp, err = runner.HTTP.MakeRequest("GET", "/p/", "", nil)
	if err != nil {
		t.Fatalf("Failed to request malformed slug: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertStatusCode(t, resp, http.StatusBadRequest)

	// Test sitemap with no posts
	_, err = runner.DB.DB.Exec("DELETE FROM posts")
	if err != nil {
		t.Fatalf("Failed to delete all posts: %v", err)
	}

	resp, err = runner.HTTP.MakeRequest("GET", "/sitemap.xml", "", nil)
	if err != nil {
		t.Fatalf("Failed to request sitemap: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertStatusCode(t, resp, http.StatusOK)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read sitemap body: %v", err)
	}

	bodyStr := string(body)

	// Should still contain homepage
	testutils.AssertContains(t, bodyStr, "<loc>http://localhost:8080/</loc>")

	// Should be valid XML
	testutils.AssertContains(t, bodyStr, `<?xml version="1.0" encoding="UTF-8"?>`)
	testutils.AssertContains(t, bodyStr, `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
}

func testSitemapPerformance(t *testing.T, runner *testutils.TestRunner) {
	// Create many posts to test performance
	for i := 0; i < 100; i++ {
		_, err := runner.DB.DB.Exec(`
			INSERT INTO posts (title, body, datepost, slug, created_at, updated_at) 
			VALUES (?, ?, ?, ?, ?, ?)
		`, fmt.Sprintf("Performance Test Post %d", i), fmt.Sprintf("Content for post %d", i), "Mon Jan 2 15:04:05 2006", fmt.Sprintf("performance-test-post-%d", i), "Mon Jan 2 15:04:05 2006", "Mon Jan 2 15:04:05 2006")
		if err != nil {
			t.Fatalf("Failed to insert performance test post %d: %v", i, err)
		}
	}

	start := time.Now()
	resp, err := runner.HTTP.MakeRequest("GET", "/sitemap.xml", "", nil)
	if err != nil {
		t.Fatalf("Failed to request sitemap: %v", err)
	}
	defer resp.Body.Close()

	duration := time.Since(start)

	testutils.AssertStatusCode(t, resp, http.StatusOK)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read sitemap body: %v", err)
	}

	bodyStr := string(body)

	// Should contain all posts
	for i := 0; i < 100; i++ {
		expectedURL := fmt.Sprintf("<loc>http://localhost:8080/p/performance-test-post-%d</loc>", i)
		testutils.AssertContains(t, bodyStr, expectedURL)
	}

	// Should complete within reasonable time
	if duration > 2*time.Second {
		t.Errorf("Sitemap generation took too long: %v", duration)
	}

	t.Logf("Generated sitemap for 100 posts in %v", duration)
}

// TestContentRemovalComprehensive tests comprehensive content removal scenarios
func TestContentRemovalComprehensive(t *testing.T) {
	runner := testutils.NewTestRunner(t)
	defer runner.Close()

	if err := runner.SetupTest(); err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}

	t.Run("CoursesLinksURLsComprehensive", func(t *testing.T) {
		testCoursesLinksURLsComprehensive(t, runner)
	})

	t.Run("NavigationRemovalComprehensive", func(t *testing.T) {
		testNavigationRemovalComprehensive(t, runner)
	})

	t.Run("TemplateErrorsAbsent", func(t *testing.T) {
		testTemplateErrorsAbsent(t, runner)
	})

	t.Run("DataFilesRemoved", func(t *testing.T) {
		testDataFilesRemoved(t, runner)
	})
}

func testCoursesLinksURLsComprehensive(t *testing.T, runner *testutils.TestRunner) {
	// Comprehensive list of URLs that should return 404
	urlsToTest := []struct {
		url         string
		description string
	}{
		// Courses URLs
		{"/courses", "courses root"},
		{"/courses/", "courses with trailing slash"},
		{"/courses/index", "courses index"},
		{"/courses/index.html", "courses index HTML"},
		{"/courses/programming", "courses programming"},
		{"/courses/web-development", "courses web development"},
		{"/courses/data-science", "courses data science"},
		{"/courses/1", "specific course by ID"},
		{"/courses/programming/basics", "nested course path"},
		{"/api/courses", "courses API"},
		{"/api/courses/", "courses API with slash"},
		{"/api/courses/1", "specific course API"},
		{"/api/courses/programming", "course category API"},
		{"/courses.json", "courses JSON"},
		{"/courses.xml", "courses XML"},
		{"/courses.html", "courses HTML"},
		{"/courses.php", "courses PHP"},
		{"/courses.asp", "courses ASP"},

		// Links URLs
		{"/links", "links root"},
		{"/links/", "links with trailing slash"},
		{"/links/index", "links index"},
		{"/links/index.html", "links index HTML"},
		{"/links/useful", "useful links"},
		{"/links/resources", "resource links"},
		{"/links/tools", "tool links"},
		{"/links/1", "specific link by ID"},
		{"/links/useful/programming", "nested link path"},
		{"/api/links", "links API"},
		{"/api/links/", "links API with slash"},
		{"/api/links/1", "specific link API"},
		{"/api/links/useful", "link category API"},
		{"/links.json", "links JSON"},
		{"/links.xml", "links XML"},
		{"/links.html", "links HTML"},
		{"/links.php", "links PHP"},
		{"/links.asp", "links ASP"},

		// Case variations
		{"/Courses", "courses capitalized"},
		{"/COURSES", "courses uppercase"},
		{"/Links", "links capitalized"},
		{"/LINKS", "links uppercase"},
		{"/CoUrSeS", "courses mixed case"},
		{"/LiNkS", "links mixed case"},
	}

	for _, urlTest := range urlsToTest {
		t.Run("URL_"+urlTest.url, func(t *testing.T) {
			// Test GET request
			resp, err := runner.HTTP.MakeRequest("GET", urlTest.url, "", nil)
			if err != nil {
				t.Fatalf("Failed to request %s: %v", urlTest.url, err)
			}
			defer resp.Body.Close()

			testutils.AssertStatusCode(t, resp, http.StatusNotFound)

			// Test other HTTP methods
			methods := []string{"POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
			for _, method := range methods {
				resp, err := runner.HTTP.MakeRequest(method, urlTest.url, "", nil)
				if err != nil {
					t.Fatalf("Failed to make %s request to %s: %v", method, urlTest.url, err)
				}
				defer resp.Body.Close()

				// Should not return 200 OK
				if resp.StatusCode == http.StatusOK {
					t.Errorf("Expected non-200 status for %s %s (%s), got %d", method, urlTest.url, urlTest.description, resp.StatusCode)
				}
			}
		})
	}
}

func testNavigationRemovalComprehensive(t *testing.T, runner *testutils.TestRunner) {
	// Test various pages that might contain navigation
	pagesToTest := []string{
		"/page?p=0",
		"/p/test-post-1",
		"/p/test-post-2",
		"/about",
		"/login",
	}

	for _, page := range pagesToTest {
		t.Run("Navigation_"+page, func(t *testing.T) {
			resp, err := runner.HTTP.MakeRequest("GET", page, "", nil)
			if err != nil {
				t.Fatalf("Failed to request %s: %v", page, err)
			}
			defer resp.Body.Close()

			// Skip if page returns error (some pages might require auth)
			if resp.StatusCode >= 400 {
				return
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read body for %s: %v", page, err)
			}

			bodyStr := strings.ToLower(string(body))

			// Check for navigation links to removed sections
			prohibitedPatterns := []string{
				`href="/courses"`,
				`href="/courses/"`,
				`href="/links"`,
				`href="/links/"`,
				`href="/api/courses"`,
				`href="/api/links"`,
				`href="courses"`,
				`href="links"`,
				`href="courses.html"`,
				`href="links.html"`,
			}

			for _, pattern := range prohibitedPatterns {
				if strings.Contains(bodyStr, pattern) {
					t.Errorf("Page %s should not contain navigation link: %s", page, pattern)
				}
			}

			// Check for menu text in navigation context
			if strings.Contains(bodyStr, "nav") || strings.Contains(bodyStr, "menu") {
				navigationSection := bodyStr
				if strings.Contains(navigationSection, ">courses<") || strings.Contains(navigationSection, ">links<") {
					t.Errorf("Page %s should not contain courses/links in navigation", page)
				}
			}

			// Check for JavaScript references to removed sections
			jsProhibitedPatterns := []string{
				`"/courses"`,
				`"/links"`,
				`'/courses'`,
				`'/links'`,
				`courses.html`,
				`links.html`,
			}

			for _, pattern := range jsProhibitedPatterns {
				if strings.Contains(bodyStr, pattern) {
					// Allow if it's part of legitimate content (like canonical links)
					if !strings.Contains(bodyStr, "canonical") && !strings.Contains(bodyStr, "schema.org") {
						t.Errorf("Page %s should not contain JavaScript reference to removed sections: %s", page, pattern)
					}
				}
			}
		})
	}
}

func testTemplateErrorsAbsent(t *testing.T, runner *testutils.TestRunner) {
	// Test that templates don't contain errors related to removed content
	resp, err := runner.HTTP.MakeRequest("GET", "/page?p=0", "", nil)
	if err != nil {
		t.Fatalf("Failed to request homepage: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertStatusCode(t, resp, http.StatusOK)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read homepage body: %v", err)
	}

	bodyStr := string(body)

	// Check for template errors
	templateErrors := []string{
		"template: no such file",
		"undefined variable",
		"can't evaluate field courses",
		"can't evaluate field links",
		"courses.yml",
		"links.yml",
		"executing template",
		"template error",
		"parse error",
		"runtime error",
	}

	for _, errorPattern := range templateErrors {
		testutils.AssertNotContains(t, bodyStr, errorPattern)
	}

	// Should contain valid HTML structure
	testutils.AssertContains(t, bodyStr, "<html")
	testutils.AssertContains(t, bodyStr, "</html>")
	testutils.AssertContains(t, bodyStr, "<head>")
	testutils.AssertContains(t, bodyStr, "</head>")
	testutils.AssertContains(t, bodyStr, "<body>")
	testutils.AssertContains(t, bodyStr, "</body>")

	// Should contain blog content
	testutils.AssertContains(t, bodyStr, "Test Post 1")
}

func testDataFilesRemoved(t *testing.T, runner *testutils.TestRunner) {
	// Test that data files for courses and links don't exist or aren't accessible
	dataURLs := []string{
		"/data/courses.yml",
		"/data/links.yml",
		"/app/data/courses.yml",
		"/app/data/links.yml",
		"/public/data/courses.yml",
		"/public/data/links.yml",
		"/courses.yml",
		"/links.yml",
	}

	for _, url := range dataURLs {
		resp, err := runner.HTTP.MakeRequest("GET", url, "", nil)
		if err != nil {
			t.Fatalf("Failed to request %s: %v", url, err)
		}
		defer resp.Body.Close()

		// Should return 404 or other non-200 status
		if resp.StatusCode == http.StatusOK {
			t.Errorf("Data file %s should not be accessible, got status %d", url, resp.StatusCode)
		}
	}
}
