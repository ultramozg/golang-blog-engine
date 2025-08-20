package model

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/ultramozg/golang-blog-engine/services"
	"golang.org/x/crypto/bcrypt"
)

// Test helper functions to avoid circular dependency with testutils

// createTestDB creates a test database with migrations
func createTestDB(t *testing.T) (*sql.DB, func()) {
	tempDir, err := os.MkdirTemp("", "model_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tempDir, "test.db")
	db, err := sql.Open("sqlite3", dbPath)
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
	// Create slug service for generating slugs
	slugService := services.NewSlugService(db)

	// Insert test posts with slugs
	testPosts := []Post{
		{Title: "Test Post 1", Body: "This is the body of test post 1", Date: "Mon Jan 1 12:00:00 2024"},
		{Title: "Test Post 2", Body: "This is the body of test post 2", Date: "Mon Jan 2 12:00:00 2024"},
		{Title: "Test Post 3", Body: "This is the body of test post 3", Date: "Mon Jan 3 12:00:00 2024"},
	}

	for _, post := range testPosts {
		// Generate slug for the post
		slug := slugService.GenerateSlug(post.Title)
		post.Slug = slugService.EnsureUniqueSlug(slug, 0) // 0 for new post

		// Insert post with slug
		result, err := db.Exec(`insert into posts (title, body, datepost, slug, created_at, updated_at) values ($1, $2, $3, $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, post.Title, post.Body, post.Date, post.Slug)
		if err != nil {
			return err
		}

		// Get the ID of the newly created post
		id, err := result.LastInsertId()
		if err != nil {
			return err
		}
		post.ID = int(id)
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

// createTempFile creates a temporary file with given content
func createTempFile(t *testing.T, content string) string {
	tmpFile, err := os.CreateTemp("", "test_*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	return tmpFile.Name()
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
	db, err := sql.Open("sqlite3", ":memory:")
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

func TestConverYamlToStruct(t *testing.T) {
	// Create temporary YAML file
	yamlContent := `infos:
  - title: "Test Title 1"
    link: "https://example.com/1"
    description: "Test description 1"
  - title: "Test Title 2"
    link: "https://example.com/2"
    description: "Test description 2"`

	tmpFile := createTempFile(t, yamlContent)
	defer func() {
		if err := os.Remove(tmpFile); err != nil {
			t.Logf("Warning: failed to remove temp file: %v", err)
		}
	}()

	tests := []struct {
		name        string
		filePath    string
		expectError bool
		expectedLen int
	}{
		{
			name:        "Valid YAML file",
			filePath:    tmpFile,
			expectError: false,
			expectedLen: 2,
		},
		{
			name:        "Non-existing file",
			filePath:    "nonexistent.yml",
			expectError: true,
			expectedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConverYamlToStruct(tt.filePath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(result.List) != tt.expectedLen {
					t.Errorf("Expected %d items, got %d", tt.expectedLen, len(result.List))
				}

				// Verify content for valid file
				if tt.expectedLen > 0 {
					if result.List[0].Title != "Test Title 1" {
						t.Errorf("Expected first title 'Test Title 1', got '%s'", result.List[0].Title)
					}
					if result.List[0].Link != "https://example.com/1" {
						t.Errorf("Expected first link 'https://example.com/1', got '%s'", result.List[0].Link)
					}
					if result.List[0].Description != "Test description 1" {
						t.Errorf("Expected first description 'Test description 1', got '%s'", result.List[0].Description)
					}
				}
			}
		})
	}
}
