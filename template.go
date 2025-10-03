package i18n

import (
	"reflect"
)

// MissingTranslationHandler decides what string should be emitted when
// Translator.Translate returns an error
type MissingTranslationHandler func(locale, key string, args []any, err error) string

// HelperConfig configures template helper exports
type HelperConfig struct {
	// LocaleKey selects the context key used to infer locale from template data.
	LocaleKey string
	// Registry allows callers to supply a custom formatter registry.
	Registry *FormatterRegistry
	// OnMissing controls the string returned when a translation is missing.
	OnMissing MissingTranslationHandler
	// TemplateHelperKey customizes the translator helper name (defaults to "translate").
	TemplateHelperKey string
}

// TemplateHelpers exposes translator + formatter helpers for go-template
func TemplateHelpers(t Translator, cfg HelperConfig) map[string]any {
	registry := cfg.Registry
	if registry == nil {
		registry = NewFormatterRegistry()
	}

	helpers := make(map[string]any)

	translateKey := cfg.TemplateHelperKey
	if translateKey == "" {
		translateKey = "translate"
	}

	helpers[translateKey] = func(localeSrc, key string, args ...any) string {
		if key == "" {
			return ""
		}

		locale := resolveLocale(localeSrc, cfg.LocaleKey)

		if t == nil {
			return handleMissing(cfg.OnMissing, locale, key, args, ErrMissingTranslation)
		}

		msg, err := t.Translate(locale, key, args...)
		if err != nil {
			return handleMissing(cfg.OnMissing, locale, key, args, err)
		}
		return msg
	}

	helpers["current_locale"] = func(localeSrc any) string {
		return resolveLocale(localeSrc, cfg.LocaleKey)
	}

	for name, fn := range registry.FuncMap("") {
		if fn == nil {
			continue
		}
		helpers[name] = wrapFormatter(registry, name, fn)
	}

	return helpers
}

func handleMissing(handler MissingTranslationHandler, locale, key string, args []any, err error) string {
	if handler != nil {
		return handler(locale, key, args, err)
	}
	return key
}

func resolveLocale(src any, key string) string {
	if src == nil {
		return ""
	}

	if str, ok := src.(string); ok {
		return str
	}

	if key == "" {
		return ""
	}

	switch data := src.(type) {
	case map[string]any:
		if v, ok := data[key]; ok {
			if str, ok := v.(string); ok {
				return str
			}
		}
	case map[string]string:
		if v, ok := data[key]; ok {
			return v
		}
	}

	value := reflect.ValueOf(src)
	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return ""
		}
		value = value.Elem()
	}

	if value.Kind() == reflect.Struct {
		field := value.FieldByNameFunc(func(name string) bool {
			return name == key
		})
		if field.IsValid() && field.Kind() == reflect.String {
			return field.String()
		}
	}

	return ""
}

func wrapFormatter(registry *FormatterRegistry, name string, base any) any {
	baseValue := reflect.ValueOf(base)
	if !baseValue.IsValid() || baseValue.Kind() != reflect.Func {
		return base
	}

	fnType := baseValue.Type()

	wrapper := reflect.MakeFunc(fnType, func(args []reflect.Value) []reflect.Value {
		locale := ""
		if len(args) > 0 && args[0].Kind() == reflect.String {
			locale = args[0].String()
		}

		impl, ok := registry.Formatter(name, locale)
		if !ok {
			impl = base
		}

		implValue := reflect.ValueOf(impl)
		if !implValue.IsValid() || implValue.Kind() != reflect.Func || !implValue.Type().AssignableTo(fnType) {
			implValue = baseValue
		}

		return implValue.Call(args)
	})

	return wrapper.Interface()
}
