package i18n

import (
	"encoding/json"
	"errors"
	"os"
	"testing"
)

type translatorFixture struct {
	DefaultLocale string                       `json:"default_locale"`
	Translations  map[string]map[string]string `json:"translations"`
}

type translatorGolden struct {
	Lookups []struct {
		Locale string `json:"locale"`
		Key    string `json:"key"`
		Args   []any  `json:"args"`
		Want   string `json:"want"`
	} `json:"lookups"`
	Missing []struct {
		Locale string `json:"locale"`
		Key    string `json:"key"`
	} `json:"missing"`
}

func TestTranslatorContract_BasicFixture(t *testing.T) {
	t.Skip("pending translator implementation")

	fixture := loadTranslatorFixture(t, "testdata/translator_basic_fixture.json")
	golden := loadTranslatorGolden(t, "testdata/translator_basic_golden.json")

	translator := newContractTranslator(t, fixture)

	for _, tc := range golden.Lookups {
		got, err := translator.Translate(tc.Locale, tc.Key, tc.Args...)
		if err != nil {
			t.Fatalf("Translate(%q,%q): unexpected err: %v", tc.Locale, tc.Key, err)
		}
		if got != tc.Want {
			t.Fatalf("Translate(%q,%q) = %q want %q", tc.Locale, tc.Key, got, tc.Want)
		}
	}

	for _, tc := range golden.Missing {
		_, err := translator.Translate(tc.Locale, tc.Key)
		if !errors.Is(err, ErrMissingTranslation) {
			t.Fatalf("missing Translate(%q,%q) err = %v want ErrMissingTranslation", tc.Locale, tc.Key, err)
		}
	}
}

func loadTranslatorFixture(t *testing.T, path string) translatorFixture {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	var fx translatorFixture
	if err := json.Unmarshal(data, &fx); err != nil {
		t.Fatalf("unmarshal fixture %s: %v", path, err)
	}
	return fx
}

func loadTranslatorGolden(t *testing.T, path string) translatorGolden {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v", path, err)
	}
	var g translatorGolden
	if err := json.Unmarshal(data, &g); err != nil {
		t.Fatalf("unmarshal golden %s: %v", path, err)
	}
	return g
}

func newContractTranslator(t *testing.T, fx translatorFixture) Translator {
	t.Helper()
	t.Fatalf("translator construction pending implementation")
	return nil
}
