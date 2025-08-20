package app

import (
	"os"
	"testing"
)

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultVal   string
		envValue     string
		setEnv       bool
		expected     string
	}{
		{
			name:       "Environment variable exists",
			key:        "TEST_VAR",
			defaultVal: "default",
			envValue:   "custom_value",
			setEnv:     true,
			expected:   "custom_value",
		},
		{
			name:       "Environment variable does not exist",
			key:        "NON_EXISTENT_VAR",
			defaultVal: "default_value",
			setEnv:     false,
			expected:   "default_value",
		},
		{
			name:       "Environment variable is empty string",
			key:        "EMPTY_VAR",
			defaultVal: "default",
			envValue:   "",
			setEnv:     true,
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up environment
			originalValue := os.Getenv(tt.key)
			defer func() {
				if originalValue != "" {
					os.Setenv(tt.key, originalValue)
				} else {
					os.Unsetenv(tt.key)
				}
			}()

			// Set environment variable if needed
			if tt.setEnv {
				os.Setenv(tt.key, tt.envValue)
			} else {
				os.Unsetenv(tt.key)
			}

			result := getEnv(tt.key, tt.defaultVal)
			if result != tt.expected {
				t.Errorf("getEnv(%s, %s) = %s, expected %s", tt.key, tt.defaultVal, result, tt.expected)
			}
		})
	}
}

