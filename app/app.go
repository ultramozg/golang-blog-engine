package app

import (
	"context"
	"crypto/tls"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"text/template"
	"time"

	"github.com/google/go-github/github"
	"github.com/ultramozg/golang-blog-engine/middleware"
	"github.com/ultramozg/golang-blog-engine/model"
	"github.com/ultramozg/golang-blog-engine/services"
	"github.com/ultramozg/golang-blog-engine/session"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	_ "modernc.org/sqlite"
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
	Router      http.Handler
	DB          *sql.DB
	Temp        *template.Template
	Sessions    *session.SessionDB
	Config      *Config
	stop        chan os.Signal
	OAuth       *oauth2.Config
	Courses     model.Infos
	Links       model.Infos
	SlugService services.SlugService
}

// NewApp return App struct
func NewApp() App {
	return App{}
}

// Initialize Is using to initialize the app(connect to DB, initialize routes,logs, sessions and etc.
func (a *App) Initialize() {
	var err error
	a.Config = newConfig()

	a.DB, err = sql.Open("sqlite", a.Config.DBURI)
	log.Println("Trying connect to DB:", a.Config.DBURI)
	if err != nil {
		log.Fatal("Error connecting to dabase", err)
	}

	model.MigrateDatabase(a.DB)

	u := &model.User{Name: "admin", Type: session.ADMIN}

	//check if Admin account exists if not create one
	if !u.IsUserExist(a.DB) {
		if ok, hash := HashPassword(a.Config.AdminPass); ok {
			err = u.CreateUser(a.DB, hash)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	//Convert Yaml data to struct
	a.Courses, err = model.ConverYamlToStruct("data/courses.yml")
	if err != nil {
		log.Println(err)
	}
	a.Links, err = model.ConverYamlToStruct("data/links.yml")
	if err != nil {
		log.Println(err)
	}

	a.initializeRoutes()

	a.Temp = template.Must(template.ParseGlob(a.Config.Templates))
	a.Sessions = session.NewSessionDB()
	a.SlugService = services.NewSlugService(a.DB)

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

// Run is using to launch and serve app web requests
func (a *App) Run() {
	//Get the cert
	cert := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(a.Config.Domain),
		Cache:      autocert.DirCache("cert"),
	}

	secureServer := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		Addr:         a.Config.Server.Addr + a.Config.Server.Https,
		TLSConfig: &tls.Config{
			GetCertificate: cert.GetCertificate,
		},
		Handler: a.Router,
	}

	httpHandler := a.Router
	if a.Config.Production == "true" {
		httpHandler = middleware.RedirectTLSMiddleware(httpHandler)
	}
	httpHandler = cert.HTTPHandler(httpHandler)

	httpServer := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		Addr:         a.Config.Server.Addr + a.Config.Server.Http,
		Handler:      httpHandler,
	}

	log.Println("Starting application with auto TLS support")
	log.Println("Listening on the addr", a.Config.Server.Addr+a.Config.Server.Http)
	log.Println("Listening TLS on the addr", a.Config.Server.Addr+a.Config.Server.Https)

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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
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
	mux.HandleFunc("/p/", a.getPostBySlug)
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
	mux.Handle("/public/", http.StripPrefix("/public/", middleware.CacheControlMiddleware(fs)))

	a.Router = middleware.LogMiddleware(a.securityMiddleware(middleware.PostRedirectMiddleware(a.DB)(middleware.GzipMiddleware(middleware.SetHeaderMiddleware(mux)))))
}

func (a *App) root(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Opps something did wrong", http.StatusNotFound)
		return
	}
	http.Redirect(w, r, "/page?p=0", http.StatusFound)
}

