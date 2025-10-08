package i18n

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/currency"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

// RegisterXTextFormatters registers locale aware formatters backed by golang.org/x/text
func RegisterXTextFormatters(registry *FormatterRegistry, rulesProvider *FormattingRulesProvider, locales ...string) {
	if registry == nil {
		return
	}

	for _, locale := range locales {
		trimmed := strings.TrimSpace(locale)
		if trimmed == "" {
			continue
		}

		registry.RegisterTypedProvider(trimmed, newXTextProvider(trimmed, rulesProvider))
	}
}

type xtextProvider struct {
	locale       string
	tag          language.Tag
	printer      *message.Printer
	rules        *FormattingRules
	funcs        map[string]any
	capabilities FormatterCapabilities
}

func newXTextProvider(locale string, rulesProvider *FormattingRulesProvider) *xtextProvider {
	tag := language.Make(locale)

	// Load formatting rules
	var rules *FormattingRules
	if rulesProvider != nil {
		rules = rulesProvider.Get(locale)
	} else {
		rules = loadFormattingRules(locale)
	}

	provider := &xtextProvider{
		locale:  locale,
		tag:     tag,
		printer: message.NewPrinter(tag),
		rules:   rules,
	}

	provider.capabilities = FormatterCapabilities{
		Number:   true,
		Currency: true,
		Date:     true,
		DateTime: true,
		Time:     true,
	}

	provider.funcs = map[string]any{
		"format_number":   provider.formatNumber,
		"format_currency": provider.formatCurrency,
		"format_date":     provider.formatDate,
		"format_datetime": provider.formatDateTime,
		"format_time":     provider.formatTime,
	}

	return provider
}

func (p *xtextProvider) Formatter(name string) (any, bool) {
	if p == nil {
		return nil, false
	}
	fn, ok := p.funcs[name]
	return fn, ok
}

func (p *xtextProvider) FuncMap() map[string]any {
	if p == nil {
		return nil
	}
	return cloneFuncMap(p.funcs)
}

func (p *xtextProvider) Capabilities() FormatterCapabilities {
	if p == nil {
		return FormatterCapabilities{}
	}
	return p.capabilities
}

func (p *xtextProvider) formatNumber(_ string, value float64, decimals int) string {
	// If we have custom formatting rules, use them for decimal/thousand separators
	if p.rules != nil {
		return p.formatNumberWithRules(value, decimals)
	}

	// Fallback to golang.org/x/text formatting
	opts := []number.Option{}
	if decimals >= 0 {
		opts = append(opts, number.MinFractionDigits(decimals), number.MaxFractionDigits(decimals))
	}
	return p.printer.Sprintf("%v", number.Decimal(value, opts...))
}

func (p *xtextProvider) formatNumberWithRules(value float64, decimals int) string {
	if decimals < 0 {
		decimals = 2
	}

	// Format with standard Go formatting first
	formatted := fmt.Sprintf("%.*f", decimals, value)

	// Apply custom separators
	decimalSep := p.rules.CurrencyRules.DecimalSep
	thousandSep := p.rules.CurrencyRules.ThousandSep

	// Replace decimal separator
	if decimalSep != "." {
		formatted = strings.Replace(formatted, ".", decimalSep, 1)
	}

	// Add thousand separators if needed
	if thousandSep != "" {
		// Split into integer and decimal parts
		parts := strings.Split(formatted, decimalSep)
		integerPart := parts[0]

		// Add thousand separators to integer part (from right to left)
		if len(integerPart) > 3 {
			var result strings.Builder
			for i, digit := range integerPart {
				if i > 0 && (len(integerPart)-i)%3 == 0 {
					result.WriteString(thousandSep)
				}
				result.WriteRune(digit)
			}
			integerPart = result.String()
		}

		// Reconstruct the number
		if len(parts) > 1 {
			formatted = integerPart + decimalSep + parts[1]
		} else {
			formatted = integerPart
		}
	}

	return formatted
}

func (p *xtextProvider) formatCurrency(_ string, amount float64, code string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return p.formatNumber(p.locale, amount, 2)
	}

	unit, err := currency.ParseISO(code)
	if err != nil || unit.String() == "XXX" {
		return strings.ToUpper(code) + " " + p.formatNumber(p.locale, amount, 2)
	}

	// Format the amount using locale-specific number formatting
	formattedAmount := p.formatNumber(p.locale, amount, 2)

	// Get the currency symbol using golang.org/x/text/currency
	// Try to extract symbol from x/text formatting
	value := unit.Amount(amount)
	fullFormat := p.printer.Sprintf("%v", currency.Symbol(value))

	// Extract symbol from the full format
	// The fullFormat contains both the symbol and a formatted amount from x/text
	opts := []number.Option{number.MinFractionDigits(2), number.MaxFractionDigits(2)}
	xtextAmount := p.printer.Sprintf("%v", number.Decimal(amount, opts...))

	// Remove the x/text formatted amount to get the symbol
	symbol := strings.TrimSpace(strings.ReplaceAll(fullFormat, xtextAmount, ""))

	// If symbol extraction failed or returned currency code, try a known locale
	if symbol == "" || symbol == unit.String() {
		// Try with English locale as fallback for symbol extraction
		englishPrinter := message.NewPrinter(language.English)
		englishFormat := englishPrinter.Sprintf("%v", currency.Symbol(value))
		englishAmount := englishPrinter.Sprintf("%v", number.Decimal(amount, opts...))
		symbol = strings.TrimSpace(strings.ReplaceAll(englishFormat, englishAmount, ""))

		// Still no symbol? Use currency code
		if symbol == "" {
			symbol = unit.String()
		}
	}

	// Apply locale-specific symbol placement from our formatting rules
	if p.rules != nil && p.rules.CurrencyRules.SymbolPosition == "after" {
		return formattedAmount + " " + symbol
	}

	// Default: symbol before amount
	return symbol + " " + formattedAmount
}

func (p *xtextProvider) formatDate(_ string, t time.Time) string {
	pattern := p.rules.DatePatterns.Pattern
	monthName := p.rules.MonthNames[t.Month()-1]

	result := strings.ReplaceAll(pattern, "{day}", strconv.Itoa(t.Day()))
	result = strings.ReplaceAll(result, "{month}", monthName)
	result = strings.ReplaceAll(result, "{year}", strconv.Itoa(t.Year()))

	return result
}

func (p *xtextProvider) formatTime(_ string, t time.Time) string {
	if p.rules.TimeFormat.Use24Hour {
		return t.Format("15:04")
	}
	return t.Format("3:04 PM")
}

func (p *xtextProvider) formatDateTime(locale string, t time.Time) string {
	return fmt.Sprintf("%s %s", p.formatDate(locale, t), p.formatTime(locale, t))
}
