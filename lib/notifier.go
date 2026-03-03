package polochon

import (
	"github.com/sirupsen/logrus"
)

// Notifier is an interface to notify when a video is added
type Notifier interface {
	Module
	Notify(any, *logrus.Entry) error
}
