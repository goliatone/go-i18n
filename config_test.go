package i18n

import "testing"

func TestNewConfigDefaults(t *testing.T) {
	cfg, err := NewConfig(
		WithLocales("es", "en", "en"),
		WithDefaultLocale("es"),
	)
	if err != nil {
		t.Fatalf("NewConfig: %v", err)
	}

	if cfg.DefaultLocale != "es" {
		t.Fatalf("DefaultLocale = %q", cfg.DefaultLocale)
	}

	expected := []string{"en", "es"}
	if len(cfg.Locales) != len(expected) {
		t.Fatalf("Locales length = %d, want %d", len(cfg.Locales), len(expected))
	}
	for i, locale := range expected {
		if cfg.Locales[i] != locale {
			t.Fatalf("Locales[%d] = %q, want %q", i, cfg.Locales[i], locale)
		}
	}

	if cfg.Store == nil {
		t.Fatal("expected default store")
	}

	if cfg.Formatter == nil {
		t.Fatal("expected default formatter")
	}

	if cfg.Resolver == nil {
		t.Fatal("expected fallback resolver")
	}
}

func TestNewConfigWithLoader(t *testing.T) {
	loader := LoaderFunc(func() (Translations, error) {
		return Translations{
			"en": {"home.title": "Welcome"},
		}, nil
	})

	cfg, err := NewConfig(WithLoader(loader))
	if err != nil {
		t.Fatalf("NewConfig with loader: %v", err)
	}

	msg, ok := cfg.Store.Get("en", "home.title")
	if !ok || msg != "Welcome" {
		t.Fatalf("store lookup returned %q,%v", msg, ok)
	}
}

func TestConfigWithFallbackOption(t *testing.T) {
	cfg, err := NewConfig(
		WithFallback("es", "en", "fr", "en"),
	)
	if err != nil {
		t.Fatalf("NewConfig: %v", err)
	}

	chain := cfg.Resolver.Resolve("es")

	expected := []string{"en", "fr"}
	if len(chain) != len(expected) {
		t.Fatalf("fallback chain length = %d want %d", len(chain), len(expected))
	}

	for i, locale := range expected {
		if chain[i] != locale {
			t.Fatalf("fallback[%d] = %q want %q", i, chain[i], locale)
		}
	}
}

func TestBuildTranslator(t *testing.T) {
	cfg, err := NewConfig(
		WithLocales("en"),
		WithDefaultLocale("en"),
	)
	if err != nil {
		t.Fatalf("NewConfig: %v", err)
	}

	t.Setenv("_", "unused")

	translator, err := cfg.BuildTranslator()
	if err != nil {
		t.Fatalf("BuildTranslator: %v", err)
	}

	if translator == nil {
		t.Fatal("expected translator instance")
	}
}

func TestBuildTranslatorUsesFallback(t *testing.T) {
	store := NewStaticStore(Translations{
		"en": {"home.title": "Welcome"},
	})

	cfg, err := NewConfig(
		WithStore(store),
		WithDefaultLocale("en"),
		WithFallback("es", "en"),
	)
	if err != nil {
		t.Fatalf("NewConfig: %v", err)
	}

	translator, err := cfg.BuildTranslator()
	if err != nil {
		t.Fatalf("BuildTranslator: %v", err)
	}

	got, err := translator.Translate("es", "home.title")
	if err != nil {
		t.Fatalf("Translate with fallback: %v", err)
	}

	if got != "Welcome" {
		t.Fatalf("Translate() = %q want Welcome", got)
	}
}

func TestConfigBuildTranslatorNil(t *testing.T) {
	var cfg *Config
	translator, err := cfg.BuildTranslator()
	if err != ErrNotImplemented || translator != nil {
		t.Fatalf("expected ErrNotImplemented, got (%v, %v)", err, translator)
	}
}

func TestBuildTranslatorAppliesHooks(t *testing.T) {
	store := NewStaticStore(Translations{
		"en": {"home.title": "Welcome"},
	})

	var before, after int
	hook := TranslationHookFuncs{
		Before: func(ctx *TranslatorHookContext) { before++ },
		After: func(ctx *TranslatorHookContext) {
			after++
		},
	}

	cfg, err := NewConfig(
		WithStore(store),
		WithDefaultLocale("en"),
		WithTranslatorHooks(hook),
	)
	if err != nil {
		t.Fatalf("NewConfig: %v", err)
	}

	translator, err := cfg.BuildTranslator()
	if err != nil {
		t.Fatalf("BuildTranslator: %v", err)
	}

	if _, err := translator.Translate("en", "home.title"); err != nil {
		t.Fatalf("Translate: %v", err)
	}

	if before != 1 || after != 1 {
		t.Fatalf("expected hook counts 1/1, got %d/%d", before, after)
	}
}
