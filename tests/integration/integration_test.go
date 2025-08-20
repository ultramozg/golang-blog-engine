package integration

import (
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/ultramozg/golang-blog-engine/testutils"
)

// TestBlogPlatformIntegration tests the complete blog platform functionality
func TestBlogPlatformIntegration(t *testing.T) {
	runner := testutils.NewTestRunner(t)
	defer runner.Close()

	if err := runner.SetupTest(); err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}

	t.Run("HomePage", func(t *testing.T) {
		testHomePage(t, runner)
	})

	t.Run("PostOperations", func(t *testing.T) {
		testPostOperations(t, runner)
	})

	t.Run("CommentOperations", func(t *testing.T) {
		testCommentOperations(t, runner)
	})

	t.Run("Authentication", func(t *testing.T) {
		testAuthentication(t, runner)
	})

	t.Run("Authorization", func(t *testing.T) {
		testAuthorization(t, runner)
	})

	t.Run("SlugBasedRouting", func(t *testing.T) {
		testSlugBasedRouting(t, runner)
	})
}

func testHomePage(t *testing.T, runner *testutils.TestRunner) {
	// Test root redirect
	resp, err := runner.HTTP.MakeRequest("GET", "/", "", nil)
	if err != nil {
		t.Fatalf("Failed to make request to root: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertRedirect(t, resp, "/page?p=0")

	// Test page listing
	resp, err = runner.HTTP.MakeRequest("GET", "/page?p=0", "", nil)
	if err != nil {
		t.Fatalf("Failed to make request to page: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertStatusCode(t, resp, http.StatusOK)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	bodyStr := string(body)
	testutils.AssertContains(t, bodyStr, "Test Post 1")
	testutils.AssertContains(t, bodyStr, "Test Post 2")
	testutils.AssertContains(t, bodyStr, "Test Post 3")
	
	// Verify that the page contains slug-based links instead of ID-based links
	testutils.AssertContains(t, bodyStr, "/p/test-post-1")
	testutils.AssertContains(t, bodyStr, "/p/test-post-2")
	testutils.AssertContains(t, bodyStr, "/p/test-post-3")
}

func testPostOperations(t *testing.T, runner *testutils.TestRunner) {
	// Login as admin first
	sessionCookie, err := runner.HTTP.LoginAsAdmin()
	if err != nil {
		t.Fatalf("Failed to login as admin: %v", err)
	}

	// Test post creation form
	resp, err := runner.HTTP.MakeRequestWithCookies("GET", "/create", "", nil, []*http.Cookie{sessionCookie})
	if err != nil {
		t.Fatalf("Failed to get create form: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertStatusCode(t, resp, http.StatusOK)

	// Test post creation
	formData := "title=Integration+Test+Post&body=This+is+a+test+post+created+during+integration+testing"
	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}

	resp, err = runner.HTTP.MakeRequestWithCookies("POST", "/create", formData, headers, []*http.Cookie{sessionCookie})
	if err != nil {
		t.Fatalf("Failed to create post: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertRedirect(t, resp, "/")

	// Verify post was created in database
	var count int
	err = runner.DB.DB.QueryRow("SELECT COUNT(*) FROM posts WHERE title = ?", "Integration Test Post").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query new post: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 new post, got %d", count)
	}

	// Get the post ID for further testing
	var postID int
	err = runner.DB.DB.QueryRow("SELECT id FROM posts WHERE title = ?", "Integration Test Post").Scan(&postID)
	if err != nil {
		t.Fatalf("Failed to get post ID: %v", err)
	}

	// Test post viewing
	resp, err = runner.HTTP.MakeRequest("GET", "/post?id="+string(rune(postID+'0')), "", nil)
	if err != nil {
		t.Fatalf("Failed to view post: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertStatusCode(t, resp, http.StatusOK)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read post body: %v", err)
	}

	bodyStr := string(body)
	testutils.AssertContains(t, bodyStr, "Integration Test Post")
	testutils.AssertContains(t, bodyStr, "This is a test post created during integration testing")

	// Test post update form
	resp, err = runner.HTTP.MakeRequestWithCookies("GET", "/update?id="+string(rune(postID+'0')), "", nil, []*http.Cookie{sessionCookie})
	if err != nil {
		t.Fatalf("Failed to get update form: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertStatusCode(t, resp, http.StatusOK)

	// Test post update
	updateData := "id=" + string(rune(postID+'0')) + "&title=Updated+Integration+Test+Post&body=This+post+has+been+updated"
	resp, err = runner.HTTP.MakeRequestWithCookies("POST", "/update", updateData, headers, []*http.Cookie{sessionCookie})
	if err != nil {
		t.Fatalf("Failed to update post: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertRedirect(t, resp, "/")

	// Verify post was updated
	var updatedTitle string
	err = runner.DB.DB.QueryRow("SELECT title FROM posts WHERE id = ?", postID).Scan(&updatedTitle)
	if err != nil {
		t.Fatalf("Failed to query updated post: %v", err)
	}
	if updatedTitle != "Updated Integration Test Post" {
		t.Errorf("Expected 'Updated Integration Test Post', got '%s'", updatedTitle)
	}

	// Test post deletion
	resp, err = runner.HTTP.MakeRequestWithCookies("GET", "/delete?id="+string(rune(postID+'0')), "", nil, []*http.Cookie{sessionCookie})
	if err != nil {
		t.Fatalf("Failed to delete post: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertRedirect(t, resp, "/")

	// Verify post was deleted
	err = runner.DB.DB.QueryRow("SELECT COUNT(*) FROM posts WHERE id = ?", postID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query deleted post: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 posts after deletion, got %d", count)
	}
}

func testCommentOperations(t *testing.T, runner *testutils.TestRunner) {
	// Login as admin to create a session
	sessionCookie, err := runner.HTTP.LoginAsAdmin()
	if err != nil {
		t.Fatalf("Failed to login as admin: %v", err)
	}

	// Test comment creation
	formData := "id=1&name=Integration+Tester&comment=This+is+an+integration+test+comment"
	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
		"Referer":      "/post?id=1",
	}

	resp, err := runner.HTTP.MakeRequestWithCookies("POST", "/create-comment", formData, headers, []*http.Cookie{sessionCookie})
	if err != nil {
		t.Fatalf("Failed to create comment: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertRedirect(t, resp, "/post?id=1")

	// Verify comment was created
	var count int
	err = runner.DB.DB.QueryRow("SELECT COUNT(*) FROM comments WHERE name = ? AND comment = ?",
		"Integration Tester", "This is an integration test comment").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query new comment: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 new comment, got %d", count)
	}

	// Get comment ID for deletion test
	var commentID int
	err = runner.DB.DB.QueryRow("SELECT commentid FROM comments WHERE name = ? AND comment = ?",
		"Integration Tester", "This is an integration test comment").Scan(&commentID)
	if err != nil {
		t.Fatalf("Failed to get comment ID: %v", err)
	}

	// Test comment deletion (admin only)
	resp, err = runner.HTTP.MakeRequestWithCookies("GET", "/delete-comment?id="+string(rune(commentID+'0')), "",
		map[string]string{"Referer": "/post?id=1"}, []*http.Cookie{sessionCookie})
	if err != nil {
		t.Fatalf("Failed to delete comment: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertRedirect(t, resp, "/post?id=1")

	// Verify comment was deleted
	err = runner.DB.DB.QueryRow("SELECT COUNT(*) FROM comments WHERE commentid = ?", commentID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query deleted comment: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 comments after deletion, got %d", count)
	}
}

func testAuthentication(t *testing.T, runner *testutils.TestRunner) {
	// Test login form
	resp, err := runner.HTTP.MakeRequest("GET", "/login", "", nil)
	if err != nil {
		t.Fatalf("Failed to get login form: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertStatusCode(t, resp, http.StatusOK)

	// Test successful login
	loginData := "login=admin&password=" + runner.HTTP.App.Config.AdminPass
	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}

	resp, err = runner.HTTP.MakeRequest("POST", "/login", loginData, headers)
	if err != nil {
		t.Fatalf("Failed to login: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertRedirect(t, resp, "/")
	sessionCookie := testutils.AssertCookieExists(t, resp, "session")

	if sessionCookie == nil {
		t.Fatal("Session cookie should exist after successful login")
	}

	// Test failed login
	failedLoginData := "login=admin&password=wrongpassword"
	resp, err = runner.HTTP.MakeRequest("POST", "/login", failedLoginData, headers)
	if err != nil {
		t.Fatalf("Failed to make failed login request: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertStatusCode(t, resp, http.StatusUnauthorized)

	// Test logout
	resp, err = runner.HTTP.MakeRequestWithCookies("GET", "/logout", "", nil, []*http.Cookie{sessionCookie})
	if err != nil {
		t.Fatalf("Failed to logout: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertRedirect(t, resp, "/")
}

func testAuthorization(t *testing.T, runner *testutils.TestRunner) {
	// Test unauthorized access to admin endpoints
	adminEndpoints := []string{"/create", "/update?id=1", "/delete?id=999"}

	for _, endpoint := range adminEndpoints {
		resp, err := runner.HTTP.MakeRequest("GET", endpoint, "", nil)
		if err != nil {
			t.Fatalf("Failed to make request to %s: %v", endpoint, err)
		}
		resp.Body.Close()

		testutils.AssertStatusCode(t, resp, http.StatusUnauthorized)
	}

	// Test unauthorized comment operations
	commentData := "id=1&name=Test&comment=Test"
	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}

	resp, err := runner.HTTP.MakeRequest("POST", "/create-comment", commentData, headers)
	if err != nil {
		t.Fatalf("Failed to make unauthorized comment request: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertStatusCode(t, resp, http.StatusUnauthorized)

	// Test that admin can access protected endpoints
	sessionCookie, err := runner.HTTP.LoginAsAdmin()
	if err != nil {
		t.Fatalf("Failed to login as admin: %v", err)
	}

	for _, endpoint := range adminEndpoints {
		resp, err := runner.HTTP.MakeRequestWithCookies("GET", endpoint, "", nil, []*http.Cookie{sessionCookie})
		if err != nil {
			t.Fatalf("Failed to make authorized request to %s: %v", endpoint, err)
		}
		resp.Body.Close()

		// Should not be unauthorized (could be 200, 404, etc. depending on endpoint)
		if resp.StatusCode == http.StatusUnauthorized {
			t.Errorf("Admin should be authorized to access %s", endpoint)
		}
	}
}

// TestErrorHandling tests various error conditions
func TestErrorHandling(t *testing.T) {
	runner := testutils.NewTestRunner(t)
	defer runner.Close()

	if err := runner.SetupTest(); err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}

	// Test invalid post ID
	resp, err := runner.HTTP.MakeRequest("GET", "/post?id=invalid", "", nil)
	if err != nil {
		t.Fatalf("Failed to make request with invalid ID: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertStatusCode(t, resp, http.StatusBadRequest)

	// Test non-existent post
	resp, err = runner.HTTP.MakeRequest("GET", "/post?id=99999", "", nil)
	if err != nil {
		t.Fatalf("Failed to make request for non-existent post: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertStatusCode(t, resp, http.StatusNotFound)

	// Test invalid page parameter
	resp, err = runner.HTTP.MakeRequest("GET", "/page?p=invalid", "", nil)
	if err != nil {
		t.Fatalf("Failed to make request with invalid page: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertStatusCode(t, resp, http.StatusBadRequest)

	// Test 404 for non-existent routes
	resp, err = runner.HTTP.MakeRequest("GET", "/nonexistent", "", nil)
	if err != nil {
		t.Fatalf("Failed to make request to non-existent route: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertStatusCode(t, resp, http.StatusNotFound)
}

// TestConcurrentAccess tests concurrent access to the application
func TestConcurrentAccess(t *testing.T) {
	runner := testutils.NewTestRunner(t)
	defer runner.Close()

	if err := runner.SetupTest(); err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}

	// Test concurrent page requests
	const numRequests = 10
	results := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			resp, err := runner.HTTP.MakeRequest("GET", "/page?p=0", "", nil)
			if err != nil {
				results <- err
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				results <- fmt.Errorf("expected status 200, got %d", resp.StatusCode)
				return
			}

			results <- nil
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		if err := <-results; err != nil {
			t.Errorf("Concurrent request failed: %v", err)
		}
	}
}
func testSlugBasedRouting(t *testing.T, runner *testutils.TestRunner) {
	// Test accessing posts by slug
	// The test data should have posts with slugs like "test-post-1", "test-post-2", etc.
	
	// Test accessing first post by slug
	resp, err := runner.HTTP.MakeRequest("GET", "/p/test-post-1", "", nil)
	if err != nil {
		t.Fatalf("Failed to access post by slug: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertStatusCode(t, resp, http.StatusOK)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	bodyStr := string(body)
	testutils.AssertContains(t, bodyStr, "Test Post 1")
	testutils.AssertContains(t, bodyStr, "This is the body of test post 1")

	// Test accessing second post by slug
	resp, err = runner.HTTP.MakeRequest("GET", "/p/test-post-2", "", nil)
	if err != nil {
		t.Fatalf("Failed to access second post by slug: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertStatusCode(t, resp, http.StatusOK)

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	bodyStr = string(body)
	testutils.AssertContains(t, bodyStr, "Test Post 2")
	testutils.AssertContains(t, bodyStr, "This is the body of test post 2")

	// Test accessing non-existent slug
	resp, err = runner.HTTP.MakeRequest("GET", "/p/non-existent-slug", "", nil)
	if err != nil {
		t.Fatalf("Failed to make request for non-existent slug: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertStatusCode(t, resp, http.StatusNotFound)

	// Test empty slug
	resp, err = runner.HTTP.MakeRequest("GET", "/p/", "", nil)
	if err != nil {
		t.Fatalf("Failed to make request for empty slug: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertStatusCode(t, resp, http.StatusBadRequest)

	// Test that new posts get slugs when created
	sessionCookie, err := runner.HTTP.LoginAsAdmin()
	if err != nil {
		t.Fatalf("Failed to login as admin: %v", err)
	}

	// Create a new post with a title that should generate a specific slug
	formData := "title=My+Awesome+New+Post&body=This+is+a+test+post+for+slug+generation"
	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}

	resp, err = runner.HTTP.MakeRequestWithCookies("POST", "/create", formData, headers, []*http.Cookie{sessionCookie})
	if err != nil {
		t.Fatalf("Failed to create post: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertRedirect(t, resp, "/")

	// Verify the post was created with a slug
	var slug string
	err = runner.DB.DB.QueryRow("SELECT slug FROM posts WHERE title = ?", "My Awesome New Post").Scan(&slug)
	if err != nil {
		t.Fatalf("Failed to query new post slug: %v", err)
	}

	expectedSlug := "my-awesome-new-post"
	if slug != expectedSlug {
		t.Errorf("Expected slug '%s', got '%s'", expectedSlug, slug)
	}

	// Test accessing the new post by its slug
	resp, err = runner.HTTP.MakeRequest("GET", "/p/"+slug, "", nil)
	if err != nil {
		t.Fatalf("Failed to access new post by slug: %v", err)
	}
	defer resp.Body.Close()

	testutils.AssertStatusCode(t, resp, http.StatusOK)

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	bodyStr = string(body)
	testutils.AssertContains(t, bodyStr, "My Awesome New Post")
	testutils.AssertContains(t, bodyStr, "This is a test post for slug generation")
}