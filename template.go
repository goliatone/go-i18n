package i18n

import "maps"

// MissingTranslationHandler decides what string should be emitted when
// Translator.Translate returns an error
type MissingTranslationHandler func(locale, key string, args []any, err error) string

// HelperConfig configures template helper exports
type HelperConfig struct {
	LocaleKey         string
	Registry          *FormatterRegistry
	OnMissing         MissingTranslationHandler
	TemplateHelperKey string
}

// TemplateHelpers exposes translator + formatter helpers for go-template
func TemplateHelpers(t Translator, cfg HelperConfig) map[string]any {
	helpers := make(map[string]any)

	key := "translate"
	if cfg.TemplateHelperKey != "" {
		key = cfg.TemplateHelperKey
	}

	helpers[key] = func(locale, key string, args ...any) string {
		if key == "" {
			return ""
		}

		if t == nil {
			return handleMissing(cfg.OnMissing, locale, key, args, ErrMissingTranslation)
		}

		msg, err := t.Translate(locale, key, args...)
		if err != nil {
			return handleMissing(cfg.OnMissing, locale, key, args, err)
		}
		return msg
	}

	registry := cfg.Registry
	if registry == nil {
		registry = NewFormatterRegistry()
	}

	maps.Copy(helpers, registry.FuncMap(""))

	return helpers
}

func handleMissing(handler MissingTranslationHandler, locale, key string, args []any, err error) string {
	if handler != nil {
		return handler(locale, key, args, err)
	}
	return key
}
