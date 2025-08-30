# Enhanced Testing Infrastructure Guide

This document provides a comprehensive guide to the enhanced testing infrastructure for the Go blog platform.

## Overview

The enhanced testing infrastructure provides comprehensive utilities for:
- Database setup, seeding, and teardown with advanced features
- HTTP request/response testing with fluent API
- Test configuration management with environment-specific settings
- Enhanced assertion helpers for comprehensive validation
- Performance testing and benchmarking utilities

## Quick Start

### Basic Usage with Enhanced Test Runner

```go
func TestMyFeature(t *testing.T) {
    // Create enhanced test runner
    runner := testutils.NewEnhancedTestRunner(t)
    defer runner.Close()

    // Setup test data
    if err := runner.SetupTest(); err != nil {
        t.Fatalf("Failed to setup test: %v", err)
    }

    // Your test code here
    resp, err := runner.HTTP.NewRequest().
        GET("/page").
        Query("p", "0").
        Execute()
    
    testutils.AssertNil(t, err, "Request should succeed")
    defer resp.Body.Close()
    
    testutils.AssertHTTPStatusCode(t, resp, http.StatusOK)
}
```

## Enhanced Database Testing

### DatabaseTestHelper Features

The `DatabaseTestHelper` provides advanced database testing capabilities:

```go
func TestDatabaseOperations(t *testing.T) {
    dh := testutils.NewDatabaseTestHelper(t)
    defer dh.Close()

    // Seed test data with various configurations
    err := dh.SeedTestPosts(10)
    testutils.AssertNil(t, err, "Should seed posts")
    
    err = dh.SeedTestComments(1, 5) // 5 comments for post ID 1
    testutils.AssertNil(t, err, "Should seed comments")
    
    err = dh.SeedTestUsers()
    testutils.AssertNil(t, err, "Should seed users")
    
    err = dh.SeedTestFiles(3)
    testutils.AssertNil(t, err, "Should seed files")

    // Use enhanced assertions
    dh.AssertRecordCount(t, "posts", 10)
    dh.AssertRecordExists(t, "posts", map[string]interface{}{
        "title": "Test Post 1",
    })

    // Test transactions with automatic rollback
    dh.Transaction(t, func(tx *sql.Tx) {
        _, err := tx.Exec("INSERT INTO posts (title, body, datepost, slug) VALUES (?, ?, ?, ?)",
            "Temp Post", "Temp Body", "2024-01-01", "temp-post")
        testutils.AssertNil(t, err, "Should insert in transaction")
        
        // This will be rolled back automatically
    })
    
    // Verify rollback
    dh.AssertRecordCount(t, "posts", 10) // Still 10, not 11

    // Backup and restore functionality
    backupPath, err := dh.BackupDatabase()
    testutils.AssertNil(t, err, "Should create backup")
    
    err = dh.ClearAllTables()
    testutils.AssertNil(t, err, "Should clear tables")
    
    err = dh.RestoreDatabase(backupPath)
    testutils.AssertNil(t, err, "Should restore database")
    
    dh.AssertRecordCount(t, "posts", 10) // Restored
}
```

### Advanced Database Features

- **Automatic Schema Creation**: Creates test-specific tables
- **Transaction Testing**: Automatic rollback for isolated testing
- **Backup/Restore**: Database state management
- **Wait Conditions**: Wait for database conditions to be met
- **Record Assertions**: Convenient database validation helpers

## Enhanced HTTP Testing

### HTTPTestClient Features

The `HTTPTestClient` provides a fluent API for HTTP testing:

