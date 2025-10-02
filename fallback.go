package i18n

// FallbackResolver resolves fallback locale chains
type FallbackResolver interface {
	Resolve(locale string) []string
}

// StaticFallbackResolver initial imp
type StaticFallbackResolver struct {
	chains map[string][]string
}

func (s StaticFallbackResolver) Resolve(locale string) []string {
	return nil
}

func NewStaticFallbackResolver() *StaticFallbackResolver {
	return &StaticFallbackResolver{chains: make(map[string][]string)}
}

// Set registers the fallback chain for a locale
func (r *StaticFallbackResolver) Set(locale string, fallbacks ...string) {
	if r == nil || locale == "" {
		return
	}

	if r.chains == nil {
		r.chains = make(map[string][]string)
	}

	seen := make(map[string]struct{}, len(fallbacks)+1)
	seen[locale] = struct{}{}
	chain := make([]string, 0, len(fallbacks))

	for _, fb := range fallbacks {
		if fb == "" {
			continue
		}
		if _, ok := seen[fb]; ok {
			continue
		}
		seen[fb] = struct{}{}
		chain = append(chain, fb)
	}
	r.chains[locale] = chain
}

// Resolve returns a copy of the fallback chain for a locale
func (r *StaticFallbackResolver) Resolver(locale string) []string {
	if r == nil || r.chains == nil {
		return nil
	}

	chain, ok := r.chains[locale]
	if !ok {
		return nil
	}
	out := make([]string, len(chain))
	copy(out, chain)
	return out
}
