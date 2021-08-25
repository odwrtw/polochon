package nfo

import (
	polochon "github.com/odwrtw/polochon/lib"
)

func subFromMetadata(video polochon.Video, langs []polochon.Language) []*polochon.Subtitle {
	if len(langs) == 0 {
		return nil
	}

	subs := make([]*polochon.Subtitle, len(langs))
	for i, lang := range langs {
		subs[i] = &polochon.Subtitle{
			Embedded: true,
			Video:    video,
			Lang:     lang,
		}
	}

	return subs
}
