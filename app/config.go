package app

import (
	"os"
)

type Server struct {
	Addr  string
	Http  string
	Https string
}

type OAuth struct {
	GithubAuthorizeURL string
	GithubTokenURL     string
	RedirectURL        string
	ClientID           string
	ClientSecret       string
}

//Config is strcuct which holds necesary data such as server conf
//database, log, cert, oauth
type Config struct {
	Server     Server
	OAuth      OAuth
	Production string
	DBURI      string
	Domain     string
}

//NewConfig create config structure
func NewConfig() *Config {
	return &Config{
		Server: Server{
			Addr:  getEnv("IP_ADDR", "0.0.0.0"),
			Http:  getEnv("HTTP_PORT", ":8080"),
			Https: getEnv("HTTPS_PORT", "8443"),
		},
		OAuth: OAuth{
			GithubAuthorizeURL: getEnv("GITHUB_AUTHORIZE_URL", ""),
			GithubTokenURL:     getEnv("GITHUB_TOKEN_URL", ""),
			RedirectURL:        getEnv("REDIRECT_URL", ""),
			ClientID:           getEnv("CLIENT_ID", ""),
			ClientSecret:       getEnv("CLIENT_SECRET", ""),
		},
		Production: getEnv("PRODUCTION", "false"),
		DBURI:      getEnv("DBURI", "file:database/database.sqlite"),
		Domain:     getEnv("DOMAIN", ""),
	}
}

// Simple helper function to read an environment or return a default value
func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultVal
}
