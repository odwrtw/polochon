package eztv

import (
	"reflect"
	"testing"

	"github.com/odwrtw/eztv"
	polochon "github.com/odwrtw/polochon/lib"
)

func TestEztvShowList(t *testing.T) {
	e := &Eztv{}
	eztvListShows = func(page int) ([]*eztv.Show, error) {
		return []*eztv.Show{
			{
				ImdbID: "tt2562232",
				Title:  "The Movie",
				TvdbID: "aa123",
				Year:   "aa123",
			},
		}, nil
	}

	list, err := e.GetShowList("", fakeLogEntry)
	if err != nil {
		t.Fatalf("Expected no errors, got %q", err)
	}

	expectedShowList := []*polochon.Show{
		{
			ShowConfig: polochon.ShowConfig{},
			Title:      "The Movie",
			ImdbID:     "tt2562232",
		},
	}

	if !reflect.DeepEqual(expectedShowList, list) {
		t.Errorf("Failed to get show list from eztv\nExpected %+v\nGot %+v", expectedShowList, list)
	}
}
