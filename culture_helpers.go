package i18n

import (
	"fmt"
	"reflect"
)

// CultureHelpers returns template helper functions for culture data
func CultureHelpers(service CultureService, localeKey string) map[string]any {
	return map[string]any{
		"resolve_currency": func(data any) (string, error) {
			locale := extractLocale(data, localeKey)
			return service.GetCurrencyCode(locale)
		},

		"culture_value": func(data any, key string) (string, error) {
			locale := extractLocale(data, localeKey)
			switch key {
			case "support_number":
				return service.GetSupportNumber(locale)
			case "currency":
				return service.GetCurrencyCode(locale)
			default:
				return "", fmt.Errorf("unknown culture key: %q", key)
			}
		},

		"culture_list": func(data any, name string) ([]string, error) {
			locale := extractLocale(data, localeKey)
			return service.GetList(locale, name)
		},

		"preferred_measurement": func(data any, value float64, fromUnit, measurementType string) (string, error) {
			locale := extractLocale(data, localeKey)
			converted, unit, err := service.ConvertMeasurement(locale, value, fromUnit, measurementType)
			if err != nil {
				return "", err
			}
			// Use measurement formatter so locale-specific separators are applied.
			return FormatMeasurement(locale, converted, unit), nil
		},
	}
}

// extractLocale extracts the locale from template data using the configured key
// This function handles both map[string]any and struct types (like PageData)
func extractLocale(data any, localeKey string) string {
	if data == nil {
		return "en"
	}

	if localeKey == "" {
		localeKey = "Locale"
	}

	// Handle string directly
	if str, ok := data.(string); ok {
		return str
	}

	// Try map access
	switch d := data.(type) {
	case map[string]any:
		if v, ok := d[localeKey]; ok {
			if str, ok := v.(string); ok {
				return str
			}
		}
	case map[string]string:
		if v, ok := d[localeKey]; ok {
			return v
		}
	}

	// Handle struct types using reflection (like PageData)
	value := reflect.ValueOf(data)
	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return "en"
		}
		value = value.Elem()
	}

	if value.Kind() == reflect.Struct {
		field := value.FieldByNameFunc(func(name string) bool {
			return name == localeKey
		})
		if field.IsValid() && field.Kind() == reflect.String {
			return field.String()
		}
	}

	return "en" // Default fallback
}