func TestNewConfig(t *testing.T) {
	// Save original environment variables
	originalVars := map[string]string{
		"IP_ADDR":               os.Getenv("IP_ADDR"),
		"HTTP_PORT":             os.Getenv("HTTP_PORT"),
		"HTTPS_PORT":            os.Getenv("HTTPS_PORT"),
		"GITHUB_AUTHORIZE_URL":  os.Getenv("GITHUB_AUTHORIZE_URL"),
		"GITHUB_TOKEN_URL":      os.Getenv("GITHUB_TOKEN_URL"),
		"REDIRECT_URL":          os.Getenv("REDIRECT_URL"),
		"CLIENT_ID":             os.Getenv("CLIENT_ID"),
		"CLIENT_SECRET":         os.Getenv("CLIENT_SECRET"),
		"TEMPLATES":             os.Getenv("TEMPLATES"),
		"PRODUCTION":            os.Getenv("PRODUCTION"),
		"DBURI":                 os.Getenv("DBURI"),
		"DOMAIN":                os.Getenv("DOMAIN"),
		"ADMIN_PASSWORD":        os.Getenv("ADMIN_PASSWORD"),
	}

	// Restore environment variables after test
	defer func() {
		for key, value := range originalVars {
			if value != "" {
				os.Setenv(key, value)
			} else {
				os.Unsetenv(key)
			}
		}
	}()

	t.Run("Default configuration", func(t *testing.T) {
		// Clear all environment variables
		for key := range originalVars {
			os.Unsetenv(key)
		}

		config := newConfig()

		// Test default values
		if config.Server.Addr != "0.0.0.0" {
			t.Errorf("Expected default IP_ADDR '0.0.0.0', got '%s'", config.Server.Addr)
		}
		if config.Server.Http != ":8080" {
			t.Errorf("Expected default HTTP_PORT ':8080', got '%s'", config.Server.Http)
		}
		if config.Server.Https != ":8443" {
			t.Errorf("Expected default HTTPS_PORT ':8443', got '%s'", config.Server.Https)
		}
		if config.Templates != "templates/*.gohtml" {
			t.Errorf("Expected default TEMPLATES 'templates/*.gohtml', got '%s'", config.Templates)
		}
		if config.Production != "false" {
			t.Errorf("Expected default PRODUCTION 'false', got '%s'", config.Production)
		}
		if config.DBURI != "file:database/database.sqlite" {
			t.Errorf("Expected default DBURI 'file:database/database.sqlite', got '%s'", config.DBURI)
		}
		if config.AdminPass != "12345" {
			t.Errorf("Expected default ADMIN_PASSWORD '12345', got '%s'", config.AdminPass)
		}
		if config.Domain != "" {
			t.Errorf("Expected default DOMAIN '', got '%s'", config.Domain)
		}
	})

	t.Run("Custom configuration from environment", func(t *testing.T) {
		// Set custom environment variables
		os.Setenv("IP_ADDR", "127.0.0.1")
		os.Setenv("HTTP_PORT", ":3000")
		os.Setenv("HTTPS_PORT", ":3443")
		os.Setenv("GITHUB_AUTHORIZE_URL", "https://github.com/login/oauth/authorize")
		os.Setenv("GITHUB_TOKEN_URL", "https://github.com/login/oauth/access_token")
		os.Setenv("REDIRECT_URL", "http://localhost:3000/auth-callback")
		os.Setenv("CLIENT_ID", "test_client_id")
		os.Setenv("CLIENT_SECRET", "test_client_secret")
		os.Setenv("TEMPLATES", "custom/templates/*.html")
		os.Setenv("PRODUCTION", "true")
		os.Setenv("DBURI", "file:custom/database.sqlite")
		os.Setenv("DOMAIN", "example.com")
		os.Setenv("ADMIN_PASSWORD", "custom_password")

		config := newConfig()

		// Test custom values
		if config.Server.Addr != "127.0.0.1" {
			t.Errorf("Expected custom IP_ADDR '127.0.0.1', got '%s'", config.Server.Addr)
		}
		if config.Server.Http != ":3000" {
			t.Errorf("Expected custom HTTP_PORT ':3000', got '%s'", config.Server.Http)
		}
		if config.Server.Https != ":3443" {
			t.Errorf("Expected custom HTTPS_PORT ':3443', got '%s'", config.Server.Https)
		}
		if config.OAuth.GithubAuthorizeURL != "https://github.com/login/oauth/authorize" {
			t.Errorf("Expected custom GITHUB_AUTHORIZE_URL, got '%s'", config.OAuth.GithubAuthorizeURL)
		}
		if config.OAuth.GithubTokenURL != "https://github.com/login/oauth/access_token" {
			t.Errorf("Expected custom GITHUB_TOKEN_URL, got '%s'", config.OAuth.GithubTokenURL)
		}
		if config.OAuth.RedirectURL != "http://localhost:3000/auth-callback" {
			t.Errorf("Expected custom REDIRECT_URL, got '%s'", config.OAuth.RedirectURL)
		}
		if config.OAuth.ClientID != "test_client_id" {
			t.Errorf("Expected custom CLIENT_ID 'test_client_id', got '%s'", config.OAuth.ClientID)
		}
		if config.OAuth.ClientSecret != "test_client_secret" {
			t.Errorf("Expected custom CLIENT_SECRET 'test_client_secret', got '%s'", config.OAuth.ClientSecret)
		}
		if config.Templates != "custom/templates/*.html" {
			t.Errorf("Expected custom TEMPLATES 'custom/templates/*.html', got '%s'", config.Templates)
		}
		if config.Production != "true" {
			t.Errorf("Expected custom PRODUCTION 'true', got '%s'", config.Production)
		}
		if config.DBURI != "file:custom/database.sqlite" {
			t.Errorf("Expected custom DBURI 'file:custom/database.sqlite', got '%s'", config.DBURI)
		}
		if config.Domain != "example.com" {
			t.Errorf("Expected custom DOMAIN 'example.com', got '%s'", config.Domain)
		}
		if config.AdminPass != "custom_password" {
			t.Errorf("Expected custom ADMIN_PASSWORD 'custom_password', got '%s'", config.AdminPass)
		}
	})

	t.Run("Mixed configuration", func(t *testing.T) {
		// Clear all environment variables
		for key := range originalVars {
			os.Unsetenv(key)
		}

		// Set only some environment variables
		os.Setenv("IP_ADDR", "192.168.1.1")
		os.Setenv("PRODUCTION", "true")
		os.Setenv("CLIENT_ID", "mixed_client_id")

		config := newConfig()

		// Test mixed values (some custom, some default)
		if config.Server.Addr != "192.168.1.1" {
			t.Errorf("Expected custom IP_ADDR '192.168.1.1', got '%s'", config.Server.Addr)
		}
		if config.Server.Http != ":8080" {
			t.Errorf("Expected default HTTP_PORT ':8080', got '%s'", config.Server.Http)
		}
		if config.Production != "true" {
			t.Errorf("Expected custom PRODUCTION 'true', got '%s'", config.Production)
		}
		if config.OAuth.ClientID != "mixed_client_id" {
			t.Errorf("Expected custom CLIENT_ID 'mixed_client_id', got '%s'", config.OAuth.ClientID)
		}
		if config.OAuth.ClientSecret != "" {
			t.Errorf("Expected default CLIENT_SECRET '', got '%s'", config.OAuth.ClientSecret)
		}
		if config.AdminPass != "12345" {
			t.Errorf("Expected default ADMIN_PASSWORD '12345', got '%s'", config.AdminPass)
		}
	})
}