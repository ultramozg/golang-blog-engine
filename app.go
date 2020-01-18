package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"text/template"
	"time"

	"github.com/google/go-github/github"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
)

const (
	PostsPerPage = 8
)

/*
App The main app structure which holds all necessary Data within
conf := NewConfig()
conf.ReadConfig(<PATH>)

app := App{}
a.Initialize(conf)
a.Run()
*/
type App struct {
	Router   http.Handler
	DB       *sql.DB
	Temp     *template.Template
	Sessions *SessionDB
	Log      Logging
	Config   *Config
	stop     chan os.Signal
	OAuth    *oauth2.Config
}

//Initialize Is using to initialize the app(connect to DB, initialize routes,logs, sessions and etc.
func (a *App) Initialize(c *Config) {
	var err error
	a.Config = c

	a.DB, err = sql.Open("sqlite3", a.Config.Database.DBpath)
	if err != nil {
		log.Fatal("Error connecting to dabase", err)
	}

	migrateDatabase(a.DB)

	u := &User{userName: "admin", userType: ADMIN}

	//check if Admin account exists if not create one
	if !u.isUserExist(a.DB) {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter Admin password: ")
		pass, _ := reader.ReadString('\n')
		pass = strings.Replace(pass, "\n", "", -1)

		if ok, hash := HashPassword(pass); ok {
			err = u.createUser(a.DB, hash)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	a.initializeRoutes()

	a.Temp = template.Must(template.ParseGlob(a.Config.Template.TmPath))
	a.Sessions = NewSessionDB()
	a.Log = NewLogging(a.Config.Log.LogPath)

	//Setting up OAuth authentication via github
	a.OAuth = &oauth2.Config{
		ClientID:     a.Config.OAuth.ClientID,
		ClientSecret: a.Config.OAuth.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  a.Config.OAuth.GithubAuthorizeURL,
			TokenURL: a.Config.OAuth.GithubTokenURL,
		},
		RedirectURL: a.Config.OAuth.RedirectURL,
		Scopes:      []string{"read:user"},
	}
	//======END OAUTH CONFIGURATION======

	//setting up signal capturing
	a.stop = make(chan os.Signal, 1)
	signal.Notify(a.stop, os.Interrupt)
	signal.Notify(a.stop, syscall.SIGTERM)
}

//Run is using to launch and serve app web requests
func (a *App) Run() {
	//Get the cert
	cert := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(a.Config.Cert.Domain),
		Cache:      autocert.DirCache("cert"),
	}

	secureServer := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		Addr:         a.Config.Server.SAddr,
		TLSConfig: &tls.Config{
			GetCertificate: cert.GetCertificate,
		},
		Handler: a.Router,
	}

	httpHandler := a.Router
	//if this is a test environment disable 301 redirect to https
	if a.Config.Production == "true" {
		httpHandler = a.redirectTLSMiddleware(httpHandler)
	}
	httpHandler = cert.HTTPHandler(httpHandler)

	httpServer := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		Addr:         a.Config.Server.Addr,
		Handler:      httpHandler,
	}

	log.Println("Starting application with auto TLS support")
	log.Println("Listening on the addr", a.Config.Server.Addr)
	log.Println("Listening TLS on the addr", a.Config.Server.SAddr)

	//Launch standart http, to fetch cert Let's Encrypt with 301 -> https
	go func() {
		if err := httpServer.ListenAndServe(); err != nil {
			log.Fatal("Unable to listen on http port: ", err)
		}
	}()

	//Launch https
	go func() {
		if err := secureServer.ListenAndServeTLS("", ""); err != nil {
			log.Fatal("Unable to listen on https port: ", err)
		}
	}()

	//Listen to catch sigint signal to gracefully stop the app
	<-a.stop
	log.Println("Caught SIGINT or SIGTERM stopping the app")

	//close all connections
	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
	if err := secureServer.Shutdown(ctx); err != nil {
		log.Println("Unable to shutdown http server")
	}
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Println("Unable to shutdown http server")
	}
	a.DB.Close()
	os.Exit(0)
}

func (a *App) initializeRoutes() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", a.root)
	mux.HandleFunc("/page", a.getPage)
	mux.HandleFunc("/login", a.login)
	mux.HandleFunc("/logout", a.logout)
	mux.HandleFunc("/post", a.getPost)
	mux.HandleFunc("/update", a.updatePost)
	mux.HandleFunc("/create", a.createPost)
	mux.HandleFunc("/delete", a.deletePost)
	mux.HandleFunc("/about", a.about)
	mux.HandleFunc("/links", a.links)
	mux.HandleFunc("/courses", a.courses)
	mux.HandleFunc("/auth-callback", a.oauth)
	mux.HandleFunc("/create-comment", a.createComment)
	mux.HandleFunc("/delete-comment", a.deleteComment)

	//Register Fileserver
	fs := http.FileServer(http.Dir("public/"))
	mux.Handle("/public/", http.StripPrefix("/public/", cacheControlMiddleware(fs)))

	a.Router = gzipMiddleware(setHeaderMiddleware(a.Log.logMiddleware(mux)))
}

