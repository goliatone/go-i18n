package i18n

import "fmt"

// CultureData contains locale-specific business/cultural information
type CultureData struct {
	SchemaVersion          string                              `json:"schema_version"`
	DefaultLocale          string                              `json:"default_locale"`
	Locales                map[string]LocaleDefinition         `json:"locales"`
	Currencies             map[string]CurrencyInfo             `json:"currencies"`
	SupportNumbers         map[string]string                   `json:"support_numbers"`
	Lists                  map[string]map[string][]string      `json:"lists"`
	MeasurementPreferences map[string]MeasurementPreferenceSet `json:"measurement_preferences"`
	FormattingRules        map[string]FormattingRules          `json:"formatting_rules"`
}

// LocaleDefinition represents the raw locale metadata as defined in culture data files.
type LocaleDefinition struct {
	DisplayName string         `json:"display_name"`
	Active      *bool          `json:"active,omitempty"`
	Fallbacks   []string       `json:"fallbacks,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// CurrencyInfo describes currency metadata for a locale.
type CurrencyInfo struct {
	Code   string `json:"code"`
	Symbol string `json:"symbol"`
}

// MeasurementPreferenceSet groups unit preferences by measurement type.
type MeasurementPreferenceSet map[string]UnitPreference

// UnitPreference specifies preferred unit and conversion
type UnitPreference struct {
	Unit           string             `json:"unit"`
	Symbol         string             `json:"symbol"`
	ConversionFrom map[string]float64 `json:"conversion_from,omitempty"`
}

// CultureService provides access to cultural/business data
type CultureService interface {
	// GetCurrencyCode returns the currency code for a locale
	GetCurrencyCode(locale string) (string, error)

	// GetCurrency returns currency metadata for a locale
	GetCurrency(locale string) (CurrencyInfo, error)

	// GetSupportNumber returns the support contact for a locale
	GetSupportNumber(locale string) (string, error)

	// GetList returns a locale-specific list by name
	GetList(locale, name string) ([]string, error)

	// GetMeasurementPreference returns preferred units for a locale
	GetMeasurementPreference(locale, measurementType string) (*UnitPreference, error)

	// ConvertMeasurement converts a value to the preferred unit for a locale
	ConvertMeasurement(locale string, value float64, fromUnit, measurementType string) (float64, string, string, error)
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

// GetCurrency returns the currency metadata for a locale.
func (s *cultureService) GetCurrency(locale string) (CurrencyInfo, error) {
	if s.data == nil || s.data.Currencies == nil {
		return CurrencyInfo{}, fmt.Errorf("no currency for locale %q", locale)
	}

	candidates := s.resolveCandidates(locale)

	for _, candidate := range candidates {
		if info, ok := s.data.Currencies[candidate]; ok {
			if info.Code != "" || info.Symbol != "" {
				return info, nil
			}
		}
	}

	if info, ok := s.data.Currencies["default"]; ok {
		if info.Code != "" || info.Symbol != "" {
			return info, nil
		}
	}

	return CurrencyInfo{}, fmt.Errorf("no currency for locale %q", locale)
}

// GetCurrencyCode returns the currency code for a locale
func (s *cultureService) GetCurrencyCode(locale string) (string, error) {
	info, err := s.GetCurrency(locale)
	if err != nil {
		return "", err
	}
	if info.Code == "" {
		return "", fmt.Errorf("no currency code for locale %q", locale)
	}
	return info.Code, nil
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

	primary := s.collectLocaleChain(locale, nil)
	if pref := s.selectMeasurementPreference(primary, measurementType); pref != nil {
		return pref, nil
	}

	if pref := s.selectMeasurementPreference([]string{"default"}, measurementType); pref != nil {
		return pref, nil
	}

	var fallbackChain []string
	if s.resolver != nil {
		seen := make(map[string]struct{})
		for _, code := range primary {
			seen[code] = struct{}{}
		}
		for _, fallback := range s.resolver.Resolve(locale) {
			fallbackChain = append(fallbackChain, s.collectLocaleChain(fallback, seen)...)
		}
	}

	if pref := s.selectMeasurementPreference(fallbackChain, measurementType); pref != nil {
		return pref, nil
	}

	return nil, fmt.Errorf("no measurement preference for %q in locale %q", measurementType, locale)
}

func (s *cultureService) collectLocaleChain(locale string, seen map[string]struct{}) []string {
	if locale == "" {
		return nil
	}

	if seen == nil {
		seen = make(map[string]struct{})
	}

	appendLocale := func(dst *[]string, value string) {
		if value == "" {
			return
		}
		if _, exists := seen[value]; exists {
			return
		}
		seen[value] = struct{}{}
		*dst = append(*dst, value)
	}

	var chain []string
	appendLocale(&chain, locale)
	for _, parent := range localeParentChain(locale) {
		appendLocale(&chain, parent)
	}

	return chain
}

func (s *cultureService) selectMeasurementPreference(locales []string, measurementType string) *UnitPreference {
	if len(locales) == 0 {
		return nil
	}

	for _, candidate := range locales {
		prefs, ok := s.data.MeasurementPreferences[candidate]
		if !ok || prefs == nil {
			continue
		}

		if pref, ok := prefs[measurementType]; ok && pref.Unit != "" {
			copy := pref
			return &copy
		}
	}

	return nil
}

// ConvertMeasurement converts a value to the preferred unit for a locale
func (s *cultureService) ConvertMeasurement(locale string, value float64, fromUnit, measurementType string) (float64, string, string, error) {
	pref, err := s.GetMeasurementPreference(locale, measurementType)
	if err != nil {
		return value, fromUnit, "", err
	}

	// If already in preferred unit, return as-is
	if pref.Unit == fromUnit {
		return value, pref.Unit, pref.Symbol, nil
	}

	// Try to find conversion factor
	if pref.ConversionFrom != nil {
		if factor, ok := pref.ConversionFrom[fromUnit]; ok {
			return value * factor, pref.Unit, pref.Symbol, nil
		}
	}

	return value, fromUnit, pref.Symbol, fmt.Errorf("no conversion from %q to %q", fromUnit, pref.Unit)
}

// resolveCandidates returns the list of locale candidates to try
func (s *cultureService) resolveCandidates(locale string) []string {
	if locale == "" {
		return nil
	}

	seen := make(map[string]struct{}, 4)
	candidates := make([]string, 0, 4)

	appendLocale := func(value string) {
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		candidates = append(candidates, value)
	}

	appendLocale(locale)

	for _, parent := range localeParentChain(locale) {
		appendLocale(parent)
	}

	if s.resolver != nil {
		for _, fallback := range s.resolver.Resolve(locale) {
			appendLocale(fallback)
			for _, parent := range localeParentChain(fallback) {
				appendLocale(parent)
			}
		}
	}

	return candidates
}
