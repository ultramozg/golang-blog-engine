package model

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

// Test helper functions to avoid circular dependency with testutils

// createTestDB creates a test database with migrations
func createTestDB(t *testing.T) (*sql.DB, func()) {
	tempDir, err := os.MkdirTemp("", "model_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tempDir, "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Run migrations
	MigrateDatabase(db)

	cleanup := func() {
		db.Close()
		os.RemoveAll(tempDir)
	}

	return db, cleanup
}

// seedTestData inserts test data into the database
func seedTestData(db *sql.DB) error {
	// Insert test posts with simple slugs
	testPosts := []struct {
		title, body, date, slug string
	}{
		{"Test Post 1", "This is the body of test post 1", "Mon Jan 1 12:00:00 2024", "test-post-1"},
		{"Test Post 2", "This is the body of test post 2", "Mon Jan 2 12:00:00 2024", "test-post-2"},
		{"Test Post 3", "This is the body of test post 3", "Mon Jan 3 12:00:00 2024", "test-post-3"},
	}

	for _, post := range testPosts {
		// Insert post with slug
		result, err := db.Exec(`insert into posts (title, body, datepost, slug, created_at, updated_at) values ($1, $2, $3, $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, post.title, post.body, post.date, post.slug)
		if err != nil {
			return err
		}

		// Get the ID of the newly created post
		_, err = result.LastInsertId()
		if err != nil {
			return err
		}
	}

	// Insert test comments
	testComments := []struct {
		postID              int
		name, date, comment string
	}{
		{1, "Test User", "Mon Jan 1 13:00:00 2024", "This is a test comment"},
		{1, "Another User", "Mon Jan 1 14:00:00 2024", "Another test comment"},
		{2, "Test User", "Mon Jan 2 13:00:00 2024", "Comment on second post"},
	}

	for _, comment := range testComments {
		_, err := db.Exec(`INSERT INTO comments (postid, name, date, comment) VALUES (?, ?, ?, ?)`,
			comment.postID, comment.name, comment.date, comment.comment)
		if err != nil {
			return err
		}
	}

	return nil
}

// createTestUser creates a test user in the database
func createTestUser(db *sql.DB, name, password string, userType int) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = db.Exec(`INSERT INTO users (name, type, pass) VALUES (?, ?, ?)`,
		name, userType, string(hashedPassword))
	return err
}

func TestPost_GetPost(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Seed test data
	if err := seedTestData(db); err != nil {
		t.Fatalf("Failed to seed test data: %v", err)
	}

	tests := []struct {
		name         string
		postID       int
		expectError  bool
		expectedPost Post
	}{
		{
			name:        "Get existing post",
			postID:      1,
			expectError: false,
			expectedPost: Post{
				ID:    1,
				Title: "Test Post 1",
				Body:  "This is the body of test post 1",
				Date:  "Mon Jan 1 12:00:00 2024",
			},
		},
		{
			name:        "Get non-existing post",
			postID:      999,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			post := Post{ID: tt.postID}
			err := post.GetPost(db)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if err != sql.ErrNoRows {
					t.Errorf("Expected sql.ErrNoRows, got %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if post.ID != tt.expectedPost.ID {
					t.Errorf("Expected ID %d, got %d", tt.expectedPost.ID, post.ID)
				}
				if post.Title != tt.expectedPost.Title {
					t.Errorf("Expected Title %s, got %s", tt.expectedPost.Title, post.Title)
				}
				if post.Body != tt.expectedPost.Body {
					t.Errorf("Expected Body %s, got %s", tt.expectedPost.Body, post.Body)
				}
				if post.Date != tt.expectedPost.Date {
					t.Errorf("Expected Date %s, got %s", tt.expectedPost.Date, post.Date)
				}
			}
		})
	}
}

func TestPost_GetPostBySlug(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Seed test data
	if err := seedTestData(db); err != nil {
		t.Fatalf("Failed to seed test data: %v", err)
	}

	tests := []struct {
		name         string
		slug         string
		expectError  bool
		expectedPost Post
	}{
		{
			name:        "Get existing post by slug",
			slug:        "test-post-1",
			expectError: false,
			expectedPost: Post{
				ID:    1,
				Title: "Test Post 1",
				Body:  "This is the body of test post 1",
				Date:  "Mon Jan 1 12:00:00 2024",
				Slug:  "test-post-1",
			},
		},
		{
			name:        "Get non-existing post by slug",
			slug:        "non-existing-slug",
			expectError: true,
		},
		{
			name:        "Get post with empty slug",
			slug:        "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			post := Post{Slug: tt.slug}
			err := post.GetPostBySlug(db)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if post.ID != tt.expectedPost.ID {
					t.Errorf("Expected ID %d, got %d", tt.expectedPost.ID, post.ID)
				}
				if post.Title != tt.expectedPost.Title {
					t.Errorf("Expected Title %s, got %s", tt.expectedPost.Title, post.Title)
				}
				if post.Slug != tt.expectedPost.Slug {
					t.Errorf("Expected Slug %s, got %s", tt.expectedPost.Slug, post.Slug)
				}
			}
		})
	}
}

func TestGetPostBySlug(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Seed test data
	if err := seedTestData(db); err != nil {
		t.Fatalf("Failed to seed test data: %v", err)
	}

	tests := []struct {
		name        string
		slug        string
		expectError bool
	}{
		{
			name:        "Get existing post by slug",
			slug:        "test-post-1",
			expectError: false,
		},
		{
			name:        "Get non-existing post by slug",
			slug:        "non-existing-slug",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			post, err := GetPostBySlug(db, tt.slug)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if post != nil {
					t.Errorf("Expected nil post but got %+v", post)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if post == nil {
					t.Error("Expected post to be returned")
					return
				}
				if post.Slug != tt.slug {
					t.Errorf("Expected slug %s, got %s", tt.slug, post.Slug)
				}
			}
		})
	}
}

func TestPost_CreatePost(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	tests := []struct {
		name        string
		post        Post
		expectError bool
	}{
		{
			name: "Create valid post",
			post: Post{
				Title: "New Test Post",
				Body:  "This is a new test post body",
				Date:  time.Now().Format("Mon Jan _2 15:04:05 2006"),
			},
			expectError: false,
		},
		{
			name: "Create post with empty title",
			post: Post{
				Title: "",
				Body:  "This is a test post body",
				Date:  time.Now().Format("Mon Jan _2 15:04:05 2006"),
			},
			expectError: false, // SQLite allows empty strings
		},
		{
			name: "Create post with empty body",
			post: Post{
				Title: "Test Post",
				Body:  "",
				Date:  time.Now().Format("Mon Jan _2 15:04:05 2006"),
			},
			expectError: false, // SQLite allows empty strings
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.post.CreatePost(db)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				// Verify the post was created by trying to retrieve it
				var count int
				err = db.QueryRow("SELECT COUNT(*) FROM posts WHERE title = ? AND body = ?",
					tt.post.Title, tt.post.Body).Scan(&count)
				if err != nil {
					t.Errorf("Error checking if post was created: %v", err)
				}
				if count != 1 {
					t.Errorf("Expected 1 post to be created, found %d", count)
				}
			}
		})
	}
}

func TestPost_UpdatePost(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Seed test data
	if err := seedTestData(db); err != nil {
		t.Fatalf("Failed to seed test data: %v", err)
	}

	tests := []struct {
		name        string
		post        Post
		expectError bool
	}{
		{
			name: "Update existing post",
			post: Post{
				ID:    1,
				Title: "Updated Test Post",
				Body:  "This is the updated body",
				Date:  time.Now().Format("Mon Jan _2 15:04:05 2006"),
			},
			expectError: false,
		},
		{
			name: "Update non-existing post",
			post: Post{
				ID:    999,
				Title: "Updated Test Post",
				Body:  "This is the updated body",
				Date:  time.Now().Format("Mon Jan _2 15:04:05 2006"),
			},
			expectError: false, // SQLite doesn't return error for UPDATE with no matching rows
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.post.UpdatePost(db)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				// For existing post, verify the update
				if tt.post.ID == 1 {
					var title, body, date string
					err = db.QueryRow("SELECT title, body, datepost FROM posts WHERE id = ?",
						tt.post.ID).Scan(&title, &body, &date)
					if err != nil {
						t.Errorf("Error retrieving updated post: %v", err)
					}
					if title != tt.post.Title {
						t.Errorf("Expected title %s, got %s", tt.post.Title, title)
					}
					if body != tt.post.Body {
						t.Errorf("Expected body %s, got %s", tt.post.Body, body)
					}
				}
			}
		})
	}
}

func TestPost_DeletePost(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Seed test data
	if err := seedTestData(db); err != nil {
		t.Fatalf("Failed to seed test data: %v", err)
	}

	tests := []struct {
		name        string
		postID      int
		expectError bool
	}{
		{
			name:        "Delete existing post",
			postID:      1,
			expectError: false,
		},
		{
			name:        "Delete non-existing post",
			postID:      999,
			expectError: false, // SQLite doesn't return error for DELETE with no matching rows
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			post := Post{ID: tt.postID}
			err := post.DeletePost(db)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				// For existing post, verify deletion
				if tt.postID == 1 {
					var count int
					err = db.QueryRow("SELECT COUNT(*) FROM posts WHERE id = ?", tt.postID).Scan(&count)
					if err != nil {
						t.Errorf("Error checking if post was deleted: %v", err)
					}
					if count != 0 {
						t.Errorf("Expected post to be deleted, but it still exists")
					}
				}
			}
		})
	}
}

func TestGetPosts(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Seed test data
	if err := seedTestData(db); err != nil {
		t.Fatalf("Failed to seed test data: %v", err)
	}

	tests := []struct {
		name          string
		count         int
		start         int
		expectedCount int
		expectError   bool
	}{
		{
			name:          "Get first 2 posts",
			count:         2,
			start:         0,
			expectedCount: 2,
			expectError:   false,
		},
		{
			name:          "Get posts with offset",
			count:         2,
			start:         1,
			expectedCount: 2,
			expectError:   false,
		},
		{
			name:          "Get more posts than available",
			count:         10,
			start:         0,
			expectedCount: 3, // Only 3 posts seeded
			expectError:   false,
		},
		{
			name:          "Get posts with large offset",
			count:         2,
			start:         10,
			expectedCount: 0,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			posts, err := GetPosts(db, tt.count, tt.start)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(posts) != tt.expectedCount {
					t.Errorf("Expected %d posts, got %d", tt.expectedCount, len(posts))
				}

				// Verify posts are ordered by ID desc
				for i := 1; i < len(posts); i++ {
					if posts[i-1].ID < posts[i].ID {
						t.Errorf("Posts are not ordered by ID desc")
						break
					}
				}
			}
		})
	}
}

func TestCountPosts(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Test with no posts
	count := CountPosts(db)
	if count != 0 {
		t.Errorf("Expected 0 posts, got %d", count)
	}

	// Seed test data
	if err := seedTestData(db); err != nil {
		t.Fatalf("Failed to seed test data: %v", err)
	}

	// Test with seeded posts
	count = CountPosts(db)
	if count != 3 {
		t.Errorf("Expected 3 posts, got %d", count)
	}
}

func TestComment_CreateComment(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Seed test data (need posts for comments)
	if err := seedTestData(db); err != nil {
		t.Fatalf("Failed to seed test data: %v", err)
	}

	tests := []struct {
		name        string
		comment     Comment
		expectError bool
	}{
		{
			name: "Create valid comment",
			comment: Comment{
				PostID: 1,
				Name:   "Test Commenter",
				Date:   time.Now().Format("Mon Jan _2 15:04:05 2006"),
				Data:   "This is a test comment",
			},
			expectError: false,
		},
		{
			name: "Create comment with empty name",
			comment: Comment{
				PostID: 1,
				Name:   "",
				Date:   time.Now().Format("Mon Jan _2 15:04:05 2006"),
				Data:   "This is a test comment",
			},
			expectError: false, // SQLite allows empty strings
		},
		{
			name: "Create comment with empty data",
			comment: Comment{
				PostID: 1,
				Name:   "Test Commenter",
				Date:   time.Now().Format("Mon Jan _2 15:04:05 2006"),
				Data:   "",
			},
			expectError: false, // SQLite allows empty strings
		},
		{
			name: "Create comment for non-existing post",
			comment: Comment{
				PostID: 999,
				Name:   "Test Commenter",
				Date:   time.Now().Format("Mon Jan _2 15:04:05 2006"),
				Data:   "This is a test comment",
			},
			expectError: false, // SQLite allows foreign key violations by default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.comment.CreateComment(db)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				// Verify the comment was created
				var count int
				err = db.QueryRow("SELECT COUNT(*) FROM comments WHERE postid = ? AND name = ? AND comment = ?",
					tt.comment.PostID, tt.comment.Name, tt.comment.Data).Scan(&count)
				if err != nil {
					t.Errorf("Error checking if comment was created: %v", err)
				}
				if count != 1 {
					t.Errorf("Expected 1 comment to be created, found %d", count)
				}
			}
		})
	}
}

func TestComment_DeleteComment(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Seed test data
	if err := seedTestData(db); err != nil {
		t.Fatalf("Failed to seed test data: %v", err)
	}

	tests := []struct {
		name        string
		commentID   int
		expectError bool
	}{
		{
			name:        "Delete existing comment",
			commentID:   1,
			expectError: false,
		},
		{
			name:        "Delete non-existing comment",
			commentID:   999,
			expectError: false, // SQLite doesn't return error for DELETE with no matching rows
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comment := Comment{CommentID: tt.commentID}
			err := comment.DeleteComment(db)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				// For existing comment, verify deletion
				if tt.commentID == 1 {
					var count int
					err = db.QueryRow("SELECT COUNT(*) FROM comments WHERE commentid = ?", tt.commentID).Scan(&count)
					if err != nil {
						t.Errorf("Error checking if comment was deleted: %v", err)
					}
					if count != 0 {
						t.Errorf("Expected comment to be deleted, but it still exists")
					}
				}
			}
		})
	}
}

func TestGetComments(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Seed test data
	if err := seedTestData(db); err != nil {
		t.Fatalf("Failed to seed test data: %v", err)
	}

	tests := []struct {
		name          string
		postID        int
		expectedCount int
		expectError   bool
	}{
		{
			name:          "Get comments for post with comments",
			postID:        1,
			expectedCount: 2, // Post 1 has 2 comments in seed data
			expectError:   false,
		},
		{
			name:          "Get comments for post with one comment",
			postID:        2,
			expectedCount: 1, // Post 2 has 1 comment in seed data
			expectError:   false,
		},
		{
			name:          "Get comments for post with no comments",
			postID:        3,
			expectedCount: 0, // Post 3 has no comments in seed data
			expectError:   false,
		},
		{
			name:          "Get comments for non-existing post",
			postID:        999,
			expectedCount: 0,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comments, err := GetComments(db, tt.postID)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(comments) != tt.expectedCount {
					t.Errorf("Expected %d comments, got %d", tt.expectedCount, len(comments))
				}

				// Verify all comments belong to the correct post
				for _, comment := range comments {
					if comment.PostID != tt.postID {
						t.Errorf("Expected comment PostID %d, got %d", tt.postID, comment.PostID)
					}
				}

				// Verify comments are ordered by postid desc
				for i := 1; i < len(comments); i++ {
					if comments[i-1].PostID < comments[i].PostID {
						t.Errorf("Comments are not ordered by postid desc")
						break
					}
				}
			}
		})
	}
}

func TestUser_IsUserExist(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Create a test user
	if err := createTestUser(db, "testuser", "password123", ADMIN); err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	tests := []struct {
		name     string
		username string
		expected bool
	}{
		{
			name:     "Existing user",
			username: "testuser",
			expected: true,
		},
		{
			name:     "Non-existing user",
			username: "nonexistent",
			expected: false,
		},
		{
			name:     "Empty username",
			username: "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := User{Name: tt.username}
			result := user.IsUserExist(db)

			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestUser_CreateUser(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	tests := []struct {
		name        string
		user        User
		password    string
		expectError bool
	}{
		{
			name: "Create valid admin user",
			user: User{
				Name: "admin",
				Type: ADMIN,
			},
			password:    "validpassword123",
			expectError: false,
		},
		{
			name: "Create valid github user",
			user: User{
				Name: "admin",
				Type: GITHUB,
			},
			password:    "validpassword123",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear users table first
			db.Exec("DELETE FROM users")

			// Hash the password before passing to CreateUser (as the function expects)
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(tt.password), bcrypt.DefaultCost)
			if err != nil {
				t.Fatalf("Failed to hash password: %v", err)
			}

			err = tt.user.CreateUser(db, string(hashedPassword))

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				// Verify the user was created
				var count int
				err = db.QueryRow("SELECT COUNT(*) FROM users WHERE name = ? AND type = ?",
					"admin", tt.user.Type).Scan(&count) // Note: CreateUser hardcodes "admin" as name
				if err != nil {
					t.Errorf("Error checking if user was created: %v", err)
				}
				if count != 1 {
					t.Errorf("Expected 1 user to be created, found %d", count)
				}

				// Verify password is stored as provided (already hashed)
				var storedPassword string
				err = db.QueryRow("SELECT pass FROM users WHERE name = ?", "admin").Scan(&storedPassword)
				if err != nil {
					t.Errorf("Error retrieving stored password: %v", err)
				}
				if storedPassword != string(hashedPassword) {
					t.Errorf("Stored password doesn't match provided hashed password")
				}
			}
		})
	}
}

func TestUser_IsAdmin(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Create test users
	if err := createTestUser(db, "adminuser", "password123", ADMIN); err != nil {
		t.Fatalf("Failed to create admin user: %v", err)
	}
	if err := createTestUser(db, "githubuser", "password123", GITHUB); err != nil {
		t.Fatalf("Failed to create github user: %v", err)
	}

	tests := []struct {
		name     string
		username string
		expected bool
	}{
		{
			name:     "Admin user",
			username: "adminuser",
			expected: true,
		},
		{
			name:     "GitHub user",
			username: "githubuser",
			expected: false,
		},
		{
			name:     "Non-existing user",
			username: "nonexistent",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := User{Name: tt.username}
			result := user.IsAdmin(db)

			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestUser_CheckCredentials(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Create test user with known password
	password := "testpassword123"
	if err := createTestUser(db, "testuser", password, ADMIN); err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	tests := []struct {
		name     string
		username string
		password string
		expected bool
	}{
		{
			name:     "Valid credentials",
			username: "testuser",
			password: "testpassword123",
			expected: true,
		},
		{
			name:     "Invalid password",
			username: "testuser",
			password: "wrongpassword",
			expected: false,
		},
		{
			name:     "Non-existing user",
			username: "nonexistent",
			password: "anypassword",
			expected: false,
		},
		{
			name:     "Empty password",
			username: "testuser",
			password: "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := User{Name: tt.username}
			result := user.CheckCredentials(db, tt.password)

			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestMigrateDatabase(t *testing.T) {
	// Create a fresh database without migrations
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Run migrations
	MigrateDatabase(db)

	// Verify tables were created
	tables := []string{"posts", "comments", "users"}
	for _, table := range tables {
		var count int
		query := `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`
		err := db.QueryRow(query, table).Scan(&count)
		if err != nil {
			t.Errorf("Error checking if table %s exists: %v", table, err)
		}
		if count != 1 {
			t.Errorf("Expected table %s to exist, but it doesn't", table)
		}
	}

	// Verify we can insert data into each table
	// Test posts table
	_, err = db.Exec(`INSERT INTO posts (title, body, datepost) VALUES (?, ?, ?)`,
		"Test", "Body", "Date")
	if err != nil {
		t.Errorf("Error inserting into posts table: %v", err)
	}

	// Test comments table
	_, err = db.Exec(`INSERT INTO comments (postid, name, date, comment) VALUES (?, ?, ?, ?)`,
		1, "Name", "Date", "Comment")
	if err != nil {
		t.Errorf("Error inserting into comments table: %v", err)
	}

	// Test users table
	_, err = db.Exec(`INSERT INTO users (name, type, pass) VALUES (?, ?, ?)`,
		"user", 1, "pass")
	if err != nil {
		t.Errorf("Error inserting into users table: %v", err)
	}
}

func TestPost_ValidateAndSanitizeSEOFields(t *testing.T) {
	tests := []struct {
		name         string
		post         Post
		expectedMeta string
		expectedKeys string
	}{
		{
			name: "Valid SEO fields",
			post: Post{
				MetaDescription: "This is a valid meta description",
				Keywords:        "keyword1, keyword2, keyword3",
			},
			expectedMeta: "This is a valid meta description",
			expectedKeys: "keyword1, keyword2, keyword3",
		},
		{
			name: "Long meta description gets truncated",
			post: Post{
				MetaDescription: "This is a very long meta description that exceeds the recommended 160 character limit and should be truncated to fit within the SEO guidelines for search engines",
			},
			expectedMeta: "This is a very long meta description that exceeds the recommended 160 character limit and should be truncated to fit within the SEO guidelines for search eng...",
			expectedKeys: "",
		},
		{
			name: "HTML in meta description gets escaped",
			post: Post{
				MetaDescription: "This has <script>alert('xss')</script> HTML tags",
				Keywords:        "test, <script>alert('xss')</script>",
			},
			expectedMeta: "This has &lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt; HTML tags",
			expectedKeys: "test, &lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;",
		},
		{
			name: "Too many keywords get limited",
			post: Post{
				Keywords: "k1, k2, k3, k4, k5, k6, k7, k8, k9, k10, k11, k12, k13, k14, k15",
			},
			expectedMeta: "",
			expectedKeys: "k1, k2, k3, k4, k5, k6, k7, k8, k9, k10",
		},
		{
			name: "Keywords with extra spaces get cleaned",
			post: Post{
				Keywords: "  keyword1  ,   keyword2   ,keyword3,   keyword4  ",
			},
			expectedMeta: "",
			expectedKeys: "keyword1, keyword2, keyword3, keyword4",
		},
		{
			name: "Long individual keywords get filtered out",
			post: Post{
				Keywords: "short, verylongkeywordthatexceedsfiftycharacterslimitandshouldbefilteredout, another",
			},
			expectedMeta: "",
			expectedKeys: "short, another",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.post.ValidateAndSanitizeSEOFields()
			if err != nil {
				t.Errorf("ValidateAndSanitizeSEOFields() error = %v", err)
				return
			}

			if tt.post.MetaDescription != tt.expectedMeta {
				t.Errorf("MetaDescription = %v, want %v", tt.post.MetaDescription, tt.expectedMeta)
			}

			if tt.post.Keywords != tt.expectedKeys {
				t.Errorf("Keywords = %v, want %v", tt.post.Keywords, tt.expectedKeys)
			}
		})
	}
}

func TestPost_GenerateDefaultSEOFields(t *testing.T) {
	tests := []struct {
		name         string
		post         Post
		expectedMeta string
		expectedKeys string
	}{
		{
			name: "Generate meta description from body",
			post: Post{
				Title: "Test Post",
				Body:  "This is a test post body that should be used to generate a meta description when none is provided.",
			},
			expectedMeta: "This is a test post body that should be used to generate a meta description when none is provided.",
			expectedKeys: "test, post",
		},
		{
			name: "Generate meta description from long body",
			post: Post{
				Title: "Test Post",
				Body:  "This is a very long test post body that exceeds the 150 character limit and should be truncated when generating the meta description automatically from the post content.",
			},
			expectedMeta: "This is a very long test post body that exceeds the 150 character limit and should be truncated when generating the meta description automatically...",
			expectedKeys: "test, post",
		},
		{
			name: "Generate meta description from HTML body",
			post: Post{
				Title: "Test Post",
				Body:  "<p>This is a <strong>test</strong> post with <em>HTML</em> tags that should be stripped.</p>",
			},
			expectedMeta: "This is a test post with HTML tags that should be stripped.",
			expectedKeys: "test, post",
		},
		{
			name: "Generate keywords from title",
			post: Post{
				Title: "Advanced JavaScript Programming Techniques",
				Body:  "Some body content",
			},
			expectedMeta: "Read this blog post about Advanced JavaScript Programming Techniques to learn more about the topic and get insights.",
			expectedKeys: "advanced, javascript, programming, techniques",
		},
		{
			name: "Filter short words from title keywords",
			post: Post{
				Title: "How to Use Go for Web Development",
				Body:  "Some body content",
			},
			expectedMeta: "Read this blog post about How to Use Go for Web Development to learn more about the topic and get insights.",
			expectedKeys: "development",
		},
		{
			name: "Don't override existing SEO fields",
			post: Post{
				Title:           "Test Post",
				Body:            "This is a test post body",
				MetaDescription: "Existing meta description that is long enough to not be overridden by the new logic",
				Keywords:        "existing, keywords",
			},
			expectedMeta: "Existing meta description that is long enough to not be overridden by the new logic",
			expectedKeys: "existing, keywords",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.post.GenerateDefaultSEOFields()

			if tt.post.MetaDescription != tt.expectedMeta {
				t.Errorf("MetaDescription = %v, want %v", tt.post.MetaDescription, tt.expectedMeta)
			}

			if tt.post.Keywords != tt.expectedKeys {
				t.Errorf("Keywords = %v, want %v", tt.post.Keywords, tt.expectedKeys)
			}
		})
	}
}

func TestPost_CreatePostWithSEO(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	tests := []struct {
		name        string
		post        Post
		expectError bool
	}{
		{
			name: "Create post with SEO fields",
			post: Post{
				Title:           "SEO Test Post",
				Body:            "This is a test post with SEO fields",
				Date:            time.Now().Format("Mon Jan _2 15:04:05 2006"),
				MetaDescription: "Custom meta description",
				Keywords:        "seo, test, post",
			},
			expectError: false,
		},
		{
			name: "Create post without SEO fields (auto-generated)",
			post: Post{
				Title: "Auto SEO Test Post",
				Body:  "This is a test post that should have SEO fields auto-generated from the content",
				Date:  time.Now().Format("Mon Jan _2 15:04:05 2006"),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.post.CreatePost(db)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				// Verify the post was created with proper SEO fields
				var metaDesc, keywords string
				err = db.QueryRow("SELECT meta_description, keywords FROM posts WHERE id = ?",
					tt.post.ID).Scan(&metaDesc, &keywords)
				if err != nil {
					t.Errorf("Error retrieving SEO fields: %v", err)
				}

				// For posts with explicit SEO fields, verify they were saved
				if tt.post.MetaDescription != "" {
					if metaDesc != tt.post.MetaDescription {
						t.Errorf("Expected meta description %s, got %s", tt.post.MetaDescription, metaDesc)
					}
				} else {
					// For auto-generated SEO, verify something was generated
					if metaDesc == "" {
						t.Errorf("Expected auto-generated meta description, got empty string")
					}
				}
			}
		})
	}
}

// Additional comprehensive tests for edge cases and error conditions

func TestPost_GetPost_EdgeCases(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	tests := []struct {
		name        string
		postID      int
		expectError bool
		errorType   error
	}{
		{
			name:        "Get post with ID 0",
			postID:      0,
			expectError: true,
			errorType:   sql.ErrNoRows,
		},
		{
			name:        "Get post with negative ID",
			postID:      -1,
			expectError: true,
			errorType:   sql.ErrNoRows,
		},
		{
			name:        "Get post with very large ID",
			postID:      999999999,
			expectError: true,
			errorType:   sql.ErrNoRows,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			post := Post{ID: tt.postID}
			err := post.GetPost(db)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if err != tt.errorType {
					t.Errorf("Expected error type %v, got %v", tt.errorType, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestPost_UpdatePost_EdgeCases(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Seed test data
	if err := seedTestData(db); err != nil {
		t.Fatalf("Failed to seed test data: %v", err)
	}

	tests := []struct {
		name        string
		post        Post
		expectError bool
	}{
		{
			name: "Update with very long title",
			post: Post{
				ID:    1,
				Title: strings.Repeat("A", 1000),
				Body:  "Updated body",
				Date:  time.Now().Format("Mon Jan _2 15:04:05 2006"),
			},
			expectError: false, // SQLite allows long strings
		},
		{
			name: "Update with very long body",
			post: Post{
				ID:    1,
				Title: "Updated title",
				Body:  strings.Repeat("B", 10000),
				Date:  time.Now().Format("Mon Jan _2 15:04:05 2006"),
			},
			expectError: false, // SQLite allows long strings
		},
		{
			name: "Update with special characters",
			post: Post{
				ID:    1,
				Title: "Title with special chars: !@#$%^&*()",
				Body:  "Body with unicode: 你好世界 🌍",
				Date:  time.Now().Format("Mon Jan _2 15:04:05 2006"),
			},
			expectError: false,
		},
		{
			name: "Update with malicious SEO content",
			post: Post{
				ID:              1,
				Title:           "Test Post",
				Body:            "Test body",
				Date:            time.Now().Format("Mon Jan _2 15:04:05 2006"),
				MetaDescription: "<script>alert('xss')</script>",
				Keywords:        "<script>alert('xss')</script>, malicious",
			},
			expectError: false, // Should sanitize, not error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.post.UpdatePost(db)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				// Verify malicious content was sanitized
				if strings.Contains(tt.post.MetaDescription, "<script>") {
					var metaDesc string
					err = db.QueryRow("SELECT meta_description FROM posts WHERE id = ?", tt.post.ID).Scan(&metaDesc)
					if err != nil {
						t.Errorf("Error retrieving meta description: %v", err)
					}
					if strings.Contains(metaDesc, "<script>") {
						t.Errorf("Expected script tags to be escaped, but found: %s", metaDesc)
					}
				}
			}
		})
	}
}

func TestPost_CreatePost_EdgeCases(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	tests := []struct {
		name        string
		post        Post
		expectError bool
	}{
		{
			name: "Create post with very long title",
			post: Post{
				Title: strings.Repeat("A", 1000),
				Body:  "Test body",
				Date:  time.Now().Format("Mon Jan _2 15:04:05 2006"),
			},
			expectError: false, // SQLite allows long strings
		},
		{
			name: "Create post with unicode characters",
			post: Post{
				Title: "Unicode test: 你好世界 🌍 Ñoño",
				Body:  "Unicode body: العربية русский 日本語",
				Date:  time.Now().Format("Mon Jan _2 15:04:05 2006"),
			},
			expectError: false,
		},
		{
			name: "Create post with null bytes (should be handled)",
			post: Post{
				Title: "Title with\x00null byte",
				Body:  "Body with\x00null byte",
				Date:  time.Now().Format("Mon Jan _2 15:04:05 2006"),
			},
			expectError: false, // SQLite handles null bytes
		},
		{
			name: "Create post with only whitespace title",
			post: Post{
				Title: "   \t\n   ",
				Body:  "Test body",
				Date:  time.Now().Format("Mon Jan _2 15:04:05 2006"),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.post.CreatePost(db)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				// Verify post was created and has an ID
				if tt.post.ID <= 0 {
					t.Errorf("Expected post ID to be set after creation, got %d", tt.post.ID)
				}
			}
		})
	}
}

func TestComment_CreateComment_EdgeCases(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Seed test data (need posts for comments)
	if err := seedTestData(db); err != nil {
		t.Fatalf("Failed to seed test data: %v", err)
	}

	tests := []struct {
		name        string
		comment     Comment
		expectError bool
	}{
		{
			name: "Create comment with very long name",
			comment: Comment{
				PostID: 1,
				Name:   strings.Repeat("A", 1000),
				Date:   time.Now().Format("Mon Jan _2 15:04:05 2006"),
				Data:   "Test comment",
			},
			expectError: false, // SQLite allows long strings
		},
		{
			name: "Create comment with very long data",
			comment: Comment{
				PostID: 1,
				Name:   "Test User",
				Date:   time.Now().Format("Mon Jan _2 15:04:05 2006"),
				Data:   strings.Repeat("B", 10000),
			},
			expectError: false, // SQLite allows long strings
		},
		{
			name: "Create comment with unicode characters",
			comment: Comment{
				PostID: 1,
				Name:   "用户 🙂",
				Date:   time.Now().Format("Mon Jan _2 15:04:05 2006"),
				Data:   "评论内容 with emoji 🎉",
			},
			expectError: false,
		},
		{
			name: "Create comment with HTML content",
			comment: Comment{
				PostID: 1,
				Name:   "HTML User",
				Date:   time.Now().Format("Mon Jan _2 15:04:05 2006"),
				Data:   "<script>alert('xss')</script><p>HTML content</p>",
			},
			expectError: false, // Model doesn't sanitize, that's handled at handler level
		},
		{
			name: "Create comment with zero PostID",
			comment: Comment{
				PostID: 0,
				Name:   "Test User",
				Date:   time.Now().Format("Mon Jan _2 15:04:05 2006"),
				Data:   "Test comment",
			},
			expectError: false, // SQLite allows this
		},
		{
			name: "Create comment with negative PostID",
			comment: Comment{
				PostID: -1,
				Name:   "Test User",
				Date:   time.Now().Format("Mon Jan _2 15:04:05 2006"),
				Data:   "Test comment",
			},
			expectError: false, // SQLite allows this
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.comment.CreateComment(db)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				// Verify the comment was created
				var count int
				err = db.QueryRow("SELECT COUNT(*) FROM comments WHERE postid = ? AND name = ? AND comment = ?",
					tt.comment.PostID, tt.comment.Name, tt.comment.Data).Scan(&count)
				if err != nil {
					t.Errorf("Error checking if comment was created: %v", err)
				}
				if count != 1 {
					t.Errorf("Expected 1 comment to be created, found %d", count)
				}
			}
		})
	}
}

func TestUser_IsUserExist_EdgeCases(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Create test users with various names
	testUsers := []struct {
		name     string
		password string
		userType int
	}{
		{"normaluser", "password", ADMIN},
		{"user with spaces", "password", ADMIN},
		{"用户", "password", ADMIN}, // Unicode username
		{"user@email.com", "password", ADMIN},
		{"", "password", ADMIN}, // Empty username
	}

	for _, user := range testUsers {
		if err := createTestUser(db, user.name, user.password, user.userType); err != nil {
			t.Fatalf("Failed to create test user %s: %v", user.name, err)
		}
	}

	tests := []struct {
		name     string
		username string
		expected bool
	}{
		{
			name:     "Normal username",
			username: "normaluser",
			expected: true,
		},
		{
			name:     "Username with spaces",
			username: "user with spaces",
			expected: true,
		},
		{
			name:     "Unicode username",
			username: "用户",
			expected: true,
		},
		{
			name:     "Email-like username",
			username: "user@email.com",
			expected: true,
		},
		{
			name:     "Empty username",
			username: "",
			expected: true, // Empty username was created
		},
		{
			name:     "Case sensitive check",
			username: "NORMALUSER",
			expected: false, // SQLite is case-sensitive by default
		},
		{
			name:     "Username with special chars",
			username: "user!@#$%",
			expected: false,
		},
		{
			name:     "Very long username",
			username: strings.Repeat("A", 1000),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := User{Name: tt.username}
			result := user.IsUserExist(db)

			if result != tt.expected {
				t.Errorf("Expected %v, got %v for username '%s'", tt.expected, result, tt.username)
			}
		})
	}
}

func TestUser_CheckCredentials_EdgeCases(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Create test users with various passwords
	testCases := []struct {
		username string
		password string
		userType int
	}{
		{"user1", "simplepass", ADMIN},
		{"user2", "complex!@#$%^&*()_+", ADMIN},
		{"user3", "unicode密码🔐", ADMIN},
		{"user4", strings.Repeat("A", 70), ADMIN}, // Long password (within bcrypt limit)
		{"user5", "", ADMIN},                       // Empty password (bcrypt will hash it)
	}

	for _, tc := range testCases {
		if err := createTestUser(db, tc.username, tc.password, tc.userType); err != nil {
			t.Fatalf("Failed to create test user %s: %v", tc.username, err)
		}
	}

	tests := []struct {
		name     string
		username string
		password string
		expected bool
	}{
		{
			name:     "Correct simple password",
			username: "user1",
			password: "simplepass",
			expected: true,
		},
		{
			name:     "Correct complex password",
			username: "user2",
			password: "complex!@#$%^&*()_+",
			expected: true,
		},
		{
			name:     "Correct unicode password",
			username: "user3",
			password: "unicode密码🔐",
			expected: true,
		},
		{
			name:     "Correct long password",
			username: "user4",
			password: strings.Repeat("A", 70),
			expected: true,
		},
		{
			name:     "Correct empty password",
			username: "user5",
			password: "",
			expected: true,
		},
		{
			name:     "Wrong password",
			username: "user1",
			password: "wrongpass",
			expected: false,
		},
		{
			name:     "Case sensitive password",
			username: "user1",
			password: "SIMPLEPASS",
			expected: false,
		},
		{
			name:     "Password with extra characters",
			username: "user1",
			password: "simplepass123",
			expected: false,
		},
		{
			name:     "Non-existent user",
			username: "nonexistent",
			password: "anypassword",
			expected: false,
		},
		{
			name:     "Empty username",
			username: "",
			password: "anypassword",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := User{Name: tt.username}
			result := user.CheckCredentials(db, tt.password)

			if result != tt.expected {
				t.Errorf("Expected %v, got %v for user '%s' with password '%s'", tt.expected, result, tt.username, tt.password)
			}
		})
	}
}

func TestUser_CreateUser_EdgeCases(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	tests := []struct {
		name        string
		user        User
		password    string
		expectError bool
		setupFunc   func() // Function to run before test (e.g., create conflicting user)
	}{
		{
			name: "Create user with very long password",
			user: User{
				Name: "admin",
				Type: ADMIN,
			},
			password:    strings.Repeat("A", 70), // bcrypt limit is 72 bytes
			expectError: false,
		},
		{
			name: "Create user with unicode password",
			user: User{
				Name: "admin",
				Type: GITHUB,
			},
			password:    "密码🔐unicode",
			expectError: false,
		},
		{
			name: "Create user with empty password",
			user: User{
				Name: "admin",
				Type: ADMIN,
			},
			password:    "",
			expectError: false, // bcrypt can hash empty strings
		},
		{
			name: "Create duplicate user",
			user: User{
				Name: "admin",
				Type: ADMIN,
			},
			password:    "password123",
			expectError: true, // Should fail due to unique constraint
			setupFunc: func() {
				// Create a user first
				hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("existing"), bcrypt.DefaultCost)
				db.Exec("INSERT INTO users (name, type, pass) VALUES (?, ?, ?)", "admin", ADMIN, string(hashedPassword))
			},
		},
		{
			name: "Create user with invalid type",
			user: User{
				Name: "admin",
				Type: 999, // Invalid user type
			},
			password:    "password123",
			expectError: false, // SQLite allows any integer
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear users table first
			db.Exec("DELETE FROM users")

			// Run setup function if provided
			if tt.setupFunc != nil {
				tt.setupFunc()
			}

			// Hash the password before passing to CreateUser
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(tt.password), bcrypt.DefaultCost)
			if err != nil {
				t.Fatalf("Failed to hash password: %v", err)
			}

			err = tt.user.CreateUser(db, string(hashedPassword))

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestGetPosts_EdgeCases(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Seed test data
	if err := seedTestData(db); err != nil {
		t.Fatalf("Failed to seed test data: %v", err)
	}

	tests := []struct {
		name          string
		count         int
		start         int
		expectedCount int
		expectError   bool
	}{
		{
			name:          "Negative count",
			count:         -1,
			start:         0,
			expectedCount: 3, // SQLite treats negative limit as no limit
			expectError:   false,
		},
		{
			name:          "Zero count",
			count:         0,
			start:         0,
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:          "Negative start",
			count:         2,
			start:         -1,
			expectedCount: 2, // SQLite treats negative offset as 0
			expectError:   false,
		},
		{
			name:          "Very large count",
			count:         999999,
			start:         0,
			expectedCount: 3, // Only 3 posts available
			expectError:   false,
		},
		{
			name:          "Very large start",
			count:         2,
			start:         999999,
			expectedCount: 0,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			posts, err := GetPosts(db, tt.count, tt.start)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(posts) != tt.expectedCount {
					t.Errorf("Expected %d posts, got %d", tt.expectedCount, len(posts))
				}
			}
		})
	}
}

func TestGetComments_EdgeCases(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Seed test data
	if err := seedTestData(db); err != nil {
		t.Fatalf("Failed to seed test data: %v", err)
	}

	tests := []struct {
		name          string
		postID        int
		expectedCount int
		expectError   bool
	}{
		{
			name:          "Zero post ID",
			postID:        0,
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:          "Negative post ID",
			postID:        -1,
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:          "Very large post ID",
			postID:        999999,
			expectedCount: 0,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comments, err := GetComments(db, tt.postID)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(comments) != tt.expectedCount {
					t.Errorf("Expected %d comments, got %d", tt.expectedCount, len(comments))
				}
			}
		})
	}
}

// Test File model methods

func TestFile_CreateFile(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	tests := []struct {
		name        string
		file        File
		expectError bool
	}{
		{
			name: "Create valid file",
			file: File{
				UUID:         "test-uuid-1",
				OriginalName: "test.txt",
				StoredName:   "stored-test.txt",
				Path:         "/uploads/test.txt",
				Size:         1024,
				MimeType:     "text/plain",
				IsImage:      false,
			},
			expectError: false,
		},
		{
			name: "Create valid image file",
			file: File{
				UUID:         "test-uuid-2",
				OriginalName: "image.jpg",
				StoredName:   "stored-image.jpg",
				Path:         "/uploads/image.jpg",
				Size:         2048,
				MimeType:     "image/jpeg",
				IsImage:      true,
				Width:        func() *int { w := 800; return &w }(),
				Height:       func() *int { h := 600; return &h }(),
				ThumbnailPath: func() *string { p := "/uploads/thumb.jpg"; return &p }(),
				AltText:      func() *string { a := "Test image"; return &a }(),
			},
			expectError: false,
		},
		{
			name: "Create file with duplicate UUID",
			file: File{
				UUID:         "test-uuid-1", // Duplicate UUID
				OriginalName: "test2.txt",
				StoredName:   "stored-test2.txt",
				Path:         "/uploads/test2.txt",
				Size:         512,
				MimeType:     "text/plain",
				IsImage:      false,
			},
			expectError: true, // Should fail due to unique constraint
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.file.CreateFile(db)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				// Verify the file was created and has an ID
				if tt.file.ID <= 0 {
					t.Errorf("Expected file ID to be set after creation, got %d", tt.file.ID)
				}

				// Verify file can be retrieved
				retrievedFile := File{ID: tt.file.ID}
				err = retrievedFile.GetFile(db)
				if err != nil {
					t.Errorf("Error retrieving created file: %v", err)
				}
				if retrievedFile.UUID != tt.file.UUID {
					t.Errorf("Expected UUID %s, got %s", tt.file.UUID, retrievedFile.UUID)
				}
			}
		})
	}
}

func TestFile_GetFile(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Create a test file
	testFile := File{
		UUID:         "test-get-uuid",
		OriginalName: "test-get.txt",
		StoredName:   "stored-get.txt",
		Path:         "/uploads/get.txt",
		Size:         1024,
		MimeType:     "text/plain",
		IsImage:      false,
	}
	err := testFile.CreateFile(db)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name        string
		fileID      int
		expectError bool
	}{
		{
			name:        "Get existing file",
			fileID:      testFile.ID,
			expectError: false,
		},
		{
			name:        "Get non-existing file",
			fileID:      999,
			expectError: true,
		},
		{
			name:        "Get file with zero ID",
			fileID:      0,
			expectError: true,
		},
		{
			name:        "Get file with negative ID",
			fileID:      -1,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := File{ID: tt.fileID}
			err := file.GetFile(db)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if file.UUID != testFile.UUID {
					t.Errorf("Expected UUID %s, got %s", testFile.UUID, file.UUID)
				}
			}
		})
	}
}

func TestFile_GetFileByUUID(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Create a test file
	testFile := File{
		UUID:         "test-uuid-get",
		OriginalName: "test-uuid.txt",
		StoredName:   "stored-uuid.txt",
		Path:         "/uploads/uuid.txt",
		Size:         1024,
		MimeType:     "text/plain",
		IsImage:      false,
	}
	err := testFile.CreateFile(db)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name        string
		uuid        string
		expectError bool
	}{
		{
			name:        "Get existing file by UUID",
			uuid:        testFile.UUID,
			expectError: false,
		},
		{
			name:        "Get non-existing file by UUID",
			uuid:        "non-existing-uuid",
			expectError: true,
		},
		{
			name:        "Get file with empty UUID",
			uuid:        "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := File{UUID: tt.uuid}
			err := file.GetFileByUUID(db)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if file.ID != testFile.ID {
					t.Errorf("Expected ID %d, got %d", testFile.ID, file.ID)
				}
			}
		})
	}
}

func TestFile_DeleteFile(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Create test files
	testFile1 := File{
		UUID:         "test-delete-1",
		OriginalName: "delete1.txt",
		StoredName:   "stored-delete1.txt",
		Path:         "/uploads/delete1.txt",
		Size:         1024,
		MimeType:     "text/plain",
		IsImage:      false,
	}
	err := testFile1.CreateFile(db)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name        string
		fileID      int
		expectError bool
	}{
		{
			name:        "Delete existing file",
			fileID:      testFile1.ID,
			expectError: false,
		},
		{
			name:        "Delete non-existing file",
			fileID:      999,
			expectError: false, // SQLite doesn't error on DELETE with no matches
		},
		{
			name:        "Delete file with zero ID",
			fileID:      0,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := File{ID: tt.fileID}
			err := file.DeleteFile(db)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				// For existing file, verify deletion
				if tt.fileID == testFile1.ID {
					var count int
					err = db.QueryRow("SELECT COUNT(*) FROM files WHERE id = ?", tt.fileID).Scan(&count)
					if err != nil {
						t.Errorf("Error checking if file was deleted: %v", err)
					}
					if count != 0 {
						t.Errorf("Expected file to be deleted, but it still exists")
					}
				}
			}
		})
	}
}

func TestFile_IncrementDownloadCount(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Create a test file
	testFile := File{
		UUID:          "test-download-uuid",
		OriginalName:  "download.txt",
		StoredName:    "stored-download.txt",
		Path:          "/uploads/download.txt",
		Size:          1024,
		MimeType:      "text/plain",
		DownloadCount: 0,
		IsImage:       false,
	}
	err := testFile.CreateFile(db)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name        string
		fileID      int
		expectError bool
	}{
		{
			name:        "Increment download count for existing file",
			fileID:      testFile.ID,
			expectError: false,
		},
		{
			name:        "Increment download count for non-existing file",
			fileID:      999,
			expectError: false, // SQLite doesn't error on UPDATE with no matches
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := File{ID: tt.fileID}
			err := file.IncrementDownloadCount(db)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				// For existing file, verify download count was incremented
				if tt.fileID == testFile.ID {
					var downloadCount int
					err = db.QueryRow("SELECT download_count FROM files WHERE id = ?", tt.fileID).Scan(&downloadCount)
					if err != nil {
						t.Errorf("Error checking download count: %v", err)
					}
					if downloadCount != 1 {
						t.Errorf("Expected download count to be 1, got %d", downloadCount)
					}
				}
			}
		})
	}
}

func TestGetFiles(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Create test files
	testFiles := []File{
		{
			UUID:         "file-1",
			OriginalName: "file1.txt",
			StoredName:   "stored1.txt",
			Path:         "/uploads/file1.txt",
			Size:         1024,
			MimeType:     "text/plain",
			IsImage:      false,
		},
		{
			UUID:         "file-2",
			OriginalName: "file2.jpg",
			StoredName:   "stored2.jpg",
			Path:         "/uploads/file2.jpg",
			Size:         2048,
			MimeType:     "image/jpeg",
			IsImage:      true,
		},
		{
			UUID:         "file-3",
			OriginalName: "file3.pdf",
			StoredName:   "stored3.pdf",
			Path:         "/uploads/file3.pdf",
			Size:         4096,
			MimeType:     "application/pdf",
			IsImage:      false,
		},
	}

	for i := range testFiles {
		err := testFiles[i].CreateFile(db)
		if err != nil {
			t.Fatalf("Failed to create test file %d: %v", i, err)
		}
	}

	tests := []struct {
		name          string
		limit         int
		offset        int
		expectedCount int
		expectError   bool
	}{
		{
			name:          "Get first 2 files",
			limit:         2,
			offset:        0,
			expectedCount: 2,
			expectError:   false,
		},
		{
			name:          "Get files with offset",
			limit:         2,
			offset:        1,
			expectedCount: 2,
			expectError:   false,
		},
		{
			name:          "Get all files",
			limit:         10,
			offset:        0,
			expectedCount: 3,
			expectError:   false,
		},
		{
			name:          "Get files with large offset",
			limit:         2,
			offset:        10,
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:          "Get files with zero limit",
			limit:         0,
			offset:        0,
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:          "Get files with negative limit",
			limit:         -1,
			offset:        0,
			expectedCount: 3, // SQLite treats negative limit as no limit
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := GetFiles(db, tt.limit, tt.offset)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(files) != tt.expectedCount {
					t.Errorf("Expected %d files, got %d", tt.expectedCount, len(files))
				}

				// Verify files are ordered by created_at desc (newest first)
				for i := 1; i < len(files); i++ {
					// Since we're using CURRENT_TIMESTAMP, we can't easily test exact ordering
					// but we can verify the structure is correct
					if files[i].ID <= 0 {
						t.Errorf("File %d has invalid ID: %d", i, files[i].ID)
					}
				}
			}
		})
	}
}

func TestPost_CreatePostWithSEOIntegration(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	tests := []struct {
		name string
		post Post
	}{
		{
			name: "Create post with SEO fields",
			post: Post{
				Title:           "Test Post with SEO",
				Body:            "This is a test post body",
				Date:            "Mon Jan 1 12:00:00 2024",
				MetaDescription: "Custom meta description",
				Keywords:        "test, seo, post",
			},
		},
		{
			name: "Create post without SEO fields (auto-generated)",
			post: Post{
				Title: "Another Test Post for SEO Generation",
				Body:  "This is another test post body that should generate default SEO fields automatically when none are provided.",
				Date:  "Mon Jan 2 12:00:00 2024",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.post.CreatePost(db)
			if err != nil {
				t.Errorf("CreatePost() error = %v", err)
				return
			}

			// Verify the post was created with SEO fields
			var retrievedPost Post
			retrievedPost.ID = tt.post.ID
			err = retrievedPost.GetPost(db)
			if err != nil {
				t.Errorf("GetPost() error = %v", err)
				return
			}

			// Check that SEO fields are present (either provided or auto-generated)
			if retrievedPost.MetaDescription == "" {
				t.Error("MetaDescription should not be empty after creation")
			}

			if retrievedPost.Keywords == "" {
				t.Error("Keywords should not be empty after creation")
			}

			// Verify other fields
			if retrievedPost.Title != tt.post.Title {
				t.Errorf("Title = %v, want %v", retrievedPost.Title, tt.post.Title)
			}
		})
	}
}

func TestPost_UpdatePostWithSEO(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Create a test post first
	post := Post{
		Title: "Original Title",
		Body:  "Original body",
		Date:  "Mon Jan 1 12:00:00 2024",
	}
	err := post.CreatePost(db)
	if err != nil {
		t.Fatalf("Failed to create test post: %v", err)
	}

	// Update the post with SEO fields
	post.Title = "Updated Title"
	post.MetaDescription = "Updated meta description"
	post.Keywords = "updated, keywords, test"

	err = post.UpdatePost(db)
	if err != nil {
		t.Errorf("UpdatePost() error = %v", err)
		return
	}

	// Verify the update
	var retrievedPost Post
	retrievedPost.ID = post.ID
	err = retrievedPost.GetPost(db)
	if err != nil {
		t.Errorf("GetPost() error = %v", err)
		return
	}

	if retrievedPost.MetaDescription != "Updated meta description" {
		t.Errorf("MetaDescription = %v, want %v", retrievedPost.MetaDescription, "Updated meta description")
	}

	if retrievedPost.Keywords != "updated, keywords, test" {
		t.Errorf("Keywords = %v, want %v", retrievedPost.Keywords, "updated, keywords, test")
	}
}

func TestPostSEOFields(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Insert a test post with SEO fields
	_, err := db.Exec(`INSERT INTO posts (title, body, datepost, slug, meta_description, keywords, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		"SEO Test Post", "Test body with SEO fields", "Mon Jan 2 15:04:05 2006", "seo-test-post",
		"This is a meta description for SEO", "seo, test, blog, post")
	if err != nil {
		t.Fatalf("Failed to insert test post with SEO fields: %v", err)
	}

	// Test getting post with SEO fields
	post := Post{Slug: "seo-test-post"}
	err = post.GetPostBySlug(db)
	if err != nil {
		t.Errorf("Failed to get post by slug: %v", err)
	}

	if post.Title != "SEO Test Post" {
		t.Errorf("Expected title 'SEO Test Post', got '%s'", post.Title)
	}

	if post.MetaDescription != "This is a meta description for SEO" {
		t.Errorf("Expected meta description 'This is a meta description for SEO', got '%s'", post.MetaDescription)
	}

	if post.Keywords != "seo, test, blog, post" {
		t.Errorf("Expected keywords 'seo, test, blog, post', got '%s'", post.Keywords)
	}

	// Test that created_at and updated_at are populated
	if post.CreatedAt == "" {
		t.Error("Expected CreatedAt to be populated")
	}

	if post.UpdatedAt == "" {
		t.Error("Expected UpdatedAt to be populated")
	}
}

func TestGetPostsForSitemap(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Insert test posts - some with slugs, some without
	testPosts := []struct {
		title, body, slug string
	}{
		{"Post with Slug 1", "Body 1", "post-with-slug-1"},
		{"Post with Slug 2", "Body 2", "post-with-slug-2"},
		{"Post without Slug", "Body 3", ""}, // This should be excluded from sitemap
	}

	for _, post := range testPosts {
		var query string
		var args []interface{}

		if post.slug != "" {
			query = `INSERT INTO posts (title, body, datepost, slug, created_at, updated_at) VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`
			args = []interface{}{post.title, post.body, "Mon Jan 2 15:04:05 2006", post.slug}
		} else {
			query = `INSERT INTO posts (title, body, datepost, created_at, updated_at) VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`
			args = []interface{}{post.title, post.body, "Mon Jan 2 15:04:05 2006"}
		}

		_, err := db.Exec(query, args...)
		if err != nil {
			t.Fatalf("Failed to insert test post: %v", err)
		}
	}

	// Query posts suitable for sitemap (only those with slugs)
	rows, err := db.Query(`
		SELECT id, title, body, datepost, slug, created_at, updated_at 
		FROM posts 
		WHERE slug IS NOT NULL AND slug != '' 
		ORDER BY id DESC
	`)
	if err != nil {
		t.Fatalf("Failed to query posts for sitemap: %v", err)
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		err := rows.Scan(&post.ID, &post.Title, &post.Body, &post.Date, &post.Slug, &post.CreatedAt, &post.UpdatedAt)
		if err != nil {
			t.Fatalf("Failed to scan post: %v", err)
		}
		posts = append(posts, post)
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

	if !slugs["post-with-slug-1"] {
		t.Error("Expected 'post-with-slug-1' to be in results")
	}

	if !slugs["post-with-slug-2"] {
		t.Error("Expected 'post-with-slug-2' to be in results")
	}
}

func TestPostTimestamps(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Create a post
	post := Post{
		Title: "Timestamp Test Post",
		Body:  "Testing timestamp functionality",
		Date:  "Mon Jan 2 15:04:05 2006",
		Slug:  "timestamp-test-post",
	}

	// Insert post
	result, err := db.Exec(`INSERT INTO posts (title, body, datepost, slug, created_at, updated_at) 
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		post.Title, post.Body, post.Date, post.Slug)
	if err != nil {
		t.Fatalf("Failed to insert post: %v", err)
	}

	// Get the ID
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get post ID: %v", err)
	}
	post.ID = int(id)

	// Retrieve the post
	retrievedPost := Post{ID: post.ID}
	err = retrievedPost.GetPost(db)
	if err != nil {
		t.Fatalf("Failed to retrieve post: %v", err)
	}

	// Check that timestamps are populated
	if retrievedPost.CreatedAt == "" {
		t.Error("Expected CreatedAt to be populated")
	}

	if retrievedPost.UpdatedAt == "" {
		t.Error("Expected UpdatedAt to be populated")
	}

	// Update the post
	retrievedPost.Title = "Updated Timestamp Test Post"
	_, err = db.Exec(`UPDATE posts SET title = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		retrievedPost.Title, retrievedPost.ID)
	if err != nil {
		t.Fatalf("Failed to update post: %v", err)
	}

	// Retrieve again
	updatedPost := Post{ID: post.ID}
	err = updatedPost.GetPost(db)
	if err != nil {
		t.Fatalf("Failed to retrieve updated post: %v", err)
	}

	// Check that updated_at changed but created_at remained the same
	if updatedPost.CreatedAt != retrievedPost.CreatedAt {
		t.Error("CreatedAt should not change on update")
	}

	// Note: In a real test, we might want to add a small delay to ensure updated_at changes
	// For now, we just check that both timestamps exist
	if updatedPost.UpdatedAt == "" {
		t.Error("Expected UpdatedAt to be populated after update")
	}
}

// TestPostModelSEOFields tests Post model with SEO-related fields
func TestPostModelSEOFields(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	if err := seedTestData(db); err != nil {
		t.Fatalf("Failed to seed test data: %v", err)
	}

	t.Run("GetPostBySlugWithSEOFields", func(t *testing.T) {
		// Insert post with SEO fields
		_, err := db.Exec(`
			INSERT INTO posts (title, body, datepost, slug, created_at, updated_at, meta_description, keywords) 
			VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, ?, ?)
		`, "SEO Test Post", "Content for SEO testing", "Mon Jan 4 12:00:00 2024", "seo-test-post", "This is a meta description", "seo, test, keywords")
		if err != nil {
			t.Fatalf("Failed to insert SEO test post: %v", err)
		}

		post := Post{Slug: "seo-test-post"}
		err = post.GetPostBySlug(db)
		if err != nil {
			t.Errorf("Failed to get SEO post by slug: %v", err)
		}

		if post.Title != "SEO Test Post" {
			t.Errorf("Expected title 'SEO Test Post', got '%s'", post.Title)
		}
	})

	t.Run("PostSlugUniqueness", func(t *testing.T) {
		// Try to insert post with duplicate slug
		_, err := db.Exec(`
			INSERT INTO posts (title, body, datepost, slug, created_at, updated_at) 
			VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, "Duplicate Slug Post", "Content with duplicate slug", "Mon Jan 5 12:00:00 2024", "test-post-1")

		// Should fail due to unique constraint
		if err == nil {
			t.Error("Expected error when inserting post with duplicate slug")
		}
	})

	t.Run("PostWithSpecialCharactersInSlug", func(t *testing.T) {
		// Insert post with special characters in slug
		_, err := db.Exec(`
			INSERT INTO posts (title, body, datepost, slug, created_at, updated_at) 
			VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, "Special Characters Post", "Content with special chars", "Mon Jan 6 12:00:00 2024", "post-with-special-chars-symbols")
		if err != nil {
			t.Fatalf("Failed to insert post with special chars in slug: %v", err)
		}

		post := Post{Slug: "post-with-special-chars-symbols"}
		err = post.GetPostBySlug(db)
		if err != nil {
			t.Errorf("Failed to get post with special chars slug: %v", err)
		}

		if post.Title != "Special Characters Post" {
			t.Errorf("Expected title 'Special Characters Post', got '%s'", post.Title)
		}
	})
}

// TestPostModelTimestamps tests Post model timestamp handling
func TestPostModelTimestamps(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	t.Run("PostCreationTimestamps", func(t *testing.T) {
		// Insert post with explicit timestamps
		createdAt := "Mon Jan 1 12:00:00 2024"
		updatedAt := "Mon Jan 2 12:00:00 2024"

		result, err := db.Exec(`
			INSERT INTO posts (title, body, datepost, slug, created_at, updated_at) 
			VALUES (?, ?, ?, ?, ?, ?)
		`, "Timestamp Test Post", "Content for timestamp testing", "Mon Jan 1 12:00:00 2024", "timestamp-test-post", createdAt, updatedAt)
		if err != nil {
			t.Fatalf("Failed to insert timestamp test post: %v", err)
		}

		postID, err := result.LastInsertId()
		if err != nil {
			t.Fatalf("Failed to get post ID: %v", err)
		}

		post := Post{ID: int(postID)}
		err = post.GetPost(db)
		if err != nil {
			t.Errorf("Failed to get post: %v", err)
		}

		if post.CreatedAt != createdAt {
			t.Errorf("Expected created_at '%s', got '%s'", createdAt, post.CreatedAt)
		}

		if post.UpdatedAt != updatedAt {
			t.Errorf("Expected updated_at '%s', got '%s'", updatedAt, post.UpdatedAt)
		}
	})

	t.Run("PostUpdateTimestamp", func(t *testing.T) {
		// Insert initial post
		result, err := db.Exec(`
			INSERT INTO posts (title, body, datepost, slug, created_at, updated_at) 
			VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, "Update Test Post", "Initial content", "Mon Jan 1 12:00:00 2024", "update-test-post")
		if err != nil {
			t.Fatalf("Failed to insert update test post: %v", err)
		}

		postID, err := result.LastInsertId()
		if err != nil {
			t.Fatalf("Failed to get post ID: %v", err)
		}

		// Update the post
		newUpdatedAt := "Mon Jan 3 12:00:00 2024"
		_, err = db.Exec(`
			UPDATE posts SET title = ?, body = ?, updated_at = ? WHERE id = ?
		`, "Updated Test Post", "Updated content", newUpdatedAt, postID)
		if err != nil {
			t.Fatalf("Failed to update post: %v", err)
		}

		post := Post{ID: int(postID)}
		err = post.GetPost(db)
		if err != nil {
			t.Errorf("Failed to get updated post: %v", err)
		}

		if post.Title != "Updated Test Post" {
			t.Errorf("Expected updated title 'Updated Test Post', got '%s'", post.Title)
		}

		if post.UpdatedAt != newUpdatedAt {
			t.Errorf("Expected updated_at '%s', got '%s'", newUpdatedAt, post.UpdatedAt)
		}
	})
}

// TestPostModelErrorHandling tests Post model error handling
func TestPostModelErrorHandling(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	t.Run("GetPostInvalidID", func(t *testing.T) {
		post := Post{ID: 99999}
		err := post.GetPost(db)
		if err == nil {
			t.Error("Expected error for invalid post ID")
		}
	})

	t.Run("GetPostZeroID", func(t *testing.T) {
		post := Post{ID: 0}
		err := post.GetPost(db)
		if err == nil {
			t.Error("Expected error for zero post ID")
		}
	})

	t.Run("GetPostBySlugInvalid", func(t *testing.T) {
		post := Post{Slug: "invalid-slug-that-does-not-exist"}
		err := post.GetPostBySlug(db)
		if err == nil {
			t.Error("Expected error for invalid slug")
		}
	})

	t.Run("CreatePostWithoutRequiredFields", func(t *testing.T) {
		// Try to insert post without required fields
		_, err := db.Exec(`INSERT INTO posts (title) VALUES (?)`, "Incomplete Post")
		if err == nil {
			t.Error("Expected error when inserting post without required fields")
		}
	})

	t.Run("UpdateNonExistentPost", func(t *testing.T) {
		result, err := db.Exec(`UPDATE posts SET title = ? WHERE id = ?`, "Updated Title", 99999)
		if err != nil {
			t.Errorf("Unexpected error updating non-existent post: %v", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			t.Errorf("Failed to get rows affected: %v", err)
		}

		if rowsAffected != 0 {
			t.Errorf("Expected 0 rows affected for non-existent post, got %d", rowsAffected)
		}
	})

	t.Run("DeleteNonExistentPost", func(t *testing.T) {
		post := Post{ID: 99999}
		err := post.DeletePost(db)
		// Should not error, but should affect 0 rows
		if err != nil {
			t.Errorf("Unexpected error deleting non-existent post: %v", err)
		}
	})
}

// TestPostModelPerformance tests Post model performance aspects
func TestPostModelPerformance(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Seed test data first
	if err := seedTestData(db); err != nil {
		t.Fatalf("Failed to seed test data: %v", err)
	}

	// Create many posts for performance testing
	numPosts := 1000
	for i := 0; i < numPosts; i++ {
		_, err := db.Exec(`
			INSERT INTO posts (title, body, datepost, slug, created_at, updated_at) 
			VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, fmt.Sprintf("Performance Test Post %d", i), fmt.Sprintf("Content for post %d", i), "Mon Jan 1 12:00:00 2024", fmt.Sprintf("performance-test-post-%d", i))
		if err != nil {
			t.Fatalf("Failed to insert performance test post %d: %v", i, err)
		}
	}

	t.Run("GetPostBySlugPerformance", func(t *testing.T) {
		start := time.Now()

		// Test getting posts by slug
		for i := 0; i < 100; i++ {
			slug := fmt.Sprintf("performance-test-post-%d", i)
			post := Post{Slug: slug}
			err := post.GetPostBySlug(db)
			if err != nil {
				t.Errorf("Failed to get post by slug %s: %v", slug, err)
			}
		}

		duration := time.Since(start)

		// Should complete within reasonable time
		if duration > 1*time.Second {
			t.Errorf("Getting 100 posts by slug took too long: %v", duration)
		}

		t.Logf("Retrieved 100 posts by slug in %v", duration)
	})

	t.Run("GetPostsPerformance", func(t *testing.T) {
		start := time.Now()

		// Test getting posts with pagination
		posts, err := GetPosts(db, 50, 0)
		if err != nil {
			t.Errorf("Failed to get posts: %v", err)
		}

		duration := time.Since(start)

		if len(posts) != 50 {
			t.Errorf("Expected 50 posts, got %d", len(posts))
		}

		// Should complete within reasonable time
		if duration > 500*time.Millisecond {
			t.Errorf("Getting 50 posts took too long: %v", duration)
		}

		t.Logf("Retrieved 50 posts in %v", duration)
	})

	t.Run("CountPostsPerformance", func(t *testing.T) {
		start := time.Now()

		count := CountPosts(db)

		duration := time.Since(start)

		expectedCount := numPosts + 3 // +3 from seedTestData
		if count != expectedCount {
			t.Errorf("Expected %d posts, got %d", expectedCount, count)
		}

		// Should complete very quickly
		if duration > 100*time.Millisecond {
			t.Errorf("Counting posts took too long: %v", duration)
		}

		t.Logf("Counted %d posts in %v", count, duration)
	})
}

// TestPostModelSEOIntegration tests SEO-related functionality in the model
func TestPostModelSEOIntegration(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	t.Run("PostsForSitemapGeneration", func(t *testing.T) {
		// Insert posts with and without slugs
		testPosts := []struct {
			title, body, slug string
			hasSlug           bool
		}{
			{"Post with Slug 1", "Content 1", "post-with-slug-1", true},
			{"Post with Slug 2", "Content 2", "post-with-slug-2", true},
			{"Post without Slug", "Content 3", "", false},
		}

		for _, post := range testPosts {
			if post.hasSlug {
				_, err := db.Exec(`
					INSERT INTO posts (title, body, datepost, slug, created_at, updated_at) 
					VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
				`, post.title, post.body, "Mon Jan 1 12:00:00 2024", post.slug)
				if err != nil {
					t.Fatalf("Failed to insert post with slug: %v", err)
				}
			} else {
				_, err := db.Exec(`
					INSERT INTO posts (title, body, datepost, created_at, updated_at) 
					VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
				`, post.title, post.body, "Mon Jan 1 12:00:00 2024")
				if err != nil {
					t.Fatalf("Failed to insert post without slug: %v", err)
				}
			}
		}

		// Query posts for sitemap (only those with slugs)
		rows, err := db.Query(`
			SELECT id, title, body, datepost, slug, created_at, updated_at 
			FROM posts 
			WHERE slug IS NOT NULL AND slug != '' 
			ORDER BY id DESC
		`)
		if err != nil {
			t.Fatalf("Failed to query posts for sitemap: %v", err)
		}
		defer rows.Close()

		var posts []Post
		for rows.Next() {
			var post Post
			err := rows.Scan(&post.ID, &post.Title, &post.Body, &post.Date, &post.Slug, &post.CreatedAt, &post.UpdatedAt)
			if err != nil {
				t.Fatalf("Failed to scan post: %v", err)
			}
			posts = append(posts, post)
		}

		// Should only return posts with slugs
		if len(posts) != 2 {
			t.Errorf("Expected 2 posts with slugs, got %d", len(posts))
		}

		for _, post := range posts {
			if post.Slug == "" {
				t.Error("Expected all posts to have slugs")
			}
		}
	})

	t.Run("PostsWithTimestampsForSEO", func(t *testing.T) {
		// Insert post with specific timestamps
		createdAt := "Mon Jan 1 12:00:00 2024"
		updatedAt := "Mon Jan 2 12:00:00 2024"

		_, err := db.Exec(`
			INSERT INTO posts (title, body, datepost, slug, created_at, updated_at) 
			VALUES (?, ?, ?, ?, ?, ?)
		`, "SEO Timestamp Post", "Content with timestamps", "Mon Jan 1 12:00:00 2024", "seo-timestamp-post", createdAt, updatedAt)
		if err != nil {
			t.Fatalf("Failed to insert SEO timestamp post: %v", err)
		}

		post := Post{Slug: "seo-timestamp-post"}
		err = post.GetPostBySlug(db)
		if err != nil {
			t.Errorf("Failed to get post by slug: %v", err)
		}

		// Verify timestamps are available for SEO processing
		if post.CreatedAt == "" {
			t.Error("Expected created_at to be available for SEO")
		}

		if post.UpdatedAt == "" {
			t.Error("Expected updated_at to be available for SEO")
		}

		if post.CreatedAt != createdAt {
			t.Errorf("Expected created_at '%s', got '%s'", createdAt, post.CreatedAt)
		}

		if post.UpdatedAt != updatedAt {
			t.Errorf("Expected updated_at '%s', got '%s'", updatedAt, post.UpdatedAt)
		}
	})

	t.Run("PostContentForSEOProcessing", func(t *testing.T) {
		// Insert post with content that needs SEO processing
		complexContent := `<p>This is a <strong>complex</strong> post with <em>HTML</em> tags.</p>
			<p>It also contains [file:document.pdf] references and <a href="https://example.com">links</a>.</p>
			<img src="/image.jpg" alt="Test image">`

		_, err := db.Exec(`
			INSERT INTO posts (title, body, datepost, slug, created_at, updated_at) 
			VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, "Complex Content Post", complexContent, "Mon Jan 1 12:00:00 2024", "complex-content-post")
		if err != nil {
			t.Fatalf("Failed to insert complex content post: %v", err)
		}

		post := Post{Slug: "complex-content-post"}
		err = post.GetPostBySlug(db)
		if err != nil {
			t.Errorf("Failed to get post by slug: %v", err)
		}

		// Verify content is available for SEO processing
		if post.Body == "" {
			t.Error("Expected post body to be available for SEO processing")
		}

		if !strings.Contains(post.Body, "<strong>") {
			t.Error("Expected HTML tags to be preserved in post body")
		}

		if !strings.Contains(post.Body, "[file:document.pdf]") {
			t.Error("Expected file references to be preserved in post body")
		}
	})
}
