package polochon

import "errors"

// Language type
type Language string

var ErrInvalidLanguage = errors.New("polochon: invalid lang")

// Based on unix locale
const (
	EN Language = "en_US"
	FR Language = "fr_FR"
)

// LangInfo represents differents infos of a Lang
type LangInfo struct {
	ShortForm string
}

var langInfo = map[Language]LangInfo{
	EN: {ShortForm: "en"},
	FR: {ShortForm: "fr"},
}

// ShortForm returns the short form of a lang
func (l Language) ShortForm() string {
	if info, ok := langInfo[l]; ok {
		return info.ShortForm
	}
	// If there is no LangInfo for this lang, return its string form
	return string(l)
}

func NewLanguage(l string) (Language, error) {
	if _, ok := langInfo[Language(l)]; ok {
		return Language(l), nil
	}
	// If there is no LangInfo for this lang, return its string form
	return "", ErrInvalidLanguage
}
