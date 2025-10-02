package i18n

// Translator resolves a string for a given locale and message key.
type Translator interface {
	Translate(locale, key string, args ...any) (string, error)
}
