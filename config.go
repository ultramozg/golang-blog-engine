package main

import (
	"encoding/json"
	"log"
	"os"
)

type Config struct {
	Addr   string `json:"addr"`
	SAddr  string `json:"saddr"`
	Domain string `json:"domain"`
	DBpath string `json:"dbpath"`
	TmPath string `json:"tmpath"`
	Log    string `json:"log"`

	//OAuth config details
	GithubAuthorizeURL string `json:"githubauthorizeurl"`
	GithubTokenURL     string `json:"githubtokenurl"`
	RedirectURL        string `json:"redirecturl"`
	ClientID           string `json:"clientid"`
	ClientSecret       string `json:"clientsecret"`
}

func NewConfig() *Config {
	return &Config{}
}

func (c *Config) ReadConfig(path string) {
	file, err := os.Open(path)
	if err != nil {
		log.Fatal("Unable to read configuration file: ", err)
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(&c)
	if err != nil {
		log.Fatal("Invalid config format: ", err)
	}
}
