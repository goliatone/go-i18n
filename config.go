package i18n

import "fmt"

// Config captures translator and formatter setup
type Config struct {
	DefaultLocale       string
	Locales             []string
	Loader              Loader
	Store               Store
	Resolver            FallbackResolver
	Formatter           Formatter
	Hooks               []TranslationHook
	enablePlural        bool
	pluralRules         []string
	seedPluralFallbacks bool

	formatterLocales   []string
	formatterProviders map[string]FormatterProvider
	formatterRegistry  *FormatterRegistry

	cultureDataPath  string
	cultureOverrides map[string]string
	cultureService   CultureService
	cultureData      *CultureData
	localeCatalog    *LocaleCatalog
}

type pluralRuleLoader interface {
	WithPluralRules(paths ...string) Loader
}

// Option mutates Config during construction
type Option func(*Config) error

// NewConfig builds Config via supplied options
func NewConfig(opts ...Option) (*Config, error) {
	cfg := &Config{}

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}

	if err := cfg.applyLocaleCatalog(); err != nil {
		return nil, err
	}

	cfg.normalizeLocales()
	cfg.applyPluralRuleOptions()

	if cfg.Store == nil {
		if cfg.Loader != nil {
			store, err := NewStaticStoreFromLoader(cfg.Loader)
			if err != nil {
				return nil, err
			}
			cfg.Store = store
		} else {
			cfg.Store = NewStaticStore(nil)
		}
	}

	if cfg.Resolver == nil {
		cfg.Resolver = NewStaticFallbackResolver()
	}

	if cfg.Formatter == nil {
		cfg.Formatter = FormatterFunc(sprintfFormatter)
	}

	if cfg.DefaultLocale == "" && len(cfg.Locales) > 0 {
		cfg.DefaultLocale = cfg.Locales[0]
	}

	return cfg, nil
}

// WithDefaultLocale sets the default locale in Config
func WithDefaultLocale(locale string) Option {
	return func(c *Config) error {
		c.DefaultLocale = locale
		return nil
	}
}

// WithLocales registers supported locales
func WithLocales(locales ...string) Option {
	return func(c *Config) error {
		c.Locales = append(c.Locales, locales...)
		return nil
	}
}

func WithLoader(loader Loader) Option {
	return func(c *Config) error {
		c.Loader = loader
		return nil
	}
}

func WithStore(store Store) Option {
	return func(c *Config) error {
		c.Store = store
		return nil
	}
}

func WithFallbackResolver(resolver FallbackResolver) Option {
	return func(c *Config) error {
		c.Resolver = resolver
		return nil
	}
}

func WithFallback(locale string, fallbacks ...string) Option {
	return func(c *Config) error {
		if locale == "" {
			return nil
		}
		resolver, ok := c.Resolver.(*StaticFallbackResolver)
		if !ok {
			if c.Resolver != nil {
				return nil
			}
			resolver = NewStaticFallbackResolver()
			c.Resolver = resolver
		}
		resolver.Set(locale, fallbacks...)
		return nil
	}
}

func WithFormatter(formatter Formatter) Option {
	return func(c *Config) error {
		c.Formatter = formatter
		return nil
	}
}

func WithFormatterLocales(locales ...string) Option {
	return func(c *Config) error {
		if len(locales) == 0 {
			return nil
		}
		c.formatterLocales = append(c.formatterLocales, locales...)
		c.formatterLocales = normalizeLocales(c.formatterLocales)
		c.formatterRegistry = nil
		return nil
	}
}

func WithFormatterProvider(locale string, provider FormatterProvider) Option {
	return func(c *Config) error {
		if locale == "" || provider == nil {
			return nil
		}
		if c.formatterProviders == nil {
			c.formatterProviders = make(map[string]FormatterProvider)
		}
		c.formatterProviders[locale] = provider
		c.formatterRegistry = nil
		return nil
	}
}

func WithTranslatorHooks(hooks ...TranslationHook) Option {
	return func(c *Config) error {
		for _, hook := range hooks {
			if hook == nil {
				continue
			}
			c.Hooks = append(c.Hooks, hook)
		}
		return nil
	}
}

// EnablePluralization wires pluralization defaults, optionally registering CLDR rule fixtures via loader aware options.
func EnablePluralization(rulePaths ...string) Option {
	return func(c *Config) error {
		c.enablePlural = true
		if len(rulePaths) > 0 {
			c.pluralRules = append(c.pluralRules, rulePaths...)
		}
		return nil
	}
}

