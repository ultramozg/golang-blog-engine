package app

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	os.Setenv("DBURI", "file:../database/database.sqlite")
	os.Setenv("TEMPLATES", "../templates/*.gohtml")

	os.Exit(m.Run())
}

func TestRoot(t *testing.T) {
	conf := NewConfig()
	a := NewApp()
	a.Initialize(conf)

	req, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(a.root)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusFound {
		t.Errorf("Root handler returned wrong status code: got %v want %v", status, http.StatusFound)
	}
	expectedURI := "/page?p=0"
	if rr.Header().Get("Location") != expectedURI {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expectedURI)
	}
}

func TestGetPage(t *testing.T) {
	conf := NewConfig()
	a := NewApp()
	a.Initialize(conf)

	req, err := http.NewRequest(http.MethodGet, "/page?p=0", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(a.getPage)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("GetPage handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	expected := `<p>Powered by Golang net/http package</p>`
	if !strings.Contains(rr.Body.String(), expected) {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}

func TestFailedLogin(t *testing.T) {
	conf := NewConfig()
	a := NewApp()
	a.Initialize(conf)

	payload := url.Values{}
	payload.Set("login", "admin")
	payload.Set("password", "blabla")

	req, err := http.NewRequest(http.MethodPost, "/login", strings.NewReader(payload.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(a.login)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("login handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
	}
}

func TestSuccesfullLogin(t *testing.T) {
	conf := NewConfig()
	a := NewApp()
	a.Initialize(conf)

	payload := url.Values{}
	payload.Set("login", "admin")
	payload.Set("password", "12345")

	req, err := http.NewRequest(http.MethodPost, "/login", strings.NewReader(payload.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(a.login)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusSeeOther {
		t.Errorf("login handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	cookies := rr.Result().Cookies()
	if len(cookies) == 0 {
		t.Errorf("login handler returned empty cookie: got %v", cookies)
	}

	if c := cookies[0]; c.Name != "session" {
		t.Errorf("login handler 'session' cookies hasn't been set got %v want %v", c.Name, "session")
	}
}
