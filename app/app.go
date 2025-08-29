package app

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
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
	SlugService services.SlugService
	FileService services.FileService
	SEOService  services.SEOService
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

	// Create template with custom functions
	funcMap := template.FuncMap{
		"processFileReferences": a.processFileReferences,
		"extractExcerpt":        a.extractExcerpt,
	}
	a.Temp = template.Must(template.New("").Funcs(funcMap).ParseGlob(a.Config.Templates))
	a.Sessions = session.NewSessionDB()
	a.SlugService = services.NewSlugService(a.DB)
	a.FileService = services.NewFileService(a.DB, "uploads", 10*1024*1024) // 10MB max file size
	// Use domain from config or default to localhost for development
	domain := a.Config.Domain
	if domain == "" {
		domain = "http://localhost" + a.Config.Server.Http
	} else {
		// Ensure domain has proper protocol
		if !strings.HasPrefix(domain, "http://") && !strings.HasPrefix(domain, "https://") {
			if a.Config.Production == "true" {
				domain = "https://" + domain
			} else {
				domain = "http://" + domain
			}
		} else if strings.HasPrefix(domain, "http://") && a.Config.Production == "true" {
			// Convert http to https in production
			domain = strings.Replace(domain, "http://", "https://", 1)
		}
	}
	a.SEOService = services.NewSEOService(a.DB, domain)

	// Ensure upload directories exist
	if err := a.FileService.EnsureUploadDirectories(); err != nil {
		log.Printf("Warning: Failed to create upload directories: %v", err)
	}

	a.initializeRoutes()

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
			MinVersion:     tls.VersionTLS12,
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
	if err := a.DB.Close(); err != nil {
		log.Printf("Error closing database: %v", err)
	}
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
	mux.HandleFunc("/auth-callback", a.oauth)
	mux.HandleFunc("/create-comment", a.createComment)
	mux.HandleFunc("/delete-comment", a.deleteComment)
	mux.HandleFunc("/upload-file", a.uploadFile)
	mux.HandleFunc("/files/", a.serveFile)
	mux.HandleFunc("/api/files", a.listFiles)
	mux.HandleFunc("/api/files/alt-text", a.updateFileAltText)
	mux.HandleFunc("/sitemap.xml", a.serveSitemap)
	mux.HandleFunc("/robots.txt", a.serveRobotsTxt)

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
		// Set canonical URL header
		canonicalURL := a.SEOService.GetCanonicalURL(&p)
		w.Header().Set("Link", fmt.Sprintf("<%s>; rel=\"canonical\"", canonicalURL))

		comms, err := model.GetComments(a.DB, id)
		if err != nil {
			log.Println("Grab comment error: ", err.Error())
		}

		// Generate SEO data
		metaTags := a.SEOService.GenerateMetaTags(&p)
		structuredData := a.SEOService.GenerateStructuredData(&p)
		openGraphTags := a.SEOService.GenerateOpenGraphTags(&p)

		data := struct {
			Post           model.Post
			Comms          []model.Comment
			LogAsAdmin     bool
			LogAsUser      bool
			AuthURL        string
			ClientID       string
			RedirectURL    string
			MetaTags       map[string]string
			StructuredData template.HTML
			OpenGraphTags  map[string]string
			CanonicalURL   string
		}{
			p,
			comms,
			a.Sessions.IsAdmin(r),
			a.Sessions.IsLoggedin(r),
			a.Config.OAuth.GithubAuthorizeURL,
			a.Config.OAuth.ClientID,
			a.Config.OAuth.RedirectURL,
			metaTags,
			template.HTML(structuredData),
			openGraphTags,
			canonicalURL,
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
		// Set canonical URL header
		canonicalURL := a.SEOService.GetCanonicalURL(&p)
		w.Header().Set("Link", fmt.Sprintf("<%s>; rel=\"canonical\"", canonicalURL))

		comms, err := model.GetComments(a.DB, p.ID)
		if err != nil {
			log.Println("Grab comment error: ", err.Error())
		}

		// Generate SEO data
		metaTags := a.SEOService.GenerateMetaTags(&p)
		structuredData := a.SEOService.GenerateStructuredData(&p)
		openGraphTags := a.SEOService.GenerateOpenGraphTags(&p)

		data := struct {
			Post           model.Post
			Comms          []model.Comment
			LogAsAdmin     bool
			LogAsUser      bool
			AuthURL        string
			ClientID       string
			RedirectURL    string
			MetaTags       map[string]string
			StructuredData template.HTML
			OpenGraphTags  map[string]string
			CanonicalURL   string
		}{
			p,
			comms,
			a.Sessions.IsAdmin(r),
			a.Sessions.IsLoggedin(r),
			a.Config.OAuth.GithubAuthorizeURL,
			a.Config.OAuth.ClientID,
			a.Config.OAuth.RedirectURL,
			metaTags,
			template.HTML(structuredData),
			openGraphTags,
			canonicalURL,
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

		// Generate SEO fields if not provided
		p.GenerateDefaultSEOFields()
		if err := p.ValidateAndSanitizeSEOFields(); err != nil {
			http.Error(w, "Invalid SEO data: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Create post with slug and SEO fields
		result, err := a.DB.Exec(`insert into posts (title, body, datepost, slug, meta_description, keywords, created_at, updated_at) values ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, p.Title, p.Body, p.Date, p.Slug, p.MetaDescription, p.Keywords)
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

		// Generate SEO fields if not provided
		p.GenerateDefaultSEOFields()
		if err := p.ValidateAndSanitizeSEOFields(); err != nil {
			http.Error(w, "Invalid SEO data: "+err.Error(), http.StatusBadRequest)
			return
		}

		// If title changed, regenerate slug
		if currentTitle != title {
			newSlug := a.SlugService.GenerateSlug(title)
			p.Slug = a.SlugService.EnsureUniqueSlug(newSlug, p.ID)

			// Update post with new slug and SEO fields
			_, err = a.DB.Exec(`update posts set title = $1, body = $2, datepost = $3, slug = $4, meta_description = $5, keywords = $6, updated_at = CURRENT_TIMESTAMP where id = $7`, p.Title, p.Body, p.Date, p.Slug, p.MetaDescription, p.Keywords, p.ID)
		} else {
			// Update post without changing slug but with SEO fields
			_, err = a.DB.Exec(`update posts set title = $1, body = $2, datepost = $3, meta_description = $4, keywords = $5, updated_at = CURRENT_TIMESTAMP where id = $6`, p.Title, p.Body, p.Date, p.MetaDescription, p.Keywords, p.ID)
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

func (a *App) uploadFile(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		// Parse multipart form with 32MB max memory
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			http.Error(w, "Failed to parse multipart form", http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Failed to get file from form", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Upload file using file service
		fileRecord, err := a.FileService.UploadFile(file, header)
		if err != nil {
			http.Error(w, "Failed to upload file: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Return JSON response with file information
		w.Header().Set("Content-Type", "application/json")
		response := struct {
			Success      bool    `json:"success"`
			UUID         string  `json:"uuid"`
			OriginalName string  `json:"original_name"`
			Size         int64   `json:"size"`
			MimeType     string  `json:"mime_type"`
			DownloadURL  string  `json:"download_url"`
			IsImage      bool    `json:"is_image"`
			Width        *int    `json:"width,omitempty"`
			Height       *int    `json:"height,omitempty"`
			ThumbnailURL *string `json:"thumbnail_url,omitempty"`
		}{
			Success:      true,
			UUID:         fileRecord.UUID,
			OriginalName: fileRecord.OriginalName,
			Size:         fileRecord.Size,
			MimeType:     fileRecord.MimeType,
			DownloadURL:  "/files/" + fileRecord.UUID,
			IsImage:      fileRecord.IsImage,
			Width:        fileRecord.Width,
			Height:       fileRecord.Height,
		}

		// Add thumbnail URL if available
		if fileRecord.IsImage && fileRecord.ThumbnailPath != nil {
			thumbnailURL := "/files/" + fileRecord.UUID + "/thumbnail"
			response.ThumbnailURL = &thumbnailURL
		}

		// Use proper JSON encoding
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
			return
		}

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
}

func (a *App) serveFile(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Extract UUID and check for thumbnail request from URL path
		path := strings.TrimPrefix(r.URL.Path, "/files/")
		if path == "" {
			http.Error(w, "Invalid file UUID", http.StatusBadRequest)
			return
		}

		// Check if this is a thumbnail request
		var uuid string
		var isThumbnail bool
		if strings.HasSuffix(path, "/thumbnail") {
			uuid = strings.TrimSuffix(path, "/thumbnail")
			isThumbnail = true
		} else {
			uuid = path
		}

		// Get file information
		fileRecord, err := a.FileService.GetFile(uuid)
		if err != nil {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}

		var filePath string
		var mimeType string
		var filename string

		if isThumbnail && fileRecord.IsImage && fileRecord.ThumbnailPath != nil {
			// Serve thumbnail
			filePath = filepath.Join("uploads", *fileRecord.ThumbnailPath)
			mimeType = fileRecord.MimeType // Keep original mime type
			filename = "thumb_" + fileRecord.OriginalName
		} else {
			// Serve original file
			filePath, err = a.FileService.GetFilePath(uuid)
			if err != nil {
				http.Error(w, "File not found", http.StatusNotFound)
				return
			}
			mimeType = fileRecord.MimeType
			filename = fileRecord.OriginalName

			// Increment download count only for original files
			if err := fileRecord.IncrementDownloadCount(a.DB); err != nil {
				log.Printf("Failed to increment download count for file %s: %v", uuid, err)
			}
		}

		// Set appropriate headers
		w.Header().Set("Content-Type", mimeType)

		// For images, set inline disposition; for others, set attachment
		if fileRecord.IsImage && !isThumbnail {
			w.Header().Set("Content-Disposition", `inline; filename="`+filename+`"`)
		} else if fileRecord.IsImage && isThumbnail {
			w.Header().Set("Content-Disposition", `inline; filename="`+filename+`"`)
		} else {
			w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
		}

		// Get file info for content length
		if fileInfo, err := os.Stat(filePath); err == nil {
			w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))
		}

		// Serve the file
		http.ServeFile(w, r, filePath)

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
}

func (a *App) listFiles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Parse pagination parameters
		limitStr := r.URL.Query().Get("limit")
		offsetStr := r.URL.Query().Get("offset")

		limit := 20 // default
		offset := 0 // default

		if limitStr != "" {
			if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
				limit = parsedLimit
			}
		}

		if offsetStr != "" {
			if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
				offset = parsedOffset
			}
		}

		// Get files from service
		files, err := a.FileService.ListFiles(limit, offset)
		if err != nil {
			http.Error(w, "Failed to list files", http.StatusInternalServerError)
			return
		}

		// Build JSON response
		w.Header().Set("Content-Type", "application/json")

		// Create response structure
		type FileResponse struct {
			UUID          string `json:"uuid"`
			OriginalName  string `json:"original_name"`
			Size          int64  `json:"size"`
			MimeType      string `json:"mime_type"`
			DownloadCount int    `json:"download_count"`
			CreatedAt     string `json:"created_at"`
			DownloadURL   string `json:"download_url"`
		}

		fileResponses := make([]FileResponse, len(files))
		for i, file := range files {
			fileResponses[i] = FileResponse{
				UUID:          file.UUID,
				OriginalName:  file.OriginalName,
				Size:          file.Size,
				MimeType:      file.MimeType,
				DownloadCount: file.DownloadCount,
				CreatedAt:     file.CreatedAt,
				DownloadURL:   "/files/" + file.UUID,
			}
		}

		response := struct {
			Files []FileResponse `json:"files"`
		}{
			Files: fileResponses,
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
			return
		}

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
}

// processFileReferences processes [file:filename] references in post content
func (a *App) processFileReferences(content string) template.HTML {
	// Regular expression to match [file:filename] patterns
	fileRefRegex := regexp.MustCompile(`\[file:([^\]]+)\]`)

	// Replace file references with appropriate HTML based on file type
	processedContent := fileRefRegex.ReplaceAllStringFunc(content, func(match string) string {
		// Extract filename from the match
		filename := fileRefRegex.FindStringSubmatch(match)[1]

		// Query database to find file by original name with image metadata
		rows, err := a.DB.Query("SELECT uuid, original_name, is_image, thumbnail_path, alt_text, width, height FROM files WHERE original_name = ? ORDER BY created_at DESC LIMIT 1", filename)
		if err != nil {
			log.Printf("Error querying file: %v", err)
			return match // Return original if error
		}
		defer rows.Close()

		if rows.Next() {
			var uuid, originalName string
			var isImage bool
			var thumbnailPath, altText *string
			var width, height *int

			if err := rows.Scan(&uuid, &originalName, &isImage, &thumbnailPath, &altText, &width, &height); err != nil {
				log.Printf("Error scanning file: %v", err)
				return match
			}

			// If it's an image, render as responsive image
			if isImage {
				alt := originalName
				if altText != nil && *altText != "" {
					alt = *altText
				}

				// Create responsive image HTML with thumbnail fallback
				imageHTML := `<div class="blog-image-container">`

				// Use thumbnail if available, otherwise use original
				imageSrc := "/files/" + uuid
				if thumbnailPath != nil && *thumbnailPath != "" {
					// Create thumbnail serving endpoint
					imageSrc = "/files/" + uuid + "/thumbnail"
				}

				imageHTML += `<img src="` + imageSrc + `" alt="` + alt + `" class="blog-image" loading="lazy"`

				// Add dimensions if available
				if width != nil && height != nil {
					imageHTML += ` data-width="` + strconv.Itoa(*width) + `" data-height="` + strconv.Itoa(*height) + `"`
				}

				imageHTML += ` onclick="openImageModal('` + "/files/" + uuid + `', '` + alt + `')">`
				imageHTML += `</div>`

				return imageHTML
			} else {
				// Return HTML download link for non-images
				return `<a href="/files/` + uuid + `" target="_blank">📎 ` + originalName + `</a>`
			}
		}

		// If file not found, return original text
		return match
	})

	return template.HTML(processedContent) // #nosec G203 - Content allows HTML for rich formatting
}

// extractExcerpt creates a safe plain text excerpt for list views
func (a *App) extractExcerpt(content string) string {
	// Remove HTML tags
	htmlTagRegex := regexp.MustCompile(`<[^>]*>`)
	plainText := htmlTagRegex.ReplaceAllString(content, "")

	// Remove file references
	fileRefRegex := regexp.MustCompile(`\[file:[^\]]+\]`)
	plainText = fileRefRegex.ReplaceAllString(plainText, "")

	// Clean up whitespace and newlines
	spaceRegex := regexp.MustCompile(`\s+`)
	plainText = spaceRegex.ReplaceAllString(plainText, " ")
	plainText = strings.TrimSpace(plainText)

	// Limit to 500 characters for excerpt (longer than meta description)
	if len(plainText) > 500 {
		plainText = plainText[:497] + "..."
	}

	if plainText == "" {
		return "No content available"
	}

	return plainText
}

func (a *App) updateFileAltText(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		// Parse JSON request
		var request struct {
			UUID    string `json:"uuid"`
			AltText string `json:"alt_text"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if request.UUID == "" {
			http.Error(w, "UUID is required", http.StatusBadRequest)
			return
		}

		// Get file record
		fileRecord, err := a.FileService.GetFile(request.UUID)
		if err != nil {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}

		// Only allow alt text for images
		if !fileRecord.IsImage {
			http.Error(w, "Alt text can only be set for images", http.StatusBadRequest)
			return
		}

		// Update alt text in database
		_, err = a.DB.Exec("UPDATE files SET alt_text = ? WHERE uuid = ?", request.AltText, request.UUID)
		if err != nil {
			http.Error(w, "Failed to update alt text", http.StatusInternalServerError)
			return
		}

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		response := struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
		}{
			Success: true,
			Message: "Alt text updated successfully",
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
}

func (a *App) serveSitemap(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Get all posts with slugs for sitemap
		posts, err := a.getAllPostsForSitemap()
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Generate sitemap XML
		sitemapXML, err := a.SEOService.GenerateSitemap(posts)
		if err != nil {
			http.Error(w, "Failed to generate sitemap", http.StatusInternalServerError)
			return
		}

		// Set appropriate headers
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour

		// Write sitemap
		if _, err := w.Write(sitemapXML); err != nil {
			http.Error(w, "Failed to write sitemap", http.StatusInternalServerError)
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

func (a *App) serveRobotsTxt(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Generate robots.txt content
		robotsTxt := a.SEOService.GenerateRobotsTxt()

		// Set appropriate headers
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=86400") // Cache for 24 hours

		// Write robots.txt
		if _, err := w.Write([]byte(robotsTxt)); err != nil {
			http.Error(w, "Failed to write robots.txt", http.StatusInternalServerError)
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

// getAllPostsForSitemap retrieves all posts with slugs for sitemap generation
func (a *App) getAllPostsForSitemap() ([]*model.Post, error) {
	rows, err := a.DB.Query(`
		SELECT id, title, body, datepost, slug, 
		       COALESCE(created_at, ''), COALESCE(updated_at, ''),
		       COALESCE(meta_description, ''), COALESCE(keywords, '')
		FROM posts 
		WHERE slug IS NOT NULL AND slug != '' 
		ORDER BY id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []*model.Post
	for rows.Next() {
		post := &model.Post{}
		err := rows.Scan(&post.ID, &post.Title, &post.Body, &post.Date, &post.Slug, 
			&post.CreatedAt, &post.UpdatedAt, &post.MetaDescription, &post.Keywords)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}

	return posts, nil
}

func (app *App) securityMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if match, _ := regexp.MatchString("/(create|delete)-comment", r.URL.RequestURI()); match {
			if !app.Sessions.IsLoggedin(r) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		} else if match, _ := regexp.MatchString("/(delete|update|create|upload-file)", r.URL.RequestURI()); match {
			if !app.Sessions.IsAdmin(r) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
		h.ServeHTTP(w, r)
	})
}