// EnablePluralFallbackSeeding opts into automatic fallback chain seeding when pluralization is enabled.
func EnablePluralFallbackSeeding() Option {
	return func(c *Config) error {
		c.seedPluralFallbacks = true
		return nil
	}
}

// WithCultureData configures culture data loading
func WithCultureData(path string) Option {
	return func(c *Config) error {
		c.cultureDataPath = path
		c.cultureService = nil // Invalidate cached service
		c.cultureData = nil
		c.localeCatalog = nil
		return nil
	}
}

// WithCultureOverride adds locale-specific culture data override
func WithCultureOverride(locale, path string) Option {
	return func(c *Config) error {
		if c.cultureOverrides == nil {
			c.cultureOverrides = make(map[string]string)
		}
		c.cultureOverrides[locale] = path
		c.cultureService = nil // Invalidate cached service
		c.cultureData = nil
		c.localeCatalog = nil
		return nil
	}
}

func (cfg *Config) BuildTranslator() (Translator, error) {
	if cfg == nil {
		return nil, ErrNotImplemented
	}

	base, err := NewSimpleTranslator(cfg.Store,
		WithTranslatorDefaultLocale(cfg.DefaultLocale),
		WithTranslatorFormatter(cfg.Formatter),
		WithTranslatorFallbackResolver(cfg.Resolver))
	if err != nil {
		return nil, err
	}

	var translator Translator = base

	if len(cfg.Hooks) > 0 {
		translator = WrapTranslatorWithHooks(translator, cfg.Hooks...)
	}

	cfg.seedResolverFallbacks()
	cfg.ensureFormatterRegistry()

	return translator, nil
}

func (cfg *Config) normalizeLocales() {
	cfg.Locales = normalizeLocales(cfg.Locales)
}

func (cfg *Config) FormatterRegistry() *FormatterRegistry {
	if cfg == nil {
		return nil
	}
	cfg.ensureFormatterRegistry()
	return cfg.formatterRegistry
}

// CultureService returns the culture service
func (cfg *Config) CultureService() CultureService {
	if cfg == nil {
		return nil
	}
	cfg.ensureCultureService()
	return cfg.cultureService
}

// LocaleCatalog exposes the immutable locale metadata snapshot loaded from culture data.
func (cfg *Config) LocaleCatalog() *LocaleCatalog {
	if cfg == nil {
		return nil
	}
	return cfg.localeCatalog
}

func (cfg *Config) TemplateHelpers(t Translator, helperCfg HelperConfig) map[string]any {
	if cfg == nil {
		return TemplateHelpers(t, helperCfg)
	}
	if helperCfg.Registry == nil {
		helperCfg.Registry = cfg.FormatterRegistry()
	}

	// Get base helpers from TemplateHelpers
	result := TemplateHelpers(t, helperCfg)

	// Add culture helpers if culture service is configured
	cultureService := cfg.CultureService()
	if cultureService != nil {
		cultureHelpers := CultureHelpers(cultureService, helperCfg.LocaleKey)
		for name, fn := range cultureHelpers {
			result[name] = fn
		}
	}

	return result
}

func (cfg *Config) applyLocaleCatalog() error {
	if cfg == nil {
		return nil
	}

	data, err := cfg.loadCultureData()
	if err != nil {
		return err
	}

	catalog, err := newLocaleCatalog(data.DefaultLocale, data.Locales)
	if err != nil {
		return err
	}
	cfg.localeCatalog = catalog

	if catalog == nil {
		cfg.DefaultLocale = normalizeLocale(cfg.DefaultLocale)
		return nil
	}

	if len(cfg.Locales) > 0 {
		for i, locale := range cfg.Locales {
			cfg.Locales[i] = normalizeLocale(locale)
		}
		for _, locale := range cfg.Locales {
			if !catalog.Has(locale) {
				return fmt.Errorf("i18n: locale %q is not defined in culture data", locale)
			}
		}
	} else {
		cfg.Locales = catalog.ActiveLocaleCodes()
	}

	if cfg.DefaultLocale != "" {
		cfg.DefaultLocale = normalizeLocale(cfg.DefaultLocale)
		if !catalog.Has(cfg.DefaultLocale) {
			return fmt.Errorf("i18n: default locale %q is not defined in culture data", cfg.DefaultLocale)
		}
	} else if catalog.DefaultLocale() != "" {
		cfg.DefaultLocale = catalog.DefaultLocale()
	}

	if cfg.DefaultLocale != "" && !catalog.IsActive(cfg.DefaultLocale) {
		return fmt.Errorf("i18n: default locale %q is not marked active", cfg.DefaultLocale)
	}

	cfg.applyCatalogFallbacks(catalog)

	return nil
}

