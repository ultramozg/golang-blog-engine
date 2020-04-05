package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	os.Setenv("DBURI", "file:../database/database.sqlite")
	os.Setenv("TEMPLATES", "../templates/*.gohtml")

	os.Exit(m.Run())
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
		t.Errorf("Root handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	expected := `<p>Powered by Golang net/http package</p>`
	if !strings.Contains(rr.Body.String(), expected) {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}
