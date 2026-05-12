package index

import (
	polochon "github.com/odwrtw/polochon/lib"
)

func upsertSubtitle(subs []*polochon.Subtitle, sub *polochon.Subtitle) []*polochon.Subtitle {
	if sub == nil {
		return subs
	}

	if subs == nil {
		return []*polochon.Subtitle{sub}
	}

	idx := -1
	newSubs := subs
	for i, oldSub := range subs {
		if oldSub.Embedded {
			continue
		}

		if sub.Lang == oldSub.Lang {
			idx = i
			break
		}
	}

	if idx >= 0 {
		newSubs[idx] = sub
	} else {
		newSubs = append(subs, sub)
	}

	return newSubs
}
