package i18n

import "testing"

func TestFormatterRegistryProvider(t *testing.T) {
	registry := NewFormatterRegistry()

	registry.RegisterProvider("fr", func(locale string) map[string]any {
		return map[string]any{
			"format_number": func(_ string, value float64, decimals int) string {
				return "fr-" + FormatNumber("", value, decimals)
			},
		}
	})

	fnAny, ok := registry.Formatter("format_number", "fr")
	if !ok {
		t.Fatalf("expected provider formatter")
	}

	fn := fnAny.(func(string, float64, int) string)
	if got := fn("fr", 12.3, 1); got != "fr-12.3" {
		t.Fatalf("provider formatter = %q", got)
	}

	funcs := registry.FuncMap("fr")
	if funcs["format_number"].(func(string, float64, int) string)("fr", 1.2, 1) != "fr-1.2" {
		t.Fatalf("func map should reflect provider override")
	}
}

func TestFormatterRegistryProviderOverrideOrder(t *testing.T) {
	registry := NewFormatterRegistry()

	registry.RegisterProvider("fr", func(locale string) map[string]any {
		return map[string]any{
			"format_number": func(_ string, value float64, decimals int) string {
				return "provider"
			},
		}
	})

	registry.RegisterLocale("fr", "format_number", func(_ string, value float64, decimals int) string {
		return "override"
	})

	fnAny, ok := registry.Formatter("format_number", "fr")
	if !ok {
		t.Fatal("expected formatter")
	}

	fn := fnAny.(func(string, float64, int) string)
	if got := fn("fr", 1, 0); got != "override" {
		t.Fatalf("override should win, got %q", got)
	}
}
