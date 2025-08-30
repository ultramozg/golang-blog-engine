package testutils

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// TestConfigManager manages test configuration with environment-specific settings
type TestConfigManager struct {
	config       *EnhancedTestConfig
	originalEnv  map[string]string
	tempDirs     []string
	configFile   string
}

// EnhancedTestConfig provides comprehensive test configuration
type EnhancedTestConfig struct {
	// Database configuration
	Database DatabaseConfig `json:"database"`
	
	// Server configuration
	Server ServerConfig `json:"server"`
	
	// Authentication configuration
	Auth AuthConfig `json:"auth"`
	
	// File upload configuration
	FileUpload FileUploadConfig `json:"file_upload"`
	
	// Test-specific configuration
	Test TestSpecificConfig `json:"test"`
	
	// Environment configuration
	Environment EnvironmentConfig `json:"environment"`
}

// DatabaseConfig holds database-related test configuration
type DatabaseConfig struct {
	Driver          string        `json:"driver"`
	DSN             string        `json:"dsn"`
	MaxConnections  int           `json:"max_connections"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
	MigrationsPath  string        `json:"migrations_path"`
	SeedData        bool          `json:"seed_data"`
}

// ServerConfig holds server-related test configuration
type ServerConfig struct {
	Host         string        `json:"host"`
	Port         int           `json:"port"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	TLSEnabled   bool          `json:"tls_enabled"`
}

// AuthConfig holds authentication-related test configuration
type AuthConfig struct {
	AdminUsername    string `json:"admin_username"`
	AdminPassword    string `json:"admin_password"`
	SessionTimeout   time.Duration `json:"session_timeout"`
	JWTSecret        string `json:"jwt_secret"`
	OAuthClientID    string `json:"oauth_client_id"`
	OAuthClientSecret string `json:"oauth_client_secret"`
}

// FileUploadConfig holds file upload-related test configuration
type FileUploadConfig struct {
	MaxFileSize      int64    `json:"max_file_size"`
	AllowedMimeTypes []string `json:"allowed_mime_types"`
	UploadPath       string   `json:"upload_path"`
	ThumbnailSize    int      `json:"thumbnail_size"`
}

// TestSpecificConfig holds test-specific configuration
type TestSpecificConfig struct {
	Timeout           time.Duration `json:"timeout"`
	RetryAttempts     int           `json:"retry_attempts"`
	RetryDelay        time.Duration `json:"retry_delay"`
	ParallelTests     int           `json:"parallel_tests"`
	CleanupOnFailure  bool          `json:"cleanup_on_failure"`
	VerboseLogging    bool          `json:"verbose_logging"`
	CoverageThreshold float64       `json:"coverage_threshold"`
}

// EnvironmentConfig holds environment-specific configuration
type EnvironmentConfig struct {
	Name        string            `json:"name"`
	Variables   map[string]string `json:"variables"`
	TempDirBase string            `json:"temp_dir_base"`
	LogLevel    string            `json:"log_level"`
}

// NewTestConfigManager creates a new test configuration manager
func NewTestConfigManager() *TestConfigManager {
	return &TestConfigManager{
		originalEnv: make(map[string]string),
		tempDirs:    make([]string, 0),
		config:      getDefaultTestConfig(),
	}
}

