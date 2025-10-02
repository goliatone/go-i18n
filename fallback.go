package i18n

// FallbackResolver resolves fallback locale chains
type FallbackResolver interface {
	Resolve(locale string) []string
}

// StaticFallbackResolver initial imp
type StaticFallbackResolver struct{}

func (s StaticFallbackResolver) Resolve(locale string) []string {
	return nil
}
