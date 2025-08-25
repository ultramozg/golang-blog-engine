package middleware

import (
	"compress/gzip"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

var gzPool = sync.Pool{
	New: func() interface{} {
		w := gzip.NewWriter(io.Discard)
		return w
	},
}

func SetHeaderMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.RequestURI(), ".css") {
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		} else {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		}
		h.ServeHTTP(w, r)
	})
}

func CacheControlMiddleware(h http.Handler) http.Handler {
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

func GzipMiddleware(h http.Handler) http.Handler {
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

// TODO domain hardcoded need to get it from config.
func RedirectTLSMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://"+"dcandu.name"+r.RequestURI, http.StatusMovedPermanently)
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{w, http.StatusOK}
}

// WriterHeader catch status code
func (l *loggingResponseWriter) WriteHeader(code int) {
	l.statusCode = code
	l.ResponseWriter.WriteHeader(code)
}

func LogMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := newLoggingResponseWriter(w)
		h.ServeHTTP(l, r)

		_, err := fmt.Printf("%s %v %s %s %s\n", time.Now().Format("Mon Jan _2 15:04:05 2006"), l.statusCode, r.RemoteAddr, r.Method, r.URL.RequestURI())
		if err != nil {
			log.Println("Cannot write to file", err)
		}
	})
}

// PostRedirectMiddleware handles redirects from old ID-based URLs to new slug-based URLs with SEO compliance
func PostRedirectMiddleware(db *sql.DB) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if this is a request to /post with an id parameter
			if r.URL.Path == "/post" && r.Method == "GET" {
				idStr := r.URL.Query().Get("id")
				if idStr != "" {
					id, err := strconv.Atoi(idStr)
					if err == nil {
						// Get the post slug from database
						var slug string
						err = db.QueryRow("SELECT slug FROM posts WHERE id = ?", id).Scan(&slug)
						if err == nil && slug != "" {
							// Sanitize and validate the slug for security
							sanitizedSlug := sanitizeSlug(slug)
							if sanitizedSlug != "" {
								canonicalURL := "/p/" + sanitizedSlug

								// Set canonical URL header for SEO
								w.Header().Set("Link", fmt.Sprintf("<%s>; rel=\"canonical\"", canonicalURL))

								// Add cache control headers for SEO
								w.Header().Set("Cache-Control", "public, max-age=31536000") // 1 year cache for redirects

								// Redirect to slug-based URL with 301 (permanent redirect)
								http.Redirect(w, r, canonicalURL, http.StatusMovedPermanently)
								return
							}
						}
					}
				}
			}

			// Continue with normal request processing
			h.ServeHTTP(w, r)
		})
	}
}

// sanitizeSlug validates and sanitizes a slug to prevent security issues
func sanitizeSlug(slug string) string {
	// Remove any potentially dangerous characters
	// Allow only alphanumeric characters, hyphens, and underscores
	validSlugRegex := regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)

	if !validSlugRegex.MatchString(slug) {
		return ""
	}

	// Additional validation: slug should not be empty and not too long
	if len(slug) == 0 || len(slug) > 200 {
		return ""
	}

	// Prevent directory traversal attempts
	if strings.Contains(slug, "..") || strings.Contains(slug, "/") || strings.Contains(slug, "\\") {
		return ""
	}

	return slug
}
