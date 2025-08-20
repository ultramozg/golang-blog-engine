package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
)

func TestTemplateInitialization(t *testing.T) {
	// Create temporary directory for test templates
	tempDir, err := os.MkdirTemp("", "template_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name          string
		templateFiles map[string]string
		templateGlob  string
		expectError   bool
	}{
		{
			name: "Valid templates",
			templateFiles: map[string]string{
				"test1.gohtml": `<!DOCTYPE html><html><head><title>Test 1</title></head><body><h1>{{.Title}}</h1></body></html>`,
				"test2.gohtml": `<!DOCTYPE html><html><head><title>Test 2</title></head><body><p>{{.Content}}</p></body></html>`,
			},
			templateGlob: filepath.Join(tempDir, "*.gohtml"),
			expectError:  false,
		},
		{
			name: "Template with syntax error",
			templateFiles: map[string]string{
				"invalid.gohtml": `<!DOCTYPE html><html><head><title>Invalid</title></head><body><h1>{{.Title</h1></body></html>`,
			},
			templateGlob: filepath.Join(tempDir, "*.gohtml"),
			expectError:  true,
		},
		{
			name:          "No templates found",
			templateFiles: map[string]string{},
			templateGlob:  filepath.Join(tempDir, "*.gohtml"),
			expectError:   true,
		},
		{
			name: "Empty template file",
			templateFiles: map[string]string{
				"empty.gohtml": "",
			},
			templateGlob: filepath.Join(tempDir, "*.gohtml"),
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean temp directory
			files, _ := filepath.Glob(filepath.Join(tempDir, "*"))
			for _, file := range files {
				os.Remove(file)
			}

			// Create test template files
			for filename, content := range tt.templateFiles {
				filePath := filepath.Join(tempDir, filename)
				if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
					t.Fatalf("Failed to create test template file: %v", err)
				}
			}

			// Test template parsing
			defer func() {
				if r := recover(); r != nil {
					if !tt.expectError {
						t.Errorf("Unexpected panic: %v", r)
					}
				}
			}()

			tmpl, err := template.ParseGlob(tt.templateGlob)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if tmpl == nil {
					t.Errorf("Expected template to be created")
				}
			}
		})
	}
}

func TestTemplateExecution(t *testing.T) {
	// Create temporary directory for test templates
	tempDir, err := os.MkdirTemp("", "template_exec_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test templates
	templates := map[string]string{
		"simple.gohtml":     `<h1>{{.Title}}</h1><p>{{.Content}}</p>`,
		"with_logic.gohtml": `{{if .IsAdmin}}<p>Admin Panel</p>{{else}}<p>User Panel</p>{{end}}`,
		"with_loop.gohtml":  `<ul>{{range .Items}}<li>{{.}}</li>{{end}}</ul>`,
		"nested.gohtml":     `<div>{{template "simple.gohtml" .}}</div>`,
	}

	for filename, content := range templates {
		filePath := filepath.Join(tempDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
			t.Fatalf("Failed to create test template file: %v", err)
		}
	}

	// Parse templates
	tmpl, err := template.ParseGlob(filepath.Join(tempDir, "*.gohtml"))
	if err != nil {
		t.Fatalf("Failed to parse templates: %v", err)
	}

	tests := []struct {
		name            string
		templateName    string
		data            interface{}
		expectedContent string
		expectError     bool
	}{
		{
			name:         "Simple template execution",
			templateName: "simple.gohtml",
			data: struct {
				Title   string
				Content string
			}{
				Title:   "Test Title",
				Content: "Test Content",
			},
			expectedContent: "<h1>Test Title</h1><p>Test Content</p>",
			expectError:     false,
		},
		{
			name:         "Template with conditional logic - admin",
			templateName: "with_logic.gohtml",
			data: struct {
				IsAdmin bool
			}{
				IsAdmin: true,
			},
			expectedContent: "<p>Admin Panel</p>",
			expectError:     false,
		},
		{
			name:         "Template with conditional logic - user",
			templateName: "with_logic.gohtml",
			data: struct {
				IsAdmin bool
			}{
				IsAdmin: false,
			},
			expectedContent: "<p>User Panel</p>",
			expectError:     false,
		},
		{
			name:         "Template with loop",
			templateName: "with_loop.gohtml",
			data: struct {
				Items []string
			}{
				Items: []string{"Item 1", "Item 2", "Item 3"},
			},
			expectedContent: "<ul><li>Item 1</li><li>Item 2</li><li>Item 3</li></ul>",
			expectError:     false,
		},
		{
			name:         "Template with empty loop",
			templateName: "with_loop.gohtml",
			data: struct {
				Items []string
			}{
				Items: []string{},
			},
			expectedContent: "<ul></ul>",
			expectError:     false,
		},
		{
			name:         "Non-existent template",
			templateName: "nonexistent.gohtml",
			data:         struct{}{},
			expectError:  true,
		},
		{
			name:         "Template with missing data field",
			templateName: "simple.gohtml",
			data: struct {
				Title string
				// Missing Content field
			}{
				Title: "Test Title",
			},
			expectError: true, // Go templates error on missing fields
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tmpl.ExecuteTemplate(&buf, tt.templateName, tt.data)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				result := buf.String()
				if result != tt.expectedContent {
					t.Errorf("Expected content '%s', got '%s'", tt.expectedContent, result)
				}
			}
		})
	}
}

