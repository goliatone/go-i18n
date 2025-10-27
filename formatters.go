package i18n

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

func FormatDate(locale string, t time.Time) string {
	return t.Format("2006-01-02")
}

func FormatDateTime(locale string, t time.Time) string {
	return t.Format(time.RFC3339)
}

func FormatTime(locale string, t time.Time) string {
	return t.Format("15:04")
}

func FormatCurrency(locale string, amount float64, currency string) string {
	formatted := FormatNumber(locale, amount, 2)
	currency = strings.TrimSpace(currency)
	if currency == "" {
		return formatted
	}
	return currency + " " + formatted
}

func FormatNumber(locale string, value float64, decimals int) string {
	prec := decimals
	if prec < 0 {
		prec = -1
	}
	return strconv.FormatFloat(value, 'f', prec, 64)
}

func FormatPercent(locale string, value float64, decimals int) string {
	if fn, ok := resolveFormatter[func(string, float64, int) string]("format_percent", locale); ok {
		return fn(locale, value, decimals)
	}
	return formatPercentISO(locale, value, decimals)
}

func FormatOrdinal(locale string, value int) string {
	if fn, ok := resolveFormatter[func(string, int) string]("format_ordinal", locale); ok {
		return fn(locale, value)
	}
	return formatOrdinalISO(locale, value)
}

func FormatList(locale string, items []string) string {
	if fn, ok := resolveFormatter[func(string, []string) string]("format_list", locale); ok {
		return fn(locale, items)
	}
	return formatListISO(locale, items)
}

func FormatPhone(locale, raw string) string {
	if fn, ok := resolveFormatter[func(string, string) string]("format_phone", locale); ok {
		return fn(locale, raw)
	}
	return formatPhoneISO(locale, raw)
}

func FormatMeasurement(locale string, value float64, unit string) string {
	if fn, ok := resolveFormatter[func(string, float64, string) string]("format_measurement", locale); ok {
		return fn(locale, value, unit)
	}
	return formatMeasurementISO(locale, value, unit)
}

func formatPercentISO(locale string, value float64, decimals int) string {
	formatted := FormatNumber(locale, value*100, decimals)
	return formatted + "%"
}

func formatOrdinalISO(_ string, value int) string {
	suffix := ordinalSuffic(value)
	return fmt.Sprintf("%d%s", value, suffix)
}

func formatListISO(_ string, items []string) string {
	count := len(items)
	switch count {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return items[0] + " and " + items[1]
	default:
		head := strings.Join(items[:count-1], ", ")
		return head + ", and " + items[count-1]
	}
}

func formatPhoneISO(_ string, raw string) string {
	return raw
}

func formatMeasurementISO(locale string, value float64, unit string) string {
	formatted := FormatNumber(locale, value, -1)
	unit = strings.TrimSpace(unit)
	if unit == "" {
		return formatted
	}
	return formatted + " " + unit
}

func ordinalSuffic(value int) string {
	abs := value
	if abs < 0 {
		abs = -abs
	}
	mod100 := abs % 100
	if mod100 >= 11 && mod100 <= 13 {
		return "th"
	}
	switch abs % 10 {
	case 1:
		return "st"
	case 2:
		return "nd"
	case 3:
		return "rd"
	default:
		return "th"
	}
}

var (
	defaultFormatterRegistryOnce sync.Once
	defaultFormatterRegistry     *FormatterRegistry
)

func globalFormatterRegistry() *FormatterRegistry {
	defaultFormatterRegistryOnce.Do(func() {
		resolver := NewStaticFallbackResolver()
		defaultFormatterRegistry = NewFormatterRegistry(
			WithFormatterRegistryResolver(resolver),
			WithFormatterRegistryLocales(defaultFormatterLocales...),
		)
	})
	return defaultFormatterRegistry
}

func ensureLocaleFallback(registry *FormatterRegistry, locale string) {
	if registry == nil || locale == "" {
		return
	}

	resolver, ok := registry.resolver.(*StaticFallbackResolver)
	if !ok || resolver == nil {
		return
	}

	registry.mu.Lock()
	defer registry.mu.Unlock()

	if len(resolver.Resolve(locale)) > 0 {
		return
	}

	if parents := localeParentChain(locale); len(parents) > 0 {
		resolver.Set(locale, parents...)
	}
}

func resolveFormatter[T any](name, locale string) (T, bool) {
	var zero T
	registry := globalFormatterRegistry()
	ensureLocaleFallback(registry, locale)

	fn, ok := registry.Formatter(name, locale)
	if !ok || fn == nil {
		return zero, false
	}

	typed, ok := fn.(T)
	if !ok {
		return zero, false
	}

	return typed, true
}
