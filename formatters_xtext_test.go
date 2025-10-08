package i18n

import (
	"testing"
)

// TestXTextProvider_IsSpanish validates that isSpanish() correctly detects
// Spanish locales including regional variants
func TestXTextProvider_IsSpanish(t *testing.T) {
	tests := []struct {
		locale   string
		expected bool
	}{
		{"es", true},
		{"es-MX", true},
		{"es-ES", true},
		{"en", false},
		{"el", false},
		{"ar", false},
		{"fr", false},
	}

	for _, tt := range tests {
		t.Run(tt.locale, func(t *testing.T) {
			provider := newXTextProvider(tt.locale)
			got := provider.isSpanish()
			if got != tt.expected {
				t.Errorf("isSpanish() for locale %q = %v; want %v", tt.locale, got, tt.expected)
			}
		})
	}
}
