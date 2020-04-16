package papi

import (
	"fmt"

	polochon "github.com/odwrtw/polochon/lib"
)

// Module represents modules status as seen by polochon
type Module struct {
	Name   string                `json:"name"`
	Status polochon.ModuleStatus `json:"status"`
	Error  string                `json:"error"`
}

// GetModulesStatus return polochon's modules status
func (c *Client) GetModulesStatus() (map[string]map[string][]Module, error) {
	result := map[string]map[string][]Module{}
	url := fmt.Sprintf("%s/%s/%s", c.endpoint, "modules", "status")

	if err := c.get(url, &result); err != nil {
		return nil, err
	}

	return result, nil
}
