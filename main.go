package main

import (
	"database/sql"
	"fmt"
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
)

const (
	postsPerPage = 8
	logFilePath  = "log/access.log"
)

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
	db      *sql.DB
	logFile *os.File
)

func main() {
	var err error
	db = initializeDatabase("database/database.sqlite")
	migrateDatabase(db)

	//Init logging
	logFile, err = initLogging(logFilePath)
	if err != nil {
		log.Fatal("Could not open file", err)
	}
	defer logFile.Close()
	fmt.Fprintln(logFile, "Begin logging")

	mux := http.NewServeMux()
	mux.HandleFunc("/", getPage)
	mux.HandleFunc("/post", getPost)
	mux.HandleFunc("/delete", deletePost)

	//Set Admin and Logging middleware
	adminHandler := logMiddleware(authMiddleware(mux))
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
		p, err := strconv.Atoi(r.FormValue("p"))
		if err != nil {
			http.Error(w, "Query error", 400)
			return
		}

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

		fmt.Fprintln(w, posts)

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

func loggedIn() bool {
	return true
}

func authMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if strings.HasPrefix(r.URL.Path, "/delete") && !loggedIn() || r.Method != "GET" && !loggedIn() {
			http.Error(w, "Method Not Allowed", 405)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func logMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := NewLoggingResponseWriter(w)
		h.ServeHTTP(l, r)

		_, err := fmt.Fprintf(logFile, "%s %v %s %s %s\n", time.Now().Format("Mon Jan _2 15:04:05 2006"), l.statusCode, r.Host, r.Method, r.URL.RequestURI())
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