```go
func TestHTTPOperations(t *testing.T) {
    runner := testutils.NewEnhancedTestRunner(t)
    defer runner.Close()
    
    err := runner.SetupTest()
    testutils.AssertNil(t, err, "Should setup test")

    // Fluent request building
    resp, err := runner.HTTP.NewRequest().
        GET("/page").
        Query("p", "0").
        Header("Accept", "text/html").
        Execute()
    
    testutils.AssertNil(t, err, "Request should succeed")
    defer resp.Body.Close()
    
    // Enhanced response handling
    helper, err := testutils.NewResponseHelper(resp)
    testutils.AssertNil(t, err, "Should create response helper")
    
    testutils.AssertEqual(t, http.StatusOK, helper.StatusCode())
    testutils.AssertTrue(t, helper.ContainsString("Test Post"))

    // Session management
    sessionHelper := runner.HTTP.NewSessionHelper()
    sessionCookie, err := sessionHelper.LoginAsAdmin("admin", "testpass123")
    testutils.AssertNil(t, err, "Should login successfully")
    
    // Authenticated requests
    resp, err = runner.HTTP.NewRequest().
        POST("/create").
        Cookie(sessionCookie).
        Form("title", "New Post").
        Form("body", "New Body").
        Execute()
    
    testutils.AssertNil(t, err, "Should create post")
    defer resp.Body.Close()
    
    testutils.AssertHTTPRedirect(t, resp, "/")

    // Multipart form uploads
    fileContent := []byte("test file content")
    resp, err = runner.HTTP.NewRequest().
        POST("/upload-file").
        Cookie(sessionCookie).
        Multipart().
        Field("description", "Test file").
        File("file", "test.txt", fileContent, "text/plain").
        Execute()
    
    testutils.AssertNil(t, err, "Should upload file")
    defer resp.Body.Close()
}
```

### Performance Testing

```go
func TestPerformance(t *testing.T) {
    runner := testutils.NewEnhancedTestRunner(t)
    defer runner.Close()
    
    // Measure response time
    resp, duration, err := runner.HTTP.MeasureResponseTime(func() (*http.Response, error) {
        return runner.HTTP.NewRequest().
            GET("/page").
            Query("p", "0").
            Execute()
    })
    
    testutils.AssertNil(t, err, "Request should succeed")
    defer resp.Body.Close()
    
    testutils.AssertResponseTime(t, duration, 100*time.Millisecond)

    // Concurrent testing
    responses, errors := runner.HTTP.ConcurrentRequests(10, func(index int) (*http.Response, error) {
        return runner.HTTP.NewRequest().
            GET("/page").
            Query("p", fmt.Sprintf("%d", index)).
            Execute()
    })
    
    for i, err := range errors {
        testutils.AssertNil(t, err, fmt.Sprintf("Request %d should succeed", i))
    }
    
    for _, resp := range responses {
        if resp != nil {
            testutils.AssertHTTPStatusCode(t, resp, http.StatusOK)
            resp.Body.Close()
        }
    }
}
```

## Test Configuration Management

### TestConfigManager Features

The `TestConfigManager` provides comprehensive configuration management:

```go
func TestWithCustomConfig(t *testing.T) {
    // Create configuration manager
    config := testutils.NewTestConfigManager()
    defer config.Cleanup()
    
    // Load from environment variables
    config.LoadFromEnvironment()
    
    // Customize configuration
    config.SetCustomEnvironmentVariable("CUSTOM_VAR", "custom_value")
    
    dbConfig := config.GetDatabaseConfig()
    dbConfig.MaxConnections = 20
    
    authConfig := config.GetAuthConfig()
    authConfig.AdminPassword = "custom_password"
    
    // Apply configuration
    err := config.SetEnvironmentVariables()
    testutils.AssertNil(t, err, "Should set environment variables")
    
    // Validate configuration
    err = config.Validate()
    testutils.AssertNil(t, err, "Configuration should be valid")
    
    // Create temporary directories
    tempDir, err := config.CreateTempDirectory("test_")
    testutils.AssertNil(t, err, "Should create temp directory")
    testutils.AssertNotEmpty(t, tempDir, "Temp directory should not be empty")
}
```

### Configuration File Management

