package i18n

// Locale metadata placeholder pending richer implementation
type Locale struct {
	Code string
	Name string
}

// TranslationKey models identifier metadata
type TranslationKey struct {
	ID          string
	Description string
}
