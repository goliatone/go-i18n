package i18n

import (
	"golang.org/x/text/language"
)

// formattingRulesData contains hardcoded formatting rules for supported locales
// In the future, this could be generated from CLDR data or loaded from JSON files
var formattingRulesData = map[string]FormattingRules{
	"en": {
		Locale: "en",
		DatePatterns: DatePatternRules{
			Pattern:    "{month} {day}, {year}",
			DayFirst:   false,
			MonthStyle: "name",
		},
		CurrencyRules: CurrencyFormatRules{
			Pattern:        "{symbol} {amount}",
			SymbolPosition: "before",
			DecimalSep:     ".",
			ThousandSep:    ",",
			Decimals:       2,
		},
		MonthNames: []string{
			"January", "February", "March", "April", "May", "June",
			"July", "August", "September", "October", "November", "December",
		},
		TimeFormat: TimeFormatRules{
			Use24Hour: false,
			Pattern:   "3:04 PM",
		},
	},
	"es": {
		Locale: "es",
		DatePatterns: DatePatternRules{
			Pattern:    "{day} de {month} de {year}",
			DayFirst:   true,
			MonthStyle: "name",
		},
		CurrencyRules: CurrencyFormatRules{
			Pattern:        "{amount} {symbol}",
			SymbolPosition: "after",
			DecimalSep:     ",",
			ThousandSep:    ".",
			Decimals:       2,
		},
		MonthNames: []string{
			"enero", "febrero", "marzo", "abril", "mayo", "junio",
			"julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre",
		},
		TimeFormat: TimeFormatRules{
			Use24Hour: true,
			Pattern:   "15:04",
		},
	},
}

// FormattingRulesProvider provides formatting rules for locales
type FormattingRulesProvider struct {
	rules    map[string]FormattingRules
	resolver FallbackResolver
}

// NewFormattingRulesProvider creates a provider from culture data
func NewFormattingRulesProvider(cultureData *CultureData, resolver FallbackResolver) *FormattingRulesProvider {
	rules := make(map[string]FormattingRules)

	// Start with hardcoded defaults as ultimate fallback
	for k, v := range formattingRulesData {
		rules[k] = v
	}

	// Override with culture data if provided
	if cultureData != nil && cultureData.FormattingRules != nil {
		for k, v := range cultureData.FormattingRules {
			rules[k] = v
		}
	}

	return &FormattingRulesProvider{
		rules:    rules,
		resolver: resolver,
	}
}

// Get loads formatting rules for a locale
// It tries exact match, then base language, then falls back to English
func (p *FormattingRulesProvider) Get(locale string) *FormattingRules {
	if p == nil || p.rules == nil {
		// Ultimate fallback: use hardcoded English
		rules := formattingRulesData["en"]
		return &rules
	}

	// Try exact match
	if rules, ok := p.rules[locale]; ok {
		return &rules
	}

	// Try resolver candidates
	if p.resolver != nil {
		for _, candidate := range p.resolver.Resolve(locale) {
			if rules, ok := p.rules[candidate]; ok {
				return &rules
			}
		}
	}

	// Try base language
	tag := language.Make(locale)
	base, _ := tag.Base()
	baseStr := base.String()
	if rules, ok := p.rules[baseStr]; ok {
		return &rules
	}

	// Fallback to English
	if rules, ok := p.rules["en"]; ok {
		return &rules
	}

	// Ultimate fallback: hardcoded English
	rules := formattingRulesData["en"]
	return &rules
}

// loadFormattingRules loads formatting rules for a locale (deprecated, use FormattingRulesProvider)
// It tries exact match, then base language, then falls back to English
func loadFormattingRules(locale string) *FormattingRules {
	// Try exact match
	if rules, ok := formattingRulesData[locale]; ok {
		return &rules
	}

	// Try base language
	tag := language.Make(locale)
	base, _ := tag.Base()
	baseStr := base.String()
	if rules, ok := formattingRulesData[baseStr]; ok {
		return &rules
	}

	// Fallback to English
	rules := formattingRulesData["en"]
	return &rules
}
