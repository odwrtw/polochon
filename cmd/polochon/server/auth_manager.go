package server

import (
	"io"

	yaml "gopkg.in/yaml.v2"
)

type authRight string

const (
	authRightRead  authRight = "read"
	authRightWrite authRight = "write"
	authRightDebug authRight = "debug"
)

type authToken struct {
	name  string
	read  bool
	write bool
	debug bool
}

type authManager struct {
	tokens map[string]*authToken // keyed by token value
}

func newAuthManager(r io.Reader) (*authManager, error) {
	m := &authManager{tokens: map[string]*authToken{}}
	return m, yaml.NewDecoder(r).Decode(m)
}

// UnmarshalYAML implements the unmarshaler interface.
func (m *authManager) UnmarshalYAML(unmarshal func(any) error) error {
	data := []struct {
		Role   string `yaml:"role"`
		Read   bool   `yaml:"read"`
		Write  bool   `yaml:"write"`
		Debug  bool   `yaml:"debug"`
		Tokens []struct {
			Name  string `yaml:"name"`
			Value string `yaml:"value"`
		} `yaml:"token"`
	}{}

	if err := unmarshal(&data); err != nil {
		return err
	}

	for _, d := range data {
		for _, t := range d.Tokens {
			m.tokens[t.Value] = &authToken{
				name:  t.Name,
				read:  d.Read,
				write: d.Write,
				debug: d.Debug,
			}
		}
	}
	return nil
}

// isAllowed checks if the given right is allowed for the given token value.
// Returns the token name and whether access is granted.
func (m *authManager) isAllowed(tokenValue string, right authRight) (string, bool) {
	t, ok := m.tokens[tokenValue]
	if !ok {
		return "", false
	}
	var granted bool
	switch right {
	case authRightRead:
		granted = t.read
	case authRightWrite:
		granted = t.write
	case authRightDebug:
		granted = t.debug
	}
	if !granted {
		return "", false
	}
	return t.name, true
}
