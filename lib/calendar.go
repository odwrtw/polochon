package polochon

import (
	"log"
	"time"

	"github.com/Sirupsen/logrus"
)

// Calendar is an interface to get the calendar for movies and shows
type Calendar interface {
	Module
	GetShowCalendar(show *Show) (*ShowCalendar, error)
}

// RegisterCalendar helps register a new calendar
func RegisterCalendar(name string, f func(params map[string]interface{}, log *logrus.Entry) (Calendar, error)) {
	if _, ok := registeredModules.Calendars[name]; ok {
		log.Panicf("modules: %q of type %q is already registered", name, TypeCalendar)
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
