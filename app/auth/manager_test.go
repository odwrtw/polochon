package auth

import (
	"strings"
	"testing"
)

var testConfigData = `
- role: guest
  read: true
  write: false
  debug: false
  token:
  - name: guest1
    value: guest1token
  - name: guest2
    value: guest2token

- role: user
  read: true
  write: true
  debug: false
  token:
  - name: user1
    value: user1token

- role: admin
  read: true
  write: true
  debug: true
  token:
  - name: admin1
    value: admin1token
`

func TestIsAllowed(t *testing.T) {
	manager, err := New(strings.NewReader(testConfigData))
	if err != nil {
		t.Fatalf("expected no error, got %s", err)
	}

	tt := []struct {
		name     string
		token    string
		right    Right
		wantOK   bool
		wantName string
	}{
		{"unknown token", "bad", RightRead, false, ""},
		{"guest can read", "guest1token", RightRead, true, "guest1"},
		{"guest cannot write", "guest1token", RightWrite, false, ""},
		{"guest cannot debug", "guest1token", RightDebug, false, ""},
		{"user can write", "user1token", RightWrite, true, "user1"},
		{"user cannot debug", "user1token", RightDebug, false, ""},
		{"admin can debug", "admin1token", RightDebug, true, "admin1"},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			name, ok := manager.IsAllowed(tc.token, tc.right)
			if ok != tc.wantOK {
				t.Fatalf("IsAllowed: want %t, got %t", tc.wantOK, ok)
			}
			if name != tc.wantName {
				t.Fatalf("token name: want %q, got %q", tc.wantName, name)
			}
		})
	}
}
