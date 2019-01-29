package main

import (
	"compress/gzip"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
)

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

func cacheControlMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=2592000")
		h.ServeHTTP(w, r)
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

func (a *App) redirectTLSMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://"+a.Config.Domain+r.RequestURI, http.StatusMovedPermanently)
	})
}
