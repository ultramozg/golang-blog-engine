package testutils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"
)

// Enhanced assertion helpers for comprehensive testing

// AssertEqual checks if two values are equal
func AssertEqual(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		message := formatMessage("Values are not equal", msgAndArgs...)
		t.Errorf("%s\nExpected: %v\nActual: %v", message, expected, actual)
	}
}

// AssertNotEqual checks if two values are not equal
func AssertNotEqual(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if reflect.DeepEqual(expected, actual) {
		message := formatMessage("Values should not be equal", msgAndArgs...)
		t.Errorf("%s\nBoth values: %v", message, expected)
	}
}

// AssertTrue checks if a condition is true
func AssertTrue(t *testing.T, condition bool, msgAndArgs ...interface{}) {
	t.Helper()
	if !condition {
		message := formatMessage("Condition should be true", msgAndArgs...)
		t.Error(message)
	}
}

// AssertFalse checks if a condition is false
func AssertFalse(t *testing.T, condition bool, msgAndArgs ...interface{}) {
	t.Helper()
	if condition {
		message := formatMessage("Condition should be false", msgAndArgs...)
		t.Error(message)
	}
}

// AssertNil checks if a value is nil
func AssertNil(t *testing.T, value interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if !isNil(value) {
		message := formatMessage("Value should be nil", msgAndArgs...)
		t.Errorf("%s\nActual: %v", message, value)
	}
}

// AssertNotNil checks if a value is not nil
func AssertNotNil(t *testing.T, value interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if isNil(value) {
		message := formatMessage("Value should not be nil", msgAndArgs...)
		t.Error(message)
	}
}

// AssertEmpty checks if a value is empty
func AssertEmpty(t *testing.T, value interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if !isEmpty(value) {
		message := formatMessage("Value should be empty", msgAndArgs...)
		t.Errorf("%s\nActual: %v", message, value)
	}
}

// AssertNotEmpty checks if a value is not empty
func AssertNotEmpty(t *testing.T, value interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if isEmpty(value) {
		message := formatMessage("Value should not be empty", msgAndArgs...)
		t.Error(message)
	}
}

// AssertLen checks if a collection has the expected length
func AssertLen(t *testing.T, collection interface{}, expectedLen int, msgAndArgs ...interface{}) {
	t.Helper()
	actualLen := getLength(collection)
	if actualLen != expectedLen {
		message := formatMessage("Collection length mismatch", msgAndArgs...)
		t.Errorf("%s\nExpected length: %d\nActual length: %d", message, expectedLen, actualLen)
	}
}

// AssertContainsSubstring checks if a string contains a substring
func AssertContainsSubstring(t *testing.T, str, substr string, msgAndArgs ...interface{}) {
	t.Helper()
	if !strings.Contains(str, substr) {
		message := formatMessage("String should contain substring", msgAndArgs...)
		t.Errorf("%s\nString: %s\nSubstring: %s", message, str, substr)
	}
}

// AssertNotContainsSubstring checks if a string does not contain a substring
func AssertNotContainsSubstring(t *testing.T, str, substr string, msgAndArgs ...interface{}) {
	t.Helper()
	if strings.Contains(str, substr) {
		message := formatMessage("String should not contain substring", msgAndArgs...)
		t.Errorf("%s\nString: %s\nSubstring: %s", message, str, substr)
	}
}

// AssertMatchesRegex checks if a string matches a regular expression
func AssertMatchesRegex(t *testing.T, str, pattern string, msgAndArgs ...interface{}) {
	t.Helper()
	matched, err := regexp.MatchString(pattern, str)
	if err != nil {
		t.Fatalf("Invalid regex pattern: %s", pattern)
	}
	if !matched {
		message := formatMessage("String should match regex pattern", msgAndArgs...)
		t.Errorf("%s\nString: %s\nPattern: %s", message, str, pattern)
	}
}

// AssertNotMatchesRegex checks if a string does not match a regular expression
func AssertNotMatchesRegex(t *testing.T, str, pattern string, msgAndArgs ...interface{}) {
	t.Helper()
	matched, err := regexp.MatchString(pattern, str)
	if err != nil {
		t.Fatalf("Invalid regex pattern: %s", pattern)
	}
	if matched {
		message := formatMessage("String should not match regex pattern", msgAndArgs...)
		t.Errorf("%s\nString: %s\nPattern: %s", message, str, pattern)
	}
}

// AssertPanics checks if a function panics
func AssertPanics(t *testing.T, fn func(), msgAndArgs ...interface{}) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			message := formatMessage("Function should panic", msgAndArgs...)
			t.Error(message)
		}
	}()
	fn()
}

