package main

import "github.com/ultramozg/golang-blog-engine/app"

func main() {
	conf := app.NewConfig()
	conf.ReadConfig("conf.d/conf.json")

	a := app.NewApp()
	a.Initialize(conf)
	a.Run()
}
