package i18n

// Config captures translator and formatter setup
type Config struct {
	DefaultLocale string
	Locales       []string
}

// Option mutates Config during construction
type Option func(*Config) error

// NewConfig builds Config via supplied options
func NewConfig(opts ...Option) (*Config, error) {
	return nil, ErrNotImplemented
}

// WithDefaultLocale sets the default locale in Config
func WithDefaultLocale(locale string) Option {
	return func(c *Config) error {
		return ErrNotImplemented
	}
}

// WithLocales registers supported locales
func WithLocales(locales ...string) Option {
	return func(c *Config) error {
		return ErrNotImplemented
	}
}
