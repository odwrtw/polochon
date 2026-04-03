package server

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
	manager, err := newAuthManager(strings.NewReader(testConfigData))
	if err != nil {
		t.Fatalf("expected no error, got %s", err)
	}

	tt := []struct {
		name     string
		token    string
		right    authRight
		wantOK   bool
		wantName string
	}{
		{"unknown token", "bad", authRightRead, false, ""},
		{"guest can read", "guest1token", authRightRead, true, "guest1"},
		{"guest cannot write", "guest1token", authRightWrite, false, ""},
		{"guest cannot debug", "guest1token", authRightDebug, false, ""},
		{"user can write", "user1token", authRightWrite, true, "user1"},
		{"user cannot debug", "user1token", authRightDebug, false, ""},
		{"admin can debug", "admin1token", authRightDebug, true, "admin1"},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			name, ok := manager.isAllowed(tc.token, tc.right)
			if ok != tc.wantOK {
				t.Fatalf("isAllowed: want %t, got %t", tc.wantOK, ok)
			}
			if name != tc.wantName {
				t.Fatalf("token name: want %q, got %q", tc.wantName, name)
			}
		})
	}
}
