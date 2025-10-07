package i18n

import (
	"strings"
	"testing"
)

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

func TestFormatterRegistryCompositeTypedProvider(t *testing.T) {
	registry := NewFormatterRegistry()

	p1 := newStubTypedProvider(map[string]any{
		"format_percent": func(_ string, value float64, decimals int) string {
			return "provider-one"
		},
	}, FormatterCapabilities{Number: true})

	p2 := newStubTypedProvider(map[string]any{
		"format_measurement": func(_ string, value float64, unit string) string {
			return "provider-two"
		},
	}, FormatterCapabilities{Measurement: true})

	registry.RegisterTypedProvider("zz", p1)
	registry.RegisterTypedProvider("zz", p2)

	percentAny, ok := registry.Formatter("format_percent", "zz")
	if !ok {
		t.Fatal("expected composite percent formatter")
	}
	if got := percentAny.(func(string, float64, int) string)("zz", 1.0, 0); got != "provider-one" {
		t.Fatalf("composite percent formatter = %q", got)
	}

	measurementAny, ok := registry.Formatter("format_measurement", "zz")
	if !ok {
		t.Fatal("expected composite measurement formatter")
	}
	if got := measurementAny.(func(string, float64, string) string)("zz", 1.0, "kg"); got != "provider-two" {
		t.Fatalf("composite measurement formatter = %q", got)
	}

	funcs := registry.FuncMap("zz")
	if funcs["format_percent"].(func(string, float64, int) string)("zz", 1.0, 0) != "provider-one" {
		t.Fatal("func map should expose first provider formatter")
	}
	if funcs["format_measurement"].(func(string, float64, string) string)("zz", 1.0, "kg") != "provider-two" {
		t.Fatal("func map should expose second provider formatter")
	}
}

func TestFormatterRegistryMissingProviderPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when configured locale lacks provider")
		}
	}()

	NewFormatterRegistry(
		WithFormatterRegistryResolver(NewStaticFallbackResolver()),
		WithFormatterRegistryLocales("fr"),
	)
}

func TestFormatterRegistryCLDRBundles(t *testing.T) {
	registry := NewFormatterRegistry()

	fmEs := registry.FuncMap("es")
	listFn, ok := fmEs["format_list"].(func(string, []string) string)
	if !ok {
		t.Fatalf("format_list signature mismatch: %T", fmEs["format_list"])
	}

	if got := listFn("es", []string{"uno", "dos", "tres"}); got != "uno, dos y tres" {
		t.Fatalf("format_list es = %q", got)
	}

	ordinalFn, ok := fmEs["format_ordinal"].(func(string, int) string)
	if !ok {
		t.Fatalf("format_ordinal signature mismatch: %T", fmEs["format_ordinal"])
	}

	if got := ordinalFn("es", 1); got != "1º" {
		t.Fatalf("format_ordinal es = %q", got)
	}

	measurementFn, ok := fmEs["format_measurement"].(func(string, float64, string) string)
	if !ok {
		t.Fatalf("format_measurement signature mismatch: %T", fmEs["format_measurement"])
	}

	if got := measurementFn("es", 12.34, "km"); got != "12,34 kilómetros" {
		t.Fatalf("format_measurement es = %q", got)
	}

	phoneFn, ok := fmEs["format_phone"].(func(string, string) string)
	if !ok {
		t.Fatalf("format_phone signature mismatch: %T", fmEs["format_phone"])
	}

	if got := phoneFn("es", "+34123456789"); got != "+34123456789" {
		t.Fatalf("format_phone es = %q", got)
	}

	fmEn := registry.FuncMap("en")
	ordinalEn := fmEn["format_ordinal"].(func(string, int) string)
	if got := ordinalEn("en", 21); got != "21st" {
		t.Fatalf("format_ordinal en = %q", got)
	}

	listEn := fmEn["format_list"].(func(string, []string) string)
	if got := listEn("en", []string{"a", "b", "c"}); got != "a, b, and c" {
		t.Fatalf("format_list en = %q", got)
	}

	measurementEn := fmEn["format_measurement"].(func(string, float64, string) string)
	if got := measurementEn("en", 12.34, "km"); got != "12.34 kilometers" {
		t.Fatalf("format_measurement en = %q", got)
	}

	phoneEn := fmEn["format_phone"].(func(string, string) string)
	if got := phoneEn("en", "1234567890"); got != "1234567890" {
		t.Fatalf("format_phone en = %q", got)
	}
}

func TestFormatterRegistryLocaleFallbackResolution(t *testing.T) {
	resolver := NewStaticFallbackResolver()
	registry := NewFormatterRegistry(
		WithFormatterRegistryResolver(resolver),
		WithFormatterRegistryLocales("en", "en-GB"),
	)

	registry.RegisterLocale("en", "format_list", func(_ string, items []string) string {
		return "fallback-" + strings.Join(items, "|")
	})

	fnAny, ok := registry.Formatter("format_list", "en-GB")
	if !ok {
		t.Fatal("expected formatter via fallback chain")
	}
	got := fnAny.(func(string, []string) string)("en-GB", []string{"a", "b"})
	if got != "fallback-a|b" {
		t.Fatalf("fallback formatting mismatch: %q", got)
	}
}

func TestFormatterRegistryFallsBackToDefaults(t *testing.T) {
	registry := NewFormatterRegistry()

	fnAny, ok := registry.Formatter("format_phone", "fr")
	if !ok {
		t.Fatal("expected default formatter for format_phone")
	}

	fn := fnAny.(func(string, string) string)
	if got := fn("fr", "123"); got != "123" {
		t.Fatalf("default formatter mismatch: %q", got)
	}
}

type stubTypedProvider struct {
	funcs map[string]any
	caps  FormatterCapabilities
}

func newStubTypedProvider(funcs map[string]any, caps FormatterCapabilities) *stubTypedProvider {
	return &stubTypedProvider{
		funcs: funcs,
		caps:  caps,
	}
}

func (p *stubTypedProvider) Formatter(name string) (any, bool) {
	if p == nil || p.funcs == nil {
		return nil, false
	}
	fn, ok := p.funcs[name]
	return fn, ok
}

func (p *stubTypedProvider) FuncMap() map[string]any {
	result := make(map[string]any, len(p.funcs))
	for key, value := range p.funcs {
		result[key] = value
	}
	return result
}

func (p *stubTypedProvider) Capabilities() FormatterCapabilities {
	if p == nil {
		return FormatterCapabilities{}
	}
	return p.caps
}
