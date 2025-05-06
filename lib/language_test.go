package polochon

import (
	"testing"
)

func TestShortForm(t *testing.T) {
	for _, l := range []struct {
		lang      Language
		longForm  string
		shortForm string
		err       error
	}{
		{
			longForm:  "en_US",
			shortForm: "en",
			lang:      EN,
		},
		{
			longForm:  "fr_FR",
			shortForm: "fr",
			lang:      FR,
		},
		{
			longForm: "pwet",
			err:      ErrInvalidLanguage,
		},
	} {
		lang, err := NewLanguage(l.longForm)
		if err != l.err {
			t.Errorf("Expected error %#v, got %#v", l.err, err)
		}
		if lang != l.lang {
			t.Errorf("Expected lang %#v, got %#v", l.lang, lang)
		}
		short := lang.ShortForm()
		if short != l.shortForm {
			t.Errorf("Expected %#v, got %#v", l.shortForm, short)
		}
	}
}