```go
func TestConfigFiles(t *testing.T) {
    config := testutils.NewTestConfigManager()
    
    // Save configuration to file
    configFile := "test_config.json"
    err := config.SaveToFile(configFile)
    testutils.AssertNil(t, err, "Should save config file")
    defer os.Remove(configFile)
    
    // Load configuration from file
    newConfig := testutils.NewTestConfigManager()
    err = newConfig.LoadFromFile(configFile)
    testutils.AssertNil(t, err, "Should load config file")
    
    // Verify loaded configuration
    originalAuth := config.GetAuthConfig()
    loadedAuth := newConfig.GetAuthConfig()
    testutils.AssertEqual(t, originalAuth.AdminUsername, loadedAuth.AdminUsername)
}
```

## Enhanced Assertions

### Basic Assertions

```go
func TestAssertions(t *testing.T) {
    // Value assertions
    testutils.AssertEqual(t, 42, 42, "Values should be equal")
    testutils.AssertNotEqual(t, 42, 43, "Values should not be equal")
    testutils.AssertTrue(t, true, "Condition should be true")
    testutils.AssertFalse(t, false, "Condition should be false")
    testutils.AssertNil(t, nil, "Value should be nil")
    testutils.AssertNotNil(t, "not nil", "Value should not be nil")
    
    // Collection assertions
    testutils.AssertEmpty(t, "", "String should be empty")
    testutils.AssertNotEmpty(t, "not empty", "String should not be empty")
    testutils.AssertLen(t, []int{1, 2, 3}, 3, "Slice should have length 3")
    
    // String assertions
    text := "Hello, World!"
    testutils.AssertContainsSubstring(t, text, "Hello", "Should contain substring")
    testutils.AssertNotContainsSubstring(t, text, "Goodbye", "Should not contain substring")
    testutils.AssertMatchesRegex(t, text, `Hello.*World`, "Should match regex")
    testutils.AssertNotMatchesRegex(t, text, `^Goodbye`, "Should not match regex")
}
```

### HTTP-Specific Assertions

```go
func TestHTTPAssertions(t *testing.T) {
    runner := testutils.NewEnhancedTestRunner(t)
    defer runner.Close()
    
    resp, err := runner.HTTP.NewRequest().GET("/page").Query("p", "0").Execute()
    testutils.AssertNil(t, err, "Request should succeed")
    defer resp.Body.Close()
    
    // HTTP status assertions
    testutils.AssertHTTPStatusCode(t, resp, http.StatusOK)
    
    // Header assertions
    testutils.AssertHTTPHeaderExists(t, resp, "Content-Type")
    testutils.AssertHTTPHeader(t, resp, "Content-Type", "text/html; charset=utf-8")
    
    // Body assertions
    body, err := io.ReadAll(resp.Body)
    testutils.AssertNil(t, err, "Should read body")
    
    testutils.AssertHTTPBodyContains(t, body, "Test Post")
    testutils.AssertHTTPBodyNotContains(t, body, "Secret Content")
    
    // JSON response assertions
    var jsonData map[string]interface{}
    testutils.AssertHTTPBodyJSON(t, body, &jsonData) // Will fail for HTML, but demonstrates usage
}
```

### Time-Based Assertions

```go
func TestTimeBasedAssertions(t *testing.T) {
    counter := 0
    
    // Wait for condition to become true
    testutils.AssertEventually(t, func() bool {
        counter++
        return counter >= 5
    }, 1*time.Second, 10*time.Millisecond, "Counter should reach 5")
    
    // Ensure condition never becomes true
    stable := true
    testutils.AssertNever(t, func() bool {
        return !stable
    }, 100*time.Millisecond, 10*time.Millisecond, "Stable should remain true")
}
```

### Performance Assertions