func (a *App) root(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Opps something did wrong", http.StatusNotFound)
		return
	}
	http.Redirect(w, r, "/page?p=0", http.StatusFound)
	return
}

func (a *App) getPost(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		http.Error(w, "Invalid Blog id", http.StatusBadRequest)
		return
	}

	p := Post{ID: id}
	if err = p.getPost(a.DB); err != nil {
		switch err {
		case sql.ErrNoRows:
			http.Error(w, "Not Found", http.StatusNotFound)
		default:
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
		return
	}

	switch r.Method {
	case http.MethodGet:

		comms, err := getComments(a.DB, id)
		if err != nil {
			log.Println("Grab comment error: ", err.Error())
		}

		data := struct {
			Post        Post
			Comms       []Comment
			LogAsAdmin  bool
			LogAsUser   bool
			AuthURL     string
			ClientID    string
			RedirectURL string
		}{
			p,
			comms,
			a.Sessions.isAdmin(r),
			a.Sessions.isLoggedin(r),
			a.Config.OAuth.GithubAuthorizeURL,
			a.Config.OAuth.ClientID,
			a.Config.OAuth.RedirectURL,
		}
		err = a.Temp.ExecuteTemplate(w, "post.gohtml", data)
		if err != nil {
			log.Println(err.Error())
		}
	case http.MethodHead:
		w.WriteHeader(http.StatusOK)
		return

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
}

func (a *App) getPage(w http.ResponseWriter, r *http.Request) {
	var page int
	var err error
	page, err = strconv.Atoi(r.FormValue("p"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	posts, err := getPosts(a.DB, PostsPerPage, page*PostsPerPage)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(posts) == 0 {
		http.Error(w, "No data", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		data := struct {
			Posts      []Post
			LoggedIn   bool
			IsNextPage bool
			PrevPage   int
			NextPage   int
		}{
			posts,
			a.Sessions.isAdmin(r),
			isNextPage(page+1, countPosts(a.DB)),
			absolute(page - 1),
			absolute(page + 1),
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

		p := Post{ID: id}
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

		p := Post{ID: id, Title: title, Body: body, Date: time.Now().Format("Mon Jan _2 15:04:05 2006")}
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

		p := Post{ID: id}
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

func (a *App) courses(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.Temp.ExecuteTemplate(w, "courses.gohtml", a.Sessions.isAdmin(r))
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
		u := &User{userName: login}

		if u.checkCredentials(a.DB, pass) && u.isAdmin(a.DB) {
			c := a.Sessions.createSession(User{userType: ADMIN, userName: "admin"})
			http.SetCookie(w, c)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, "Invalid login credentials", http.StatusUnauthorized)
		return

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

func (a *App) oauth(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		token, err := a.OAuth.Exchange(oauth2.NoContext, r.URL.Query().Get("code"))
		if err != nil {
			log.Println(w, "there was an issue getting your token: ", err.Error())
			return
		}
		if !token.Valid() {
			log.Println(w, "retreived invalid token")
			return
		}

		client := github.NewClient(a.OAuth.Client(oauth2.NoContext, token))
		user, _, err := client.Users.Get(context.Background(), "")
		if err != nil {
			log.Println(w, "error getting name")
			return
		}

		c := a.Sessions.createSession(User{userType: GITHUB, userName: *(user.Login)})
		http.SetCookie(w, c)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		log.Println("You have loged int as github user :", *(user.Login))
		return

	case http.MethodHead:
		w.WriteHeader(http.StatusOK)
		return
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
}

func (a *App) createComment(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		if !(a.Sessions.isAdmin(r) || a.Sessions.isLoggedin(r)) {
			http.Error(w, "Not Authorized", http.StatusUnauthorized)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		id, err := strconv.Atoi(r.FormValue("id"))
		if err != nil {
			http.Error(w, "Invalid Id", http.StatusBadRequest)
			return
		}

		name := r.FormValue("name")
		comment := r.FormValue("comment")
		if name == "" || comment == "" {
			http.Error(w, "Bad Request", 400)
			return
		}

		p := Comment{PostID: id, Name: name, Date: time.Now().Format("Mon Jan _2 15:04:05 2006"), Data: comment}
		if err := p.createComment(a.DB); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
}

func (a *App) deleteComment(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !a.Sessions.isAdmin(r) {
			http.Error(w, "Not Authorized", http.StatusUnauthorized)
			return
		}

		id, err := strconv.Atoi(r.FormValue("id"))
		if err != nil {
			http.Error(w, "Invalid Id", http.StatusBadRequest)
			return
		}

		c := Comment{CommentID: id}
		if err := c.deleteComment(a.DB); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
}

func absolute(i int) int {
	if i <= 0 {
		return 0
	}
	return i
}

func isNextPage(nextPage, totalPosts int) bool {
	return (totalPosts / PostsPerPage) > nextPage
}

func HashPassword(password string) (bool, string) {

	var hashedPassword, err = bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Unable to generate hashed password")
		return false, password
	}

	return true, string(hashedPassword)
}
