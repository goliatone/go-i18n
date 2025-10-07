package i18n

import (
	"sort"
	"strings"
)

// Config captures translator and formatter setup
type Config struct {
	DefaultLocale string
	Locales       []string
	Loader        Loader
	Store         Store
	Resolver      FallbackResolver
	Formatter     Formatter
	Hooks         []TranslationHook
	enablePlural  bool
	pluralRules   []string

	formatterLocales   []string
	formatterProviders map[string]FormatterProvider
	formatterRegistry  *FormatterRegistry
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
	if len(cfg.Locales) == 0 {
		return
	}

	seen := make(map[string]struct{}, len(cfg.Locales))
	dedeped := cfg.Locales[:0]
	for _, locale := range cfg.Locales {
		if locale == "" {
			continue
		}

		if _, ok := seen[locale]; ok {
			continue
		}
		seen[locale] = struct{}{}
		dedeped = append(dedeped, locale)
	}
	sort.Strings(dedeped)
	cfg.Locales = dedeped
}

func (cfg *Config) FormatterRegistry() *FormatterRegistry {
	if cfg == nil {
		return nil
	}
	cfg.ensureFormatterRegistry()
	return cfg.formatterRegistry
}

func (cfg *Config) TemplateHelpers(t Translator, helperCfg HelperConfig) map[string]any {
	if cfg == nil {
		return TemplateHelpers(t, helperCfg)
	}
	if helperCfg.Registry == nil {
		helperCfg.Registry = cfg.FormatterRegistry()
	}
	return TemplateHelpers(t, helperCfg)
}

func (cfg *Config) applyPluralRuleOptions() {
	if !cfg.enablePlural || len(cfg.pluralRules) == 0 || cfg.Loader == nil {
		return
	}

	if loader, ok := cfg.Loader.(pluralRuleLoader); ok {
		cfg.Loader = loader.WithPluralRules(cfg.pluralRules...)
	}
}

func (cfg *Config) seedResolverFallbacks() {
	if !cfg.enablePlural {
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
		chain := deriveLocaleParents(locale)
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

	options := []FormatterRegistryOption{
		WithFormatterRegistryResolver(cfg.Resolver),
		WithFormatterRegistryLocales(locales...),
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

func deriveLocaleParents(locale string) []string {
	var chain []string
	current := locale
	for {
		parent := collapseLocaleParent(current)
		if parent == "" {
			break
		}
		chain = append(chain, parent)
		current = parent
	}
	return chain
}

func collapseLocaleParent(locale string) string {
	if idx := strings.LastIndex(locale, "-"); idx > 0 {
		return locale[:idx]
	}
	return ""
}
