package testutils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ultramozg/golang-blog-engine/app"
)

// HTTPTestClient provides enhanced HTTP testing capabilities
type HTTPTestClient struct {
	App        *app.App
	Server     *httptest.Server
	Client     *http.Client
	BaseURL    string
	DefaultHeaders map[string]string
}

// NewHTTPTestClient creates a new enhanced HTTP test client
func NewHTTPTestClient(t *testing.T, testApp *app.App) *HTTPTestClient {
	server := httptest.NewServer(testApp.Router)

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects by default
		},
		Timeout: 30 * time.Second,
	}

	return &HTTPTestClient{
		App:     testApp,
		Server:  server,
		Client:  client,
		BaseURL: server.URL,
		DefaultHeaders: map[string]string{
			"User-Agent": "TestClient/1.0",
		},
	}
}

// Close shuts down the test server
func (c *HTTPTestClient) Close() {
	if c.Server != nil {
		c.Server.Close()
	}
}

// RequestBuilder provides a fluent interface for building HTTP requests
type RequestBuilder struct {
	client      *HTTPTestClient
	method      string
	path        string
	body        io.Reader
	headers     map[string]string
	cookies     []*http.Cookie
	queryParams url.Values
	formData    url.Values
	jsonData    interface{}
	multipart   *MultipartBuilder
}

// NewRequest creates a new request builder
func (c *HTTPTestClient) NewRequest() *RequestBuilder {
	return &RequestBuilder{
		client:      c,
		headers:     make(map[string]string),
		cookies:     make([]*http.Cookie, 0),
		queryParams: make(url.Values),
		formData:    make(url.Values),
	}
}

// Method sets the HTTP method
func (rb *RequestBuilder) Method(method string) *RequestBuilder {
	rb.method = method
	return rb
}

// GET sets the method to GET
func (rb *RequestBuilder) GET(path string) *RequestBuilder {
	rb.method = "GET"
	rb.path = path
	return rb
}

// POST sets the method to POST
func (rb *RequestBuilder) POST(path string) *RequestBuilder {
	rb.method = "POST"
	rb.path = path
	return rb
}

// PUT sets the method to PUT
func (rb *RequestBuilder) PUT(path string) *RequestBuilder {
	rb.method = "PUT"
	rb.path = path
	return rb
}

// DELETE sets the method to DELETE
func (rb *RequestBuilder) DELETE(path string) *RequestBuilder {
	rb.method = "DELETE"
	rb.path = path
	return rb
}

// Path sets the request path
func (rb *RequestBuilder) Path(path string) *RequestBuilder {
	rb.path = path
	return rb
}

// Header adds a header
func (rb *RequestBuilder) Header(key, value string) *RequestBuilder {
	rb.headers[key] = value
	return rb
}

// Headers adds multiple headers
func (rb *RequestBuilder) Headers(headers map[string]string) *RequestBuilder {
	for k, v := range headers {
		rb.headers[k] = v
	}
	return rb
}

// Cookie adds a cookie
func (rb *RequestBuilder) Cookie(cookie *http.Cookie) *RequestBuilder {
	rb.cookies = append(rb.cookies, cookie)
	return rb
}

// Cookies adds multiple cookies
func (rb *RequestBuilder) Cookies(cookies []*http.Cookie) *RequestBuilder {
	rb.cookies = append(rb.cookies, cookies...)
	return rb
}

// Query adds a query parameter
func (rb *RequestBuilder) Query(key, value string) *RequestBuilder {
	rb.queryParams.Add(key, value)
	return rb
}

// QueryParams adds multiple query parameters
func (rb *RequestBuilder) QueryParams(params map[string]string) *RequestBuilder {
	for k, v := range params {
		rb.queryParams.Add(k, v)
	}
	return rb
}

// Form adds form data
func (rb *RequestBuilder) Form(key, value string) *RequestBuilder {
	rb.formData.Add(key, value)
	return rb
}

// FormData sets form data from a map
func (rb *RequestBuilder) FormData(data map[string]string) *RequestBuilder {
	rb.formData = make(url.Values)
	for k, v := range data {
		rb.formData.Add(k, v)
	}
	return rb
}

// JSON sets JSON body data
func (rb *RequestBuilder) JSON(data interface{}) *RequestBuilder {
	rb.jsonData = data
	rb.headers["Content-Type"] = "application/json"
	return rb
}

// Body sets raw body data
func (rb *RequestBuilder) Body(body io.Reader) *RequestBuilder {
	rb.body = body
	return rb
}

// BodyString sets string body data
func (rb *RequestBuilder) BodyString(body string) *RequestBuilder {
	rb.body = strings.NewReader(body)
	return rb
}

