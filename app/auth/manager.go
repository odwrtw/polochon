package auth

import (
	"errors"
	"io"

	yaml "gopkg.in/yaml.v2"
)

// Custom errors
var (
	ErrRoleAlreadyDefined = errors.New("auth: role already defined")
	ErrRoleIncludeInvalid = errors.New("auth: invalid role in the include statement")
)

type token struct {
	name     string
	value    string
	routeMap map[string]struct{}
}

func newToken(name, value string, routes []string) *token {
	t := &token{
		name:     name,
		value:    value,
		routeMap: map[string]struct{}{},
	}

	for _, r := range routes {
		t.routeMap[r] = struct{}{}
	}

	return t
}

// Manager stores, and checks token
type Manager struct {
	tokens map[string]*token
}

// New returns a new authentication maanger
func New(r io.Reader) (*Manager, error) {
	m := &Manager{
		tokens: map[string]*token{},
	}
	return m, yaml.NewDecoder(r).Decode(m)
}

// UnmarshalYAML implements the unmarshaler interface
func (m *Manager) UnmarshalYAML(unmarshal func(interface{}) error) error {
	data := []struct {
		Role    string   `yaml:"role"`
		Allowed []string `yaml:"allowed"`
		Tokens  []struct {
			Name  string `yaml:"name"`
			Value string `yaml:"value"`
		} `yaml:"token"`
		Include []string `yaml:"include"`
	}{}

	if err := unmarshal(&data); err != nil {
		return err
	}

	roles := map[string][]string{}
	for _, d := range data {
		// Check if the role is already defined
		_, ok := roles[d.Role]
		if ok {
			return ErrRoleAlreadyDefined
		}

		// Add the allowed routes to the role
		roles[d.Role] = d.Allowed

		// If the role includes another role, add theses routes as well
		for _, i := range d.Include {
			routes, ok := roles[i]
			if !ok {
				return ErrRoleIncludeInvalid
			}

			roles[d.Role] = append(roles[d.Role], routes...)
		}

		// Now that the routes are gathered, lets setup the tokens
		for _, t := range d.Tokens {
			m.tokens[t.Value] = newToken(t.Name, t.Value, roles[d.Role])
		}
	}

	return nil
}

// IsAllowed checks if the given route is allowed with the given token
func (m *Manager) IsAllowed(token, route string) (string, bool) {
	t, ok := m.tokens[token]
	if !ok {
		return "", false
	}

	_, ok = t.routeMap[route]
	if !ok {
		return "", false
	}
	return t.name, ok
}

// GetAllowed returns the allowed routes names for a token
func (m *Manager) GetAllowed(token string) []string {
	t, ok := m.tokens[token]
	if !ok {
		return []string{}
	}

	routes := []string{}
	for k := range t.routeMap {
		routes = append(routes, k)
	}

	return routes
}
