package polochon

// Language typc
type Language string

// Based on unix loacle
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
