package main

import (
	"compress/gzip"
	"crypto/tls"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/satori/go.uuid"
	"golang.org/x/crypto/acme/autocert"
)

const (
	postsPerPage = 8
	logFilePath  = "log/access.log"
)

const (
	ADMIN = iota
	USER
)

type Admin struct {
	login  string
	passwd string
}

type Post struct {
	Id    int
	Title string
	Body  string
	Date  string
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

var gzPool = sync.Pool{
	New: func() interface{} {
		w := gzip.NewWriter(ioutil.Discard)
		return w
	},
}

var (
	db         *sql.DB
	logFile    *os.File
	admin      *Admin
	dbSessions map[string]int
	tpl        *template.Template
)

func init() {
	tpl = template.Must(template.ParseGlob("templates/*.gohtml"))
}

func main() {
	//Parse Keys
	addr := flag.String("ip", "localhost", "Ip address to bind(localhost)")
	port := flag.String("p", "8080", "port listen to (:8080)")
	sport := flag.String("sp", "8443", "port to listen ssl (:8443)")
	flag.Parse()

	var err error
	db = initializeDatabase("database/database.sqlite")
	migrateDatabase(db)

	//User admin
	admin = &Admin{"admin", "abcd"}
	dbSessions = make(map[string]int)

	//Init logging
	logFile, err = initLogging(logFilePath)
	if err != nil {
		log.Fatal("Could not open file", err)
	}
	defer logFile.Close()
	fmt.Fprintln(logFile, "Begin logging")

	//Get the cert
	cert := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist("dcandu.name"),
		Cache:      autocert.DirCache("cert"),
	}

	//Register MUX
	mux := http.NewServeMux()
	mux.HandleFunc("/", getPage)
	mux.HandleFunc("/login", login)
	mux.HandleFunc("/logout", logout)
	mux.HandleFunc("/post", getPost)
	mux.HandleFunc("/publish", publish)
	mux.HandleFunc("/about", about)
	mux.HandleFunc("/links", links)
	mux.HandleFunc("/delete", deletePost)

	//Register Fileserver
	fs := http.FileServer(http.Dir("public/"))
	mux.Handle("/public/", http.StripPrefix("/public/", fs))

	//Set Admin and Logging middleware
	mainHandler := gzipMiddleware(logMiddleware(setHeaderMiddleware(authMiddleware(mux))))

	server := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		Addr:         *addr + ":" + *sport,
		TLSConfig: &tls.Config{
			GetCertificate: cert.GetCertificate,
		},
		Handler: mainHandler,
	}

	log.Println("Listening on addr:port: ", *addr+":"+*port)
	log.Println("Listening SSL on addr:port: ", *addr+":"+*sport)

	//Launch standart http and https protocols
	go http.ListenAndServe(*addr+":"+*port, cert.HTTPHandler(mainHandler))
	log.Fatal(server.ListenAndServeTLS("", ""))
}

func initializeDatabase(filepath string) *sql.DB {
	db, err := sql.Open("sqlite3", filepath)
	if err != nil {
		panic("Error connecting to database")
	}

	return db
}

func migrateDatabase(db *sql.DB) {
	sql := `
	create table if not exists posts (
	id integer primary key autoincrement,
	title string not null,
	body string not null,
	datepost string not null);
	`

	_, err := db.Exec(sql)

	if err != nil {
		panic(err)
	}
}

func getPost(w http.ResponseWriter, r *http.Request) {
	v := r.FormValue("id")
	if _, err := strconv.Atoi(v); err != nil {
		http.Error(w, "Query error", 400)
		return
	}

	p := Post{}

	switch r.Method {
	case http.MethodGet:
		s := `select * from posts where id = ?`
		row := db.QueryRow(s, v)
		err := row.Scan(&p.Id, &p.Title, &p.Body, &p.Date)

		switch err {
		case sql.ErrNoRows:
			fmt.Fprintln(w, "No row was returned!")
		case nil:
			data := struct {
				Post     Post
				LoggedIn bool
			}{
				p,
				loggedInAsAdmin(r),
			}
			tpl.ExecuteTemplate(w, "post.gohtml", data)
		default:
			panic(err)
		}
	default:
		http.Error(w, "Method not Allowed", 405)
		return
	}
}

