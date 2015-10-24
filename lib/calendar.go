package polochon

import (
	"fmt"
	"time"

	"github.com/Sirupsen/logrus"
)

// Calendar is an interface to get the calendar for movies and shows
type Calendar interface {
	Module
	GetShowCalendar(*Show, *logrus.Entry) (*ShowCalendar, error)
}

// RegisterCalendar helps register a new calendar
func RegisterCalendar(name string, f func(params []byte) (Calendar, error)) {
	if _, ok := registeredModules.Calendars[name]; ok {
		panic(fmt.Sprintf("modules: %q of type %q is already registered", name, TypeCalendar))
	}

	// Register the module
	registeredModules.Calendars[name] = f
}

// ShowCalendar holds the calendar for a show
type ShowCalendar struct {
	ImdbID   string
	Episodes []*ShowCalendarEpisode
}

// NewShowCalendar returns a new show calendar
func NewShowCalendar(imdbID string) *ShowCalendar {
	return &ShowCalendar{
		ImdbID:   imdbID,
		Episodes: []*ShowCalendarEpisode{},
	}
}

// ShowCalendarEpisode holds the episode calendar infos
type ShowCalendarEpisode struct {
	Season    int
	Episode   int
	AiredDate *time.Time
}

// IsAvailable tells if the episode is currently available
func (sc *ShowCalendarEpisode) IsAvailable() bool {
	// No info on aired, let's say it's available, it might actually be true
	if sc.AiredDate == nil {
		return true
	}

	return sc.AiredDate.Before(time.Now())
}

// IsOlder returns true if the given show is older than the calendar episode
func (sc *ShowCalendarEpisode) IsOlder(ws *WishedShow) bool {
	if sc.Season < ws.Season {
		return true
	}

	if sc.Season == ws.Season {
		if sc.Episode < ws.Episode {
			return true
		}
	}

	return false
}
