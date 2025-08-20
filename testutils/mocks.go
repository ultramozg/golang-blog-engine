package testutils

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/ultramozg/golang-blog-engine/model"
	"github.com/ultramozg/golang-blog-engine/session"
)

// MockSessionDB provides a mock implementation of session.SessionDB for testing
type MockSessionDB struct {
	sessions map[string]model.User
	adminSessions map[string]bool
	userSessions map[string]bool
}

// NewMockSessionDB creates a new mock session database
func NewMockSessionDB() *MockSessionDB {
	return &MockSessionDB{
		sessions: make(map[string]model.User),
		adminSessions: make(map[string]bool),
		userSessions: make(map[string]bool),
	}
}

// CreateSession creates a mock session and returns a cookie
func (m *MockSessionDB) CreateSession(user model.User) *http.Cookie {
	sessionID := fmt.Sprintf("test_session_%d", time.Now().UnixNano())
	m.sessions[sessionID] = user
	
	if user.Type == session.ADMIN {
		m.adminSessions[sessionID] = true
	} else {
		m.userSessions[sessionID] = true
	}

	return &http.Cookie{
		Name:  "session",
		Value: sessionID,
		Path:  "/",
	}
}

// IsAdmin checks if the request has admin session
func (m *MockSessionDB) IsAdmin(r *http.Request) bool {
	cookie, err := r.Cookie("session")
	if err != nil {
		return false
	}
	return m.adminSessions[cookie.Value]
}

// IsLoggedin checks if the request has any valid session
func (m *MockSessionDB) IsLoggedin(r *http.Request) bool {
	cookie, err := r.Cookie("session")
	if err != nil {
		return false
	}
	_, exists := m.sessions[cookie.Value]
	return exists
}

// DelSession removes a session
func (m *MockSessionDB) DelSession(sessionID string) {
	delete(m.sessions, sessionID)
	delete(m.adminSessions, sessionID)
	delete(m.userSessions, sessionID)
}

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct {
	counter int
}

// NewTestDataGenerator creates a new test data generator
func NewTestDataGenerator() *TestDataGenerator {
	return &TestDataGenerator{counter: 0}
}

// GeneratePost creates a test post with unique data
func (g *TestDataGenerator) GeneratePost() model.Post {
	g.counter++
	return model.Post{
		ID:    g.counter,
		Title: fmt.Sprintf("Test Post %d", g.counter),
		Body:  fmt.Sprintf("This is the body content for test post %d. It contains some sample text for testing purposes.", g.counter),
		Date:  time.Now().Format("Mon Jan _2 15:04:05 2006"),
	}
}

// GenerateComment creates a test comment with unique data
func (g *TestDataGenerator) GenerateComment(postID int) model.Comment {
	g.counter++
	return model.Comment{
		PostID:    postID,
		CommentID: g.counter,
		Name:      fmt.Sprintf("Test User %d", g.counter),
		Date:      time.Now().Format("Mon Jan _2 15:04:05 2006"),
		Data:      fmt.Sprintf("This is test comment %d content.", g.counter),
	}
}

// GenerateUser creates a test user with unique data
func (g *TestDataGenerator) GenerateUser(userType int) model.User {
	g.counter++
	return model.User{
		Name: fmt.Sprintf("testuser%d", g.counter),
		Type: userType,
	}
}

// MockHTTPHandler provides utilities for creating mock HTTP handlers
type MockHTTPHandler struct {
	responses map[string]*MockResponse
}

// MockResponse represents a mock HTTP response
type MockResponse struct {
	StatusCode int
	Body       string
	Headers    map[string]string
	Cookies    []*http.Cookie
}

// NewMockHTTPHandler creates a new mock HTTP handler
func NewMockHTTPHandler() *MockHTTPHandler {
	return &MockHTTPHandler{
		responses: make(map[string]*MockResponse),
	}
}

// SetResponse sets a mock response for a specific path
func (m *MockHTTPHandler) SetResponse(path string, response *MockResponse) {
	m.responses[path] = response
}

// ServeHTTP implements http.Handler interface
func (m *MockHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	response, exists := m.responses[r.URL.Path]
	if !exists {
		http.NotFound(w, r)
		return
	}

	// Set headers
	for key, value := range response.Headers {
		w.Header().Set(key, value)
	}

	// Set cookies
	for _, cookie := range response.Cookies {
		http.SetCookie(w, cookie)
	}

	// Set status code
	w.WriteHeader(response.StatusCode)

	// Write body
	w.Write([]byte(response.Body))
}

// DatabaseMock provides utilities for mocking database operations
type DatabaseMock struct {
	posts    []model.Post
	comments []model.Comment
	users    []model.User
	queries  []string
}

