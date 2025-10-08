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
	l.overrides[locale] = path
}

// mergeCultureDataInto merges source into dest (source takes precedence)
func mergeCultureDataInto(dest, source *CultureData) {
	if dest == nil || source == nil {
		return
	}

	if source.CurrencyCodes != nil {
		if dest.CurrencyCodes == nil {
			dest.CurrencyCodes = make(map[string]string, len(source.CurrencyCodes))
		}
		maps.Copy(dest.CurrencyCodes, source.CurrencyCodes)
	}

	if source.SupportNumbers != nil {
		if dest.SupportNumbers == nil {
			dest.SupportNumbers = make(map[string]string, len(source.SupportNumbers))
		}
		maps.Copy(dest.SupportNumbers, source.SupportNumbers)
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
			maps.Copy(dest.Lists[listName], localeMap)
		}
	}

	if source.MeasurementPreferences != nil {
		if dest.MeasurementPreferences == nil {
			dest.MeasurementPreferences = make(map[string]MeasurementPrefs, len(source.MeasurementPreferences))
		}
		maps.Copy(dest.MeasurementPreferences, source.MeasurementPreferences)
	}

	if source.FormattingRules != nil {
		if dest.FormattingRules == nil {
			dest.FormattingRules = make(map[string]FormattingRules, len(source.FormattingRules))
		}
		maps.Copy(dest.FormattingRules, source.FormattingRules)
	}
}

// mergeCultureData merges source into dest (source takes precedence)
func (l *CultureDataLoader) mergeCultureData(dest, source *CultureData) {
	mergeCultureDataInto(dest, source)
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
	mergeCultureDataInto(base, &override)

	return nil
}