func TestTemplateWithComplexData(t *testing.T) {
	// Create temporary directory for test templates
	tempDir, err := os.MkdirTemp("", "template_complex_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a complex template similar to what the blog app uses
	templateContent := `<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}}</title>
</head>
<body>
    {{if .LoggedIn}}
        <p>Welcome, {{.Username}}!</p>
        {{if .IsAdmin}}
            <a href="/create">Create Post</a>
        {{end}}
    {{else}}
        <a href="/login">Login</a>
    {{end}}
    
    <div class="posts">
        {{range .Posts}}
            <article>
                <h2>{{.Title}}</h2>
                <p>{{.Body}}</p>
                <small>{{.Date}}</small>
            </article>
        {{end}}
    </div>
    
    {{if .HasNextPage}}
        <a href="/page?p={{.NextPage}}">Next</a>
    {{end}}
</body>
</html>`

	templatePath := filepath.Join(tempDir, "complex.gohtml")
	if err := os.WriteFile(templatePath, []byte(templateContent), 0600); err != nil {
		t.Fatalf("Failed to create test template file: %v", err)
	}

	// Parse template
	tmpl, err := template.ParseGlob(filepath.Join(tempDir, "*.gohtml"))
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	// Test data structure similar to what the blog app uses
	type Post struct {
		Title string
		Body  string
		Date  string
	}

	data := struct {
		Title       string
		LoggedIn    bool
		Username    string
		IsAdmin     bool
		Posts       []Post
		HasNextPage bool
		NextPage    int
	}{
		Title:    "Test Blog",
		LoggedIn: true,
		Username: "testuser",
		IsAdmin:  true,
		Posts: []Post{
			{Title: "Post 1", Body: "Body 1", Date: "2024-01-01"},
			{Title: "Post 2", Body: "Body 2", Date: "2024-01-02"},
		},
		HasNextPage: true,
		NextPage:    2,
	}

	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "complex.gohtml", data)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}

	result := buf.String()

	// Check that expected content is present
	expectedStrings := []string{
		"<title>Test Blog</title>",
		"Welcome, testuser!",
		"Create Post",
		"<h2>Post 1</h2>",
		"<p>Body 1</p>",
		"<small>2024-01-01</small>",
		"<h2>Post 2</h2>",
		"<p>Body 2</p>",
		"<small>2024-01-02</small>",
		`<a href="/page?p=2">Next</a>`,
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(result, expected) {
			t.Errorf("Expected result to contain '%s', but it didn't. Result: %s", expected, result)
		}
	}

	// Check that login link is not present (since user is logged in)
	if strings.Contains(result, `<a href="/login">Login</a>`) {
		t.Errorf("Result should not contain login link when user is logged in")
	}
}

func TestTemplateErrorHandling(t *testing.T) {
	// Create temporary directory for test templates
	tempDir, err := os.MkdirTemp("", "template_error_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create template with potential runtime errors
	templateContent := `{{.User.Name}} - {{.User.Email}} - {{.User.Profile.Bio}}`

	templatePath := filepath.Join(tempDir, "error_test.gohtml")
	if err := os.WriteFile(templatePath, []byte(templateContent), 0600); err != nil {
		t.Fatalf("Failed to create test template file: %v", err)
	}

	// Parse template
	tmpl, err := template.ParseGlob(filepath.Join(tempDir, "*.gohtml"))
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	tests := []struct {
		name        string
		data        interface{}
		expectError bool
	}{
		{
			name: "Valid nested data",
			data: struct {
				User struct {
					Name    string
					Email   string
					Profile struct {
						Bio string
					}
				}
			}{
				User: struct {
					Name    string
					Email   string
					Profile struct {
						Bio string
					}
				}{
					Name:  "John Doe",
					Email: "john@example.com",
					Profile: struct {
						Bio string
					}{
						Bio: "Software Developer",
					},
				},
			},
			expectError: false,
		},
		{
			name:        "Nil data",
			data:        nil,
			expectError: false, // Go templates can handle nil data, just produces empty output
		},
		{
			name: "Missing nested field",
			data: struct {
				User struct {
					Name  string
					Email string
					// Missing Profile field
				}
			}{
				User: struct {
					Name  string
					Email string
				}{
					Name:  "John Doe",
					Email: "john@example.com",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tmpl.ExecuteTemplate(&buf, "error_test.gohtml", tt.data)

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
