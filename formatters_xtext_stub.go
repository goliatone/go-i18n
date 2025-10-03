//go:build !xtext

package i18n

// RegisterXTextFormatters is a no-op when golang.org/x/text integration is not enabled.
func RegisterXTextFormatters(*FormatterRegistry, ...string) {}
