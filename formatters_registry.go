package i18n

import (
	"maps"
	"sync"
)

type FormatterProvider func(locale string) map[string]any

// FormatRegistry manages formatter functions and locale specific overrides
type FormatterRegistry struct {
	mu        sync.RWMutex
	defaults  map[string]any
	overrides map[string]map[string]any
	providers map[string]FormatterProvider
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
		overrides: make(map[string]map[string]any),
		providers: make(map[string]FormatterProvider),
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

	if r.overrides == nil {
		r.overrides = make(map[string]map[string]any)
	}

	helpers := r.overrides[locale]

	if helpers == nil {
		helpers = make(map[string]any)
		r.overrides[locale] = helpers
	}
	helpers[name] = fn
}

func (r *FormatterRegistry) RegisterProvider(locale string, provider FormatterProvider) {
	if locale == "" || provider == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.providers == nil {
		r.providers = make(map[string]FormatterProvider)
	}

	r.providers[locale] = provider
}

// Formatter returns the helper implementation for the given name and locale
func (r *FormatterRegistry) Formatter(name, locale string) (any, bool) {
	if name == "" {
		return nil, false
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if locale != "" {
		if fn := r.lookupLocaleLocked(name, locale); fn != nil {
			return fn, true
		}
	}

	fn, ok := r.defaults[name]
	return fn, ok
}

func (r *FormatterRegistry) lookupLocaleLocked(name, locale string) any {
	if r.overrides != nil {
		if helpers, ok := r.overrides[locale]; ok {
			if fn, ok := helpers[name]; ok {
				return fn
			}
		}
	}

	if r.providers != nil {
		if provider, ok := r.providers[locale]; ok && provider != nil {
			if helpers := provider(locale); helpers != nil {
				if fn, ok := helpers[name]; ok {
					return fn
				}
			}
		}
	}
	return nil
}

// FUncMap returns all helper functions applicable to the locale
func (r *FormatterRegistry) FuncMap(locale string) map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]any, len(r.defaults))
	maps.Copy(result, r.defaults)

	if locale != "" && r.providers != nil {
		if provider, ok := r.providers[locale]; ok && provider != nil {
			if helpers := provider(locale); helpers != nil {
				maps.Copy(result, helpers)
			}
		}
	}

	if locale != "" && r.overrides != nil {
		if helpers, ok := r.overrides[locale]; ok {
			maps.Copy(result, helpers)
		}
	}

	return result
}