func publish(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case http.MethodGet:
		//First we need check if this post id already exist
		//fetch id, try to load into template
		id, _ := strconv.Atoi(r.FormValue("id"))

		p := Post{}
		s := `select * from posts where id = ?`
		row := db.QueryRow(s, id)
		err := row.Scan(&p.Id, &p.Title, &p.Body, &p.Date)

		switch err {
		case sql.ErrNoRows:
			//No data return empty form
			data := struct {
				Post     Post
				LoggedIn bool
				IsUpdate bool
			}{
				Post{},
				loggedInAsAdmin(r),
				false,
			}
			tpl.ExecuteTemplate(w, "publish.gohtml", data)
		case nil:
			data := struct {
				Post     Post
				LoggedIn bool
				IsUpdate bool
			}{
				p,
				loggedInAsAdmin(r),
				true,
			}
			tpl.ExecuteTemplate(w, "publish.gohtml", data)
		default:
			http.Error(w, "Internal Server error", 500)
		}

		tpl.ExecuteTemplate(w, "publish.gohtml", loggedInAsAdmin(r))
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad Request", 400)
			return
		}

		id, ok := strconv.Atoi(r.FormValue("id"))
		title := r.FormValue("title")
		body := r.FormValue("body")
		if title == "" || body == "" {
			http.Error(w, "Bad Request", 400)
			return
		}
		//check if id was passed, if so need to change query
		var s string
		var err error

		if ok != nil {
			s = `insert into posts (title, body, datepost) values ($1, $2, $3)`
			_, err = db.Exec(s, title, body, time.Now().Format("Mon Jan _2 15:04:05 2006"))
		} else {
			s = `update posts set title = $1, body = $2, datepost = $3 where id = $4`
			_, err = db.Exec(s, title, body, time.Now().Format("Mon Jan _2 15:04:05 2006"), id)
		}
		if err != nil {
			http.Error(w, "Internal Server Error", 500)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	default:
		http.Error(w, "Method Not Allowed", 405)
		return
	}
}

func getPage(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		p, _ := strconv.Atoi(r.FormValue("p"))

		posts := []Post{}
		s := `select id, title, substr(body,1,300), datepost from posts order by id desc limit 8 offset ?;`
		rows, err := db.Query(s, p*postsPerPage)
		if err != nil {
			http.Error(w, "Internal Server error", 500)
		}

		defer rows.Close()
		for rows.Next() {
			p := Post{}
			err := rows.Scan(&p.Id, &p.Title, &p.Body, &p.Date)
			if err != nil {
				http.Error(w, "Bad Request", 400)
			}
			/*
				Shrink p.Body to display only words
				because we have limit in sql query substr
			*/
			re := regexp.MustCompile(`\b+\w+\s+?$`)
			p.Body = re.ReplaceAllString(p.Body, " ...")
			posts = append(posts, p)
		}

		data := struct {
			Posts    []Post
			LoggedIn bool
			PrevPage int
			NextPage int
		}{
			posts,
			loggedInAsAdmin(r),
			add(p, -1),
			add(p, +1),
		}
		tpl.ExecuteTemplate(w, "page.gohtml", data)

	default:
		http.Error(w, "Method Not Allowed", 405)

	}

}

func deletePost(w http.ResponseWriter, r *http.Request) {
	v := r.FormValue("id")
	if _, err := strconv.Atoi(v); err != nil {
		http.Error(w, "Query error", 400)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s := `delete from posts where id = ?`
		_, err := db.Exec(s, v)

		switch err {
		case sql.ErrNoRows:
			fmt.Fprintln(w, "No row was returned!")
		case nil:
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		default:
			panic(err)
		}
	default:
		http.Error(w, "Method not Allowed", 405)
		return
	}
}

func about(w http.ResponseWriter, r *http.Request) {
	tpl.ExecuteTemplate(w, "about.gohtml", loggedInAsAdmin(r))
}

func links(w http.ResponseWriter, r *http.Request) {
	tpl.ExecuteTemplate(w, "links.gohtml", loggedInAsAdmin(r))
}

