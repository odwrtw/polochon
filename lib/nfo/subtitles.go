package nfo

import (
	polochon "github.com/odwrtw/polochon/lib"
)

func subToLang(subs []*polochon.Subtitle) []polochon.Language {
	out := []polochon.Language{}
	for _, s := range subs {
		if !s.Embedded {
			continue
		}

		out = append(out, s.Lang)
	}

	if len(out) > 0 {
		return out
	}

	return nil
}

func langToSub(video polochon.Video, langs []polochon.Language) []*polochon.Subtitle {
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