// Multipart creates a multipart form builder
func (rb *RequestBuilder) Multipart() *MultipartBuilder {
	rb.multipart = &MultipartBuilder{
		requestBuilder: rb,
		fields:         make(map[string]string),
		files:          make(map[string]MultipartFile),
	}
	return rb.multipart
}

// Execute builds and executes the HTTP request
func (rb *RequestBuilder) Execute() (*http.Response, error) {
	// Build URL with query parameters
	fullURL := rb.client.BaseURL + rb.path
	if len(rb.queryParams) > 0 {
		fullURL += "?" + rb.queryParams.Encode()
	}

	// Prepare body
	var body io.Reader
	if rb.multipart != nil {
		var err error
		body, rb.headers["Content-Type"], err = rb.multipart.build()
		if err != nil {
			return nil, fmt.Errorf("failed to build multipart body: %v", err)
		}
	} else if rb.jsonData != nil {
		jsonBytes, err := json.Marshal(rb.jsonData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal JSON: %v", err)
		}
		body = bytes.NewReader(jsonBytes)
	} else if len(rb.formData) > 0 {
		body = strings.NewReader(rb.formData.Encode())
		rb.headers["Content-Type"] = "application/x-www-form-urlencoded"
	} else if rb.body != nil {
		body = rb.body
	}

	// Create request
	req, err := http.NewRequest(rb.method, fullURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set default headers
	for k, v := range rb.client.DefaultHeaders {
		req.Header.Set(k, v)
	}

	// Set custom headers
	for k, v := range rb.headers {
		req.Header.Set(k, v)
	}

	// Set cookies
	for _, cookie := range rb.cookies {
		req.AddCookie(cookie)
	}

	// Execute request
	return rb.client.Client.Do(req)
}

// MultipartBuilder helps build multipart form data
type MultipartBuilder struct {
	requestBuilder *RequestBuilder
	fields         map[string]string
	files          map[string]MultipartFile
}

// MultipartFile represents a file in multipart form
type MultipartFile struct {
	FieldName string
	FileName  string
	Content   []byte
	MimeType  string
}

// Field adds a form field
func (mb *MultipartBuilder) Field(name, value string) *MultipartBuilder {
	mb.fields[name] = value
	return mb
}

// File adds a file field
func (mb *MultipartBuilder) File(fieldName, fileName string, content []byte, mimeType string) *MultipartBuilder {
	mb.files[fieldName] = MultipartFile{
		FieldName: fieldName,
		FileName:  fileName,
		Content:   content,
		MimeType:  mimeType,
	}
	return mb
}

// Execute builds and executes the multipart request
func (mb *MultipartBuilder) Execute() (*http.Response, error) {
	return mb.requestBuilder.Execute()
}

// build creates the multipart body and content type
func (mb *MultipartBuilder) build() (io.Reader, string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add form fields
	for name, value := range mb.fields {
		err := writer.WriteField(name, value)
		if err != nil {
			return nil, "", fmt.Errorf("failed to write field %s: %v", name, err)
		}
	}

	// Add files
	for _, file := range mb.files {
		part, err := writer.CreateFormFile(file.FieldName, file.FileName)
		if err != nil {
			return nil, "", fmt.Errorf("failed to create form file %s: %v", file.FieldName, err)
		}

		// Set content type if specified
		if file.MimeType != "" {
			h := make(textproto.MIMEHeader)
			h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, file.FieldName, file.FileName))
			h.Set("Content-Type", file.MimeType)
			part, err = writer.CreatePart(h)
			if err != nil {
				return nil, "", fmt.Errorf("failed to create part with content type: %v", err)
			}
		}

		_, err = part.Write(file.Content)
		if err != nil {
			return nil, "", fmt.Errorf("failed to write file content: %v", err)
		}
	}

	err := writer.Close()
	if err != nil {
		return nil, "", fmt.Errorf("failed to close multipart writer: %v", err)
	}

	return &buf, writer.FormDataContentType(), nil
}

// ResponseHelper provides utilities for working with HTTP responses
type ResponseHelper struct {
	Response *http.Response
	Body     []byte
}

// NewResponseHelper creates a response helper
func NewResponseHelper(resp *http.Response) (*ResponseHelper, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}
	resp.Body.Close()

	return &ResponseHelper{
		Response: resp,
		Body:     body,
	}, nil
}

// StatusCode returns the response status code
func (rh *ResponseHelper) StatusCode() int {
	return rh.Response.StatusCode
}

// BodyString returns the response body as a string
func (rh *ResponseHelper) BodyString() string {
	return string(rh.Body)
}

// BodyBytes returns the response body as bytes
func (rh *ResponseHelper) BodyBytes() []byte {
	return rh.Body
}

// Header returns a response header value
func (rh *ResponseHelper) Header(name string) string {
	return rh.Response.Header.Get(name)
}

