package session

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ultramozg/golang-blog-engine/model"
)

func TestNewSessionDB(t *testing.T) {
	sessionDB := NewSessionDB()

	if sessionDB == nil {
		t.Errorf("NewSessionDB() returned nil")
	}

	if sessionDB.Len() != 0 {
		t.Errorf("NewSessionDB() should return empty map, got length %d", sessionDB.Len())
	}
}

func TestSessionDB_CreateSession(t *testing.T) {
	sessionDB := NewSessionDB()

	tests := []struct {
		name string
		user model.User
	}{
		{
			name: "Create admin session",
			user: model.User{Type: ADMIN, Name: "admin"},
		},
		{
			name: "Create GitHub session",
			user: model.User{Type: GITHUB, Name: "github_user"},
		},
		{
			name: "Create session with empty name",
			user: model.User{Type: ADMIN, Name: ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cookie := sessionDB.CreateSession(tt.user)

			// Verify cookie properties
			if cookie.Name != "session" {
				t.Errorf("Expected cookie name 'session', got '%s'", cookie.Name)
			}

			if cookie.Value == "" {
				t.Errorf("Expected non-empty cookie value")
			}

			// Verify session was stored
			storedUser, exists := sessionDB.GetSession(cookie.Value)
			if !exists {
				t.Errorf("Session was not stored in SessionDB")
			}

			if storedUser.Type != tt.user.Type {
				t.Errorf("Expected user type %d, got %d", tt.user.Type, storedUser.Type)
			}

			if storedUser.Name != tt.user.Name {
				t.Errorf("Expected user name '%s', got '%s'", tt.user.Name, storedUser.Name)
			}
		})
	}
}

func TestSessionDB_IsAdmin(t *testing.T) {
	sessionDB := NewSessionDB()

	// Create test sessions
	adminUser := model.User{Type: ADMIN, Name: "admin"}
	githubUser := model.User{Type: GITHUB, Name: "github_user"}

	adminCookie := sessionDB.CreateSession(adminUser)
	githubCookie := sessionDB.CreateSession(githubUser)

	tests := []struct {
		name     string
		cookie   *http.Cookie
		expected bool
	}{
		{
			name:     "Admin session returns true",
			cookie:   adminCookie,
			expected: true,
		},
		{
			name:     "GitHub session returns false",
			cookie:   githubCookie,
			expected: false,
		},
		{
			name:     "Invalid session returns false",
			cookie:   &http.Cookie{Name: "session", Value: "invalid_session_id"},
			expected: false,
		},
		{
			name:     "No session cookie returns false",
			cookie:   nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}

			result := sessionDB.IsAdmin(req)
			if result != tt.expected {
				t.Errorf("IsAdmin() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestSessionDB_IsLoggedin(t *testing.T) {
	sessionDB := NewSessionDB()

	// Create test sessions
	adminUser := model.User{Type: ADMIN, Name: "admin"}
	githubUser := model.User{Type: GITHUB, Name: "github_user"}

	adminCookie := sessionDB.CreateSession(adminUser)
	githubCookie := sessionDB.CreateSession(githubUser)

	tests := []struct {
		name     string
		cookie   *http.Cookie
		expected bool
	}{
		{
			name:     "Admin session returns true",
			cookie:   adminCookie,
			expected: true,
		},
		{
			name:     "GitHub session returns true",
			cookie:   githubCookie,
			expected: true,
		},
		{
			name:     "Invalid session returns false",
			cookie:   &http.Cookie{Name: "session", Value: "invalid_session_id"},
			expected: false,
		},
		{
			name:     "No session cookie returns false",
			cookie:   nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}

			result := sessionDB.IsLoggedin(req)
			if result != tt.expected {
				t.Errorf("IsLoggedin() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestSessionDB_DelSession(t *testing.T) {
	sessionDB := NewSessionDB()

	// Create a test session
	user := model.User{Type: ADMIN, Name: "admin"}
	cookie := sessionDB.CreateSession(user)
	sessionID := cookie.Value

	// Verify session exists
	if _, exists := sessionDB.GetSession(sessionID); !exists {
		t.Fatalf("Session should exist before deletion")
	}

	tests := []struct {
		name      string
		sessionID string
	}{
		{
			name:      "Delete existing session",
			sessionID: sessionID,
		},
		{
			name:      "Delete non-existing session",
			sessionID: "non_existing_session",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deleteCookie := sessionDB.DelSession(tt.sessionID)

			// Verify delete cookie properties
			if deleteCookie.Name != "session" {
				t.Errorf("Expected delete cookie name 'session', got '%s'", deleteCookie.Name)
			}

			if deleteCookie.Value != "" {
				t.Errorf("Expected delete cookie value to be empty, got '%s'", deleteCookie.Value)
			}

			if deleteCookie.MaxAge != -1 {
				t.Errorf("Expected delete cookie MaxAge to be -1, got %d", deleteCookie.MaxAge)
			}

			// Verify session was removed (for existing session)
			if tt.sessionID == sessionID {
				if _, exists := sessionDB.GetSession(sessionID); exists {
					t.Errorf("Session should be deleted from SessionDB")
				}
			}
		})
	}
}

func TestSessionDB_ConcurrentAccess(t *testing.T) {
	sessionDB := NewSessionDB()

	// Test concurrent session creation and access
	done := make(chan bool, 10)

	// Create multiple sessions concurrently
	for i := 0; i < 10; i++ {
		go func(id int) {
			user := model.User{Type: ADMIN, Name: "admin"}
			cookie := sessionDB.CreateSession(user)

			// Verify the session can be accessed
			req := httptest.NewRequest("GET", "/", nil)
			req.AddCookie(cookie)

			if !sessionDB.IsAdmin(req) {
				t.Errorf("Session %d should be admin", id)
			}

			if !sessionDB.IsLoggedin(req) {
				t.Errorf("Session %d should be logged in", id)
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all sessions were created
	if sessionDB.Len() != 10 {
		t.Errorf("Expected 10 sessions, got %d", sessionDB.Len())
	}
}

func TestSessionDB_SessionLifecycle(t *testing.T) {
	sessionDB := NewSessionDB()

	// Create session
	user := model.User{Type: ADMIN, Name: "admin"}
	cookie := sessionDB.CreateSession(user)
	sessionID := cookie.Value

	// Test session exists and works
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(cookie)

	if !sessionDB.IsAdmin(req) {
		t.Errorf("Session should be admin")
	}

	if !sessionDB.IsLoggedin(req) {
		t.Errorf("Session should be logged in")
	}

	// Delete session
	sessionDB.DelSession(sessionID)

	// Test session no longer works
	if sessionDB.IsAdmin(req) {
		t.Errorf("Session should not be admin after deletion")
	}

	if sessionDB.IsLoggedin(req) {
		t.Errorf("Session should not be logged in after deletion")
	}
}

func TestSessionConstants(t *testing.T) {
	if ADMIN != 1 {
		t.Errorf("Expected ADMIN constant to be 1, got %d", ADMIN)
	}

	if GITHUB != 2 {
		t.Errorf("Expected GITHUB constant to be 2, got %d", GITHUB)
	}
}
