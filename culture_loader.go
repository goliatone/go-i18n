package i18n

import (
	_ "embed"
	"encoding/json"
	"fmt"
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
	l.overrides[locale] = path
}

// mergeCultureData merges source into dest (source takes precedence)
func (l *CultureDataLoader) mergeCultureData(dest, source *CultureData) {
	if source.CurrencyCodes != nil {
		if dest.CurrencyCodes == nil {
			dest.CurrencyCodes = make(map[string]string)
		}
		for k, v := range source.CurrencyCodes {
			dest.CurrencyCodes[k] = v
		}
	}

	if source.SupportNumbers != nil {
		if dest.SupportNumbers == nil {
			dest.SupportNumbers = make(map[string]string)
		}
		for k, v := range source.SupportNumbers {
			dest.SupportNumbers[k] = v
		}
	}

	if source.Lists != nil {
		if dest.Lists == nil {
			dest.Lists = make(map[string]map[string][]string)
		}
		for listName, localeMap := range source.Lists {
			if dest.Lists[listName] == nil {
				dest.Lists[listName] = make(map[string][]string)
			}
			for loc, list := range localeMap {
				dest.Lists[listName][loc] = list
			}
		}
	}

	if source.MeasurementPreferences != nil {
		if dest.MeasurementPreferences == nil {
			dest.MeasurementPreferences = make(map[string]MeasurementPrefs)
		}
		for k, v := range source.MeasurementPreferences {
			dest.MeasurementPreferences[k] = v
		}
	}

	if source.FormattingRules != nil {
		if dest.FormattingRules == nil {
			dest.FormattingRules = make(map[string]FormattingRules)
		}
		for k, v := range source.FormattingRules {
			dest.FormattingRules[k] = v
		}
	}
}

// loadOverride loads and merges a locale-specific override file
func (l *CultureDataLoader) loadOverride(base *CultureData, locale, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("load culture override for %q: %w", locale, err)
	}

	var override CultureData
	if err := json.Unmarshal(data, &override); err != nil {
		return fmt.Errorf("parse culture override for %q: %w", locale, err)
	}

	// Merge override data into base
	if override.CurrencyCodes != nil {
		if base.CurrencyCodes == nil {
			base.CurrencyCodes = make(map[string]string)
		}
		for k, v := range override.CurrencyCodes {
			base.CurrencyCodes[k] = v
		}
	}

	if override.SupportNumbers != nil {
		if base.SupportNumbers == nil {
			base.SupportNumbers = make(map[string]string)
		}
		for k, v := range override.SupportNumbers {
			base.SupportNumbers[k] = v
		}
	}

	if override.Lists != nil {
		if base.Lists == nil {
			base.Lists = make(map[string]map[string][]string)
		}
		for listName, localeMap := range override.Lists {
			if base.Lists[listName] == nil {
				base.Lists[listName] = make(map[string][]string)
			}
			for loc, list := range localeMap {
				base.Lists[listName][loc] = list
			}
		}
	}

	if override.MeasurementPreferences != nil {
		if base.MeasurementPreferences == nil {
			base.MeasurementPreferences = make(map[string]MeasurementPrefs)
		}
		for k, v := range override.MeasurementPreferences {
			base.MeasurementPreferences[k] = v
		}
	}

	if override.FormattingRules != nil {
		if base.FormattingRules == nil {
			base.FormattingRules = make(map[string]FormattingRules)
		}
		for k, v := range override.FormattingRules {
			base.FormattingRules[k] = v
		}
	}

	return nil
}