// AssertNotPanics checks if a function does not panic
func AssertNotPanics(t *testing.T, fn func(), msgAndArgs ...interface{}) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			message := formatMessage("Function should not panic", msgAndArgs...)
			t.Errorf("%s\nPanic: %v", message, r)
		}
	}()
	fn()
}

// AssertEventually checks if a condition becomes true within a timeout
func AssertEventually(t *testing.T, condition func() bool, timeout time.Duration, interval time.Duration, msgAndArgs ...interface{}) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(interval)
	}
	message := formatMessage("Condition did not become true within timeout", msgAndArgs...)
	t.Errorf("%s\nTimeout: %v", message, timeout)
}

// AssertNever checks if a condition never becomes true within a duration
func AssertNever(t *testing.T, condition func() bool, duration time.Duration, interval time.Duration, msgAndArgs ...interface{}) {
	t.Helper()
	deadline := time.Now().Add(duration)
	for time.Now().Before(deadline) {
		if condition() {
			message := formatMessage("Condition should never become true", msgAndArgs...)
			t.Error(message)
			return
		}
		time.Sleep(interval)
	}
}

// HTTP-specific assertions

// AssertHTTPStatusCode checks HTTP response status code
func AssertHTTPStatusCode(t *testing.T, resp *http.Response, expectedStatus int, msgAndArgs ...interface{}) {
	t.Helper()
	if resp.StatusCode != expectedStatus {
		message := formatMessage("HTTP status code mismatch", msgAndArgs...)
		t.Errorf("%s\nExpected: %d\nActual: %d", message, expectedStatus, resp.StatusCode)
	}
}

// AssertHTTPHeader checks if HTTP response has expected header value
func AssertHTTPHeader(t *testing.T, resp *http.Response, headerName, expectedValue string, msgAndArgs ...interface{}) {
	t.Helper()
	actualValue := resp.Header.Get(headerName)
	if actualValue != expectedValue {
		message := formatMessage("HTTP header mismatch", msgAndArgs...)
		t.Errorf("%s\nHeader: %s\nExpected: %s\nActual: %s", message, headerName, expectedValue, actualValue)
	}
}

// AssertHTTPHeaderExists checks if HTTP response has a header
func AssertHTTPHeaderExists(t *testing.T, resp *http.Response, headerName string, msgAndArgs ...interface{}) {
	t.Helper()
	if resp.Header.Get(headerName) == "" {
		message := formatMessage("HTTP header should exist", msgAndArgs...)
		t.Errorf("%s\nHeader: %s", message, headerName)
	}
}

// AssertHTTPHeaderNotExists checks if HTTP response does not have a header
func AssertHTTPHeaderNotExists(t *testing.T, resp *http.Response, headerName string, msgAndArgs ...interface{}) {
	t.Helper()
	if resp.Header.Get(headerName) != "" {
		message := formatMessage("HTTP header should not exist", msgAndArgs...)
		t.Errorf("%s\nHeader: %s\nValue: %s", message, headerName, resp.Header.Get(headerName))
	}
}

// AssertHTTPCookie checks if HTTP response has expected cookie
func AssertHTTPCookie(t *testing.T, resp *http.Response, cookieName, expectedValue string, msgAndArgs ...interface{}) {
	t.Helper()
	for _, cookie := range resp.Cookies() {
		if cookie.Name == cookieName {
			if cookie.Value == expectedValue {
				return
			}
			message := formatMessage("HTTP cookie value mismatch", msgAndArgs...)
			t.Errorf("%s\nCookie: %s\nExpected: %s\nActual: %s", message, cookieName, expectedValue, cookie.Value)
			return
		}
	}
	message := formatMessage("HTTP cookie not found", msgAndArgs...)
	t.Errorf("%s\nCookie: %s", message, cookieName)
}

// AssertHTTPCookieExists checks if HTTP response has a cookie
func AssertHTTPCookieExists(t *testing.T, resp *http.Response, cookieName string, msgAndArgs ...interface{}) {
	t.Helper()
	for _, cookie := range resp.Cookies() {
		if cookie.Name == cookieName {
			return
		}
	}
	message := formatMessage("HTTP cookie should exist", msgAndArgs...)
	t.Errorf("%s\nCookie: %s", message, cookieName)
}

// AssertHTTPRedirect checks if HTTP response is a redirect to expected location
func AssertHTTPRedirect(t *testing.T, resp *http.Response, expectedLocation string, msgAndArgs ...interface{}) {
	t.Helper()
	if resp.StatusCode < 300 || resp.StatusCode >= 400 {
		message := formatMessage("Response should be a redirect", msgAndArgs...)
		t.Errorf("%s\nStatus: %d", message, resp.StatusCode)
		return
	}
	
	location := resp.Header.Get("Location")
	if location != expectedLocation {
		message := formatMessage("Redirect location mismatch", msgAndArgs...)
		t.Errorf("%s\nExpected: %s\nActual: %s", message, expectedLocation, location)
	}
}