// Headers returns all response headers
func (rh *ResponseHelper) Headers() http.Header {
	return rh.Response.Header
}

// Cookie returns a response cookie by name
func (rh *ResponseHelper) Cookie(name string) *http.Cookie {
	for _, cookie := range rh.Response.Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

// Cookies returns all response cookies
func (rh *ResponseHelper) Cookies() []*http.Cookie {
	return rh.Response.Cookies()
}

// JSON unmarshals the response body as JSON
func (rh *ResponseHelper) JSON(v interface{}) error {
	return json.Unmarshal(rh.Body, v)
}

// ContainsString checks if the response body contains a string
func (rh *ResponseHelper) ContainsString(s string) bool {
	return strings.Contains(rh.BodyString(), s)
}

// MatchesPattern checks if the response body matches a pattern
func (rh *ResponseHelper) MatchesPattern(pattern string) bool {
	// Simple pattern matching - could be enhanced with regex
	return strings.Contains(rh.BodyString(), pattern)
}

// SessionHelper provides utilities for session management in tests
type SessionHelper struct {
	client *HTTPTestClient
}

// NewSessionHelper creates a session helper
func (c *HTTPTestClient) NewSessionHelper() *SessionHelper {
	return &SessionHelper{client: c}
}

// LoginAsAdmin performs admin login and returns session cookie
func (sh *SessionHelper) LoginAsAdmin(username, password string) (*http.Cookie, error) {
	resp, err := sh.client.NewRequest().
		POST("/login").
		Form("login", username).
		Form("password", password).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("login request failed: %v", err)
	}
	defer resp.Body.Close()

	// Check for redirect status codes (302, 303, etc.)
	if resp.StatusCode < 300 || resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("login failed with status: %d, body: %s", resp.StatusCode, string(body))
	}

	// Extract session cookie
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "session" {
			return cookie, nil
		}
	}

	return nil, fmt.Errorf("session cookie not found")
}

// LoginAsUser performs user login with custom credentials
func (sh *SessionHelper) LoginAsUser(username, password string) (*http.Cookie, error) {
	return sh.LoginAsAdmin(username, password) // Same process for now
}

// Logout performs logout
func (sh *SessionHelper) Logout(sessionCookie *http.Cookie) error {
	resp, err := sh.client.NewRequest().
		GET("/logout").
		Cookie(sessionCookie).
		Execute()
	if err != nil {
		return fmt.Errorf("logout request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("logout failed with status: %d", resp.StatusCode)
	}

	return nil
}

// Enhanced assertion helpers for HTTP responses
func AssertJSONResponse(t *testing.T, resp *http.Response, expectedStatus int, target interface{}) {
	helper, err := NewResponseHelper(resp)
	if err != nil {
		t.Fatalf("Failed to create response helper: %v", err)
	}

	if helper.StatusCode() != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, helper.StatusCode())
	}

	contentType := helper.Header("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected JSON content type, got %s", contentType)
	}

	if target != nil {
		err = helper.JSON(target)
		if err != nil {
			t.Errorf("Failed to unmarshal JSON response: %v", err)
		}
	}
}

// AssertRedirectResponse checks for redirect responses
func AssertRedirectResponse(t *testing.T, resp *http.Response, expectedLocation string) {
	if resp.StatusCode < 300 || resp.StatusCode >= 400 {
		t.Errorf("Expected redirect status code (3xx), got %d", resp.StatusCode)
		return
	}

	location := resp.Header.Get("Location")
	if location != expectedLocation {
		t.Errorf("Expected redirect to '%s', got '%s'", expectedLocation, location)
	}
}

// AssertHTTPResponseTime checks if response time is within acceptable limits
func AssertHTTPResponseTime(t *testing.T, duration time.Duration, maxDuration time.Duration) {
	if duration > maxDuration {
		t.Errorf("Response time %v exceeded maximum %v", duration, maxDuration)
	}
}

// MeasureResponseTime measures the time taken for a request
func (c *HTTPTestClient) MeasureResponseTime(requestFunc func() (*http.Response, error)) (*http.Response, time.Duration, error) {
	start := time.Now()
	resp, err := requestFunc()
	duration := time.Since(start)
	return resp, duration, err
}

// ConcurrentRequests executes multiple requests concurrently
func (c *HTTPTestClient) ConcurrentRequests(count int, requestFunc func(int) (*http.Response, error)) ([]*http.Response, []error) {
	responses := make([]*http.Response, count)
	errors := make([]error, count)
	done := make(chan struct{})

	for i := 0; i < count; i++ {
		go func(index int) {
			defer func() { done <- struct{}{} }()
			responses[index], errors[index] = requestFunc(index)
		}(i)
	}

	// Wait for all requests to complete
	for i := 0; i < count; i++ {
		<-done
	}

	return responses, errors
}