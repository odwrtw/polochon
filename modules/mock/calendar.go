package mock

import (
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

// GetShowCalendar implements the calendar interface
func (mock *Mock) GetShowCalendar(*polochon.Show, *logrus.Entry) (*polochon.ShowCalendar, error) {
	return nil, nil
}
