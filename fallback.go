package i18n

import "sync"

// FallbackResolver resolves fallback locale chains
type FallbackResolver interface {
	Resolve(locale string) []string
}

var _ FallbackResolver = &StaticFallbackResolver{}

// StaticFallbackResolver initial imp
type StaticFallbackResolver struct {
	chains map[string][]string
	mu     sync.RWMutex
}

func NewStaticFallbackResolver() *StaticFallbackResolver {
	return &StaticFallbackResolver{chains: make(map[string][]string)}
}

// Set registers the fallback chain for a locale
func (r *StaticFallbackResolver) Set(locale string, fallbacks ...string) {
	normalizedLocale := NormalizeLocale(locale)
	if r == nil || normalizedLocale == "" {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.chains == nil {
		r.chains = make(map[string][]string)
	}

	seen := make(map[string]struct{}, len(fallbacks)+1)
	seen[normalizedLocale] = struct{}{}
	chain := make([]string, 0, len(fallbacks))

	for _, fb := range fallbacks {
		normalizedFallback := NormalizeLocale(fb)
		if normalizedFallback == "" {
			continue
		}
		if _, ok := seen[normalizedFallback]; ok {
			continue
		}
		seen[normalizedFallback] = struct{}{}
		chain = append(chain, normalizedFallback)
	}
	r.chains[normalizedLocale] = chain
}

// Resolve returns a copy of the fallback chain for a locale
func (r *StaticFallbackResolver) Resolve(locale string) []string {
	if r == nil {
		return nil
	}

	normalizedLocale := NormalizeLocale(locale)
	if normalizedLocale == "" {
		return nil
	}

	r.mu.RLock()
	chain := r.chains[normalizedLocale]
	r.mu.RUnlock()

	if len(chain) == 0 {
		return nil
	}
	out := make([]string, len(chain))
	copy(out, chain)
	return out
}
