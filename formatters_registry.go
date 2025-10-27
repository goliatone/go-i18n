package i18n

import (
	"fmt"
	"maps"
	"sync"
)

type FormatterCapabilities struct {
	Number      bool
	Currency    bool
	Date        bool
	DateTime    bool
	Time        bool
	List        bool
	Ordinal     bool
	Measurement bool
	Phone       bool
}

func mergeCapabilities(a, b FormatterCapabilities) FormatterCapabilities {
	return FormatterCapabilities{
		Number:      a.Number || b.Number,
		Currency:    a.Currency || b.Currency,
		Date:        a.Date || b.Date,
		DateTime:    a.DateTime || b.DateTime,
		Time:        a.Time || b.Time,
		List:        a.List || b.List,
		Ordinal:     a.Ordinal || b.Ordinal,
		Measurement: a.Measurement || b.Measurement,
		Phone:       a.Phone || b.Phone,
	}
}

type compositeTypedProvider struct {
	providers []TypedFormatterProvider
}

func newCompositeTypedProvider(providers ...TypedFormatterProvider) TypedFormatterProvider {
	flattened := make([]TypedFormatterProvider, 0, len(providers))
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		if composite, ok := provider.(*compositeTypedProvider); ok {
			flattened = append(flattened, composite.providers...)
			continue
		}
		flattened = append(flattened, provider)
	}

	switch len(flattened) {
	case 0:
		return nil
	case 1:
		return flattened[0]
	default:
		return &compositeTypedProvider{providers: flattened}
	}
}

func (c *compositeTypedProvider) Formatter(name string) (any, bool) {
	if c == nil {
		return nil, false
	}
	for i := len(c.providers) - 1; i >= 0; i-- {
		if fn, ok := c.providers[i].Formatter(name); ok {
			return fn, true
		}
	}
	return nil, false
}

func (c *compositeTypedProvider) FuncMap() map[string]any {
	result := make(map[string]any)
	if c == nil {
		return result
	}
	for _, provider := range c.providers {
		for key, value := range provider.FuncMap() {
			result[key] = value
		}
	}
	return result
}

func (c *compositeTypedProvider) Capabilities() FormatterCapabilities {
	var caps FormatterCapabilities
	if c == nil {
		return caps
	}
	for _, provider := range c.providers {
		caps = mergeCapabilities(caps, provider.Capabilities())
	}
	return caps
}

type FormatterProvider func(locale string) map[string]any

type TypedFormatterProvider interface {
	Formatter(name string) (any, bool)
	FuncMap() map[string]any
	Capabilities() FormatterCapabilities
}

// FormatRegistry manages formatter functions and locale specific overrides
type FormatterRegistry struct {
	mu            sync.RWMutex
	defaults      map[string]any
	overrides     map[string]map[string]any
	providers     map[string]FormatterProvider
	globals       map[string]any
	funcCache     map[string]map[string]any
	resolver      FallbackResolver
	locales       []string
	typed         map[string]TypedFormatterProvider
	caps          map[string]FormatterCapabilities
	rulesProvider *FormattingRulesProvider
}

var defaultFormatterLocales = []string{"en", "es"}

// formattingRulesData contains hardcoded formatting rules for the locales we ship by default.
// Keeping this map next to defaultFormatterLocales ensures new locales are added in a single place.
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

var defaultPhoneDialPlans = map[string]PhoneDialPlan{
	"en": {
		CountryCode:    "1",
		NationalPrefix: "1",
		Groups:         []int{3, 3, 4},
	},
	"es": {
		CountryCode: "34",
		Groups:      []int{3, 3, 3},
	},
}

type formatterRegistryConfig struct {
	resolver      FallbackResolver
	locales       []string
	providers     map[string]FormatterProvider
	typed         map[string]TypedFormatterProvider
	rulesProvider *FormattingRulesProvider
}

type FormatterRegistryOption func(*formatterRegistryConfig)

func WithFormatterRegistryResolver(resolver FallbackResolver) FormatterRegistryOption {
	return func(frc *formatterRegistryConfig) {
		frc.resolver = resolver
	}
}

func WithFormatterRegistryLocales(locales ...string) FormatterRegistryOption {
	return func(frc *formatterRegistryConfig) {
		frc.locales = append(frc.locales, locales...)
	}
}

