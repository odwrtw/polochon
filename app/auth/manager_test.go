package auth

import (
	"io"
	"reflect"
	"sort"
	"strings"
	"testing"
)

var testConfigData = `
- role: guest
  allowed:
    - TokenGetAllowed
    - MoviesListIDs
  token:
  - name: guest1
    value: guest1token
  - name: guest2
    value: guest2token

- role: user
  include:
    - guest
  allowed:
    - TorrentsAdd
  token:
  - name: user1
    value: user1token

- role: admin
  include:
    - user
  allowed:
    - DeleteEpisode
  token:
  - name: admin1
    value: admin1token
`

func TestConfig(t *testing.T) {
	manager, err := New(strings.NewReader(testConfigData))
	if err != nil {
		t.Fatalf("expected no error, got %s", err.Error())
	}

	tt := []struct {
		name           string
		token          string
		expectedRoutes []string
		err            error
	}{
		{
			name:  "valid guest token",
			token: "guest1token",
			expectedRoutes: []string{
				"MoviesListIDs",
				"TokenGetAllowed",
			},
		},
		{
			name:  "valid user token",
			token: "user1token",
			expectedRoutes: []string{
				"MoviesListIDs",
				"TokenGetAllowed",
				"TorrentsAdd",
			},
		},
		{
			name:  "valid admin token",
			token: "admin1token",
			expectedRoutes: []string{
				"DeleteEpisode",
				"MoviesListIDs",
				"TokenGetAllowed",
				"TorrentsAdd",
			},
		},
		{
			name:           "invalid token",
			expectedRoutes: []string{},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			routes := manager.GetAllowed(tc.token)
			sort.Strings(routes)
			if !reflect.DeepEqual(routes, tc.expectedRoutes) {
				t.Fatalf("invalid route match")
			}
		})
	}
}

func TestIsAllowed(t *testing.T) {
	manager, err := New(strings.NewReader(testConfigData))
	if err != nil {
		t.Fatalf("expected no error, got %s", err.Error())
	}

	tt := []struct {
		name              string
		token             string
		route             string
		expected          bool
		expectedTokenName string
	}{
		{
			name:     "invalid token on user routes",
			token:    "invalid_token",
			route:    "MoviesListIDs",
			expected: false,
		},
		{
			name:              "guest on guest routes",
			token:             "guest1token",
			route:             "MoviesListIDs",
			expected:          true,
			expectedTokenName: "guest1",
		},
		{
			name:     "guest on user routes",
			token:    "guest1token",
			route:    "TorrentsAdd",
			expected: false,
		},
		{
			name:              "user on user routes",
			token:             "user1token",
			route:             "TorrentsAdd",
			expected:          true,
			expectedTokenName: "user1",
		},
		{
			name:     "user on admin routes",
			token:    "user1token",
			route:    "DeleteEpisode",
			expected: false,
		},
		{
			name:              "admin on admin routes",
			token:             "admin1token",
			route:             "DeleteEpisode",
			expected:          true,
			expectedTokenName: "admin1",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			tokenName, allowed := manager.IsAllowed(tc.token, tc.route)
			if allowed != tc.expected {
				t.Fatalf("expected %t, got %t", tc.expected, allowed)
			}

			if tokenName != tc.expectedTokenName {
				t.Fatalf("invalid token name expected %q, got %q",
					tc.expectedTokenName, tokenName)
			}
		})
	}
}

var testInvalidRoleInclude = strings.NewReader(`
- role: guest
  include:
  - plop
`)

var testRoleAlreadyDefined = strings.NewReader(`
- role: guest
- role: guest
`)

var testIncludeSelf = strings.NewReader(`
- role: guest
  include:
  - guest
  allowed:
    - LucasRoute
    - LucasRoute2
  token:
  - name: admin1
    value: admin1token
`)

func TestInvalidConfig(t *testing.T) {
	tt := []struct {
		name  string
		input io.Reader
		err   error
	}{
		{
			name:  "role already defined",
			input: testRoleAlreadyDefined,
			err:   ErrRoleAlreadyDefined,
		},
		{
			name:  "role include invalid",
			input: testInvalidRoleInclude,
			err:   ErrRoleIncludeInvalid,
		},
		{
			name:  "role including itself",
			input: testIncludeSelf,
			err:   nil,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			_, got := New(tc.input)
			if got != tc.err {
				t.Fatalf("expected %s, got %s", tc.err, got)
			}
		})
	}
}
