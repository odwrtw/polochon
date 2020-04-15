package transmission

import (
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/transmission"
)

func state(state int) polochon.TorrentState {
	switch state {
	case transmission.StatusStopped:
		return polochon.TorrentStateStopped
	case transmission.StatusCheckPending:
		return polochon.TorrentStatePending
	case transmission.StatusChecking:
		return polochon.TorrentStateChecking
	case transmission.StatusDownloadPending:
		return polochon.TorrentStateDownloadPending
	case transmission.StatusDownloading:
		return polochon.TorrentStateDownloading
	case transmission.StatusSeedPending:
		return polochon.TorrentStateSeedPending
	case transmission.StatusSeeding:
		return polochon.TorrentStateSeeding
	default:
		return polochon.TorrentState("unknown")
	}
}
