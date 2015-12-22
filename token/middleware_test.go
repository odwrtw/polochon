package token_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/odwrtw/polochon/token"
)

func fakeHandler(http.ResponseWriter, *http.Request) {}

func createTestNegroni() *negroni.Negroni {
	manager := createExpectedManager()

	router := mux.NewRouter()
	router.HandleFunc("/guest", fakeHandler).Name("TokenGetAllowed")
	router.HandleFunc("/user", fakeHandler).Name("TorrentsAdd")
	router.HandleFunc("/admin", fakeHandler).Name("DeleteBySlugs")
	router.HandleFunc("/noname", fakeHandler)

	tmiddleware := token.NewMiddleware(manager, router)

	n := negroni.New()
	n.Use(tmiddleware)
	n.UseHandler(router)
	return n
}

func TestMiddleware404(t *testing.T) {
	n := createTestNegroni()
	s := httptest.NewServer(n)
	defer s.Close()

	resp, err := http.Get(s.URL + "/dontexist")
	if err != nil {
		t.Error(err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Error("Expected: 404, got:", resp.StatusCode)
	}
}

func TestMiddlewareNoName(t *testing.T) {
	n := createTestNegroni()
	s := httptest.NewServer(n)
	defer s.Close()

	resp, err := http.Get(s.URL + "/noname")
	if err != nil {
		t.Error(err)
	}

	if resp.StatusCode != http.StatusForbidden {
		t.Error("Expected:", http.StatusForbidden, ", got:", resp.StatusCode)
	}

	if body, err := ioutil.ReadAll(resp.Body); err != nil {
		t.Error(err)
	} else if string(body) != "Invalid route\n" {
		t.Error("expected", "Invalid route\n", "got", fmt.Sprintf("%q", string(body)))
	}
}

func TestMiddlewareAllow(t *testing.T) {
	testMock := []struct {
		Path     string
		Value    string
		Expected int
	}{
		{"/guest", "guest1token", http.StatusOK},
		{"/user", "guest1token", http.StatusForbidden},
		{"/admin", "guest1token", http.StatusForbidden},

		{"/guest", "user1token", http.StatusOK},
		{"/user", "user1token", http.StatusOK},
		{"/admin", "user1token", http.StatusForbidden},

		{"/guest", "admin1token", http.StatusOK},
		{"/user", "admin1token", http.StatusOK},
		{"/admin", "admin1token", http.StatusOK},
	}

	n := createTestNegroni()
	s := httptest.NewServer(n)
	defer s.Close()

	for _, tt := range testMock {
		resp, err := http.Get(s.URL + tt.Path + "?token=" + tt.Value)
		if err != nil {
			t.Error(err)
		}

		if resp.StatusCode != tt.Expected {
			t.Error("Expected:", tt.Expected, ", got:", resp.StatusCode)
		}

	}
}

func TestNoToken(t *testing.T) {
	testMock := []struct {
		Path     string
		Expected int
	}{
		{"/guest", http.StatusOK},
		{"/user", http.StatusForbidden},
		{"/admin", http.StatusForbidden},
	}
	n := createTestNegroni()
	s := httptest.NewServer(n)
	defer s.Close()

	for _, tt := range testMock {
		resp, err := http.Get(s.URL + tt.Path)
		if err != nil {
			t.Error(err)
		}

		if resp.StatusCode != tt.Expected {
			t.Error("Expected:", tt.Expected, ", got:", resp.StatusCode)
		}

	}
}
