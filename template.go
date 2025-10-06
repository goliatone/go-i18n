package i18n

import (
	"fmt"
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

	helpers[translateKey] = func(localeSrc any, key string, params ...any) string {
		if key == "" {
			return ""
		}

		locale := resolveLocale(localeSrc, cfg.LocaleKey)
		call := prepareTranslateCall(params...)

		msg, _, err := executeTemplateTranslation(t, locale, key, call)
		if err != nil {
			return handleMissing(cfg.OnMissing, locale, key, append(call.args, call.optionArgs()...), err)
		}

		return msg
	}

	helpers["translate_count"] = func(localeSrc any, key string, count any, params ...any) map[string]any {
		result := map[string]any{
			"key":    key,
			"locale": resolveLocale(localeSrc, cfg.LocaleKey),
		}

		if key == "" {
			result["text"] = ""
			return result
		}

		call := prepareTranslateCall(params...)
		call.hasCount = true
		call.count = count

		text, metadata, err := executeTemplateTranslation(t, result["locale"].(string), key, call)
		if err != nil {
			result["text"] = handleMissing(cfg.OnMissing, result["locale"].(string), key, append(call.args, call.optionArgs()...), err)
			result["error"] = err.Error()
			if metadata == nil {
				metadata = map[string]any{}
			}
		} else {
			result["text"] = text
		}

		if len(metadata) > 0 {
			result["metadata"] = metadata
		}

		if plural := extractPluralMetadata(metadata, count); len(plural) > 0 {
			result["plural"] = plural
		}

		return result
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

type helperCall struct {
	args     []any
	hasCount bool
	count    any
}

func (h helperCall) optionArgs() []any {
	if !h.hasCount {
		return nil
	}

	return []any{WithCount(h.count)}
}

func prepareTranslateCall(params ...any) helperCall {
	call := helperCall{}

	for _, param := range params {
		if count, ok := extractCountOption(param); ok {
			call.hasCount = true
			call.count = count
			if residual := removeKnownOptions(param, "count"); residual != nil {
				call.args = append(call.args, residual)
			}
			continue
		}
		call.args = append(call.args, param)
	}

	return call
}

func executeTemplateTranslation(t Translator, locale, key string, call helperCall) (string, map[string]any, error) {
	if t == nil {
		return "", nil, ErrMissingTranslation
	}

	args := make([]any, 0, len(call.args)+1)
	if call.hasCount {
		args = append(args, WithCount(call.count))
	}
	args = append(args, call.args...)

	if mt, ok := t.(metadataTranslator); ok {
		return mt.TranslateWithMetadata(locale, key, args...)
	}
	result, err := t.Translate(locale, key, args...)
	return result, nil, err
}

func extractCountOption(param any) (any, bool) {
	clone, ok := toStringMap(param)
	if !ok {
		return nil, false
	}

	value, exists := clone["count"]
	if !exists {
		return nil, false
	}
	return value, true
}

func removeKnownOptions(param any, keys ...string) any {
	clone, ok := toStringMap(param)
	if !ok {
		return param
	}
	for _, key := range keys {
		delete(clone, key)
	}
	if len(clone) == 0 {
		return nil
	}
	return clone
}

func toStringMap(param any) (map[string]any, bool) {
	switch value := param.(type) {
	case map[string]any:
		clone := make(map[string]any, len(value))
		for k, v := range value {
			clone[k] = v
		}
		return clone, true
	case map[any]any:
		clone := make(map[string]any, len(value))
		for k, v := range value {
			key, ok := stringifyMapKey(k)
			if !ok {
				continue
			}
			clone[key] = v
		}
		return clone, true
	default:
		return nil, false
	}
}

func stringifyMapKey(key any) (string, bool) {
	switch v := key.(type) {
	case string:
		return v, true
	case fmt.Stringer:
		return v.String(), true
	default:
		return fmt.Sprint(v), true
	}
}

func extractPluralMetadata(metadata map[string]any, count any) map[string]any {
	if len(metadata) == 0 && count == nil {
		return nil
	}

	plural := make(map[string]any)

	if metadata != nil {
		if category, ok := metadata[metadataPluralCategory]; ok {
			plural["category"] = category
		}
		if value, ok := metadata[metadataPluralCount]; ok {
			plural["count"] = value
		}
		if message, ok := metadata[metadataPluralMessage]; ok {
			plural["message"] = message
		}
		if missing, ok := metadata[metadataPluralMissing]; ok {
			plural["missing"] = missing
		}
	}

	if _, ok := plural["count"]; !ok && count != nil {
		plural["count"] = count
	}

	if len(plural) == 0 {
		return nil
	}

	return plural
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