```go
func TestPerformanceAssertions(t *testing.T) {
    // Response time assertion
    start := time.Now()
    time.Sleep(10 * time.Millisecond)
    duration := time.Since(start)
    
    testutils.AssertResponseTime(t, duration, 50*time.Millisecond)
    
    // Memory usage assertion
    var m1, m2 runtime.MemStats
    runtime.ReadMemStats(&m1)
    
    // Allocate some memory
    _ = make([]byte, 1024)
    
    runtime.ReadMemStats(&m2)
    testutils.AssertMemoryUsage(t, m1.Alloc, m2.Alloc, 10*1024)
}
```

## Integration with Existing Tests

### Backward Compatibility

The enhanced testing infrastructure is fully backward compatible with existing tests:

```go
func TestBackwardCompatibility(t *testing.T) {
    // Existing TestRunner still works
    runner := testutils.NewTestRunner(t)
    defer runner.Close()
    
    err := runner.SetupTest()
    testutils.AssertNil(t, err, "Should setup test")
    
    // Existing assertion helpers still work
    resp, err := runner.HTTP.MakeRequest("GET", "/page?p=0", "", nil)
    testutils.AssertNil(t, err, "Request should succeed")
    defer resp.Body.Close()
    
    testutils.AssertStatusCode(t, resp, http.StatusOK)
    
    body, err := io.ReadAll(resp.Body)
    testutils.AssertNil(t, err, "Should read body")
    
    testutils.AssertContains(t, string(body), "Test Post")
}
```

### Migration Guide

To migrate existing tests to use enhanced features:

1. **Replace TestRunner with EnhancedTestRunner**:
   ```go
   // Old
   runner := testutils.NewTestRunner(t)
   
   // New
   runner := testutils.NewEnhancedTestRunner(t)
   ```

2. **Use fluent HTTP API**:
   ```go
   // Old
   resp, err := runner.HTTP.MakeRequest("GET", "/page?p=0", "", nil)
   
   // New
   resp, err := runner.HTTP.NewRequest().GET("/page").Query("p", "0").Execute()
   ```

3. **Use enhanced assertions**:
   ```go
   // Old
   if resp.StatusCode != http.StatusOK {
       t.Errorf("Expected 200, got %d", resp.StatusCode)
   }
   
   // New
   testutils.AssertHTTPStatusCode(t, resp, http.StatusOK)
   ```

## Best Practices

### 1. Test Organization

```go
func TestFeature(t *testing.T) {
    runner := testutils.NewEnhancedTestRunner(t)
    defer runner.Close()
    
    err := runner.SetupTest()
    testutils.AssertNil(t, err, "Should setup test")
    
    t.Run("SubFeature1", func(t *testing.T) {
        // Test sub-feature 1
    })
    
    t.Run("SubFeature2", func(t *testing.T) {
        // Test sub-feature 2
    })
}
```

### 2. Configuration Management

```go
func TestWithEnvironmentConfig(t *testing.T) {
    config, err := testutils.GetConfigForEnvironment("integration")
    testutils.AssertNil(t, err, "Should get config for environment")
    defer config.Cleanup()
    
    // Use environment-specific configuration
}
```

### 3. Database Testing

```go
func TestDatabaseFeature(t *testing.T) {
    dh := testutils.NewDatabaseTestHelper(t)
    defer dh.Close()
    
    // Clear data before each test
    err := dh.ClearAllTables()
    testutils.AssertNil(t, err, "Should clear tables")
    
    // Seed only the data you need
    err = dh.SeedTestPosts(5)
    testutils.AssertNil(t, err, "Should seed posts")
    
    // Use transactions for isolated testing
    dh.Transaction(t, func(tx *sql.Tx) {
        // Test database operations in isolation
    })
}
```

### 4. HTTP Testing

