# Testing Infrastructure

This package provides comprehensive testing utilities for the Go blog platform, including database setup/teardown, HTTP testing helpers, and test configuration management.

## Overview

The testing infrastructure consists of several key components:

- **TestRunner**: Main orchestrator that combines database and HTTP testing utilities
- **TestDatabase**: Database setup, seeding, and cleanup utilities
- **HTTPTestHelper**: HTTP request/response testing utilities
- **TestEnvironment**: Environment variable and configuration management
- **Mocks**: Mock implementations for testing in isolation
- **Assertions**: Helper functions for common test assertions

## Quick Start

### Basic Test Setup

```go
func TestMyFeature(t *testing.T) {
    // Create test runner with database and HTTP helpers
    runner := testutils.NewTestRunner(t)
    defer runner.Close()

    // Setup test data
    if err := runner.SetupTest(); err != nil {
        t.Fatalf("Failed to setup test: %v", err)
    }

    // Your test code here
    resp, err := runner.HTTP.MakeRequest("GET", "/page?p=0", "", nil)
    if err != nil {
        t.Fatalf("Failed to make request: %v", err)
    }
    defer resp.Body.Close()

    testutils.AssertStatusCode(t, resp, http.StatusOK)
}
```

### Database Testing

```go
func TestDatabaseOperations(t *testing.T) {
    // Create test database
    db := testutils.NewTestDatabase(t)
    defer db.Close()

    // Seed test data
    if err := db.SeedTestData(); err != nil {
        t.Fatalf("Failed to seed data: %v", err)
    }

    // Test database operations
    var count int
    err := db.DB.QueryRow("SELECT COUNT(*) FROM posts").Scan(&count)
    if err != nil {
        t.Fatalf("Failed to query posts: %v", err)
    }

    if count != 3 { // SeedTestData creates 3 posts
        t.Errorf("Expected 3 posts, got %d", count)
    }
}
```

### HTTP Testing

```go
func TestHTTPEndpoints(t *testing.T) {
    runner := testutils.NewTestRunner(t)
    defer runner.Close()

    // Test GET request
    resp, err := runner.HTTP.MakeRequest("GET", "/page?p=0", "", nil)
    if err != nil {
        t.Fatalf("Failed to make request: %v", err)
    }
    defer resp.Body.Close()

    testutils.AssertStatusCode(t, resp, http.StatusOK)

    // Test POST request with form data
    formData := "title=Test+Post&body=Test+Body"
    headers := map[string]string{
        "Content-Type": "application/x-www-form-urlencoded",
    }

    // Login first to get session cookie
    sessionCookie, err := runner.HTTP.LoginAsAdmin()
    if err != nil {
        t.Fatalf("Failed to login: %v", err)
    }

    resp, err = runner.HTTP.MakeRequestWithCookies("POST", "/create", formData, headers, []*http.Cookie{sessionCookie})
    if err != nil {
        t.Fatalf("Failed to create post: %v", err)
    }
    defer resp.Body.Close()

    testutils.AssertRedirect(t, resp, "/")
}
```

## Components

### TestRunner

The `TestRunner` is the main entry point that combines database and HTTP testing utilities:

```go
type TestRunner struct {
    DB   *TestDatabase
    HTTP *HTTPTestHelper
}
```

**Methods:**
- `NewTestRunner(t *testing.T) *TestRunner`: Creates a new test runner
- `Close()`: Cleans up all resources
- `SetupTest() error`: Performs common test setup (creates admin user, seeds data)

### TestDatabase

Provides database testing utilities:

```go
type TestDatabase struct {
    DB     *sql.DB
    Config *TestConfig
}
```

**Methods:**
- `NewTestDatabase(t *testing.T) *TestDatabase`: Creates test database
- `Close() error`: Cleans up database and temp files
- `SeedTestData() error`: Inserts test posts and comments
- `ClearTestData() error`: Removes all test data
- `CreateTestUser(name, password string, userType int) error`: Creates test user

### HTTPTestHelper

Provides HTTP testing utilities:

```go
type HTTPTestHelper struct {
    App    *app.App
    Server *httptest.Server
    Client *http.Client
}
```

**Methods:**
- `NewHTTPTestHelper(t *testing.T, testDB *TestDatabase) *HTTPTestHelper`: Creates HTTP helper
- `Close()`: Shuts down test server
- `MakeRequest(method, path, body string, headers map[string]string) (*http.Response, error)`: Makes HTTP request
- `MakeRequestWithCookies(...)`: Makes HTTP request with cookies
- `LoginAsAdmin() (*http.Cookie, error)`: Performs admin login and returns session cookie

### TestEnvironment

Manages test environment configuration:

```go
type TestEnvironment struct {
    originalEnv map[string]string
    tempDirs    []string
}
```

**Methods:**
- `NewTestEnvironment() *TestEnvironment`: Creates new environment manager
- `SetEnv(key, value string)`: Sets environment variable (remembers original)
- `SetTestEnv(dbPath, templatesPath string)`: Sets common test environment variables
- `CreateTempDir(prefix string) (string, error)`: Creates temporary directory
- `Cleanup()`: Restores environment and removes temp directories

### Assertion Helpers

Common assertion functions for testing:

