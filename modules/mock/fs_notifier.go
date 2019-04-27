package mock

import (
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

// Watch implements the FsNotifier interface
func (mock *Mock) Watch(watchPath string,
	ctx polochon.FsNotifierCtx, log *logrus.Entry) error {
	return nil
}
