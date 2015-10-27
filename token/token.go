package token

// func randToken() string {
// 	b := make([]byte, 8)
// 	rand.Read(b)
// 	return fmt.Sprintf("%x", b)
// }

// Manager stores, and checks token
type Manager struct {
	Tokens      []*Token
	Roles       []*Role
	NoTokenRole *Role
}

// IsAllowed return true if the given route name is allowed with
// the given token's value
func (m *Manager) IsAllowed(value, routeName string) bool {
	if value == "" {
		return m.NoTokenRole.IsAllowed(routeName)
	}

	for _, t := range m.Tokens {
		if t.Value == value {
			return t.IsAllowed(routeName)
		}
	}
	return false
}

// GetRole returns the Role of the given name or nil if
// it don't exist
func (m *Manager) GetRole(name string) *Role {
	for _, r := range m.Roles {
		if r.Name == name {
			return r
		}
	}
	return nil
}

// GetAllowed returns all the route name allowed with the
// given token's value
func (m *Manager) GetAllowed(value string) []string {
	for _, t := range m.Tokens {
		if t.Value == value {
			return t.GetAllowed()
		}
	}
	return []string{}
}

// Role is a type of token which allow access to
// route name and inherit from other role
type Role struct {
	Name    string
	Allowed []string
	Include []*Role
}

// IsAllowed return true if the given route name is allowed
func (r *Role) IsAllowed(routeName string) bool {
	for _, name := range r.Allowed {
		if name == routeName {
			return true
		}
	}
	for _, role := range r.Include {
		if role.IsAllowed(routeName) {
			return true
		}
	}
	return false
}

// GetAllowed returns all the route name allowed
func (r *Role) GetAllowed() []string {
	allowed := []string{}
	allowed = append(allowed, r.Allowed...)

	for _, role := range r.Include {
		allowed = append(allowed, role.GetAllowed()...)
	}

	return allowed
}

// Token have a name, a value and a role
type Token struct {
	Role  *Role
	Name  string
	Value string
}

// IsAllowed return true if the given route name is allowed
func (t *Token) IsAllowed(routeName string) bool {
	return t.Role.IsAllowed(routeName)
}

// GetAllowed returns all the route name allowed
func (t *Token) GetAllowed() []string {
	return t.Role.GetAllowed()
}
