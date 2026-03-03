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

func TestISO6392(t *testing.T) {
	for _, tc := range []struct {
		name    string
		lang    Language
		want    string
		wantErr error
	}{
		{name: "EN", lang: EN, want: "eng"},
		{name: "FR", lang: FR, want: "fre"},
		{name: "unknown", lang: Language("xx_XX"), want: "", wantErr: ErrInvalidLanguage},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.lang.ISO6392()
			if err != tc.wantErr {
				t.Errorf("expected err %v, got %v", tc.wantErr, err)
			}
			if got != tc.want {
				t.Errorf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestName(t *testing.T) {
	for _, tc := range []struct {
		name    string
		lang    Language
		want    string
		wantErr error
	}{
		{name: "EN", lang: EN, want: "English"},
		{name: "FR", lang: FR, want: "French"},
		{name: "unknown", lang: Language("xx_XX"), want: "", wantErr: ErrInvalidLanguage},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.lang.Name()
			if err != tc.wantErr {
				t.Errorf("expected err %v, got %v", tc.wantErr, err)
			}
			if got != tc.want {
				t.Errorf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestNewLanguageFromISO6392(t *testing.T) {
	for _, tc := range []struct {
		name    string
		code    string
		want    Language
		wantErr error
	}{
		{name: "eng", code: "eng", want: EN},
		{name: "fre", code: "fre", want: FR},
		{name: "unknown", code: "xyz", want: Language(""), wantErr: ErrInvalidLanguage},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NewLanguageFromISO6392(tc.code)
			if err != tc.wantErr {
				t.Errorf("expected err %v, got %v", tc.wantErr, err)
			}
			if got != tc.want {
				t.Errorf("expected %v, got %v", tc.want, got)
			}
		})
	}
}