```go
func TestHTTPFeature(t *testing.T) {
    runner := testutils.NewEnhancedTestRunner(t)
    defer runner.Close()
    
    // Use session helper for authentication
    sessionHelper := runner.HTTP.NewSessionHelper()
    sessionCookie, err := sessionHelper.LoginAsAdmin("admin", "testpass123")
    testutils.AssertNil(t, err, "Should login")
    
    // Use fluent API for requests
    resp, err := runner.HTTP.NewRequest().
        POST("/api/endpoint").
        Cookie(sessionCookie).
        JSON(map[string]string{"key": "value"}).
        Execute()
    
    testutils.AssertNil(t, err, "Request should succeed")
    defer resp.Body.Close()
    
    // Use enhanced assertions
    testutils.AssertHTTPStatusCode(t, resp, http.StatusCreated)
}
```

### 5. Performance Testing

```go
func TestPerformance(t *testing.T) {
    runner := testutils.NewEnhancedTestRunner(t)
    defer runner.Close()
    
    // Measure single request performance
    resp, duration, err := runner.HTTP.MeasureResponseTime(func() (*http.Response, error) {
        return runner.HTTP.NewRequest().GET("/page").Execute()
    })
    
    testutils.AssertNil(t, err, "Request should succeed")
    defer resp.Body.Close()
    
    testutils.AssertResponseTime(t, duration, 100*time.Millisecond)
    
    // Test concurrent performance
    responses, errors := runner.HTTP.ConcurrentRequests(10, func(index int) (*http.Response, error) {
        return runner.HTTP.NewRequest().GET("/page").Execute()
    })
    
    // Verify all requests succeeded
    for i, err := range errors {
        testutils.AssertNil(t, err, fmt.Sprintf("Request %d should succeed", i))
    }
}
```

## Environment-Specific Testing

### Development Environment

```go
func TestDevelopment(t *testing.T) {
    config, err := testutils.GetConfigForEnvironment("development")
    testutils.AssertNil(t, err, "Should get development config")
    defer config.Cleanup()
    
    // Development-specific tests
}
```

### Integration Environment

```go
func TestIntegration(t *testing.T) {
    config, err := testutils.GetConfigForEnvironment("integration")
    testutils.AssertNil(t, err, "Should get integration config")
    defer config.Cleanup()
    
    // Integration-specific tests with external dependencies
}
```

### Production-like Environment

```go
func TestProductionLike(t *testing.T) {
    config, err := testutils.GetConfigForEnvironment("production")
    testutils.AssertNil(t, err, "Should get production config")
    defer config.Cleanup()
    
    // Production-like tests with realistic data volumes
}
```

## Benchmarking

```go
func BenchmarkFeature(b *testing.B) {
    runner := testutils.NewEnhancedTestRunner(&testing.T{})
    defer runner.Close()
    
    runner.SetupTest()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        resp, err := runner.HTTP.NewRequest().GET("/page").Execute()
        if err == nil && resp != nil {
            resp.Body.Close()
        }
    }
}
```

## Troubleshooting

### Common Issues

1. **Database Connection Errors**: Ensure proper cleanup with `defer runner.Close()`
2. **Port Conflicts**: The HTTP test helper uses random ports automatically
3. **Environment Variables**: Use `TestConfigManager` for proper environment management
4. **Memory Leaks**: Always close HTTP response bodies with `defer resp.Body.Close()`

### Debug Tips

1. **Enable Verbose Logging**: Set `TEST_VERBOSE=true` environment variable
2. **Inspect Database State**: Use `dh.GetTableCount()` and `dh.QuerySQL()` for debugging
3. **Check HTTP Responses**: Use `ResponseHelper` to inspect response details
4. **Validate Configuration**: Use `config.Validate()` to check configuration issues

## Conclusion

The enhanced testing infrastructure provides comprehensive utilities for testing Go applications with:

- **Database Testing**: Advanced database setup, seeding, and validation
- **HTTP Testing**: Fluent API for HTTP request/response testing
- **Configuration Management**: Environment-specific configuration handling
- **Enhanced Assertions**: Comprehensive assertion helpers
- **Performance Testing**: Built-in performance measurement and benchmarking

This infrastructure ensures robust, maintainable, and comprehensive test coverage for the blog platform while maintaining backward compatibility with existing tests.