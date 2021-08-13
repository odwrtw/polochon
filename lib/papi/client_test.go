package papi

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
)

func TestTokenAuth(t *testing.T) {
	var foundToken string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		foundToken = r.Header.Get("X-Auth-Token")
	}))
	defer ts.Close()

	c, err := New(ts.URL)
	if err != nil {
		t.Fatalf("invalid endpoint: %q", err)
	}
	c.SetToken("token1")

	// Don't need to check the result of this call, the point is to get the
	// header
	c.GetMovies()
	if foundToken != "token1" {
		t.Fatalf("token not set in the header")
	}
}

func TestDownloadURL(t *testing.T) {
	c, err := New("http://mock.url")
	if err != nil {
		t.Fatalf("invalid endpoint: %q", err)
	}
	c.SetToken("test")

	for _, test := range []struct {
		name         string
		downloadable Downloadable
		expectedURL  string
		expectedErr  error
		withToken    bool
	}{
		{
			name:         "movie without token",
			downloadable: &Movie{Movie: &polochon.Movie{ImdbID: "tt001"}},
			expectedURL:  "http://mock.url/movies/tt001/download",
			expectedErr:  nil,
		},
		{
			name:         "movie with token",
			downloadable: &Movie{Movie: &polochon.Movie{ImdbID: "tt001"}},
			expectedURL:  "http://mock.url/movies/tt001/download?token=test",
			expectedErr:  nil,
			withToken:    true,
		},
		{
			name:         "empty movie",
			downloadable: &Movie{},
			expectedErr:  ErrMissingMovie,
		},
		{
			name:         "empty movie with token",
			downloadable: &Movie{},
			expectedErr:  ErrMissingMovie,
			withToken:    true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var f func(Downloadable) (string, error) = c.DownloadURL
			if test.withToken {
				f = c.DownloadURLWithToken
			}

			got, err := f(test.downloadable)
			if err != test.expectedErr {
				t.Fatalf("expected err %q, got %q", test.expectedErr, err)
			}

			if got != test.expectedURL {
				t.Fatalf("expected %q, got %q", test.expectedURL, got)
			}
		})
	}
}

func TestGet(t *testing.T) {
	for _, test := range []struct {
		serverHeader   int
		expectedError  error
		expectedResult []int
	}{
		{
			serverHeader:   http.StatusOK,
			expectedError:  nil,
			expectedResult: []int{1, 2},
		},
		{
			serverHeader:   http.StatusNotFound,
			expectedError:  ErrResourceNotFound,
			expectedResult: []int{},
		},
		{
			serverHeader:   http.StatusForbidden,
			expectedError:  errors.New(`papi: HTTP error status 403 Forbidden: Unknown error`),
			expectedResult: []int{},
		},
	} {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(test.serverHeader)
			w.Write([]byte("[1,2]"))
		}))
		defer ts.Close()

		client, err := New(ts.URL)
		if err != nil {
			t.Fatalf("expected no error, got %q", err)
		}

		got := []int{}
		err = client.get(ts.URL, &got)
		if err != test.expectedError {
			if err == nil {
				t.Errorf("expected no error got %q", err)
			} else {
				if err.Error() != test.expectedError.Error() {
					t.Fatalf("expected: %+v, got %+v", test.expectedError.Error(), err.Error())
				}
			}
		}

		if !reflect.DeepEqual(got, test.expectedResult) {
			t.Fatalf("expected: %+v, got %+v", test.expectedResult, got)
		}
	}
}

func TestPost(t *testing.T) {
	for _, test := range []struct {
		serverHeader   int
		data           interface{}
		expectedError  error
		expectedResult []int
	}{
		{
			serverHeader:   http.StatusNotFound,
			data:           nil,
			expectedError:  ErrResourceNotFound,
			expectedResult: []int{},
		},
		{
			serverHeader: http.StatusForbidden,
			data: struct {
				Test string
			}{
				Test: "test",
			},
			expectedError:  errors.New(`papi: HTTP error status 403 Forbidden: Unknown error`),
			expectedResult: []int{},
		},
	} {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(test.serverHeader)
			w.Write([]byte("[1,2]"))
		}))
		defer ts.Close()

		client, err := New(ts.URL)
		if err != nil {
			t.Fatalf("expected no error doing new client, got %q", err)
		}

		got := []int{}
		err = client.post(ts.URL, test.data, &got)
		if err != test.expectedError {
			if err == nil {
				t.Errorf("expected error %q, got nil", test.expectedError)
			} else {
				if err.Error() != test.expectedError.Error() {
					t.Fatalf("expected: %+v, got %+v", test.expectedError.Error(), err.Error())
				}
			}
		}

		if !reflect.DeepEqual(got, test.expectedResult) {
			t.Fatalf("expected: %+v, got %+v", test.expectedResult, got)
		}
	}
}

func TestDelete(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client, err := New(ts.URL)
	if err != nil {
		t.Fatalf("expected no error doing new client, got %q", err)
	}

	err = client.Delete(&Movie{Movie: &polochon.Movie{ImdbID: "fake_id"}})
	if err != nil {
		t.Fatalf("Expected no error, got %+v", err)
	}
}

func TestBasicAuth(t *testing.T) {
	var useBasicAuth bool
	var username, password string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, useBasicAuth = r.BasicAuth()
		w.Write([]byte("[1,2]"))
	}))
	defer ts.Close()

	client, err := New(ts.URL)
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}

	expectedUsername := "my_username"
	expectedPassword := "my_password"
	client.SetBasicAuth(expectedUsername, expectedPassword)

	got := []int{}
	if err := client.get(ts.URL, &got); err != nil {
		t.Fatalf("expected no error, got %q", err)
	}

	expected := []int{1, 2}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("expected: %+v, got %+v", expected, got)
	}

	if !useBasicAuth {
		t.Fatal("basic auth not set")
	}

	if expectedUsername != username {
		t.Fatalf("invalid username, expected %q, got %q", expectedUsername, username)
	}

	if expectedPassword != password {
		t.Fatalf("invalid password, expected %q, got %q", expectedPassword, password)
	}
}
