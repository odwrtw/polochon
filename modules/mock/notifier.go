package mock

import "context"

// Notify implements the notifier interface
func (mock *Mock) Notify(_ context.Context, _ any) error {
	return nil
}
