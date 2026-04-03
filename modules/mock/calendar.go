package mock

import (
	"context"

	polochon "github.com/odwrtw/polochon/lib"
)

// GetShowCalendar implements the calendar interface
func (mock *Mock) GetShowCalendar(_ context.Context, _ *polochon.Show) (*polochon.ShowCalendar, error) {
	return nil, nil
}
