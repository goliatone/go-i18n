package i18n

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"maps"
	"os"
)

//go:embed testdata/default_formatting_rules.json
var defaultFormattingRulesJSON []byte

// CultureDataLoader loads culture data from various sources
type CultureDataLoader struct {
	defaultPath string
	overrides   map[string]string
}

// NewCultureDataLoader creates a loader
func NewCultureDataLoader(defaultPath string) *CultureDataLoader {
	return &CultureDataLoader{
		defaultPath: defaultPath,
		overrides:   make(map[string]string),
	}
}

// Load reads culture data from JSON files
func (l *CultureDataLoader) Load() (*CultureData, error) {
	// Start with embedded default formatting rules
	var cultureData CultureData
	if err := json.Unmarshal(defaultFormattingRulesJSON, &cultureData); err != nil {
		return nil, fmt.Errorf("parse default formatting rules: %w", err)
	}

	// Load user-provided culture data if path is specified
	if l.defaultPath != "" {
		data, err := os.ReadFile(l.defaultPath)
		if err != nil {
			return nil, fmt.Errorf("load culture data: %w", err)
		}

		var userData CultureData
		if err := json.Unmarshal(data, &userData); err != nil {
			return nil, fmt.Errorf("parse culture data: %w", err)
		}

		// Merge user data into base (user data takes precedence)
		l.mergeCultureData(&cultureData, &userData)
	}

	// Apply overrides if any
	for locale, path := range l.overrides {
		if err := l.loadOverride(&cultureData, locale, path); err != nil {
			return nil, err
		}
	}

	return &cultureData, nil
}

// AddOverride adds a locale-specific override file
func (l *CultureDataLoader) AddOverride(locale, path string) {
	locale = NormalizeLocale(locale)
	if locale == "" || path == "" {
		return
	}
	l.overrides[locale] = path
}

// mergeCultureDataInto merges source into dest (source takes precedence)
func mergeCultureDataInto(dest, source *CultureData) {
	if dest == nil || source == nil {
		return
	}

	if source.SchemaVersion != "" {
		dest.SchemaVersion = source.SchemaVersion
	}

	if source.DefaultLocale != "" {
		dest.DefaultLocale = source.DefaultLocale
	}

	if len(source.Locales) > 0 {
		if dest.Locales == nil {
			dest.Locales = make(map[string]LocaleDefinition, len(source.Locales))
		}
		for code, definition := range source.Locales {
			code = NormalizeLocale(code)
			if code == "" {
				continue
			}
			existing, ok := dest.Locales[code]
			if !ok {
				dest.Locales[code] = cloneLocaleDefinition(definition)
				continue
			}
			dest.Locales[code] = mergeLocaleDefinition(existing, definition)
		}
	}

	if source.Currencies != nil {
		if dest.Currencies == nil {
			dest.Currencies = make(map[string]CurrencyInfo, len(source.Currencies))
		}
		for locale, currency := range source.Currencies {
			locale = NormalizeLocale(locale)
			if locale == "" {
				continue
			}
			dest.Currencies[locale] = currency
		}
	}

	if source.SupportNumbers != nil {
		if dest.SupportNumbers == nil {
			dest.SupportNumbers = make(map[string]string, len(source.SupportNumbers))
		}
		for locale, number := range source.SupportNumbers {
			locale = NormalizeLocale(locale)
			if locale == "" {
				continue
			}
			dest.SupportNumbers[locale] = number
		}
	}

	if source.Lists != nil {
		if dest.Lists == nil {
			dest.Lists = make(map[string]map[string][]string, len(source.Lists))
		}
		for listName, localeMap := range source.Lists {
			if localeMap == nil {
				continue
			}
			if dest.Lists[listName] == nil {
				dest.Lists[listName] = make(map[string][]string, len(localeMap))
			}
			for locale, entries := range localeMap {
				locale = NormalizeLocale(locale)
				if locale == "" {
					continue
				}
				dest.Lists[listName][locale] = cloneStrings(entries)
			}
		}
	}

	if source.MeasurementPreferences != nil {
		if dest.MeasurementPreferences == nil {
			dest.MeasurementPreferences = make(map[string]MeasurementPreferenceSet, len(source.MeasurementPreferences))
		}
		for locale, prefs := range source.MeasurementPreferences {
			locale = NormalizeLocale(locale)
			if locale == "" || prefs == nil {
				continue
			}
			if existing, ok := dest.MeasurementPreferences[locale]; ok && existing != nil {
				mergeMeasurementPreferenceSet(existing, prefs)
			} else {
				dest.MeasurementPreferences[locale] = cloneMeasurementPreferenceSet(prefs)
			}
		}
	}

	if source.FormattingRules != nil {
		if dest.FormattingRules == nil {
			dest.FormattingRules = make(map[string]FormattingRules, len(source.FormattingRules))
		}
		for locale, rules := range source.FormattingRules {
			locale = NormalizeLocale(locale)
			if locale == "" {
				continue
			}
			dest.FormattingRules[locale] = cloneFormattingRules(rules)
		}
	}
}

