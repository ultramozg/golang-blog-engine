package main

import "github.com/ultramozg/golang-blog-engine/app"

func main() {
	conf := app.NewConfig()
	a := app.NewApp()
	a.Initialize(conf)
	a.Run()
}
