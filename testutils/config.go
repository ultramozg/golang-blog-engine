package testutils

import (
	"os"
	"path/filepath"
)

// TestEnvironment manages test environment configuration
type TestEnvironment struct {
	originalEnv map[string]string
	tempDirs    []string
}

// NewTestEnvironment creates a new test environment
func NewTestEnvironment() *TestEnvironment {
	return &TestEnvironment{
		originalEnv: make(map[string]string),
		tempDirs:    make([]string, 0),
	}
}

// SetEnv sets an environment variable and remembers the original value
func (te *TestEnvironment) SetEnv(key, value string) {
	if original, exists := os.LookupEnv(key); exists {
		te.originalEnv[key] = original
	} else {
		te.originalEnv[key] = ""
	}
	os.Setenv(key, value)
}

// SetTestEnv sets up common test environment variables
func (te *TestEnvironment) SetTestEnv(dbPath, templatesPath string) {
	te.SetEnv("DBURI", dbPath)
	te.SetEnv("TEMPLATES", templatesPath)
	te.SetEnv("ADMIN_PASSWORD", "testpass123")
	te.SetEnv("PRODUCTION", "false")
	te.SetEnv("IP_ADDR", "127.0.0.1")
	te.SetEnv("HTTP_PORT", ":0") // Use random port for testing
	te.SetEnv("HTTPS_PORT", ":0")
	te.SetEnv("DOMAIN", "localhost")
}

// CreateTempDir creates a temporary directory and tracks it for cleanup
func (te *TestEnvironment) CreateTempDir(prefix string) (string, error) {
	dir, err := os.MkdirTemp("", prefix)
	if err != nil {
		return "", err
	}
	te.tempDirs = append(te.tempDirs, dir)
	return dir, nil
}

// Cleanup restores original environment variables and removes temp directories
func (te *TestEnvironment) Cleanup() {
	// Restore original environment variables
	for key, original := range te.originalEnv {
		if original == "" {
			os.Unsetenv(key)
		} else {
			os.Setenv(key, original)
		}
	}

	// Remove temporary directories
	for _, dir := range te.tempDirs {
		os.RemoveAll(dir)
	}
}

// GetTestDataPath returns the path to test data files
func GetTestDataPath() string {
	// Look for test data in common locations
	paths := []string{
		"testdata",
		"../testdata",
		"../../testdata",
		filepath.Join("testutils", "testdata"),
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// If no test data directory found, create one
	os.MkdirAll("testdata", 0755)
	return "testdata"
}

// GetTemplatesPath returns the path to template files for testing
func GetTemplatesPath() string {
	// Look for templates in common locations
	paths := []string{
		"templates/*.gohtml",
		"../templates/*.gohtml",
		"../../templates/*.gohtml",
	}

	for _, path := range paths {
		matches, _ := filepath.Glob(path)
		if len(matches) > 0 {
			return path
		}
	}

	// Default fallback
	return "templates/*.gohtml"
}

// TestFileManager helps manage test files and directories
type TestFileManager struct {
	baseDir string
	files   []string
	dirs    []string
}

// NewTestFileManager creates a new test file manager
func NewTestFileManager(baseDir string) *TestFileManager {
	return &TestFileManager{
		baseDir: baseDir,
		files:   make([]string, 0),
		dirs:    make([]string, 0),
	}
}

// CreateFile creates a test file with given content
func (tfm *TestFileManager) CreateFile(relativePath, content string) (string, error) {
	fullPath := filepath.Join(tfm.baseDir, relativePath)

	// Create directory if it doesn't exist
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	// Write file
	if err := os.WriteFile(fullPath, []byte(content), 0600); err != nil {
		return "", err
	}

	tfm.files = append(tfm.files, fullPath)
	return fullPath, nil
}

// CreateDir creates a test directory
func (tfm *TestFileManager) CreateDir(relativePath string) (string, error) {
	fullPath := filepath.Join(tfm.baseDir, relativePath)
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return "", err
	}

	tfm.dirs = append(tfm.dirs, fullPath)
	return fullPath, nil
}

// Cleanup removes all created files and directories
func (tfm *TestFileManager) Cleanup() {
	// Remove files first
	for _, file := range tfm.files {
		os.Remove(file)
	}

	// Remove directories (in reverse order)
	for i := len(tfm.dirs) - 1; i >= 0; i-- {
		os.Remove(tfm.dirs[i])
	}
}

// GetPath returns the full path for a relative path
func (tfm *TestFileManager) GetPath(relativePath string) string {
	return filepath.Join(tfm.baseDir, relativePath)
}
