package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	/*"io"
	"flag"
	*/

	_ "github.com/mattn/go-sqlite3"
	"github.com/satori/go.uuid"
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

	//Register MUX
	mux := http.NewServeMux()
	mux.HandleFunc("/", getPage)
	mux.HandleFunc("/post", getPost)
	mux.HandleFunc("/delete", deletePost)
	mux.HandleFunc("/logout", logOut)

	//Register Fileserver
	fs := http.FileServer(http.Dir("public"))
	mux.Handle("/public/", http.StripPrefix("/public/", fs))

	//Set Admin and Logging middleware
	adminHandler := logMiddleware(authMiddleware(mux))

	log.Println("Listening on port :8080")
	http.ListenAndServe(":8080", adminHandler)
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
	case "GET":
		s := `select * from posts where id = ?`
		row := db.QueryRow(s, v)
		err := row.Scan(&p.Id, &p.Title, &p.Body, &p.Date)

		switch err {
		case sql.ErrNoRows:
			fmt.Fprintln(w, "No row was returned!")
		case nil:
			fmt.Fprintln(w, p)
		default:
			panic(err)
		}
	default:
		http.Error(w, "Method not Allowed", 405)
		return
	}
}

func getPage(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		p, _ := strconv.Atoi(r.FormValue("p"))

		posts := []Post{}
		s := `select * from posts limit 8 offset ?;`
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
			posts = append(posts, p)
		}

		//fmt.Fprintln(w, posts)
		tpl.ExecuteTemplate(w, "page.gohtml", posts)

	case "POST":
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad Request", 400)
			return
		}

		title := r.FormValue("title")
		body := r.FormValue("body")
		if title == "" || body == "" {
			http.Error(w, "Bad Request", 400)
			return
		}

		s := `insert into posts (title, body, datepost) values ($1, $2, $3)`
		_, err := db.Exec(s, title, body, time.Now().Format("Mon Jan _2 15:04:05 2006"))
		if err != nil {
			http.Error(w, "Internal Server Error", 500)
		}
	default:
		http.Error(w, "Method Not Allowed", 405)

	}

}

func logOut(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("session")
	if err == http.ErrNoCookie {
		http.Error(w, "You not Logged in", 401)
		return
	}
	if r.Method == http.MethodGet {
		//delete session
		if loggedInAsAdmin(c) {
			delete(dbSessions, c.Value)
			//delete cookie
			c = &http.Cookie{
				Name:   "session",
				Value:  "",
				MaxAge: -1,
			}
			http.SetCookie(w, c)
		}
	}
}

func deletePost(w http.ResponseWriter, r *http.Request) {
	v := r.FormValue("id")
	if _, err := strconv.Atoi(v); err != nil {
		http.Error(w, "Query error", 400)
		return
	}

	switch r.Method {
	case "GET":
		s := `delete from posts where id = ?`
		_, err := db.Exec(s, v)

		switch err {
		case sql.ErrNoRows:
			fmt.Fprintln(w, "No row was returned!")
		case nil:
			fmt.Fprintln(w, v)
		default:
			panic(err)
		}
	default:
		http.Error(w, "Method not Allowed", 405)
		return
	}
}

func loggedInAsAdmin(c *http.Cookie) bool {
	if v, ok := dbSessions[c.Value]; ok && v == ADMIN {
		return true
	}
	return false
}

func authMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//-First check if session exist, if so, allow
		//-Check if this is POST request, if so fetch try to fetch login, password, if login successfull create session
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		c, err := r.Cookie("session")
		if err == http.ErrNoCookie {
			if r.Method == "POST" {
				if err := r.ParseForm(); err != nil {
					http.Error(w, "Bad Request", 400)
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
						return
					} else {
						http.Error(w, "Not Authorized", 400)
						return
					}
				}
			}
		} else {
			//Cookie exist need to check if this is match in our dbSEssions
			if strings.HasPrefix(r.URL.Path, "/delete") && !loggedInAsAdmin(c) || r.Method != "GET" && !loggedInAsAdmin(c) {
				http.Error(w, "Method Not Allowed", 405)
				return
			}
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
