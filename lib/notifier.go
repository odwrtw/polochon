package polochon

import "context"

// Notifier is an interface to notify when a video is added
type Notifier interface {
	Module
	Notify(ctx context.Context, v any) error
}
