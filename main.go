package main

import "flag"

func main() {
	addr := flag.String("addr", ":8080", "Default localhost:8080")
	saddr := flag.String("saddr", ":8443", "Default localhost:8443")
	domain := flag.String("domain", "dcandu.name", "Enter domain name like this domain.com")
	flag.Parse()

	a := App{}
	a.Initialize("database/database.sqlite", "templates/*.gohtml")
	a.Run(*domain, *addr, *saddr)
}
