package i18n

import (
	"fmt"
	"maps"
	"sort"
	"sync"
)

type FormatterCapabilities struct {
	Number    bool
	Currencty bool
	Date      bool
	DateTime  bool
	Time      bool
}

type FormatterProvider func(locale string) map[string]any

type TypedFormatterProvider interface {
	Formatter(name string) (any, bool)
	FuncMap() map[string]any
	Capabilities() FormatterCapabilities
}

// FormatRegistry manages formatter functions and locale specific overrides
type FormatterRegistry struct {
	mu        sync.RWMutex
	defaults  map[string]any
	overrides map[string]map[string]any
	providers map[string]FormatterProvider
	resolver  FallbackResolver
	locales   []string
	typed     map[string]TypedFormatterProvider
	caps      map[string]FormatterCapabilities
}

var defaultFormatterLocales = []string{"en", "es"}

type formatterRegistryConfig struct {
	resolver  FallbackResolver
	locales   []string
	providers map[string]FormatterProvider
	typed     map[string]TypedFormatterProvider
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
		"format_percent":     FormatPercent,
		"format_ordinal":     FormatOrdinal,
		"format_list":        FormatList,
		"format_phone":       FormatPhone,
		"format_measurement": FormatMeasurement,
	}

	registry := &FormatterRegistry{
		defaults:  defaults,
		overrides: make(map[string]map[string]any),
		providers: make(map[string]FormatterProvider),
		resolver:  cfg.resolver,
		locales:   cfg.locales,
	}

	registry.registerDefaults()
	registry.registerTypedProviders(cfg.typed)
	registry.registerConfiguredProviders(cfg.providers)
	registry.seedFallbacks()
	registry.ensureConfiguredProviders()

	return registry
}

func (r *FormatterRegistry) registerDefaults() {
	RegisterXTextFormatters(r, defaultFormatterLocales...)
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

func (r *FormatterRegistry) RegisterTypedProvider(locale string, provider TypedFormatterProvider) {
	if locale == "" || provider == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.typed == nil {
		r.typed = make(map[string]TypedFormatterProvider)
	}
	r.typed[locale] = provider

	if r.caps == nil {
		r.caps = make(map[string]FormatterCapabilities)
	}
	r.caps[locale] = provider.Capabilities()

	if r.providers == nil {
		r.providers = make(map[string]FormatterProvider)
	}

	r.providers[locale] = func(string) map[string]any {
		return cloneFuncMap(provider.FuncMap())
	}
}

// Formatter returns the helper implementation for the given name and locale
func (r *FormatterRegistry) Formatter(name, locale string) (any, bool) {
	if name == "" {
		return nil, false
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, candidate := range r.candidateLocales(locale) {
		if r.typed != nil {
			if provider := r.typed[candidate]; provider != nil {
				if fn, ok := provider.Formatter(name); ok {
					return fn, true
				}
			}
		}

		if fn := r.lookupLocaleLocked(name, candidate); fn != nil {
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

	for _, candidate := range r.candiateLocales(locale) {
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

	return result
}

func (r *FormatterRegistry) candiateLocales(locale string) []string {
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
		if parents := deriveLocaleParents(locale); len(parents) > 0 {
			resolver.Set(locale, parents...)
		}
	}
}

func (r *FormatterRegistry) ensureConfiguredProviders() {
	if len(r.locales) == 0 {
		return
	}

	r.mu.RLock()
	defer r.mu.Unlock()

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
		for _, candidate := range r.candiateLocales(locale) {
			if provider := r.typed[candidate]; provider != nil {
				return true
			}
		}
	}

	if r.providers == nil {
		return false
	}

	for _, candidate := range r.candiateLocales(locale) {
		if provider := r.providers[candidate]; provider != nil {
			return true
		}
	}
	return false
}

func normalizeLocales(locales []string) []string {
	if len(locales) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(locales))
	result := make([]string, 0, len(locales))
	for _, locale := range locales {
		if locale == "" {
			continue
		}
		if _, exists := seen[locale]; exists {
			continue
		}
		seen[locale] = struct{}{}
		result = append(result, locale)
	}

	sort.Strings(result)
	return result
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