// getDefaultTestConfig returns default test configuration
func getDefaultTestConfig() *EnhancedTestConfig {
	return &EnhancedTestConfig{
		Database: DatabaseConfig{
			Driver:          "sqlite",
			DSN:             ":memory:",
			MaxConnections:  10,
			ConnMaxLifetime: 30 * time.Minute,
			MigrationsPath:  "migrations",
			SeedData:        true,
		},
		Server: ServerConfig{
			Host:         "127.0.0.1",
			Port:         0, // Random port
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			TLSEnabled:   false,
		},
		Auth: AuthConfig{
			AdminUsername:     "admin",
			AdminPassword:     "testpass123",
			SessionTimeout:    24 * time.Hour,
			JWTSecret:         "test-jwt-secret-key",
			OAuthClientID:     "test-oauth-client-id",
			OAuthClientSecret: "test-oauth-client-secret",
		},
		FileUpload: FileUploadConfig{
			MaxFileSize:      10 * 1024 * 1024, // 10MB
			AllowedMimeTypes: []string{"image/jpeg", "image/png", "image/gif", "text/plain", "application/pdf"},
			UploadPath:       "test_uploads",
			ThumbnailSize:    300,
		},
		Test: TestSpecificConfig{
			Timeout:           30 * time.Second,
			RetryAttempts:     3,
			RetryDelay:        100 * time.Millisecond,
			ParallelTests:     4,
			CleanupOnFailure:  true,
			VerboseLogging:    false,
			CoverageThreshold: 80.0,
		},
		Environment: EnvironmentConfig{
			Name:        "test",
			Variables:   make(map[string]string),
			TempDirBase: os.TempDir(),
			LogLevel:    "error",
		},
	}
}

// LoadFromFile loads configuration from a JSON file
func (tcm *TestConfigManager) LoadFromFile(filename string) error {
	tcm.configFile = filename
	
	// #nosec G304 - filename is controlled by test code
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, use defaults and create it
			return tcm.SaveToFile(filename)
		}
		return fmt.Errorf("failed to read config file: %v", err)
	}

	err = json.Unmarshal(data, tcm.config)
	if err != nil {
		return fmt.Errorf("failed to parse config file: %v", err)
	}

	return nil
}

