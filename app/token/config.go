package token

import (
	"fmt"
	"io"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// gatekeeper ensures all string in slice
// are unique
type gatekeeper []string

// Append appends string to the slice
// returns false if element already exist
func (g *gatekeeper) Append(str string) bool {
	if g.Has(str) {
		return false
	}
	*g = append(*g, str)
	return true
}

func (g *gatekeeper) Has(str string) bool {
	for _, s := range *g {
		if s == str {
			return true
		}
	}
	return false
}

// LoadFromYaml returns a Manager from io.Reader which
// contain yaml
func LoadFromYaml(reader io.Reader) (*Manager, error) {
	b, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	trlist := &fileTokenRoleList{}
	err = yaml.Unmarshal(b, trlist)
	if err != nil {
		return nil, err
	}

	manager := Manager{}

	gkRole := &gatekeeper{}
	gkTokenName := &gatekeeper{}
	gkTokenValue := &gatekeeper{}

	allowNoTokenSet := false

	usedRole := []string{}

	for _, tr := range *trlist {

		if ok := gkRole.Append(tr.Role); !ok {
			return nil, fmt.Errorf("Invalid yml, role: %q already exists", tr.Role)
		}

		var role *Role
		role = manager.GetRole(tr.Role)
		if role == nil {
			role = &Role{
				Name:    tr.Role,
				Allowed: tr.Allowed,
				Include: []*Role{},
			}
			manager.Roles = append(manager.Roles, role)
		} else {
			role.Allowed = append(role.Allowed, tr.Allowed...)
		}

		for _, roleName := range tr.Include {
			var r *Role
			r = manager.GetRole(roleName)
			if r == nil {
				r = &Role{
					Name:    roleName,
					Allowed: []string{},
					Include: []*Role{},
				}
				manager.Roles = append(manager.Roles, r)
			}
			role.Include = append(role.Include, r)
			usedRole = append(usedRole, roleName)
		}

		for _, t := range tr.Token {
			if ok := gkTokenName.Append(t.Name); !ok {
				return nil, fmt.Errorf("Invalid yml, token name: %q already exists", t.Name)
			}
			if ok := gkTokenValue.Append(t.Value); !ok {
				return nil, fmt.Errorf("Invalid yml, token value: %q already exists", t.Value)
			}
			manager.Tokens = append(manager.Tokens, &Token{
				Role:  role,
				Name:  t.Name,
				Value: t.Value,
			})
		}

		if tr.AllowNoToken {
			if allowNoTokenSet {
				return nil, fmt.Errorf("No token role already declared, you can't use %q", tr.Role)
			}
			allowNoTokenSet = true
			manager.NoTokenRole = role
		}

	}

	for _, roleName := range usedRole {
		if !gkRole.Has(roleName) {
			return nil, fmt.Errorf("Invalid yml, role %q included but not defined", roleName)
		}
	}
	return &manager, nil
}

// Couple of object used for parsing the yaml file
type fileTokenRoleList []*fileTokenRole

type fileTokenRole struct {
	Role         string
	Allowed      []string
	Include      []string
	AllowNoToken bool
	Token        []*fileToken
}

type fileToken struct {
	Name  string
	Value string
}