// mergeCultureData merges source into dest (source takes precedence)
func (l *CultureDataLoader) mergeCultureData(dest, source *CultureData) {
	mergeCultureDataInto(dest, source)
}

// loadOverride loads and merges a locale-specific override file
func (l *CultureDataLoader) loadOverride(base *CultureData, locale, path string) error {
	locale = NormalizeLocale(locale)
	if locale == "" {
		return fmt.Errorf("load culture override: empty locale")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("load culture override for %q: %w", locale, err)
	}

	var override CultureData
	if err := json.Unmarshal(data, &override); err != nil {
		return fmt.Errorf("parse culture override for %q: %w", locale, err)
	}

	scoped := scopedCultureOverride(locale, &override)
	if scoped == nil {
		return nil
	}

	mergeCultureDataInto(base, scoped)

	return nil
}

func scopedCultureOverride(locale string, source *CultureData) *CultureData {
	locale = NormalizeLocale(locale)
	if locale == "" || source == nil {
		return nil
	}

	override := &CultureData{}

	if definition, ok := lookupLocaleDefinition(source.Locales, locale); ok {
		override.Locales = map[string]LocaleDefinition{
			locale: cloneLocaleDefinition(definition),
		}
	}

	if currency, ok := lookupNormalizedMapValue(source.Currencies, locale); ok {
		override.Currencies = map[string]CurrencyInfo{
			locale: currency,
		}
	}

	if number, ok := lookupNormalizedMapValue(source.SupportNumbers, locale); ok {
		override.SupportNumbers = map[string]string{
			locale: number,
		}
	}

	if len(source.Lists) > 0 {
		for listName, localeMap := range source.Lists {
			if entries, ok := lookupNormalizedMapValue(localeMap, locale); ok {
				if override.Lists == nil {
					override.Lists = make(map[string]map[string][]string)
				}
				override.Lists[listName] = map[string][]string{
					locale: cloneStrings(entries),
				}
			}
		}
	}

	if prefs, ok := lookupNormalizedMapValue(source.MeasurementPreferences, locale); ok {
		override.MeasurementPreferences = map[string]MeasurementPreferenceSet{
			locale: cloneMeasurementPreferenceSet(prefs),
		}
	}

	if rules, ok := lookupNormalizedMapValue(source.FormattingRules, locale); ok {
		override.FormattingRules = map[string]FormattingRules{
			locale: cloneFormattingRules(rules),
		}
	}

	if len(override.Locales) == 0 &&
		len(override.Currencies) == 0 &&
		len(override.SupportNumbers) == 0 &&
		len(override.Lists) == 0 &&
		len(override.MeasurementPreferences) == 0 &&
		len(override.FormattingRules) == 0 {
		return nil
	}

	return override
}

