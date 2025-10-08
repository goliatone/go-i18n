package libphonenumber

import (
	"strconv"
	"strings"

	i18n "github.com/goliatone/go-i18n"
	"github.com/nyaruka/phonenumbers"
	"golang.org/x/text/language"
)

type options struct {
	region string
	format phonenumbers.PhoneNumberFormat
}

// Option configures registration behaviour for the libphonenumber adapter.
type Option func(*options)

// WithRegion forces parsing using the provided ISO 3166-1 alpha-2 country code.
func WithRegion(region string) Option {
	return func(o *options) {
		o.region = strings.ToUpper(strings.TrimSpace(region))
	}
}

// WithFormat selects the libphonenumber output format (defaults to INTERNATIONAL).
func WithFormat(format phonenumbers.PhoneNumberFormat) Option {
	return func(o *options) {
		o.format = format
	}
}

// Register wires the libphonenumber-backed formatter for a specific locale.
func Register(locale string, opts ...Option) {
	registerLocales([]string{locale}, opts...)
}

// RegisterMany wires the libphonenumber-backed formatter for multiple locales.
func RegisterMany(locales []string, opts ...Option) {
	registerLocales(locales, opts...)
}

func registerLocales(locales []string, opts ...Option) {
	if len(locales) == 0 {
		return
	}

	for _, locale := range locales {
		trimmed := strings.TrimSpace(locale)
		if trimmed == "" {
			continue
		}

		cfg := options{format: phonenumbers.INTERNATIONAL}
		for _, opt := range opts {
			if opt != nil {
				opt(&cfg)
			}
		}

		i18n.RegisterPhoneFormatter(trimmed, makeFormatter(trimmed, cfg))
	}
}

func makeFormatter(registeredLocale string, cfg options) i18n.PhoneFormatterFunc {
	return func(locale, raw string) string {
		value := strings.TrimSpace(raw)
		if value == "" {
			return value
		}

		region := determineRegion(locale, registeredLocale, cfg.region)
		number, err := phonenumbers.Parse(value, region)
		if err != nil {
			return value
		}

		if !phonenumbers.IsPossibleNumber(number) && !phonenumbers.IsValidNumber(number) {
			return value
		}

		format := cfg.format
		if format == 0 {
			format = phonenumbers.INTERNATIONAL
		}

		formatted := phonenumbers.Format(number, format)
		if formatted == "" {
			return value
		}
		return formatted
	}
}

func determineRegion(requestedLocale, registeredLocale, explicitRegion string) string {
	if explicitRegion != "" {
		return explicitRegion
	}

	if region := regionFromLocale(requestedLocale); region != "" {
		return region
	}

	if region := regionFromLocale(registeredLocale); region != "" {
		return region
	}

	if plan, ok := i18n.DefaultPhoneDialPlan(requestedLocale); ok {
		if region := regionFromDialPlan(plan); region != "" {
			return region
		}
	}

	if plan, ok := i18n.DefaultPhoneDialPlan(registeredLocale); ok {
		if region := regionFromDialPlan(plan); region != "" {
			return region
		}
	}

	return ""
}

func regionFromLocale(locale string) string {
	if locale == "" {
		return ""
	}

	cleaned := strings.ReplaceAll(locale, "_", "-")
	tag, err := language.Parse(cleaned)
	if err != nil {
		return ""
	}

	region, _ := tag.Region()
	if region == language.Und {
		return ""
	}

	return strings.ToUpper(region.String())
}

func regionFromDialPlan(plan i18n.PhoneDialPlan) string {
	code, err := strconv.Atoi(plan.CountryCode)
	if err != nil || code <= 0 {
		return ""
	}
	region := phonenumbers.GetRegionCodeForCountryCode(code)
	return strings.ToUpper(region)
}
