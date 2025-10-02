package i18n

import "fmt"

// Translator resolves a string for a given locale and message key.
type Translator interface {
	Translate(locale, key string, args ...any) (string, error)
}

// Formatter formats a template string with positional arguments
type Formatter interface {
	Format(template string, args ...any) (string, error)
}

// FormatterFunc adapts plain functions into Formatter
type FormatterFunc func(string, ...any) (string, error)

// Format impelements Fromatter for FormatterFunc
func (fn FormatterFunc) Format(template string, args ...any) (string, error) {
	if fn == nil {
		return template, nil
	}
	return fn(template, args...)
}

// SimpleTranslatorOption configures SimpleTranslator
type SimpleTranslatorOption func(*SimpleTranslator)

// SimpleTranslator performs in memory lookups backed by a Store
type SimpleTranslator struct {
	store         Store
	defaultLocale string
	formatter     Formatter
	resolver      FallbackResolver
}

func NewSimpleTranslator(store Store, opts ...SimpleTranslatorOption) (*SimpleTranslator, error) {
	st := &SimpleTranslator{
		store:     store,
		formatter: FormatterFunc(sprintfFormatter),
		resolver:  NewStaticFallbackResolver(),
	}

	if st.store == nil {
		st.store = NewStaticStore(nil)
	}

	for _, opt := range opts {
		if opt != nil {
			opt(st)
		}
	}

	if st.formatter == nil {
		st.formatter = FormatterFunc(sprintfFormatter)
	}

	if st.resolver == nil {
		st.resolver = NewStaticFallbackResolver()
	}

	return st, nil
}

func WithTranslatorDefaultLocale(locale string) SimpleTranslatorOption {
	return func(st *SimpleTranslator) {
		st.defaultLocale = locale
	}
}

func WithTranslatorFormatter(formatter Formatter) SimpleTranslatorOption {
	return func(st *SimpleTranslator) {
		st.formatter = formatter
	}
}

func WithTranslatorFallbackResolver(resolver FallbackResolver) SimpleTranslatorOption {
	return func(st *SimpleTranslator) {
		st.resolver = resolver
	}
}

func (t *SimpleTranslator) Translate(locale, key string, args ...any) (string, error) {
	if t == nil {
		return "", ErrMissingTranslation
	}

	if key == "" {
		return "", ErrMissingTranslation
	}

	primary := locale
	if primary == "" {
		primary = t.defaultLocale
	}

	if primary == "" {
		return "", ErrMissingTranslation
	}

	for _, candidate := range t.lookupLocales(primary) {
		if msg, ok := t.store.Get(candidate, key); ok {
			if len(args) == 0 || t.formatter == nil {
				return msg, nil
			}

			formatted, err := t.formatter.Format(msg, args...)
			if err != nil {
				return "", err
			}

			return formatted, nil
		}
	}

	return "", ErrMissingTranslation
}

func (t *SimpleTranslator) lookupLocales(primary string) []string {
	order := make([]string, 0, 4)
	seen := make(map[string]struct{}, 4)

	appendLocale := func(locale string) {
		if locale == "" {
			return
		}

		if _, ok := seen[locale]; ok {
			return
		}
		seen[locale] = struct{}{}
		order = append(order, locale)
	}

	appendLocale(primary)

	if t.resolver != nil {
		for _, fb := range t.resolver.Resolve(primary) {
			appendLocale(fb)
		}
	}

	appendLocale(t.defaultLocale)

	return order
}

func sprintfFormatter(template string, args ...any) (string, error) {
	return fmt.Sprintf(template, args...), nil
}