func login(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		tpl.ExecuteTemplate(w, "login.gohtml", loggedInAsAdmin(r))
	case http.MethodPost:
		//-First check if session exist, if so, allow
		//-Check if this is POST request, if so fetch try to fetch login, password, if login successfull create session
		c, err := r.Cookie("session")
		if err == http.ErrNoCookie || !loggedInAsAdmin(r) {
			if err := r.ParseForm(); err != nil {
				http.Error(w, "Internal Server Error", 500)
				return
			}
			if r.FormValue("login") != "" && r.FormValue("password") != "" {
				if r.FormValue("login") == admin.login && r.FormValue("password") == admin.passwd {
					sID, _ := uuid.NewV4()
					c = &http.Cookie{
						Name:  "session",
						Value: sID.String(),
					}
					http.SetCookie(w, c)
					dbSessions[sID.String()] = ADMIN
					http.Redirect(w, r, "/", http.StatusSeeOther)
					return
				} else {
					http.Error(w, "Not Authorized", 400)
					http.Redirect(w, r, "/", http.StatusSeeOther)
					return
				}
			}
		} else {
			//Cookie exist need to check if this is match in our dbSEssions
			if strings.HasPrefix(r.URL.Path, "/delete") && !loggedInAsAdmin(r) || r.Method != "GET" && !loggedInAsAdmin(r) {
				http.Error(w, "Method Not Allowed", 405)
				return
			}
		}
	default:
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func logout(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("session")
	if err == http.ErrNoCookie {
		http.Error(w, "You are not Logged in", 401)
		return
	}
	switch r.Method {
	case http.MethodGet:
		//delete session
		if loggedInAsAdmin(r) {
			delete(dbSessions, c.Value)
			//delete cookie
			c = &http.Cookie{
				Name:   "session",
				Value:  "",
				MaxAge: -1,
			}
			http.SetCookie(w, c)
			http.Redirect(w, r, "/", http.StatusSeeOther)
		}
	default:
		http.Error(w, "Not authorized", 401)
	}
}

func loggedInAsAdmin(r *http.Request) bool {
	c, err := r.Cookie("session")
	if err == http.ErrNoCookie {
		return false
	} else {
		if v, ok := dbSessions[c.Value]; ok && v == ADMIN {
			return true
		}
	}
	return false
}

func setHeaderMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.RequestURI(), ".css") {
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		} else {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		}
		h.ServeHTTP(w, r)
	})
}

func logMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := NewLoggingResponseWriter(w)
		h.ServeHTTP(l, r)

		_, err := fmt.Fprintf(logFile, "%s %v %s %s %s\n", time.Now().Format("Mon Jan _2 15:04:05 2006"), l.statusCode, r.RemoteAddr, r.Method, r.URL.RequestURI())
		if err != nil {
			log.Println("Cannot write to file", err)
		}
	})
}

func authMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//check if user is Authorized to call this uri and Methods
		switch r.Method {
		case http.MethodPost:
			if loggedInAsAdmin(r) || r.URL.Path == "/login" {
				h.ServeHTTP(w, r)
			}
			return
		case http.MethodGet:
			if r.URL.Path == "/publish" || r.URL.Path == "/delete" {
				if loggedInAsAdmin(r) {
					h.ServeHTTP(w, r)
					return
				} else {
					http.Error(w, "Unauthorized", 401)
					return
				}
			}
			h.ServeHTTP(w, r)
			return
		default:
			http.Error(w, "Unauthorized", 401)
			return
		}
	})
}

func (w *gzipResponseWriter) WriteHeader(status int) {
	w.Header().Del("Content-Lenght")
	w.ResponseWriter.WriteHeader(status)
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func gzipMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			h.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Encoding", "gzip")

		gz := gzPool.Get().(*gzip.Writer)
		defer gzPool.Put(gz)

		gz.Reset(w)
		defer gz.Close()
		h.ServeHTTP(&gzipResponseWriter{ResponseWriter: w, Writer: gz}, r)
	})
}

func initLogging(path string) (*os.File, error) {
	var file *os.File
	var err error

	if _, err = os.Stat(path); os.IsNotExist(err) {
		file, err = os.Create(path)
	} else {
		file, err = os.OpenFile(path, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	}
	if err != nil {
		log.Fatal("Cannot create file", err)
		return nil, err
	}
	return file, nil
}

func NewLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{w, http.StatusOK}
}

func (l *loggingResponseWriter) WriteHeader(code int) {
	l.statusCode = code
	l.ResponseWriter.WriteHeader(code)
}

func add(i, j int) int {
	if i+j <= 0 {
		return 0
	}
	return i + j
}
