package main

func main() {
	conf := NewConfig()
	conf.ReadConfig("conf.d/conf.json")

	a := App{}
	a.Initialize(conf)
	a.Run()
}
