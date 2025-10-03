//go:build xtext

package i18n

import (
	"time"

	"golang.org/x/text/date"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// RegisterXTextFormatters registers locale aware formatters backed by golang.org/x/text
func RegisterXTextFormatters(registry *FormatterRegistry, locales ...string) {
	if registry == nil {
		return
	}

	for _, locale := range locales {
		locale := locale
		if locale == "" {
			continue
		}

		registry.RegisterProvider(locale, func(_ string) map[string]any {
			tag := language.Make(locale)
			printer := message.NewPrinter(tag)
			dateFormatter := date.NewFormatter(tag)

			return map[string]any{
				"format_number": func(_ string, value float64, decimals int) string {
					if decimals >= 0 {
						return printer.Sprintf("%.*f", decimals, value)
					}
					return printer.Sprintf("%f", value)
				},
				"format_currency": func(_ string, amount float64, currency string) string {
					if currency == "" {
						return printer.Sprintf("%g", amount)
					}
					return printer.Sprintf("%s %0.2f", currency, amount)
				},
				"format_date": func(_ string, t time.Time) string {
					return dateFormatter.Format(date.Long, t)
				},
				"format_datetime": func(_ string, t time.Time) string {
					return dateFormatter.Format(date.Long, t)
				},
				"format_time": func(_ string, t time.Time) string {
					return printer.Sprintf("%02d:%02d", t.Hour(), t.Minute())
				},
			}
		})
	}
}
