package i18n

import (
	"sort"
	"strings"

	"golang.org/x/text/language"
)

// NormalizeLocale canonicalizes a locale identifier for shared use across the package.
func NormalizeLocale(locale string) string {
	cleaned := strings.ReplaceAll(strings.TrimSpace(locale), "_", "-")
	if cleaned == "" {
		return ""
	}

	if tag, err := language.Parse(cleaned); err == nil {
		value := tag.String()
		if value != "" && value != "und" {
			return value
		}
	}

	return strings.ToLower(cleaned)
}

// NormalizeLocales canonicalizes, deduplicates, and preserves first-seen order.
func NormalizeLocales(locales []string) []string {
	if len(locales) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(locales))
	result := make([]string, 0, len(locales))
	for _, locale := range locales {
		normalized := NormalizeLocale(locale)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

// NormalizeAndSortLocales canonicalizes, deduplicates, and sorts deterministically.
func NormalizeAndSortLocales(locales []string) []string {
	result := NormalizeLocales(locales)
	if len(result) == 0 {
		return nil
	}
	sort.Strings(result)
	return result
}

// LocaleParent returns the closest parent locale, if any.
func LocaleParent(locale string) string {
	normalized := NormalizeLocale(locale)
	if normalized == "" {
		return ""
	}

	if tag, err := language.Parse(normalized); err == nil {
		parentTag := tag.Parent()
		if parentTag != language.Und {
			value := parentTag.String()
			if value != "" && value != "und" {
				parent := NormalizeLocale(value)
				if parent != "" && parent != normalized {
					return parent
				}
			}
		}
	}

	if idx := strings.LastIndex(normalized, "-"); idx > 0 {
		parent := NormalizeLocale(normalized[:idx])
		if parent != "" && parent != normalized {
			return parent
		}
	}

	return ""
}

// LocaleParentChain returns the parent locale chain from nearest parent to root.
func LocaleParentChain(locale string) []string {
	normalized := NormalizeLocale(locale)
	if normalized == "" {
		return nil
	}

	chain := make([]string, 0, 4)
	seen := make(map[string]struct{}, 4)

	for current := LocaleParent(normalized); current != ""; current = LocaleParent(current) {
		if _, exists := seen[current]; exists {
			break
		}
		seen[current] = struct{}{}
		chain = append(chain, current)
	}

	return chain
}

func localeParentTag(locale string) string {
	return LocaleParent(locale)
}

func localeParentChain(locale string) []string {
	return LocaleParentChain(locale)
}

func normalizeLocale(locale string) string {
	return NormalizeLocale(locale)
}

func appendLocaleUnique(dst *[]string, seen map[string]struct{}, locale string) bool {
	locale = NormalizeLocale(locale)
	if locale == "" {
		return false
	}

	if _, exists := seen[locale]; exists {
		return false
	}

	seen[locale] = struct{}{}
	*dst = append(*dst, locale)
	return true
}

func appendLocaleParents(dst *[]string, seen map[string]struct{}, locale string) {
	for _, parent := range LocaleParentChain(locale) {
		appendLocaleUnique(dst, seen, parent)
	}
}

func collectResolverFallbacks(dst *[]string, seen map[string]struct{}, resolver FallbackResolver, seeds ...string) {
	if resolver == nil {
		return
	}

	queue := make([]string, 0, len(seeds))
	processed := make(map[string]struct{}, len(seeds))

	for _, seed := range seeds {
		if normalized := NormalizeLocale(seed); normalized != "" {
			queue = append(queue, normalized)
		}
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if _, exists := processed[current]; exists {
			continue
		}
		processed[current] = struct{}{}

		for _, fallback := range resolver.Resolve(current) {
			fallback = NormalizeLocale(fallback)
			if fallback == "" {
				continue
			}

			appendLocaleUnique(dst, seen, fallback)
			queue = append(queue, fallback)

			for _, parent := range LocaleParentChain(fallback) {
				appendLocaleUnique(dst, seen, parent)
				queue = append(queue, parent)
			}
		}
	}
}

func normalizeLocales(locales []string) []string {
	return NormalizeLocales(locales)
}

// deriveLocaleParents returns all parent locales for the given locale,
// ordered from closest parent to root. This is an alias for localeParentChain
// to maintain backward compatibility.
func deriveLocaleParents(locale string) []string {
	return localeParentChain(locale)
}