func lookupLocaleDefinition(definitions map[string]LocaleDefinition, locale string) (LocaleDefinition, bool) {
	if len(definitions) == 0 {
		return LocaleDefinition{}, false
	}

	normalized := NormalizeLocale(locale)
	for code, definition := range definitions {
		if NormalizeLocale(code) == normalized {
			return definition, true
		}
	}

	return LocaleDefinition{}, false
}

func lookupNormalizedMapValue[T any](values map[string]T, locale string) (T, bool) {
	var zero T
	if len(values) == 0 {
		return zero, false
	}

	normalized := NormalizeLocale(locale)
	for code, value := range values {
		if NormalizeLocale(code) == normalized {
			return value, true
		}
	}

	return zero, false
}

func mergeLocaleDefinition(base, override LocaleDefinition) LocaleDefinition {
	result := cloneLocaleDefinition(base)

	if override.DisplayName != "" {
		result.DisplayName = override.DisplayName
	}

	if override.Active != nil {
		result.Active = cloneBool(override.Active)
	}

	if override.Fallbacks != nil {
		result.Fallbacks = cloneStrings(override.Fallbacks)
	}

	if override.Metadata != nil {
		if result.Metadata == nil {
			result.Metadata = make(map[string]any, len(override.Metadata))
		}
		maps.Copy(result.Metadata, override.Metadata)
	}

	return result
}

func cloneLocaleDefinition(definition LocaleDefinition) LocaleDefinition {
	clone := LocaleDefinition{
		DisplayName: definition.DisplayName,
	}

	if definition.Active != nil {
		clone.Active = cloneBool(definition.Active)
	}

	if len(definition.Fallbacks) > 0 {
		clone.Fallbacks = cloneStrings(definition.Fallbacks)
	}

	if len(definition.Metadata) > 0 {
		clone.Metadata = cloneMetadata(definition.Metadata)
	}

	return clone
}

func cloneStrings(input []string) []string {
	if len(input) == 0 {
		return nil
	}
	out := make([]string, len(input))
	copy(out, input)
	return out
}

func cloneMetadata(input map[string]any) map[string]any {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = cloneMetadataValue(value)
	}
	return out
}

func cloneMetadataValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneMetadata(typed)
	case []any:
		if len(typed) == 0 {
			return []any{}
		}
		out := make([]any, len(typed))
		for i, entry := range typed {
			out[i] = cloneMetadataValue(entry)
		}
		return out
	default:
		return typed
	}
}

func cloneBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	out := new(bool)
	*out = *value
	return out
}

func mergeMeasurementPreferenceSet(dest MeasurementPreferenceSet, source MeasurementPreferenceSet) {
	if dest == nil || source == nil {
		return
	}
	for measurement, pref := range source {
		dest[measurement] = cloneUnitPreference(pref)
	}
}

func cloneMeasurementPreferenceSet(source MeasurementPreferenceSet) MeasurementPreferenceSet {
	if len(source) == 0 {
		return nil
	}
	out := make(MeasurementPreferenceSet, len(source))
	for measurement, pref := range source {
		out[measurement] = cloneUnitPreference(pref)
	}
	return out
}

func cloneUnitPreference(pref UnitPreference) UnitPreference {
	out := UnitPreference{
		Unit:   pref.Unit,
		Symbol: pref.Symbol,
	}
	if len(pref.ConversionFrom) > 0 {
		out.ConversionFrom = make(map[string]float64, len(pref.ConversionFrom))
		maps.Copy(out.ConversionFrom, pref.ConversionFrom)
	}
	return out
}

func cloneFormattingRules(rules FormattingRules) FormattingRules {
	clone := rules
	clone.Locale = NormalizeLocale(rules.Locale)
	if len(rules.MonthNames) > 0 {
		clone.MonthNames = cloneStrings(rules.MonthNames)
	}
	return clone
}