// SaveToFile saves configuration to a JSON file
func (tcm *TestConfigManager) SaveToFile(filename string) error {
	data, err := json.MarshalIndent(tcm.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	err = os.WriteFile(filename, data, 0600)
	if err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

// LoadFromEnvironment loads configuration from environment variables
func (tcm *TestConfigManager) LoadFromEnvironment() {
	// Database configuration
	if dsn := os.Getenv("TEST_DB_DSN"); dsn != "" {
		tcm.config.Database.DSN = dsn
	}
	if driver := os.Getenv("TEST_DB_DRIVER"); driver != "" {
		tcm.config.Database.Driver = driver
	}
	if maxConn := os.Getenv("TEST_DB_MAX_CONNECTIONS"); maxConn != "" {
		if val, err := strconv.Atoi(maxConn); err == nil {
			tcm.config.Database.MaxConnections = val
		}
	}

	// Server configuration
	if host := os.Getenv("TEST_SERVER_HOST"); host != "" {
		tcm.config.Server.Host = host
	}
	if port := os.Getenv("TEST_SERVER_PORT"); port != "" {
		if val, err := strconv.Atoi(port); err == nil {
			tcm.config.Server.Port = val
		}
	}

	// Auth configuration
	if adminUser := os.Getenv("TEST_ADMIN_USERNAME"); adminUser != "" {
		tcm.config.Auth.AdminUsername = adminUser
	}
	if adminPass := os.Getenv("TEST_ADMIN_PASSWORD"); adminPass != "" {
		tcm.config.Auth.AdminPassword = adminPass
	}

	// File upload configuration
	if maxSize := os.Getenv("TEST_MAX_FILE_SIZE"); maxSize != "" {
		if val, err := strconv.ParseInt(maxSize, 10, 64); err == nil {
			tcm.config.FileUpload.MaxFileSize = val
		}
	}
	if uploadPath := os.Getenv("TEST_UPLOAD_PATH"); uploadPath != "" {
		tcm.config.FileUpload.UploadPath = uploadPath
	}

	// Test configuration
	if timeout := os.Getenv("TEST_TIMEOUT"); timeout != "" {
		if val, err := time.ParseDuration(timeout); err == nil {
			tcm.config.Test.Timeout = val
		}
	}
	if parallel := os.Getenv("TEST_PARALLEL"); parallel != "" {
		if val, err := strconv.Atoi(parallel); err == nil {
			tcm.config.Test.ParallelTests = val
		}
	}
	if verbose := os.Getenv("TEST_VERBOSE"); verbose != "" {
		tcm.config.Test.VerboseLogging = strings.ToLower(verbose) == "true"
	}

	// Environment configuration
	if envName := os.Getenv("TEST_ENVIRONMENT"); envName != "" {
		tcm.config.Environment.Name = envName
	}
	if logLevel := os.Getenv("TEST_LOG_LEVEL"); logLevel != "" {
		tcm.config.Environment.LogLevel = logLevel
	}
}

// SetEnvironmentVariables sets environment variables based on configuration
func (tcm *TestConfigManager) SetEnvironmentVariables() error {
	envVars := map[string]string{
		"DBURI":          tcm.config.Database.DSN,
		"TEMPLATES":      GetTemplatesPath(),
		"ADMIN_PASSWORD": tcm.config.Auth.AdminPassword,
		"PRODUCTION":     "false",
		"IP_ADDR":        tcm.config.Server.Host,
		"HTTP_PORT":      fmt.Sprintf(":%d", tcm.config.Server.Port),
		"HTTPS_PORT":     fmt.Sprintf(":%d", tcm.config.Server.Port+1),
		"DOMAIN":         fmt.Sprintf("http://%s:%d", tcm.config.Server.Host, tcm.config.Server.Port),
		"LOG_LEVEL":      tcm.config.Environment.LogLevel,
	}

	// Add custom environment variables
	for k, v := range tcm.config.Environment.Variables {
		envVars[k] = v
	}

	// Set environment variables and remember originals
	for key, value := range envVars {
		if original, exists := os.LookupEnv(key); exists {
			tcm.originalEnv[key] = original
		} else {
			tcm.originalEnv[key] = ""
		}
		err := os.Setenv(key, value)
		if err != nil {
			return fmt.Errorf("failed to set environment variable %s: %v", key, err)
		}
	}

	return nil
}

// CreateTempDirectory creates a temporary directory for testing
func (tcm *TestConfigManager) CreateTempDirectory(prefix string) (string, error) {
	tempDir, err := os.MkdirTemp(tcm.config.Environment.TempDirBase, prefix)
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %v", err)
	}

	tcm.tempDirs = append(tcm.tempDirs, tempDir)
	return tempDir, nil
}

// GetDatabaseConfig returns database configuration
func (tcm *TestConfigManager) GetDatabaseConfig() DatabaseConfig {
	return tcm.config.Database
}

// GetServerConfig returns server configuration
func (tcm *TestConfigManager) GetServerConfig() ServerConfig {
	return tcm.config.Server
}

// GetAuthConfig returns authentication configuration
func (tcm *TestConfigManager) GetAuthConfig() AuthConfig {
	return tcm.config.Auth
}

// GetFileUploadConfig returns file upload configuration
func (tcm *TestConfigManager) GetFileUploadConfig() FileUploadConfig {
	return tcm.config.FileUpload
}

// GetTestConfig returns test-specific configuration
func (tcm *TestConfigManager) GetTestConfig() TestSpecificConfig {
	return tcm.config.Test
}

// GetEnvironmentConfig returns environment configuration
func (tcm *TestConfigManager) GetEnvironmentConfig() EnvironmentConfig {
	return tcm.config.Environment
}

// UpdateDatabaseDSN updates the database DSN with a specific path
func (tcm *TestConfigManager) UpdateDatabaseDSN(dbPath string) {
	tcm.config.Database.DSN = dbPath
}

// UpdateUploadPath updates the file upload path
func (tcm *TestConfigManager) UpdateUploadPath(uploadPath string) {
	tcm.config.FileUpload.UploadPath = uploadPath
}

// SetCustomEnvironmentVariable sets a custom environment variable
func (tcm *TestConfigManager) SetCustomEnvironmentVariable(key, value string) {
	if tcm.config.Environment.Variables == nil {
		tcm.config.Environment.Variables = make(map[string]string)
	}
	tcm.config.Environment.Variables[key] = value
}

// Cleanup restores original environment variables and removes temp directories
func (tcm *TestConfigManager) Cleanup() {
	// Restore original environment variables
	for key, original := range tcm.originalEnv {
		if original == "" {
			if err := os.Unsetenv(key); err != nil {
				fmt.Printf("Warning: failed to unset env var %s: %v\n", key, err)
			}
		} else {
			if err := os.Setenv(key, original); err != nil {
				fmt.Printf("Warning: failed to restore env var %s: %v\n", key, err)
			}
		}
	}

	// Remove temporary directories
	for _, dir := range tcm.tempDirs {
		if err := os.RemoveAll(dir); err != nil {
			// Log error but don't fail cleanup
			fmt.Printf("Warning: failed to remove temp directory %s: %v\n", dir, err)
		}
	}

	// Clear tracking
	tcm.originalEnv = make(map[string]string)
	tcm.tempDirs = make([]string, 0)
}

// Validate validates the configuration
func (tcm *TestConfigManager) Validate() error {
	config := tcm.config

	// Validate database configuration
	if config.Database.Driver == "" {
		return fmt.Errorf("database driver cannot be empty")
	}
	if config.Database.MaxConnections <= 0 {
		return fmt.Errorf("database max connections must be positive")
	}

	// Validate server configuration
	if config.Server.Host == "" {
		return fmt.Errorf("server host cannot be empty")
	}
	if config.Server.Port < 0 || config.Server.Port > 65535 {
		return fmt.Errorf("server port must be between 0 and 65535")
	}

	// Validate auth configuration
	if config.Auth.AdminUsername == "" {
		return fmt.Errorf("admin username cannot be empty")
	}
	if config.Auth.AdminPassword == "" {
		return fmt.Errorf("admin password cannot be empty")
	}

	// Validate file upload configuration
	if config.FileUpload.MaxFileSize <= 0 {
		return fmt.Errorf("max file size must be positive")
	}
	if config.FileUpload.UploadPath == "" {
		return fmt.Errorf("upload path cannot be empty")
	}

	// Validate test configuration
	if config.Test.Timeout <= 0 {
		return fmt.Errorf("test timeout must be positive")
	}
	if config.Test.ParallelTests <= 0 {
		return fmt.Errorf("parallel tests count must be positive")
	}
	if config.Test.CoverageThreshold < 0 || config.Test.CoverageThreshold > 100 {
		return fmt.Errorf("coverage threshold must be between 0 and 100")
	}

	return nil
}

// Clone creates a deep copy of the configuration
func (tcm *TestConfigManager) Clone() (*TestConfigManager, error) {
	data, err := json.Marshal(tcm.config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config for cloning: %v", err)
	}

	newConfig := &EnhancedTestConfig{}
	err = json.Unmarshal(data, newConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config for cloning: %v", err)
	}

	return &TestConfigManager{
		config:      newConfig,
		originalEnv: make(map[string]string),
		tempDirs:    make([]string, 0),
	}, nil
}

// GetConfigForEnvironment returns configuration for a specific environment
func GetConfigForEnvironment(env string) (*TestConfigManager, error) {
	tcm := NewTestConfigManager()
	
	// Try to load environment-specific config file
	configFile := fmt.Sprintf("testconfig_%s.json", env)
	if _, err := os.Stat(configFile); err == nil {
		err = tcm.LoadFromFile(configFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load config for environment %s: %v", env, err)
		}
	}

	// Load from environment variables
	tcm.LoadFromEnvironment()

	// Set environment name
	tcm.config.Environment.Name = env

	// Validate configuration
	err := tcm.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid configuration for environment %s: %v", env, err)
	}

	return tcm, nil
}

// CreateTestConfigFile creates a sample test configuration file
func CreateTestConfigFile(filename string) error {
	tcm := NewTestConfigManager()
	return tcm.SaveToFile(filename)
}