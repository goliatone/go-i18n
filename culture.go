package i18n

import (
	"fmt"
)

// CultureData contains locale-specific business/cultural information
type CultureData struct {
	CurrencyCodes          map[string]string              `json:"currency_codes"`
	SupportNumbers         map[string]string              `json:"support_numbers"`
	Lists                  map[string]map[string][]string `json:"lists"`
	MeasurementPreferences map[string]MeasurementPrefs    `json:"measurement_preferences"`
	FormattingRules        map[string]FormattingRules     `json:"formatting_rules"`
}

// MeasurementPrefs defines preferred units for a locale
type MeasurementPrefs struct {
	Weight   UnitPreference `json:"weight"`
	Distance UnitPreference `json:"distance"`
	Volume   UnitPreference `json:"volume"`
}

// UnitPreference specifies preferred unit and conversion
type UnitPreference struct {
	Unit           string             `json:"unit"`
	ConversionFrom map[string]float64 `json:"conversion_from,omitempty"`
}

// CultureService provides access to cultural/business data
type CultureService interface {
	// GetCurrencyCode returns the currency code for a locale
	GetCurrencyCode(locale string) (string, error)

	// GetSupportNumber returns the support contact for a locale
	GetSupportNumber(locale string) (string, error)

	// GetList returns a locale-specific list by name
	GetList(locale, name string) ([]string, error)

	// GetMeasurementPreference returns preferred units for a locale
	GetMeasurementPreference(locale, measurementType string) (*UnitPreference, error)

	// ConvertMeasurement converts a value to the preferred unit for a locale
	ConvertMeasurement(locale string, value float64, fromUnit, measurementType string) (float64, string, error)
}

// cultureService implements CultureService
type cultureService struct {
	data     *CultureData
	resolver FallbackResolver
}

// NewCultureService creates a culture service from data
func NewCultureService(data *CultureData, resolver FallbackResolver) CultureService {
	return &cultureService{
		data:     data,
		resolver: resolver,
	}
}

// GetCurrencyCode returns the currency code for a locale
func (s *cultureService) GetCurrencyCode(locale string) (string, error) {
	candidates := s.resolveCandidates(locale)

	for _, candidate := range candidates {
		if code, ok := s.data.CurrencyCodes[candidate]; ok {
			return code, nil
		}
	}

	return "", fmt.Errorf("no currency code for locale %q", locale)
}

// GetSupportNumber returns the support contact for a locale
func (s *cultureService) GetSupportNumber(locale string) (string, error) {
	candidates := s.resolveCandidates(locale)

	for _, candidate := range candidates {
		if number, ok := s.data.SupportNumbers[candidate]; ok {
			return number, nil
		}
	}

	return "", fmt.Errorf("no support number for locale %q", locale)
}

// GetList returns a locale-specific list by name
func (s *cultureService) GetList(locale, name string) ([]string, error) {
	if s.data.Lists == nil {
		return nil, fmt.Errorf("no lists configured")
	}

	listData, ok := s.data.Lists[name]
	if !ok {
		return nil, fmt.Errorf("no list named %q", name)
	}

	candidates := s.resolveCandidates(locale)
	for _, candidate := range candidates {
		if list, ok := listData[candidate]; ok {
			return list, nil
		}
	}

	return nil, fmt.Errorf("no list %q for locale %q", name, locale)
}

// GetMeasurementPreference returns preferred units for a locale
func (s *cultureService) GetMeasurementPreference(locale, measurementType string) (*UnitPreference, error) {
	if s.data.MeasurementPreferences == nil {
		return nil, fmt.Errorf("no measurement preferences configured")
	}

	candidates := s.resolveCandidates(locale)

	// Insert "default" after the primary locale but before fallback locales
	// so that locale-specific defaults take precedence over fallback locale settings
	if len(candidates) > 0 {
		// Insert default after the first candidate (the requested locale)
		candidates = append(candidates[:1], append([]string{"default"}, candidates[1:]...)...)
	} else {
		candidates = []string{"default"}
	}

	for _, candidate := range candidates {
		if prefs, ok := s.data.MeasurementPreferences[candidate]; ok {
			var pref *UnitPreference
			switch measurementType {
			case "weight":
				pref = &prefs.Weight
			case "distance":
				pref = &prefs.Distance
			case "volume":
				pref = &prefs.Volume
			default:
				continue
			}

			if pref != nil && pref.Unit != "" {
				return pref, nil
			}
		}
	}

	return nil, fmt.Errorf("no measurement preference for %q in locale %q", measurementType, locale)
}

// ConvertMeasurement converts a value to the preferred unit for a locale
func (s *cultureService) ConvertMeasurement(locale string, value float64, fromUnit, measurementType string) (float64, string, error) {
	pref, err := s.GetMeasurementPreference(locale, measurementType)
	if err != nil {
		return value, fromUnit, err
	}

	// If already in preferred unit, return as-is
	if pref.Unit == fromUnit {
		return value, fromUnit, nil
	}

	// Try to find conversion factor
	if pref.ConversionFrom != nil {
		if factor, ok := pref.ConversionFrom[fromUnit]; ok {
			return value * factor, pref.Unit, nil
		}
	}

	return value, fromUnit, fmt.Errorf("no conversion from %q to %q", fromUnit, pref.Unit)
}

// resolveCandidates returns the list of locale candidates to try
func (s *cultureService) resolveCandidates(locale string) []string {
	candidates := []string{locale}
	if s.resolver != nil {
		candidates = append(candidates, s.resolver.Resolve(locale)...)
	}
	return candidates
}
