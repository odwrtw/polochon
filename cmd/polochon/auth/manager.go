package auth

import (
	"io"

	yaml "gopkg.in/yaml.v2"
)

// Right represents a permission level.
type Right string

const (
	RightRead  Right = "read"
	RightWrite Right = "write"
	RightDebug Right = "debug"
)

type token struct {
	name  string
	read  bool
	write bool
	debug bool
}

// Manager stores and checks tokens.
type Manager struct {
	tokens map[string]*token // keyed by token value
}

// New returns a new authentication manager.
func New(r io.Reader) (*Manager, error) {
	m := &Manager{tokens: map[string]*token{}}
	return m, yaml.NewDecoder(r).Decode(m)
}

// UnmarshalYAML implements the unmarshaler interface.
func (m *Manager) UnmarshalYAML(unmarshal func(any) error) error {
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
			m.tokens[t.Value] = &token{
				name:  t.Name,
				read:  d.Read,
				write: d.Write,
				debug: d.Debug,
			}
		}
	}
	return nil
}

// IsAllowed checks if the given right is allowed for the given token value.
// Returns the token name and whether access is granted.
func (m *Manager) IsAllowed(tokenValue string, right Right) (string, bool) {
	t, ok := m.tokens[tokenValue]
	if !ok {
		return "", false
	}
	var granted bool
	switch right {
	case RightRead:
		granted = t.read
	case RightWrite:
		granted = t.write
	case RightDebug:
		granted = t.debug
	}
	if !granted {
		return "", false
	}
	return t.name, true
}
