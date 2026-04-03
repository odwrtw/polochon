package mock

import polochon "github.com/odwrtw/polochon/lib"

// Watch implements the FsNotifier interface
func (mock *Mock) Watch(watchPath string, ctx polochon.FsNotifierCtx) error {
	return nil
}