func WithFormatterRegistryTypedProvider(locale string, provider TypedFormatterProvider) FormatterRegistryOption {
	return func(frc *formatterRegistryConfig) {
		if locale == "" || provider == nil {
			return
		}
		if frc.typed == nil {
			frc.typed = make(map[string]TypedFormatterProvider)
		}
		frc.typed[locale] = provider
	}
}

func WithFormatterRegistryProvider(locale string, provider FormatterProvider) FormatterRegistryOption {
	return func(frc *formatterRegistryConfig) {
		if locale == "" || provider == nil {
			return
		}
		if frc.providers == nil {
			frc.providers = make(map[string]FormatterProvider)
		}
		frc.providers[locale] = provider
	}
}

func WithFormattingRulesProvider(provider *FormattingRulesProvider) FormatterRegistryOption {
	return func(frc *formatterRegistryConfig) {
		frc.rulesProvider = provider
	}
}

// NewFormatterRegistry seeds a registry with default formatter implementations
func NewFormatterRegistry(opts ...FormatterRegistryOption) *FormatterRegistry {

	cfg := formatterRegistryConfig{}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(&cfg)
	}

	cfg.locales = normalizeLocales(cfg.locales)

	defaults := map[string]any{
		"format_date":        FormatDate,
		"format_datetime":    FormatDateTime,
		"format_time":        FormatTime,
		"format_currency":    FormatCurrency,
		"format_number":      FormatNumber,
		"format_percent":     formatPercentISO,
		"format_ordinal":     formatOrdinalISO,
		"format_list":        formatListISO,
		"format_phone":       formatPhoneISO,
		"format_measurement": formatMeasurementISO,
	}

	registry := &FormatterRegistry{
		defaults:      defaults,
		overrides:     make(map[string]map[string]any),
		providers:     make(map[string]FormatterProvider),
		resolver:      cfg.resolver,
		locales:       cfg.locales,
		rulesProvider: cfg.rulesProvider,
	}

	registry.registerDefaults(cfg.locales)
	registry.registerTypedProviders(cfg.typed)
	registry.registerConfiguredProviders(cfg.providers)
	registry.seedFallbacks()
	registry.ensureConfiguredProviders()

	return registry
}

func (r *FormatterRegistry) registerDefaults(locales []string) {
	// Use configured locales if provided, otherwise use defaults
	localesToRegister := locales
	if len(localesToRegister) == 0 {
		localesToRegister = defaultFormatterLocales
	}
	RegisterXTextFormatters(r, r.rulesProvider, localesToRegister...)
	RegisterCLDRFormatters(r, localesToRegister...)
}

func (r *FormatterRegistry) registerTypedProviders(providers map[string]TypedFormatterProvider) {
	for locale, provider := range providers {
		if locale == "" || provider == nil {
			continue
		}
		r.RegisterTypedProvider(locale, provider)
	}
}

