package token

import (
	"reflect"
	"testing"
)

func TestGetAllowed(t *testing.T) {
	manager := createExpectedManager()
	testMock := []struct {
		Value    string
		Expected []string
	}{
		{"guest1token", []string{"TokenGetAllowed", "MoviesListIDs"}},
		{"user1token", []string{"TorrentsAdd", "TokenGetAllowed", "MoviesListIDs"}},
		{"admin1token", []string{"DeleteByID", "TorrentsAdd", "TokenGetAllowed", "MoviesListIDs"}},
	}

	for _, tt := range testMock {
		allowed := manager.GetAllowed(tt.Value)
		if !reflect.DeepEqual(allowed, tt.Expected) {
			t.Error("For value:", tt.Value, "Expected:", tt.Expected, "got:", allowed)
		}
	}
}

func TestIsAllowed(t *testing.T) {
	manager := createExpectedManager()
	testMock := []struct {
		Value    string
		Route    string
		Expected bool
	}{
		{"guest1token", "TokenGetAllowed", true},
		{"guest1token", "TorrentsAdd", false},
		{"user1token", "TokenGetAllowed", true},
		{"user1token", "TorrentsAdd", true},
	}

	for _, tt := range testMock {
		allowed := manager.IsAllowed(tt.Value, tt.Route)
		if allowed != tt.Expected {
			t.Error("For value:", tt.Value, "and route:", tt.Route, "Expected:", tt.Expected, "got:", allowed)
		}
	}
}
