package i18n

import "sort"

// Config captures translator and formatter setup
type Config struct {
	DefaultLocale string
	Locales       []string
	Loader        Loader
	Store         Store
	Resolver      FallbackResolver
	Formatter     Formatter
	Hooks         []TranslationHook
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
