package main

import (
	"encoding/json"
	"log"
	"os"
)

//Config is strcuct which holds necesary data such as server conf
//database, log, cert, oauth
type Config struct {
	Server struct {
		Addr  string `json:"addr"`
		SAddr string `json:"saddr"`
	} `json:"server"`
	Database struct {
		DBpath string `json:"dbpath"`
	} `json:"database"`
	Log struct {
		LogPath string `json:"log"`
	} `json:"log"`
	Template struct {
		TmPath string `json:"tmpath"`
	} `json:"template"`
	Cert struct {
		Domain string `json:"domain"`
	} `json:"cert"`
	OAuth struct {
		GithubAuthorizeURL string `json:"githubauthorizeurl"`
		GithubTokenURL     string `json:"githubtokenurl"`
		RedirectURL        string `json:"redirecturl"`
		ClientID           string `json:"clientid"`
		ClientSecret       string `json:"clientsecret"`
	} `json:"oauth"`
	Production string `json:"production,omitempty"`
}

//NewConfig create config structure
func NewConfig() *Config {
	return &Config{}
}

//ReadConfig try to read config(json)
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
