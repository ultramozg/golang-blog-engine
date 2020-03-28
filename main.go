package main

import (
	"flag"
	"log"

	"github.com/ultramozg/golang-blog-engine/app"
)

var gitCommit string

func printVersion() {
	log.Printf("Current build version: %s", gitCommit)
}

func main() {
	versionFlag := flag.Bool("v", false, "Print the current version and exit")
	flag.Parse()

	if *versionFlag {
		printVersion()
		return
	}

	conf := app.NewConfig()
	a := app.NewApp()
	a.Initialize(conf)
	a.Run()
}
