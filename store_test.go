package i18n

import "testing"

func TestStaticStoreGet(t *testing.T) {
	store := NewStaticStore(Translations{
		"en": newStringCatalog("en", map[string]string{"home.title": "Welcome"}),
		"es": newStringCatalog("es", map[string]string{"home.title": "Bienvenido"}),
	})

	tests := []struct {
		locale string
		key    string
		want   string
		ok     bool
	}{
		{locale: "en", key: "home.title", want: "Welcome", ok: true},
		{locale: "es", key: "home.title", want: "Bienvenido", ok: true},
		{locale: "en", key: "missing", want: "", ok: false},
		{locale: "fr", key: "home.title", want: "", ok: false},
	}

	for _, tc := range tests {
		got, ok := store.Get(tc.locale, tc.key)
		if ok != tc.ok || got != tc.want {
			t.Fatalf("Get(%q,%q) = %q,%v want %q,%v", tc.locale, tc.key, got, ok, tc.want, tc.ok)
		}
	}

	locales := store.Locales()
	if len(locales) != 2 || locales[0] != "en" || locales[1] != "es" {
		t.Fatalf("Locales() = %v", locales)
	}
}

func TestNewStaticStoreCopiesInput(t *testing.T) {
	src := Translations{
		"en": newStringCatalog("en", map[string]string{"home.title": "Welcome"}),
	}

	store := NewStaticStore(src)

	src["en"].Messages["home.title"] = Message{
		MessageMetadata: MessageMetadata{ID: "home.title", Locale: "en"},
		Variants:        map[PluralCategory]MessageVariant{PluralOther: {Template: "Changed"}},
	}
	src["en"].Messages["new"] = Message{
		MessageMetadata: MessageMetadata{ID: "new", Locale: "en"},
		Variants:        map[PluralCategory]MessageVariant{PluralOther: {Template: "new"}},
	}

	got, ok := store.Get("en", "home.title")
	if !ok || got != "Welcome" {
		t.Fatalf("expected snapshot to remain unchanged, got %q, ok=%v", got, ok)
	}

	if _, ok := store.Get("en", "new"); ok {
		t.Fatal("unexpected key copied from mutated input")
	}
}

func TestNewStaticStoreFromLoader(t *testing.T) {
	called := false
	loader := LoaderFunc(func() (Translations, error) {
		called = true
		return Translations{
			"en": newStringCatalog("en", map[string]string{"home.title": "Welcome"}),
		}, nil
	})

	store, err := NewStaticStoreFromLoader(loader)
	if err != nil {
		t.Fatalf("NewStaticStoreFromLoader: %v", err)
	}

	if !called {
		t.Fatal("loader not invoked")
	}

	if msg, ok := store.Get("en", "home.title"); !ok || msg != "Welcome" {
		t.Fatalf("Get returned %q,%v", msg, ok)
	}
}

func TestNewStaticStoreFromLoaderNil(t *testing.T) {
	store, err := NewStaticStoreFromLoader(nil)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if store == nil {
		t.Fatal("expected non-nil store")
	}

	if locales := store.Locales(); len(locales) != 0 {
		t.Fatalf("expected no locales, got %v", locales)
	}
}