func (a *App) getPost(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		http.Error(w, "Invalid Blog id", http.StatusBadRequest)
		return
	}

	p := model.Post{ID: id}
	if err = p.GetPost(a.DB); err != nil {
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

		comms, err := model.GetComments(a.DB, id)
		if err != nil {
			log.Println("Grab comment error: ", err.Error())
		}

		data := struct {
			Post        model.Post
			Comms       []model.Comment
			LogAsAdmin  bool
			LogAsUser   bool
			AuthURL     string
			ClientID    string
			RedirectURL string
		}{
			p,
			comms,
			a.Sessions.IsAdmin(r),
			a.Sessions.IsLoggedin(r),
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

func (a *App) getPostBySlug(w http.ResponseWriter, r *http.Request) {
	// Extract slug from URL path /p/slug-here
	slug := strings.TrimPrefix(r.URL.Path, "/p/")
	if slug == "" {
		http.Error(w, "Invalid post slug", http.StatusBadRequest)
		return
	}

	p := model.Post{Slug: slug}
	if err := p.GetPostBySlug(a.DB); err != nil {
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
		comms, err := model.GetComments(a.DB, p.ID)
		if err != nil {
			log.Println("Grab comment error: ", err.Error())
		}

		data := struct {
			Post        model.Post
			Comms       []model.Comment
			LogAsAdmin  bool
			LogAsUser   bool
			AuthURL     string
			ClientID    string
			RedirectURL string
		}{
			p,
			comms,
			a.Sessions.IsAdmin(r),
			a.Sessions.IsLoggedin(r),
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
	posts, err := model.GetPosts(a.DB, PostsPerPage, page*PostsPerPage)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	switch r.Method {
	case http.MethodGet:
		data := struct {
			Posts      []model.Post
			LoggedIn   bool
			IsNextPage bool
			PrevPage   int
			NextPage   int
		}{
			posts,
			a.Sessions.IsAdmin(r),
			isNextPage(page, model.CountPosts(a.DB)),
			absolute(page - 1),
			absolute(page + 1),
		}
		if err := a.Temp.ExecuteTemplate(w, "posts.gohtml", data); err != nil {
			log.Println("Template execution error:", err)
		}

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
		if err := a.Temp.ExecuteTemplate(w, "create.gohtml", a.Sessions.IsAdmin(r)); err != nil {
			log.Println("Template execution error:", err)
		}

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

		p := model.Post{Title: title, Body: body, Date: time.Now().Format("Mon Jan _2 15:04:05 2006")}

		// Generate slug for the new post
		slug := a.SlugService.GenerateSlug(title)
		p.Slug = a.SlugService.EnsureUniqueSlug(slug, 0) // 0 for new post

		// Create post with slug
		result, err := a.DB.Exec(`insert into posts (title, body, datepost, slug, created_at, updated_at) values ($1, $2, $3, $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, p.Title, p.Body, p.Date, p.Slug)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Get the ID of the newly created post
		id, err := result.LastInsertId()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		p.ID = int(id)
		http.Redirect(w, r, "/", http.StatusSeeOther)

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
}

func (a *App) updatePost(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Support both slug and id parameters for backward compatibility
		slug := r.FormValue("slug")
		idStr := r.FormValue("id")

		var p model.Post
		var err error

		if slug != "" {
			// Use slug to get post
			p = model.Post{Slug: slug}
			err = p.GetPostBySlug(a.DB)
		} else if idStr != "" {
			// Fallback to ID for backward compatibility
			id, parseErr := strconv.Atoi(idStr)
			if parseErr != nil {
				http.Error(w, "Invalid ID", http.StatusBadRequest)
				return
			}
			p = model.Post{ID: id}
			err = p.GetPost(a.DB)
		} else {
			http.Error(w, "Missing post identifier", http.StatusBadRequest)
			return
		}

		if err != nil {
			switch err {
			case sql.ErrNoRows:
				http.Error(w, "Post not found", http.StatusNotFound)
			default:
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		data := struct {
			Post       model.Post
			LogAsAdmin bool
		}{
			p,
			a.Sessions.IsAdmin(r),
		}
		err = a.Temp.ExecuteTemplate(w, "update.gohtml", data)
		log.Println(err)

	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// Support both slug and id parameters for backward compatibility
		slug := r.FormValue("slug")
		idStr := r.FormValue("id")

		var p model.Post
		var err error

		if slug != "" {
			// Use slug to get post
			p = model.Post{Slug: slug}
			err = p.GetPostBySlug(a.DB)
		} else if idStr != "" {
			// Fallback to ID for backward compatibility
			id, parseErr := strconv.Atoi(idStr)
			if parseErr != nil {
				http.Error(w, "Invalid id value", http.StatusBadRequest)
				return
			}
			p = model.Post{ID: id}
			err = p.GetPost(a.DB)
		} else {
			http.Error(w, "Missing post identifier", http.StatusBadRequest)
			return
		}

		if err != nil {
			http.Error(w, "Post not found", http.StatusNotFound)
			return
		}

		title := r.FormValue("title")
		body := r.FormValue("body")
		if title == "" || body == "" {
			http.Error(w, "Empty Fields", http.StatusBadRequest)
			return
		}

		// Get current post data to check if title changed
		currentTitle := p.Title

		p.Title = title
		p.Body = body
		p.Date = time.Now().Format("Mon Jan _2 15:04:05 2006")

		// If title changed, regenerate slug
		if currentTitle != title {
			newSlug := a.SlugService.GenerateSlug(title)
			p.Slug = a.SlugService.EnsureUniqueSlug(newSlug, p.ID)

			// Update post with new slug
			_, err = a.DB.Exec(`update posts set title = $1, body = $2, datepost = $3, slug = $4, updated_at = CURRENT_TIMESTAMP where id = $5`, p.Title, p.Body, p.Date, p.Slug, p.ID)
		} else {
			// Update post without changing slug
			_, err = a.DB.Exec(`update posts set title = $1, body = $2, datepost = $3, updated_at = CURRENT_TIMESTAMP where id = $4`, p.Title, p.Body, p.Date, p.ID)
		}

		if err != nil {
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
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// Support both slug and id parameters for backward compatibility
		slug := r.FormValue("slug")
		idStr := r.FormValue("id")

		var p model.Post
		var err error

		if slug != "" {
			// Use slug to get post
			p = model.Post{Slug: slug}
			err = p.GetPostBySlug(a.DB)
		} else if idStr != "" {
			// Fallback to ID for backward compatibility
			id, parseErr := strconv.Atoi(idStr)
			if parseErr != nil {
				http.Error(w, "Invalid Id", http.StatusBadRequest)
				return
			}
			p = model.Post{ID: id}
			err = p.GetPost(a.DB)
		} else {
			http.Error(w, "Missing post identifier", http.StatusBadRequest)
			return
		}

		if err != nil {
			switch err {
			case sql.ErrNoRows:
				http.Error(w, "Not Found", http.StatusNotFound)
			default:
				http.Error(w, "Internal error", http.StatusInternalServerError)
			}
			return
		}
		if err := p.DeletePost(a.DB); err != nil {
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
		if err := a.Temp.ExecuteTemplate(w, "about.gohtml", a.Sessions.IsAdmin(r)); err != nil {
			log.Println("Template execution error:", err)
		}
		return
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
		data := struct {
			LogAsAdmin bool
			Links      []model.Info
		}{
			a.Sessions.IsAdmin(r),
			a.Links.List,
		}
		if err := a.Temp.ExecuteTemplate(w, "links.gohtml", data); err != nil {
			log.Println("Template execution error:", err)
		}
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
		data := struct {
			LogAsAdmin bool
			Courses    []model.Info
		}{
			a.Sessions.IsAdmin(r),
			a.Courses.List,
		}
		if err := a.Temp.ExecuteTemplate(w, "courses.gohtml", data); err != nil {
			log.Println("Template execution error:", err)
		}
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
		if err := a.Temp.ExecuteTemplate(w, "login.gohtml", a.Sessions.IsAdmin(r)); err != nil {
			log.Println("Template execution error:", err)
		}

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
		u := &model.User{Name: login}

		if u.CheckCredentials(a.DB, pass) && u.IsAdmin(a.DB) {
			c := a.Sessions.CreateSession(model.User{Type: session.ADMIN, Name: "admin"})
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
		if a.Sessions.IsAdmin(r) {
			c, _ := r.Cookie("session")
			a.Sessions.DelSession(c.Value)
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
		token, err := a.OAuth.Exchange(context.Background(), r.URL.Query().Get("code"))
		if err != nil {
			log.Println(w, "there was an issue getting your token: ", err.Error())
			return
		}
		if !token.Valid() {
			log.Println(w, "retreived invalid token")
			return
		}

		client := github.NewClient(a.OAuth.Client(context.Background(), token))
		user, _, err := client.Users.Get(context.Background(), "")
		if err != nil {
			log.Println(w, "error getting name")
			return
		}

		c := a.Sessions.CreateSession(model.User{Type: session.GITHUB, Name: *(user.Login)})
		http.SetCookie(w, c)
		//http.Redirect(w, r, "/", http.StatusSeeOther)
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		log.Println("You have logged in as github user :", *(user.Login))
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
		if !(a.Sessions.IsLoggedin(r)) {
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

		p := model.Comment{PostID: id, Name: name, Date: time.Now().Format("Mon Jan _2 15:04:05 2006"), Data: comment}
		if err := p.CreateComment(a.DB); err != nil {
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
		if !a.Sessions.IsAdmin(r) {
			http.Error(w, "Not Authorized", http.StatusUnauthorized)
			return
		}

		id, err := strconv.Atoi(r.FormValue("id"))
		if err != nil {
			http.Error(w, "Invalid Id", http.StatusBadRequest)
			return
		}

		c := model.Comment{CommentID: id}
		if err := c.DeleteComment(a.DB); err != nil {
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

func (app *App) securityMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if match, _ := regexp.MatchString("/(create|delete)-comment", r.URL.RequestURI()); match {
			if !app.Sessions.IsLoggedin(r) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		} else if match, _ := regexp.MatchString("/(delete|update|create)", r.URL.RequestURI()); match {
			if !app.Sessions.IsAdmin(r) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
		h.ServeHTTP(w, r)
	})
}
