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

// LangInfo represents different representations of a Language
type LangInfo struct {
	ShortForm string // ISO 639-1: "en", "fr"
	ISO6392   string // ISO 639-2/B: "eng", "fre"
	Name      string // Title case English name: "English", "French"
}

var langInfo = map[Language]LangInfo{
	EN: {ShortForm: "en", ISO6392: "eng", Name: "English"},
	FR: {ShortForm: "fr", ISO6392: "fre", Name: "French"},
}

// iso6392Index is a reverse lookup from ISO 639-2/B codes to Language
var iso6392Index = func() map[string]Language {
	m := make(map[string]Language, len(langInfo))
	for lang, info := range langInfo {
		m[info.ISO6392] = lang
	}
	return m
}()

// ShortForm returns the ISO 639-1 short form of a lang (e.g. "en", "fr")
func (l Language) ShortForm() string {
	if info, ok := langInfo[l]; ok {
		return info.ShortForm
	}
	// If there is no LangInfo for this lang, return its string form
	return string(l)
}

// ISO6392 returns the ISO 639-2/B code of the language (e.g. "eng", "fre")
func (l Language) ISO6392() (string, error) {
	if info, ok := langInfo[l]; ok {
		return info.ISO6392, nil
	}
	return "", ErrInvalidLanguage
}

// Name returns the Title case English name of the language (e.g. "English", "French")
func (l Language) Name() (string, error) {
	if info, ok := langInfo[l]; ok {
		return info.Name, nil
	}
	return "", ErrInvalidLanguage
}

// NewLanguage validates and returns a Language from its locale string (e.g. "en_US")
func NewLanguage(l string) (Language, error) {
	if _, ok := langInfo[Language(l)]; ok {
		return Language(l), nil
	}
	return "", ErrInvalidLanguage
}

// NewLanguageFromISO6392 returns a Language from an ISO 639-2/B code (e.g. "eng", "fre")
func NewLanguageFromISO6392(code string) (Language, error) {
	if l, ok := iso6392Index[code]; ok {
		return l, nil
	}
	return "", ErrInvalidLanguage
}