// AssertHTTPBodyContains checks if HTTP response body contains expected content
func AssertHTTPBodyContains(t *testing.T, body []byte, expectedContent string, msgAndArgs ...interface{}) {
	t.Helper()
	bodyStr := string(body)
	if !strings.Contains(bodyStr, expectedContent) {
		message := formatMessage("HTTP body should contain content", msgAndArgs...)
		t.Errorf("%s\nExpected content: %s\nBody: %s", message, expectedContent, bodyStr)
	}
}

// AssertHTTPBodyNotContains checks if HTTP response body does not contain content
func AssertHTTPBodyNotContains(t *testing.T, body []byte, unexpectedContent string, msgAndArgs ...interface{}) {
	t.Helper()
	bodyStr := string(body)
	if strings.Contains(bodyStr, unexpectedContent) {
		message := formatMessage("HTTP body should not contain content", msgAndArgs...)
		t.Errorf("%s\nUnexpected content: %s\nBody: %s", message, unexpectedContent, bodyStr)
	}
}

// AssertHTTPBodyJSON checks if HTTP response body is valid JSON and optionally matches expected structure
func AssertHTTPBodyJSON(t *testing.T, body []byte, target interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	err := json.Unmarshal(body, target)
	if err != nil {
		message := formatMessage("HTTP body should be valid JSON", msgAndArgs...)
		t.Errorf("%s\nError: %v\nBody: %s", message, err, string(body))
	}
}

// Performance assertions

// AssertResponseTime checks if response time is within acceptable limits
func AssertResponseTime(t *testing.T, duration time.Duration, maxDuration time.Duration, msgAndArgs ...interface{}) {
	t.Helper()
	if duration > maxDuration {
		message := formatMessage("Response time exceeded maximum", msgAndArgs...)
		t.Errorf("%s\nActual: %v\nMaximum: %v", message, duration, maxDuration)
	}
}

// AssertMemoryUsage checks if memory usage is within acceptable limits (requires runtime.ReadMemStats)
func AssertMemoryUsage(t *testing.T, beforeBytes, afterBytes uint64, maxIncrease uint64, msgAndArgs ...interface{}) {
	t.Helper()
	increase := afterBytes - beforeBytes
	if increase > maxIncrease {
		message := formatMessage("Memory usage increase exceeded maximum", msgAndArgs...)
		t.Errorf("%s\nIncrease: %d bytes\nMaximum: %d bytes", message, increase, maxIncrease)
	}
}

// Utility functions

// formatMessage formats assertion message with optional arguments
func formatMessage(defaultMessage string, msgAndArgs ...interface{}) string {
	if len(msgAndArgs) == 0 {
		return defaultMessage
	}
	
	if len(msgAndArgs) == 1 {
		if msg, ok := msgAndArgs[0].(string); ok {
			return msg
		}
	}
	
	if format, ok := msgAndArgs[0].(string); ok {
		return fmt.Sprintf(format, msgAndArgs[1:]...)
	}
	
	return defaultMessage
}

// isNil checks if a value is nil
func isNil(value interface{}) bool {
	if value == nil {
		return true
	}
	
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	}
	
	return false
}

// isEmpty checks if a value is empty
func isEmpty(value interface{}) bool {
	if isNil(value) {
		return true
	}
	
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	
	return false
}

// getLength returns the length of a collection
func getLength(collection interface{}) int {
	if collection == nil {
		return 0
	}
	
	v := reflect.ValueOf(collection)
	switch v.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
		return v.Len()
	default:
		return 0
	}
}

// Composite assertions for common patterns

// AssertValidPost checks if a post has all required fields
func AssertValidPost(t *testing.T, post interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	// This would need to be implemented based on your Post struct
	// Example implementation:
	AssertNotNil(t, post, "Post should not be nil")
	// Add more specific validations based on your Post model
}

// AssertValidUser checks if a user has all required fields
func AssertValidUser(t *testing.T, user interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	// This would need to be implemented based on your User struct
	AssertNotNil(t, user, "User should not be nil")
	// Add more specific validations based on your User model
}

// AssertValidFile checks if a file record has all required fields
func AssertValidFile(t *testing.T, file interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	// This would need to be implemented based on your File struct
	AssertNotNil(t, file, "File should not be nil")
	// Add more specific validations based on your File model
}