package main

import (
	"crypto/tls"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/acme/autocert"
	"log"
	"net/http"
	"strconv"
	"text/template"
	"time"
)

//temporary for testing need to change bcrypt
var (
	adminPass = "abcd"
)

type App struct {
	Router   http.Handler
	DB       *sql.DB
	Temp     *template.Template
	Sessions *SessionDB
	Log      Logging
	Addr     string
	SAddr    string
	Domain   string
}

func (a *App) Initialize(dbname, tmpath string) {
	var err error
	a.DB, err = sql.Open("sqlite3", dbname)
	if err != nil {
		log.Fatal("Error connecting to dabase", err)
	}

	a.InitializeRoutes()

	a.Temp = template.Must(template.ParseGlob(tmpath))
	a.Sessions = NewSessionDB()
	a.Log = NewLogging("log/access.log")
}

func (a *App) Run(domain, addr, saddr string) {
	a.Addr = addr
	a.SAddr = saddr
	a.Domain = domain

	//Get the cert
	cert := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(domain),
		Cache:      autocert.DirCache("cert"),
	}

	server := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		Addr:         a.SAddr,
		TLSConfig: &tls.Config{
			GetCertificate: cert.GetCertificate,
		},
		Handler: a.Router,
	}

	log.Println("Starting application with auto TLS support")
	log.Println("Listening on the addr", a.Addr)
	log.Println("Listening TLS on the addr", a.SAddr)

	//Launch standart http and https protocols
	go http.ListenAndServe(a.Addr, cert.HTTPHandler(a.redirectTLSMiddleware(a.Router)))
	log.Fatal(server.ListenAndServeTLS("", ""))
}

func (a *App) InitializeRoutes() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", a.getPosts)
	mux.HandleFunc("/login", a.login)
	mux.HandleFunc("/logout", a.logout)
	mux.HandleFunc("/post", a.getPost)
	mux.HandleFunc("/update", a.updatePost)
	mux.HandleFunc("/create", a.createPost)
	mux.HandleFunc("/delete", a.deletePost)
	mux.HandleFunc("/about", a.about)
	mux.HandleFunc("/links", a.links)

	//Register Fileserver
	fs := http.FileServer(http.Dir("public/"))
	mux.Handle("/public/", http.StripPrefix("/public/", cacheControlMiddleware(fs)))

	a.Router = gzipMiddleware(setHeaderMiddleware(a.Log.logMiddleware(mux)))
}

func (a *App) getPost(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		http.Error(w, "Invalid Blog id", http.StatusBadRequest)
		return
	}

	p := Post{Id: id}

	switch r.Method {
	case http.MethodGet:
		if err := p.getPost(a.DB); err != nil {
			switch err {
			case sql.ErrNoRows:
				http.Error(w, "Not Found", http.StatusNotFound)
			default:
				http.Error(w, "Internal error", http.StatusInternalServerError)
			}
			return
		}

		data := struct {
			Post     Post
			LoggedIn bool
		}{
			p,
			a.Sessions.isAdmin(r),
		}
		a.Temp.ExecuteTemplate(w, "post.gohtml", data)
	case http.MethodHead:
		w.WriteHeader(http.StatusOK)
		return

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
}

func (a *App) getPosts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		var page int
		var err error
		if r.FormValue("p") == "" {
			page = 0
		} else {
			page, err = strconv.Atoi(r.FormValue("p"))
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		if page <= 0 {
			page = 0
		}

		posts, err := getPosts(a.DB, 8, page*8)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data := struct {
			Posts    []Post
			LoggedIn bool
			PrevPage int
			NextPage int
		}{
			posts,
			a.Sessions.isAdmin(r),
			add(page, -1),
			add(page, +1),
		}
		a.Temp.ExecuteTemplate(w, "posts.gohtml", data)

	case http.MethodHead:
		w.WriteHeader(http.StatusOK)
		return

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
}

func (a *App) createPost(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !a.Sessions.isAdmin(r) {
			http.Error(w, "Not Authorized", http.StatusUnauthorized)
			return
		}
		a.Temp.ExecuteTemplate(w, "create.gohtml", a.Sessions.isAdmin(r))

	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		title := r.FormValue("title")
		body := r.FormValue("body")
		if title == "" || body == "" {
			http.Error(w, "Bad Request", 400)
			return
		}

		p := Post{Title: title, Body: body, Date: time.Now().Format("Mon Jan _2 15:04:05 2006")}
		if err := p.createPost(a.DB); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
}

func (a *App) updatePost(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		id, err := strconv.Atoi(r.FormValue("id"))
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

		p := Post{Id: id}
		if err := p.getPost(a.DB); err != nil {
			switch err {
			case sql.ErrNoRows:
				http.Error(w, "Post not found", http.StatusNotFound)
			default:
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		data := struct {
			Post     Post
			LoggedIn bool
		}{
			p,
			a.Sessions.isAdmin(r),
		}
		a.Temp.ExecuteTemplate(w, "update.gohtml", data)

	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		id, err := strconv.Atoi(r.FormValue("id"))
		if err != nil {
			http.Error(w, "Invalid id value", http.StatusBadRequest)
			return
		}
		title := r.FormValue("title")
		body := r.FormValue("body")
		if title == "" || body == "" {
			http.Error(w, "Empty Fields", http.StatusBadRequest)
			return
		}

		p := Post{Id: id, Title: title, Body: body, Date: time.Now().Format("Mon Jan _2 15:04:05 2006")}
		if err := p.updatePost(a.DB); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
}

func (a *App) deletePost(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		id, err := strconv.Atoi(r.FormValue("id"))
		if err != nil {
			http.Error(w, "Invalid Id", http.StatusBadRequest)
			return
		}

		p := Post{Id: id}
		if err := p.deletePost(a.DB); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
}

func (a *App) about(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.Temp.ExecuteTemplate(w, "about.gohtml", a.Sessions.isAdmin(r))
	case http.MethodHead:
		w.WriteHeader(http.StatusOK)
		return
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
}

func (a *App) links(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.Temp.ExecuteTemplate(w, "links.gohtml", a.Sessions.isAdmin(r))
	case http.MethodHead:
		w.WriteHeader(http.StatusOK)
		return
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
}

func (a *App) login(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.Temp.ExecuteTemplate(w, "login.gohtml", a.Sessions.isAdmin(r))

	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		login := r.FormValue("login")
		pass := r.FormValue("password")

		if login == "" || pass == "" {
			http.Error(w, "Invalid Input data", http.StatusBadRequest)
			return
		}
		/*
			check credentials first
			for the test purporses admin pass will store in the
			struct
		*/
		if login == "admin" && pass == adminPass {
			c := a.Sessions.createSession(ADMIN)
			http.SetCookie(w, c)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		} else {
			http.Error(w, "Invalid login credentials", http.StatusUnauthorized)
			return
		}

	case http.MethodHead:
		w.WriteHeader(http.StatusOK)
		return

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
}

func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if a.Sessions.isAdmin(r) {
			c, _ := r.Cookie("session")
			a.Sessions.delSession(c.Value)
			http.SetCookie(w, c)
			http.Redirect(w, r, "/", http.StatusSeeOther)
		} else {
			http.Error(w, "Not Authorized", http.StatusUnauthorized)
			return
		}
	case http.MethodHead:
		w.WriteHeader(http.StatusOK)
		return
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
}

func add(i, j int) int {
	if i+j <= 0 {
		return 0
	}
	return i + j
}