func (cfg *Config) applyPluralRuleOptions() {
	if !cfg.enablePlural || len(cfg.pluralRules) == 0 || cfg.Loader == nil {
		return
	}

	if loader, ok := cfg.Loader.(pluralRuleLoader); ok {
		cfg.Loader = loader.WithPluralRules(cfg.pluralRules...)
	}
}

func (cfg *Config) applyCatalogFallbacks(catalog *LocaleCatalog) {
	if cfg == nil || catalog == nil {
		return
	}

	resolver, ok := cfg.Resolver.(*StaticFallbackResolver)
	if !ok {
		if cfg.Resolver != nil {
			return
		}
		resolver = NewStaticFallbackResolver()
		cfg.Resolver = resolver
	}

	for _, locale := range catalog.AllLocaleCodes() {
		if chain := resolver.Resolve(locale); len(chain) > 0 {
			continue
		}
		fallbacks := catalog.Fallbacks(locale)
		if len(fallbacks) == 0 {
			continue
		}
		resolver.Set(locale, fallbacks...)
	}
}

func (cfg *Config) seedResolverFallbacks() {
	if !cfg.enablePlural || !cfg.seedPluralFallbacks {
		return
	}

	resolver, ok := cfg.Resolver.(*StaticFallbackResolver)
	if !ok || resolver == nil {
		return
	}

	seen := make(map[string]struct{}, len(cfg.Locales))
	var localeCandidates []string

	appendCandidate := func(locale string) {
		if locale == "" {
			return
		}
		if _, exists := seen[locale]; exists {
			return
		}
		seen[locale] = struct{}{}
		localeCandidates = append(localeCandidates, locale)
	}

	if cfg.Store != nil {
		for _, locale := range cfg.Store.Locales() {
			appendCandidate(locale)
		}
	}

	for _, locale := range cfg.Locales {
		appendCandidate(locale)
	}

	for _, locale := range localeCandidates {
		if locale == "" {
			continue
		}
		if existing := resolver.Resolve(locale); existing != nil {
			continue
		}
		chain := localeParentChain(locale)
		if len(chain) == 0 {
			continue
		}
		resolver.Set(locale, chain...)
	}
}

func (cfg *Config) ensureFormatterRegistry() {
	if cfg == nil || cfg.formatterRegistry != nil {
		return
	}

	locales := append([]string{}, defaultFormatterLocales...)
	if len(cfg.formatterLocales) > 0 {
		locales = append(locales, cfg.formatterLocales...)
	}
	locales = normalizeLocales(locales)

	if cfg.Resolver == nil {
		cfg.Resolver = NewStaticFallbackResolver()
	}

	// Load culture data and create formatting rules provider
	cultureData, err := cfg.loadCultureData()
	if err != nil {
		cultureData = &CultureData{}
	}

	rulesProvider := NewFormattingRulesProvider(cultureData, cfg.Resolver)

	options := []FormatterRegistryOption{
		WithFormatterRegistryResolver(cfg.Resolver),
		WithFormatterRegistryLocales(locales...),
		WithFormattingRulesProvider(rulesProvider),
	}

	if len(cfg.formatterProviders) > 0 {
		for locale, provider := range cfg.formatterProviders {
			if locale == "" || provider == nil {
				continue
			}
			options = append(options, WithFormatterRegistryProvider(locale, provider))
		}
	}

	cfg.formatterRegistry = NewFormatterRegistry(options...)
}

func (cfg *Config) ensureCultureService() {
	if cfg.cultureService != nil {
		return
	}

	data, err := cfg.loadCultureData()
	if err != nil {
		// Log error but don't fail - use empty service
		cfg.cultureService = NewCultureService(&CultureData{}, cfg.Resolver)
		return
	}

	cfg.cultureService = NewCultureService(data, cfg.Resolver)
}

func (cfg *Config) loadCultureData() (*CultureData, error) {
	if cfg == nil {
		return &CultureData{}, nil
	}

	if cfg.cultureData != nil {
		return cfg.cultureData, nil
	}

	loader := cfg.newCultureDataLoader()
	data, err := loader.Load()
	if err != nil {
		return nil, err
	}
	if data == nil {
		data = &CultureData{}
	}
	cfg.cultureData = data
	return data, nil
}

func (cfg *Config) newCultureDataLoader() *CultureDataLoader {
	loader := NewCultureDataLoader(cfg.cultureDataPath)
	for locale, path := range cfg.cultureOverrides {
		loader.AddOverride(locale, path)
	}
	return loader
}
