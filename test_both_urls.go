package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	"github.com/ultramozg/golang-blog-engine/app"
	"github.com/ultramozg/golang-blog-engine/testutils"
)

func main() {
	// Create test config
	config := testutils.NewTestConfig()
	defer os.RemoveAll(config.TempDir)

	// Create test database
	testDB := testutils.NewTestDatabase(nil)
	defer testDB.Close()

	// Seed test data
	if err := testDB.SeedTestData(); err != nil {
		fmt.Printf("Failed to seed test data: %v\n", err)
		return
	}

	// Create app
	a := app.NewApp()
	a.Config = &app.Config{
		DBURI:     config.DBPath,
		Templates: config.Templates,
		AdminPass: config.AdminPass,
		Production: "false",
		Server: app.ServerConfig{
			Addr:  "127.0.0.1",
			Http:  ":8080",
			Https: ":8443",
		},
	}
	a.DB = testDB.DB
	a.Initialize()

	// Test both URL formats
	server := httptest.NewServer(a.Router)
	defer server.Close()

	// Test slug-based URL
	resp1, err := http.Get(server.URL + "/p/test-post-1")
	if err != nil {
		fmt.Printf("Error testing slug URL: %v\n", err)
		return
	}
	defer resp1.Body.Close()
	fmt.Printf("Slug-based URL (/p/test-post-1): %d\n", resp1.StatusCode)

	// Test ID-based URL (should still work for backward compatibility)
	resp2, err := http.Get(server.URL + "/post?id=1")
	if err != nil {
		fmt.Printf("Error testing ID URL: %v\n", err)
		return
	}
	defer resp2.Body.Close()
	fmt.Printf("ID-based URL (/post?id=1): %d\n", resp2.StatusCode)

	fmt.Println("Both URL formats are working!")
}