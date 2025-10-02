package i18n

import (
	"maps"
	"sync"
)

// FormatRegistry manages formatter functions and locale specific overrides
type FormatterRegistry struct {
	mu        sync.RWMutex
	defaults  map[string]any
	ovverides map[string]map[string]any
}

// NewFormatterRegistry seeds a registry with default formatter implementations
func NewFormatterRegistry() *FormatterRegistry {
	defaults := map[string]any{
		"format_date":        FormatDate,
		"format_datetime":    FormatDateTime,
		"format_time":        FormatTime,
		"format_currency":    FormatCurrency,
		"format_number":      FormatNumber,
		"format_percent":     FormatPercent,
		"format_ordinal":     FormatOrdinal,
		"format_list":        FormatList,
		"format_phone":       FormatPhone,
		"format_measurement": FormatMeasurement,
	}

	return &FormatterRegistry{
		defaults:  defaults,
		ovverides: make(map[string]map[string]any),
	}
}

// Register sets or replaces a default ipmlementtion for <name> helper
func (r *FormatterRegistry) Register(name string, fn any) {
	if name == "" || fn == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.defaults == nil {
		r.defaults = make(map[string]any)
	}

	r.defaults[name] = fn
}

// RegisterLocale registers a locale specific ovveride for the <name> helper
func (r *FormatterRegistry) RegisterLocale(locale, name string, fn any) {
	if locale == "" || name == "" || fn == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.ovverides == nil {
		r.ovverides = make(map[string]map[string]any)
	}

	helpers := r.ovverides[locale]

	if helpers == nil {
		helpers = make(map[string]any)
		r.ovverides[locale] = helpers
	}
	helpers[name] = fn
}

// Formatter returns the helper implementation for the given name and locale
func (r *FormatterRegistry) Formatter(name, locale string) (any, bool) {
	if name == "" {
		return nil, false
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if locale != "" && r.ovverides != nil {
		if helpers, ok := r.ovverides[locale]; ok {
			if fn, ok := helpers[name]; ok {
				return fn, true
			}
		}
	}

	fn, ok := r.defaults[name]
	return fn, ok
}

// FUncMap returns all helper functions applicable to the locale
func (r *FormatterRegistry) FuncMap(locale string) map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]any, len(r.defaults))
	maps.Copy(result, r.defaults)

	if locale != "" && r.ovverides != nil {
		if helpers, ok := r.ovverides[locale]; ok {
			maps.Copy(result, helpers)
		}
	}

	return result
}