// NewDatabaseMock creates a new database mock
func NewDatabaseMock() *DatabaseMock {
	return &DatabaseMock{
		posts:    make([]model.Post, 0),
		comments: make([]model.Comment, 0),
		users:    make([]model.User, 0),
		queries:  make([]string, 0),
	}
}

// AddPost adds a post to the mock database
func (dm *DatabaseMock) AddPost(post model.Post) {
	dm.posts = append(dm.posts, post)
}

// AddComment adds a comment to the mock database
func (dm *DatabaseMock) AddComment(comment model.Comment) {
	dm.comments = append(dm.comments, comment)
}

// AddUser adds a user to the mock database
func (dm *DatabaseMock) AddUser(user model.User) {
	dm.users = append(dm.users, user)
}

// GetPosts returns all posts from the mock database
func (dm *DatabaseMock) GetPosts() []model.Post {
	return dm.posts
}

// GetComments returns all comments from the mock database
func (dm *DatabaseMock) GetComments() []model.Comment {
	return dm.comments
}

// GetUsers returns all users from the mock database
func (dm *DatabaseMock) GetUsers() []model.User {
	return dm.users
}

// GetQueries returns all executed queries
func (dm *DatabaseMock) GetQueries() []string {
	return dm.queries
}

// RecordQuery records a query execution
func (dm *DatabaseMock) RecordQuery(query string) {
	dm.queries = append(dm.queries, query)
}

// Clear clears all data from the mock database
func (dm *DatabaseMock) Clear() {
	dm.posts = make([]model.Post, 0)
	dm.comments = make([]model.Comment, 0)
	dm.users = make([]model.User, 0)
	dm.queries = make([]string, 0)
}

// HTTPRecorder extends httptest.ResponseRecorder with additional utilities
type HTTPRecorder struct {
	*httptest.ResponseRecorder
}

// NewHTTPRecorder creates a new HTTP recorder
func NewHTTPRecorder() *HTTPRecorder {
	return &HTTPRecorder{
		ResponseRecorder: httptest.NewRecorder(),
	}
}

// GetBodyString returns the response body as a string
func (hr *HTTPRecorder) GetBodyString() string {
	return hr.Body.String()
}

// GetStatusCode returns the response status code
func (hr *HTTPRecorder) GetStatusCode() int {
	return hr.Code
}

// GetHeader returns a specific header value
func (hr *HTTPRecorder) GetHeader(name string) string {
	return hr.Header().Get(name)
}

// GetCookie returns a specific cookie
func (hr *HTTPRecorder) GetCookie(name string) *http.Cookie {
	for _, cookie := range hr.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

// HasCookie checks if a cookie exists
func (hr *HTTPRecorder) HasCookie(name string) bool {
	return hr.GetCookie(name) != nil
}

// TestRequestBuilder helps build HTTP requests for testing
type TestRequestBuilder struct {
	method  string
	path    string
	body    string
	headers map[string]string
	cookies []*http.Cookie
}

// NewTestRequestBuilder creates a new test request builder
func NewTestRequestBuilder() *TestRequestBuilder {
	return &TestRequestBuilder{
		headers: make(map[string]string),
		cookies: make([]*http.Cookie, 0),
	}
}

// Method sets the HTTP method
func (trb *TestRequestBuilder) Method(method string) *TestRequestBuilder {
	trb.method = method
	return trb
}

// Path sets the request path
func (trb *TestRequestBuilder) Path(path string) *TestRequestBuilder {
	trb.path = path
	return trb
}

// Body sets the request body
func (trb *TestRequestBuilder) Body(body string) *TestRequestBuilder {
	trb.body = body
	return trb
}

// Header adds a header
func (trb *TestRequestBuilder) Header(key, value string) *TestRequestBuilder {
	trb.headers[key] = value
	return trb
}

// Cookie adds a cookie
func (trb *TestRequestBuilder) Cookie(cookie *http.Cookie) *TestRequestBuilder {
	trb.cookies = append(trb.cookies, cookie)
	return trb
}

// FormData sets form data as the body and content type
func (trb *TestRequestBuilder) FormData(data string) *TestRequestBuilder {
	trb.body = data
	trb.headers["Content-Type"] = "application/x-www-form-urlencoded"
	return trb
}

// Build creates the HTTP request
func (trb *TestRequestBuilder) Build() (*http.Request, error) {
	var bodyReader *strings.Reader
	if trb.body != "" {
		bodyReader = strings.NewReader(trb.body)
	}

	req, err := http.NewRequest(trb.method, trb.path, bodyReader)
	if err != nil {
		return nil, err
	}

	// Set headers
	for key, value := range trb.headers {
		req.Header.Set(key, value)
	}

	// Set cookies
	for _, cookie := range trb.cookies {
		req.AddCookie(cookie)
	}

	return req, nil
}