package mock

import "github.com/sirupsen/logrus"

// Notify implements the notifier interface
func (mock *Mock) Notify(any, *logrus.Entry) error {
	return nil
}
