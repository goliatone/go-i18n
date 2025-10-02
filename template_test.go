package i18n

import (
	"reflect"
	"testing"
)

func TestTemplateHelpersTranslate(t *testing.T) {
	store := NewStaticStore(Translations{
		"en": {
			"home.title": "Welcome",
		},
	})

	translator, err := NewSimpleTranslator(store, WithTranslatorDefaultLocale("en"))
	if err != nil {
		t.Fatalf("NewSimpleTranslator: %v", err)
	}

	helpers := TemplateHelpers(translator, HelperConfig{})

	translate, ok := helpers["translate"].(func(string, string, ...any) string)
	if !ok {
		t.Fatalf("translate helper missing or wrong signature: %T", helpers["translate"])
	}

	if got := translate("en", "home.title"); got != "Welcome" {
		t.Fatalf("translate helper = %q", got)
	}
}

func TestTemplateHelpersMissingTranslation(t *testing.T) {
	translator, err := NewSimpleTranslator(NewStaticStore(nil), WithTranslatorDefaultLocale("en"))
	if err != nil {
		t.Fatalf("NewSimpleTranslator: %v", err)
	}

	var called bool
	onMissing := func(locale, key string, args []any, err error) string {
		called = true
		if err != ErrMissingTranslation {
			t.Fatalf("unexpected error: %v", err)
		}
		return "missing"
	}

	helpers := TemplateHelpers(translator, HelperConfig{OnMissing: onMissing})

	translate := helpers["translate"].(func(string, string, ...any) string)

	if got := translate("en", "unknown"); got != "missing" {
		t.Fatalf("translate missing = %q", got)
	}

	if !called {
		t.Fatal("expected missing handler to be called")
	}
}

func TestTemplateHelpersRegistry(t *testing.T) {
	registry := NewFormatterRegistry()
	custom := func(locale string) string { return "custom" }
	registry.Register("format_custom", custom)

	helpers := TemplateHelpers(nil, HelperConfig{Registry: registry})

	fn, ok := helpers["format_custom"]
	if !ok {
		t.Fatal("expected custom formatter in helpers")
	}

	if reflect.ValueOf(fn).Pointer() != reflect.ValueOf(custom).Pointer() {
		t.Fatal("unexpected formatter function returned")
	}

	translate := helpers["translate"].(func(string, string, ...any) string)
	if got := translate("en", "key"); got != "key" {
		t.Fatalf("translate fallback = %q", got)
	}
}
