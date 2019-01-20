package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type Logging struct {
	File *os.File
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func NewLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{w, http.StatusOK}
}

func (l *loggingResponseWriter) WriteHeader(code int) {
	l.statusCode = code
	l.ResponseWriter.WriteHeader(code)
}

func NewLogging(path string) Logging {
	var file *os.File
	var err error

	if _, err = os.Stat(path); os.IsNotExist(err) {
		file, err = os.Create(path)
	} else {
		file, err = os.OpenFile(path, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	}
	if err != nil {
		log.Fatal("Cannot create file", err)
	}
	return Logging{File: file}
}

func (lo *Logging) logMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := NewLoggingResponseWriter(w)
		h.ServeHTTP(l, r)

		_, err := fmt.Fprintf(lo.File, "%s %v %s %s %s\n", time.Now().Format("Mon Jan _2 15:04:05 2006"), l.statusCode, r.RemoteAddr, r.Method, r.URL.RequestURI())
		if err != nil {
			log.Println("Cannot write to file", err)
		}
	})
}
