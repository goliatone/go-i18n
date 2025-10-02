package i18n

// HelperConfig configures template helper exports
type HelperConfig struct {
	LocaleKey string
}

// TemplateHelpers exposes translator + formatter helpers for go-template
func TemplateHelpers(t Translator, cfg HelperConfig) map[string]any {
	panic(ErrNotImplemented)
}
