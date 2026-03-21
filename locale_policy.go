package i18n

// MatchStrategy controls how a requested locale is mapped to a supported locale.
type MatchStrategy int

const (
	MatchExact MatchStrategy = iota
	MatchExactOrParent
	MatchBestFit
)

// LocaleScope controls whether matching considers all locales or only active locales.
type LocaleScope int

const (
	ScopeAll LocaleScope = iota
	ScopeActiveOnly
)

// MatchOptions configures scoped supported-locale selection helpers.
type MatchOptions struct {
	Scope LocaleScope
}

// ResolveLocaleOptions configures deterministic locale resolution.
type ResolveLocaleOptions struct {
	Catalog         *LocaleCatalog
	Resolver        FallbackResolver
	DefaultLocale   string
	MatchStrategy   MatchStrategy
	Scope           LocaleScope
	ExpandParents   bool
	ExpandFallbacks bool
	IncludeDefault  bool
}

// LocaleResolution describes the requested locale, any supported match, and the final ordered chain.
type LocaleResolution struct {
	Requested string
	Canonical string
	Matched   string
	Parents   []string
	Fallbacks []string
	Default   string
	Chain     []string
}

// ResolveLocale builds an explicit, deterministic locale chain.
func ResolveLocale(locale string, opts ResolveLocaleOptions) LocaleResolution {
	resolution := LocaleResolution{
		Requested: locale,
		Canonical: NormalizeLocale(locale),
		Parents:   LocaleParentChain(locale),
		Fallbacks: []string{},
	}

	if opts.Catalog != nil {
		if meta, ok := opts.Catalog.matchWithOptions(resolution.Canonical, opts.MatchStrategy, opts.Scope); ok {
			resolution.Matched = meta.Code
		}
	}

	defaultLocale := NormalizeLocale(opts.DefaultLocale)
	if defaultLocale == "" && opts.Catalog != nil {
		defaultLocale = opts.Catalog.DefaultLocale()
	}

	primary := resolution.Canonical
	if resolution.Matched != "" {
		primary = resolution.Matched
	}

	seen := make(map[string]struct{}, 8)
	appendLocale := func(locale string) {
		locale = NormalizeLocale(locale)
		if locale == "" {
			return
		}
		if _, exists := seen[locale]; exists {
			return
		}
		seen[locale] = struct{}{}
		resolution.Chain = append(resolution.Chain, locale)
	}

	appendLocale(primary)

	if opts.ExpandParents {
		for _, parent := range resolution.Parents {
			appendLocale(parent)
		}
	}

	if opts.ExpandFallbacks {
		resolution.Fallbacks = mergeLocaleFallbacks(primary, opts.Catalog, opts.Resolver)
		for _, fallback := range resolution.Fallbacks {
			appendLocale(fallback)
		}
	}

	if opts.IncludeDefault {
		resolution.Default = defaultLocale
		appendLocale(defaultLocale)
	}

	return resolution
}

func mergeLocaleFallbacks(locale string, catalog *LocaleCatalog, resolver FallbackResolver) []string {
	locale = NormalizeLocale(locale)
	if locale == "" {
		return nil
	}

	merged := make([]string, 0, 4)
	seen := map[string]struct{}{locale: {}}
	collectResolverFallbacks(&merged, seen, chainedFallbackResolver{
		catalog:  catalog,
		resolver: resolver,
	}, locale)

	if len(merged) == 0 {
		return []string{}
	}

	return merged
}

type chainedFallbackResolver struct {
	catalog  *LocaleCatalog
	resolver FallbackResolver
}

func (r chainedFallbackResolver) Resolve(locale string) []string {
	seen := make(map[string]struct{}, 4)
	chain := make([]string, 0, 4)

	appendChain := func(values []string) {
		for _, value := range values {
			if appendLocaleUnique(&chain, seen, value) {
				continue
			}
		}
	}

	if r.resolver != nil {
		appendChain(r.resolver.Resolve(locale))
	}
	if r.catalog != nil {
		appendChain(r.catalog.Fallbacks(locale))
	}

	return chain
}
