package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/urfave/negroni"
)

func TestMiddleWare(t *testing.T) {
	manager := &Manager{
		tokens: map[string]*token{
			"token1": &token{
				name:  "user token 1",
				value: "token1",
				routeMap: map[string]struct{}{
					"GetStuff": {},
				},
			},
		},
	}

	router := mux.NewRouter()
	router.HandleFunc("/stuff", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Name("GetStuff").Methods("GET")
	router.HandleFunc("/private", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}).Name("GetPrivateStuff").Methods("GET")
	router.HandleFunc("/noname", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}).Name("").Methods("GET")

	n := negroni.New()
	n.Use(NewMiddleware(manager, router))
	n.UseHandler(router)

	server := httptest.NewServer(n)

	tt := []struct {
		name           string
		path           string
		token          string
		useHeader      bool
		expectedStatus int
	}{
		{
			name:           "valid token in url and valid path",
			path:           "/stuff",
			token:          "token1",
			useHeader:      false,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "valid token in headers and valid path",
			path:           "/stuff",
			token:          "token1",
			useHeader:      true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "valid token with invalid path",
			path:           "/things",
			token:          "token1",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "unauthorized",
			path:           "/private",
			token:          "token1",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "no name",
			path:           "/noname",
			token:          "token1",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			url := server.URL + tc.path
			if !tc.useHeader {
				url = url + "?token=" + tc.token
			}

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				t.Fatal(err)
			}

			if tc.useHeader {
				req.Header.Add("X-Auth-Token", tc.token)
			}

			client := http.DefaultClient
			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}

			if resp.StatusCode != tc.expectedStatus {
				t.Fatalf("invalid status code, expected %d got %d",
					tc.expectedStatus,
					resp.StatusCode)
			}
		})
	}
}
