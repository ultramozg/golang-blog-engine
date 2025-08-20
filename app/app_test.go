package app

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ultramozg/golang-blog-engine/model"
	_ "github.com/mattn/go-sqlite3"
)

func TestMain(m *testing.M) {
	os.Setenv("DBURI", "file:../database/database.sqlite")
	os.Setenv("TEMPLATES", "../templates/*.gohtml")

	os.Exit(m.Run())
}

// setupTestData ensures the database has test posts for tests that need them
func setupTestData(t *testing.T) {
	// Connect to the database
	db, err := sql.Open("sqlite3", "../database/database.sqlite")
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Check if we have any posts, if not create some test posts
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM posts").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count posts: %v", err)
	}

	// If we have fewer than 3 posts, create some test posts
	if count < 3 {
		testPosts := []model.Post{
			{Title: "Test Post 1", Body: "test body", Date: time.Now().Format("Mon Jan _2 15:04:05 2006")},
			{Title: "Test Post 2", Body: "test body", Date: time.Now().Format("Mon Jan _2 15:04:05 2006")},
			{Title: "Test Post 3", Body: "test body", Date: time.Now().Format("Mon Jan _2 15:04:05 2006")},
		}

		for _, post := range testPosts {
			err := post.CreatePost(db)
			if err != nil {
				t.Fatalf("Failed to create test post: %v", err)
			}
		}
	}
}

// getFirstPostID returns the ID of the first available post in the database
func getFirstPostID(t *testing.T) int {
	db, err := sql.Open("sqlite3", "../database/database.sqlite")
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	var id int
	err = db.QueryRow("SELECT id FROM posts ORDER BY id LIMIT 1").Scan(&id)
	if err != nil {
		t.Fatalf("Failed to get first post ID: %v", err)
	}
	return id
}

// createTestAdmin ensures there's an admin user for login tests
func createTestAdmin(t *testing.T) {
	db, err := sql.Open("sqlite3", "../database/database.sqlite")
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Check if admin user exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE name = 'admin'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check for admin user: %v", err)
	}

	// If no admin user exists, create one
	if count == 0 {
		user := model.User{Name: "admin", Type: model.ADMIN}
		success, hashedPassword := HashPassword("12345")
		if !success {
			t.Fatalf("Failed to hash password")
		}
		err = user.CreateUser(db, hashedPassword)
		if err != nil {
			t.Fatalf("Failed to create admin user: %v", err)
		}
	}
}

func TestRoot(t *testing.T) {
	a := NewApp()
	a.Initialize()

	req, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(a.root)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusFound {
		t.Errorf("Root handler returned wrong status code: got %v want %v", status, http.StatusFound)
	}
	expectedURI := "/page?p=0"
	if rr.Header().Get("Location") != expectedURI {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expectedURI)
	}
}

func TestGetPage(t *testing.T) {
	a := NewApp()
	a.Initialize()

	req, err := http.NewRequest(http.MethodGet, "/page?p=0", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(a.getPage)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("GetPage handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	expected := `<p>Powered by Golang net/http package</p>`
	if !strings.Contains(rr.Body.String(), expected) {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}

func TestFailedLogin(t *testing.T) {
	a := NewApp()
	a.Initialize()

	payload := url.Values{}
	payload.Set("login", "admin")
	payload.Set("password", "blabla")

	req, err := http.NewRequest(http.MethodPost, "/login", strings.NewReader(payload.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(a.login)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("login handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
	}
}

func TestSuccesfullLogin(t *testing.T) {
	// Ensure admin user exists
	createTestAdmin(t)
	
	a := NewApp()
	a.Initialize()

	payload := url.Values{}
	payload.Set("login", "admin")
	payload.Set("password", "12345")

	req, err := http.NewRequest(http.MethodPost, "/login", strings.NewReader(payload.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(a.login)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusSeeOther {
		t.Errorf("login handler returned wrong status code: got %v want %v", status, http.StatusSeeOther)
	}

	cookies := rr.Result().Cookies()
	if len(cookies) == 0 {
		t.Errorf("login handler returned empty cookie: got %v", cookies)
	} else {
		if c := cookies[0]; c.Name != "session" {
			t.Errorf("login handler 'session' cookies hasn't been set got %v want %v", c.Name, "session")
		}
	}
}

func TestCreatePost(t *testing.T) {
	// Ensure admin user exists
	createTestAdmin(t)
	
	a := NewApp()
	a.Initialize()

	payload := url.Values{}
	payload.Set("login", "admin")
	payload.Set("password", "12345")

	req, err := http.NewRequest(http.MethodPost, "/login", strings.NewReader(payload.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handlerLogin := http.HandlerFunc(a.login)
	handlerLogin.ServeHTTP(rr, req)

	req.AddCookie(rr.Result().Cookies()[0])

	// create test post with cookie set
	payload = url.Values{}
	payload.Set("title", "New Post")
	payload.Set("body", "test body")

	req, err = http.NewRequest(http.MethodPost, "/create", strings.NewReader(payload.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	handlerCreatePost := http.HandlerFunc(a.createPost)
	handlerCreatePost.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusSeeOther {
		t.Errorf("GetPage handler returned wrong status code: got %v want %v", status, http.StatusSeeOther)
	}
}

func TestGetPost(t *testing.T) {
	// Ensure we have test data
	setupTestData(t)
	
	a := NewApp()
	a.Initialize()

	// Get the first available post ID
	postID := getFirstPostID(t)

	req, err := http.NewRequest(http.MethodGet, "/post?id="+strconv.Itoa(postID), nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(a.getPost)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Root handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	expectedBody := "test body"
	if !strings.Contains(rr.Body.String(), expectedBody) {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expectedBody)
	}
}

func TestDeletePost(t *testing.T) {
	// Ensure we have test data and admin user
	setupTestData(t)
	createTestAdmin(t)
	
	a := NewApp()
	a.Initialize()

	payload := url.Values{}
	payload.Set("login", "admin")
	payload.Set("password", "12345")

	req, err := http.NewRequest(http.MethodPost, "/login", strings.NewReader(payload.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handlerLogin := http.HandlerFunc(a.login)
	handlerLogin.ServeHTTP(rr, req)

	// Get a post ID to delete (use the first available post)
	postID := getFirstPostID(t)

	//delete post
	req, err = http.NewRequest(http.MethodGet, "/delete?id="+strconv.Itoa(postID), strings.NewReader(payload.Encode()))
	req.AddCookie(rr.Result().Cookies()[0])
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	handlerDeletePost := http.HandlerFunc(a.deletePost)
	handlerDeletePost.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusSeeOther {
		t.Errorf("GetPage handler returned wrong status code: got %v want %v", status, http.StatusSeeOther)
	}
}