func (r *FormatterRegistry) registerConfiguredProviders(providers map[string]FormatterProvider) {
	for locale, provider := range providers {
		if locale == "" || provider == nil {
			continue
		}
		r.RegisterProvider(locale, provider)
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

	if r.globals == nil {
		r.globals = make(map[string]any)
	}
	r.globals[name] = fn
	r.invalidateFuncCacheLocked()
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
	r.invalidateFuncCacheLocked()
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
	r.invalidateFuncCacheLocked()
}

func (r *FormatterRegistry) RegisterTypedProvider(locale string, provider TypedFormatterProvider) {
	if locale == "" || provider == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.typed == nil {
		r.typed = make(map[string]TypedFormatterProvider)
	}
	combined := newCompositeTypedProvider(r.typed[locale], provider)
	r.typed[locale] = combined

	if r.caps == nil {
		r.caps = make(map[string]FormatterCapabilities)
	}
	r.caps[locale] = mergeCapabilities(r.caps[locale], combined.Capabilities())

	if r.providers == nil {
		r.providers = make(map[string]FormatterProvider)
	}
	previous := r.providers[locale]
	r.providers[locale] = func(loc string) map[string]any {
		result := make(map[string]any)
		if previous != nil {
			if helpers := previous(loc); helpers != nil {
				maps.Copy(result, helpers)
			}
		}
		maps.Copy(result, combined.FuncMap())
		return result
	}
	r.invalidateFuncCacheLocked()
}

// Formatter returns the helper implementation for the given name and locale
func (r *FormatterRegistry) Formatter(name, locale string) (any, bool) {
	if name == "" {
		return nil, false
	}

	funcs := r.funcMapForLocale(locale)
	if funcs != nil {
		if fn, ok := funcs[name]; ok && fn != nil {
			return fn, true
		}
	}

	if r.defaults != nil {
		if fn, ok := r.defaults[name]; ok {
			return fn, true
		}
	}

	return nil, false
}

// FUncMap returns all helper functions applicable to the locale
func (r *FormatterRegistry) FuncMap(locale string) map[string]any {
	return cloneFuncMap(r.funcMapForLocale(locale))
}

func (r *FormatterRegistry) funcMapForLocale(locale string) map[string]any {
	key := locale

	r.mu.RLock()
	if r.funcCache != nil {
		if cached, ok := r.funcCache[key]; ok {
			r.mu.RUnlock()
			return cached
		}
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.funcCache == nil {
		r.funcCache = make(map[string]map[string]any)
	} else if cached, ok := r.funcCache[key]; ok {
		return cached
	}

	effective := locale
	if effective == "" {
		effective = r.defaultLocale()
	}

	result := make(map[string]any, len(r.defaults))
	if len(r.defaults) > 0 {
		maps.Copy(result, r.defaults)
	}

	if effective != "" {
		candidates := r.candidateLocales(effective)

		// REVERSE ITERATION: Start from least-specific (fallback) to most-specific (target)
		// This ensures more specific locales overwrite fallback locales
		for i := len(candidates) - 1; i >= 0; i-- {
			candidate := candidates[i]

			if r.typed != nil {
				if provider := r.typed[candidate]; provider != nil {
					maps.Copy(result, provider.FuncMap())
				}
			}

			if r.providers != nil {
				if provider, ok := r.providers[candidate]; ok && provider != nil {
					if helpers := provider(candidate); helpers != nil {
						maps.Copy(result, helpers)
					}
				}
			}

			if r.overrides != nil {
				if helpers, ok := r.overrides[candidate]; ok {
					maps.Copy(result, helpers)
				}
			}
		}
	}

	if r.globals != nil {
		maps.Copy(result, r.globals)
	}

	r.funcCache[key] = result
	return result
}

func (r *FormatterRegistry) invalidateFuncCacheLocked() {
	if r == nil {
		return
	}
	r.funcCache = nil
}

func (r *FormatterRegistry) candidateLocales(locale string) []string {
	if locale == "" {
		return nil
	}

	chain := []string{locale}
	if r.resolver != nil {
		for _, parent := range r.resolver.Resolve(locale) {
			if parent == "" || containsLocale(chain, parent) {
				continue
			}
			chain = append(chain, parent)
		}
	}

	return chain
}

func (r *FormatterRegistry) seedFallbacks() {
	resolver, ok := r.resolver.(*StaticFallbackResolver)
	if !ok || resolver == nil {
		return
	}

	for _, locale := range r.locales {
		if locale == "" {
			continue
		}

		if existing := resolver.Resolve(locale); len(existing) > 0 {
			continue
		}

		if parents := localeParentChain(locale); len(parents) > 0 {
			resolver.Set(locale, parents...)
		}
	}
}

func (r *FormatterRegistry) ensureConfiguredProviders() {
	if len(r.locales) == 0 {
		return
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, locale := range r.locales {
		if locale == "" {
			continue
		}
		if !r.hasProviderLocked(locale) {
			panic(fmt.Sprintf("i18n: formatter provider missing for locale %q", locale))
		}
	}
}

func (r *FormatterRegistry) hasProviderLocked(locale string) bool {
	if r.typed != nil {
		for _, candidate := range r.candidateLocales(locale) {
			if provider := r.typed[candidate]; provider != nil {
				return true
			}
		}
	}

	if r.providers == nil {
		return false
	}

	for _, candidate := range r.candidateLocales(locale) {
		if provider := r.providers[candidate]; provider != nil {
			return true
		}
	}

	return false
}

func containsLocale(locales []string, target string) bool {
	for _, locale := range locales {
		if locale == target {
			return true
		}
	}
	return false
}

func cloneFuncMap(source map[string]any) map[string]any {
	if len(source) == 0 {
		return map[string]any{}
	}

	target := make(map[string]any, len(source))
	for key, value := range source {
		target[key] = value
	}
	return target
}

func (r *FormatterRegistry) defaultLocale() string {
	if r == nil {
		return ""
	}

	if len(r.locales) > 0 {
		return r.locales[0]
	}

	if len(defaultFormatterLocales) > 0 {
		return defaultFormatterLocales[0]
	}

	return ""
}
