package i18n

import (
	"fmt"
	"strings"
	"time"

	"golang.org/x/text/currency"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

// RegisterXTextFormatters registers locale aware formatters backed by golang.org/x/text
func RegisterXTextFormatters(registry *FormatterRegistry, locales ...string) {
	if registry == nil {
		return
	}

	for _, locale := range locales {
		trimmed := strings.TrimSpace(locale)
		if trimmed == "" {
			continue
		}

		registry.RegisterTypedProvider(trimmed, newXTextProvider(trimmed))
	}
}

type xtextProvider struct {
	locale       string
	tag          language.Tag
	printer      *message.Printer
	funcs        map[string]any
	capabilities FormatterCapabilities
}

func newXTextProvider(locale string) *xtextProvider {
	tag := language.Make(locale)
	provider := &xtextProvider{
		locale:  locale,
		tag:     tag,
		printer: message.NewPrinter(tag),
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
	opts := []number.Option{}
	if decimals >= 0 {
		opts = append(opts, number.MinFractionDigits(decimals), number.MaxFractionDigits(decimals))
	}
	return p.printer.Sprintf("%v", number.Decimal(value, opts...))
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

	value := unit.Amount(amount)
	return p.printer.Sprintf("%v", currency.Symbol(value))
}

func (p *xtextProvider) formatDate(_ string, t time.Time) string {
	if p.isSpanish() {
		return fmt.Sprintf("%d de %s de %d", t.Day(), p.monthName(t.Month()), t.Year())
	}

	return fmt.Sprintf("%s %d, %d", p.monthName(t.Month()), t.Day(), t.Year())
}

func (p *xtextProvider) formatTime(_ string, t time.Time) string {
	if p.uses12HourClock() {
		return t.Format("3:04 PM")
	}
	return t.Format("15:04")
}

func (p *xtextProvider) formatDateTime(locale string, t time.Time) string {
	return fmt.Sprintf("%s %s", p.formatDate(locale, t), p.formatTime(locale, t))
}

func (p *xtextProvider) monthName(month time.Month) string {
	if p.isSpanish() {
		return spanishMonths[month-1]
	}
	return month.String()
}

var spanishBase, _ = language.Spanish.Base()

func (p *xtextProvider) isSpanish() bool {
	base, _ := p.tag.Base()
	return base == spanishBase
}

func (p *xtextProvider) uses12HourClock() bool {
	base, _ := p.tag.Base()
	return base != spanishBase
}

var spanishMonths = []string{
	"enero",
	"febrero",
	"marzo",
	"abril",
	"mayo",
	"junio",
	"julio",
	"agosto",
	"septiembre",
	"octubre",
	"noviembre",
	"diciembre",
}
