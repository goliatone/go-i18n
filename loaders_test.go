package i18n

import (
	"path/filepath"
	"testing"
)

func TestFileLoaderJSONAndYAML(t *testing.T) {
	loader := NewFileLoader(
		filepath.Join(".tmp", "testdata", "loader_en.json"),
		filepath.Join(".tmp", "testdata", "loader_es.yaml"),
	)

	translations, err := loader.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(translations) != 2 {
		t.Fatalf("expected 2 locales, got %d", len(translations))
	}

	if translations["es"]["home.title"] != "Bienvenido" {
		t.Fatalf("unexpected translation for es: %v", translations["es"]["home.title"])
	}

	if translations["en"]["home.greeting"] != "Hello %s" {
		t.Fatalf("unexpected translation for en: %v", translations["en"]["home.greeting"])
	}
}

func TestFileLoaderUnsupportedExtension(t *testing.T) {
	loader := NewFileLoader(filepath.Join(".tmp", "testdata", "loader_en.json"), "unsupported.txt")

	if _, err := loader.Load(); err == nil {
		t.Fatal("expected error for unsupported extension")
	}
}

func TestFileLoaderIntegration(t *testing.T) {
	loader := NewFileLoader(
		filepath.Join(".tmp", "testdata", "loader_en.json"),
		filepath.Join(".tmp", "testdata", "loader_es.yaml"),
	)

	cfg, err := NewConfig(
		WithLoader(loader),
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

	if got, err := translator.Translate("es", "home.title"); err != nil || got != "Bienvenido" {
		t.Fatalf("Translate es/home.title = %q,%v", got, err)
	}

	if got, err := translator.Translate("fr", "home.greeting", "Carlos"); err != nil || got != "Hello Carlos" {
		t.Fatalf("Translate fallback = %q,%v", got, err)
	}
}