- `AssertStatusCode(t *testing.T, resp *http.Response, expected int)`: Checks status code
- `AssertContains(t *testing.T, body, expected string)`: Checks if body contains string
- `AssertNotContains(t *testing.T, body, unexpected string)`: Checks if body doesn't contain string
- `AssertRedirect(t *testing.T, resp *http.Response, expectedLocation string)`: Checks redirect
- `AssertCookieExists(t *testing.T, resp *http.Response, cookieName string) *http.Cookie`: Checks cookie exists

### Mock Utilities

The package includes several mock implementations for isolated testing:

- `MockSessionDB`: Mock session database
- `TestDataGenerator`: Generates test data
- `MockHTTPHandler`: Mock HTTP handler
- `DatabaseMock`: Mock database operations
- `TestRequestBuilder`: Fluent API for building test requests

## Test Data

### Default Test Data

When you call `runner.SetupTest()`, the following test data is created:

**Posts:**
- "Test Post 1" with body content
- "Test Post 2" with body content  
- "Test Post 3" with body content

**Comments:**
- 3 test comments on the posts

**Users:**
- Admin user with credentials from config

### Custom Test Data

You can create custom test data using the `TestDataGenerator`:

```go
generator := testutils.NewTestDataGenerator()

// Generate test post
post := generator.GeneratePost()

// Generate test comment
comment := generator.GenerateComment(postID)

// Generate test user
user := generator.GenerateUser(session.ADMIN)
```

## Environment Configuration

The testing infrastructure automatically manages environment variables:

```go
env := testutils.NewTestEnvironment()
defer env.Cleanup()

// Set test-specific environment
tempDir, _ := env.CreateTempDir("test_")
dbPath := filepath.Join(tempDir, "test.db")
env.SetTestEnv(dbPath, "templates/*.gohtml")
```

## Integration Testing

For integration tests, use the provided integration test framework:

```go
// tests/integration/integration_test.go
func TestBlogPlatformIntegration(t *testing.T) {
    runner := testutils.NewTestRunner(t)
    defer runner.Close()

    if err := runner.SetupTest(); err != nil {
        t.Fatalf("Failed to setup test: %v", err)
    }

    t.Run("HomePage", func(t *testing.T) {
        // Test home page functionality
    })

    t.Run("PostOperations", func(t *testing.T) {
        // Test CRUD operations on posts
    })

    t.Run("Authentication", func(t *testing.T) {
        // Test login/logout functionality
    })
}
```

## Best Practices

### 1. Always Clean Up Resources

```go
func TestSomething(t *testing.T) {
    runner := testutils.NewTestRunner(t)
    defer runner.Close() // Always defer cleanup
    
    // Test code here
}
```

### 2. Use Subtests for Organization

```go
func TestFeature(t *testing.T) {
    runner := testutils.NewTestRunner(t)
    defer runner.Close()

    t.Run("SubFeature1", func(t *testing.T) {
        // Test sub-feature 1
    })

    t.Run("SubFeature2", func(t *testing.T) {
        // Test sub-feature 2
    })
}
```

### 3. Use Assertion Helpers

```go
// Instead of manual checks
if resp.StatusCode != http.StatusOK {
    t.Errorf("Expected 200, got %d", resp.StatusCode)
}

// Use assertion helpers
testutils.AssertStatusCode(t, resp, http.StatusOK)
```

### 4. Test Both Success and Error Cases

```go
func TestPostCreation(t *testing.T) {
    runner := testutils.NewTestRunner(t)
    defer runner.Close()

    t.Run("ValidPost", func(t *testing.T) {
        // Test successful post creation
    })

    t.Run("InvalidPost", func(t *testing.T) {
        // Test post creation with invalid data
    })

    t.Run("UnauthorizedPost", func(t *testing.T) {
        // Test post creation without authentication
    })
}
```

### 5. Use Test Data Generators for Variety

```go
generator := testutils.NewTestDataGenerator()

for i := 0; i < 10; i++ {
    post := generator.GeneratePost()
    // Test with different post data
}
```

## Running Tests

### Run All Tests
```bash
go test ./...
```

### Run Specific Test Package
```bash
go test ./testutils
go test ./tests/integration
```

### Run with Verbose Output
```bash
go test -v ./testutils
```

### Run with Coverage
```bash
go test -cover ./...
```

### Run Benchmarks
```bash
go test -bench=. ./testutils
```

## Troubleshooting

### Common Issues

1. **Database Lock Errors**: Make sure to call `defer runner.Close()` to clean up database connections

2. **Template Not Found**: Ensure templates path is correct in test environment:
   ```go
   env.SetEnv("TEMPLATES", "../../templates/*.gohtml")
   ```

3. **Port Already in Use**: The HTTP test helper uses random ports, but if you see port conflicts, restart your tests

4. **Permission Errors**: Ensure test has write permissions for temporary directories

### Debug Tips

1. **Enable Verbose Logging**: Set environment variable for detailed logs:
   ```bash
   VERBOSE=1 go test -v ./testutils
   ```

2. **Inspect Test Database**: The test database files are created in temp directories. You can inspect them during debugging by adding a breakpoint after `NewTestRunner()`.

3. **Check HTTP Responses**: Use `ioutil.ReadAll(resp.Body)` to inspect response bodies during debugging.

## Contributing

When adding new testing utilities:

1. Follow the existing patterns and naming conventions
2. Add comprehensive documentation and examples
3. Include both unit tests and integration tests
4. Update this README with new functionality
5. Ensure all tests pass before submitting

## Examples

See the `example_test.go` and `tests/integration/integration_test.go` files for comprehensive examples of how to use the testing infrastructure.