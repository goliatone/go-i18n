package i18n

import (
	"testing"
)

func TestGreekCurrencyFormatting_Debug(t *testing.T) {
	// Load culture data
	loader := NewCultureDataLoader("examples/web/locales/culture_data.json")
	cultureData, err := loader.Load()
	if err != nil {
		t.Fatalf("Failed to load culture data: %v", err)
	}

	// Check if Greek formatting rules are loaded
	if rules, ok := cultureData.FormattingRules["el"]; ok {
		t.Logf("Greek formatting rules found:")
		t.Logf("  Symbol position: %s", rules.CurrencyRules.SymbolPosition)
		t.Logf("  Decimal separator: %s", rules.CurrencyRules.DecimalSep)
		t.Logf("  Thousand separator: %s", rules.CurrencyRules.ThousandSep)
	} else {
		t.Fatal("Greek formatting rules not found in culture data")
	}

	// Create provider with culture data
	resolver := NewStaticFallbackResolver()
	rulesProvider := NewFormattingRulesProvider(cultureData, resolver)

	// Get Greek rules
	greekRules := rulesProvider.Get("el")
	if greekRules == nil {
		t.Fatal("Greek rules not returned by provider")
	}

	t.Logf("Provider returned Greek rules:")
	t.Logf("  Locale: %s", greekRules.Locale)
	t.Logf("  Symbol position: %s", greekRules.CurrencyRules.SymbolPosition)

	// Create xtext provider
	provider := newXTextProvider("el", rulesProvider)
	if provider.rules == nil {
		t.Fatal("XText provider has nil rules")
	}

	t.Logf("XText provider rules:")
	t.Logf("  Locale: %s", provider.rules.Locale)
	t.Logf("  Symbol position: %s", provider.rules.CurrencyRules.SymbolPosition)

	// Test currency formatting
	result := provider.formatCurrency("el", 129.95, "EUR")
	t.Logf("Formatted currency: %s", result)

	// Check if symbol is after
	if greekRules.CurrencyRules.SymbolPosition != "after" {
		t.Errorf("Expected symbol_position='after', got '%s'", greekRules.CurrencyRules.SymbolPosition)
	}
}
