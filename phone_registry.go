package i18n

import (
	"strings"

	"golang.org/x/text/language"
)

// PhoneDialPlan describes how to normalize and format phone numbers for a locale.
// CountryCode should be digits without the leading plus sign.
// NationalPrefix is optional and stripped when present.
// Groups defines the digit grouping for the national significant number.
type PhoneDialPlan struct {
	CountryCode    string
	NationalPrefix string
	Groups         []int
}

// PhoneFormatterFunc formats a raw phone number string for a locale.
type PhoneFormatterFunc func(locale, raw string) string

// RegisterPhoneFormatter registers a custom phone formatter for the given locale.
// The formatter receives the resolved locale and raw input string.
func RegisterPhoneFormatter(locale string, formatter PhoneFormatterFunc) {
	trimmedLocale := strings.TrimSpace(locale)
	if trimmedLocale == "" || formatter == nil {
		return
	}

	registry := globalFormatterRegistry()
	if registry == nil {
		return
	}

	registry.RegisterLocale(trimmedLocale, "format_phone", func(loc, raw string) string {
		if loc == "" {
			loc = trimmedLocale
		}
		return formatter(loc, raw)
	})
}

// RegisterPhoneDialPlan registers a dial plan for a locale using the shared formatter pipeline.
// The plan is converted into a formatter that formats numbers in the +<country> groups... pattern.
func RegisterPhoneDialPlan(locale string, plan PhoneDialPlan) {
	meta := plan.toMetadata()
	if meta.CountryCode == "" || len(meta.Groups) == 0 {
		return
	}

	RegisterPhoneFormatter(locale, func(_ string, raw string) string {
		return formatPhoneWithMetadata(raw, meta)
	})
}

// DefaultPhoneDialPlan exposes the built-in dial plan for a locale if available.
// It first checks the exact locale key, then falls back to the base language.
func DefaultPhoneDialPlan(locale string) (PhoneDialPlan, bool) {
	key := normalizeLocaleKey(locale)
	if plan, ok := defaultPhoneDialPlans[key]; ok {
		return plan, true
	}

	tag := language.Make(key)
	base, _ := tag.Base()
	baseKey := base.String()
	if baseKey != key {
		if plan, ok := defaultPhoneDialPlans[baseKey]; ok {
			return plan, true
		}
	}

	return PhoneDialPlan{}, false
}

func (plan PhoneDialPlan) toMetadata() cldrPhoneMetadata {
	return cldrPhoneMetadata{
		CountryCode:    strings.TrimSpace(plan.CountryCode),
		NationalPrefix: strings.TrimSpace(plan.NationalPrefix),
		Groups:         normalizePhoneGroups(plan.Groups),
	}
}

func normalizePhoneGroups(groups []int) []int {
	if len(groups) == 0 {
		return nil
	}
	result := make([]int, 0, len(groups))
	for _, g := range groups {
		if g > 0 {
			result = append(result, g)
		}
	}
	return result
}

func normalizeLocaleKey(locale string) string {
	if locale == "" {
		return ""
	}
	normalized := strings.ReplaceAll(locale, "_", "-")
	return strings.ToLower(normalized)
}
