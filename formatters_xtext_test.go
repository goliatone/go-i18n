package i18n

import (
	"testing"
	"time"
)

// TestXTextProvider_DataDriven_DateFormatting validates that date formatting
// uses data-driven rules instead of hardcoded language checks
func TestXTextProvider_DataDriven_DateFormatting(t *testing.T) {
	tests := []struct {
		locale   string
		date     time.Time
		expected string
	}{
		{
			locale:   "es",
			date:     time.Date(2025, 10, 7, 0, 0, 0, 0, time.UTC),
			expected: "7 de octubre de 2025",
		},
		{
			locale:   "es-MX",
			date:     time.Date(2025, 10, 7, 0, 0, 0, 0, time.UTC),
			expected: "7 de octubre de 2025",
		},
		{
			locale:   "en",
			date:     time.Date(2025, 10, 7, 0, 0, 0, 0, time.UTC),
			expected: "October 7, 2025",
		},
	}

	for _, tt := range tests {
		t.Run(tt.locale, func(t *testing.T) {
			provider := newXTextProvider(tt.locale)
			got := provider.formatDate(tt.locale, tt.date)
			if got != tt.expected {
				t.Errorf("formatDate(%q) = %q; want %q", tt.locale, got, tt.expected)
			}
		})
	}
}

// TestXTextProvider_DataDriven_TimeFormatting validates that time formatting
// uses data-driven rules for 12/24 hour clock preference
func TestXTextProvider_DataDriven_TimeFormatting(t *testing.T) {
	tests := []struct {
		locale   string
		time     time.Time
		expected string
	}{
		{
			locale:   "es",
			time:     time.Date(2025, 10, 7, 14, 30, 0, 0, time.UTC),
			expected: "14:30", // 24-hour format for Spanish
		},
		{
			locale:   "es-MX",
			time:     time.Date(2025, 10, 7, 14, 30, 0, 0, time.UTC),
			expected: "14:30", // 24-hour format for Spanish Mexico
		},
		{
			locale:   "en",
			time:     time.Date(2025, 10, 7, 14, 30, 0, 0, time.UTC),
			expected: "2:30 PM", // 12-hour format for English
		},
	}

	for _, tt := range tests {
		t.Run(tt.locale, func(t *testing.T) {
			provider := newXTextProvider(tt.locale)
			got := provider.formatTime(tt.locale, tt.time)
			if got != tt.expected {
				t.Errorf("formatTime(%q) = %q; want %q", tt.locale, got, tt.expected)
			}
		})
	}
}

// TestXTextProvider_FormattingRulesLoading validates that formatting rules
// are correctly loaded with fallback logic
func TestXTextProvider_FormattingRulesLoading(t *testing.T) {
	tests := []struct {
		locale       string
		expectedLang string // Expected language that rules fall back to
	}{
		{"es", "es"},
		{"es-MX", "es"}, // Falls back to base "es"
		{"es-ES", "es"}, // Falls back to base "es"
		{"en", "en"},
		{"en-US", "en"},   // Falls back to base "en"
		{"fr", "en"},      // Unknown locale falls back to "en"
		{"unknown", "en"}, // Unknown locale falls back to "en"
	}

	for _, tt := range tests {
		t.Run(tt.locale, func(t *testing.T) {
			provider := newXTextProvider(tt.locale)
			if provider.rules == nil {
				t.Fatalf("rules is nil for locale %q", tt.locale)
			}
			if provider.rules.Locale != tt.expectedLang {
				t.Errorf("rules.Locale = %q; want %q", provider.rules.Locale, tt.expectedLang)
			}
		})
	}
}
