package main

func main() {
	a := App{}
	a.Initialize("database/database.sqlite", "templates/*.gohtml")
	a.Run(":8080")
}
