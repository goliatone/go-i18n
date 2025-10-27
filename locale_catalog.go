package i18n

import (
	"fmt"
	"sort"
)

// LocaleCatalog is an immutable snapshot of locale metadata loaded from culture data.
type LocaleCatalog struct {
	defaultLocale string
	locales       map[string]localeEntry
	allCodes      []string
	activeCodes   []string
}

type localeEntry struct {
	displayName string
	active      bool
	fallbacks   []string
	metadata    map[string]any
}

// LocaleMetadata exposes the immutable metadata for a single locale.
type LocaleMetadata struct {
	Code        string
	DisplayName string
	Active      bool
	Fallbacks   []string
	Metadata    map[string]any
}

func newLocaleCatalog(defaultLocale string, definitions map[string]LocaleDefinition) (*LocaleCatalog, error) {
	if len(definitions) == 0 {
		return nil, nil
	}

	normalizedDefault := normalizeLocale(defaultLocale)
	locales := make(map[string]localeEntry, len(definitions))

	for originalCode, definition := range definitions {
		code := normalizeLocale(originalCode)
		if code == "" {
			return nil, fmt.Errorf("locale catalog: empty locale code")
		}
		if _, exists := locales[code]; exists {
			return nil, fmt.Errorf("locale catalog: duplicate locale %q", code)
		}

		entry := localeEntry{
			displayName: definition.DisplayName,
			active:      true,
		}

		if definition.Active != nil {
			entry.active = *definition.Active
		}

		if len(definition.Fallbacks) > 0 {
			entry.fallbacks = sanitizeFallbacks(code, definition.Fallbacks)
		}

		if len(definition.Metadata) > 0 {
			entry.metadata = cloneMetadata(definition.Metadata)
		}

		locales[code] = entry
	}

	if normalizedDefault != "" {
		if _, exists := locales[normalizedDefault]; !exists {
			return nil, fmt.Errorf("locale catalog: default locale %q not defined", normalizedDefault)
		}
	}

	for code, entry := range locales {
		for _, fallback := range entry.fallbacks {
			if _, exists := locales[fallback]; !exists {
				return nil, fmt.Errorf("locale catalog: %q references undefined fallback %q", code, fallback)
			}
		}
	}

	allCodes := make([]string, 0, len(locales))
	activeCodes := make([]string, 0, len(locales))
	for code, entry := range locales {
		allCodes = append(allCodes, code)
		if entry.active {
			activeCodes = append(activeCodes, code)
		}
	}
	sort.Strings(allCodes)
	sort.Strings(activeCodes)

	return &LocaleCatalog{
		defaultLocale: normalizedDefault,
		locales:       locales,
		allCodes:      allCodes,
		activeCodes:   activeCodes,
	}, nil
}

// DefaultLocale returns the configured default locale.
func (c *LocaleCatalog) DefaultLocale() string {
	if c == nil {
		return ""
	}
	return c.defaultLocale
}

// ActiveLocaleCodes returns all locales marked active, sorted alphabetically.
func (c *LocaleCatalog) ActiveLocaleCodes() []string {
	if c == nil || len(c.activeCodes) == 0 {
		return nil
	}
	out := make([]string, len(c.activeCodes))
	copy(out, c.activeCodes)
	return out
}

// AllLocaleCodes returns every locale in the catalog, sorted alphabetically.
func (c *LocaleCatalog) AllLocaleCodes() []string {
	if c == nil || len(c.allCodes) == 0 {
		return nil
	}
	out := make([]string, len(c.allCodes))
	copy(out, c.allCodes)
	return out
}

// DisplayName returns the human-friendly name for the requested locale.
func (c *LocaleCatalog) DisplayName(locale string) string {
	if c == nil {
		return ""
	}
	entry, ok := c.locales[normalizeLocale(locale)]
	if !ok {
		return ""
	}
	return entry.displayName
}

// Metadata returns a shallow copy of the custom metadata map for the locale.
func (c *LocaleCatalog) Metadata(locale string) map[string]any {
	if c == nil {
		return nil
	}
	entry, ok := c.locales[normalizeLocale(locale)]
	if !ok || len(entry.metadata) == 0 {
		return nil
	}
	return cloneMetadata(entry.metadata)
}

// Fallbacks returns the configured fallback chain for the locale.
func (c *LocaleCatalog) Fallbacks(locale string) []string {
	if c == nil {
		return nil
	}
	entry, ok := c.locales[normalizeLocale(locale)]
	if !ok || len(entry.fallbacks) == 0 {
		return nil
	}
	out := make([]string, len(entry.fallbacks))
	copy(out, entry.fallbacks)
	return out
}

// IsActive reports whether the locale is marked active.
func (c *LocaleCatalog) IsActive(locale string) bool {
	if c == nil {
		return false
	}
	entry, ok := c.locales[normalizeLocale(locale)]
	if !ok {
		return false
	}
	return entry.active
}

// Has reports whether the locale exists in the catalog.
func (c *LocaleCatalog) Has(locale string) bool {
	if c == nil {
		return false
	}
	_, ok := c.locales[normalizeLocale(locale)]
	return ok
}

// Locale returns the full metadata payload for a locale.
func (c *LocaleCatalog) Locale(locale string) (LocaleMetadata, bool) {
	if c == nil {
		return LocaleMetadata{}, false
	}
	normalized := normalizeLocale(locale)
	entry, ok := c.locales[normalized]
	if !ok {
		return LocaleMetadata{}, false
	}
	meta := LocaleMetadata{
		Code:        normalized,
		DisplayName: entry.displayName,
		Active:      entry.active,
	}
	if len(entry.fallbacks) > 0 {
		meta.Fallbacks = append([]string(nil), entry.fallbacks...)
	}
	if len(entry.metadata) > 0 {
		meta.Metadata = cloneMetadata(entry.metadata)
	}
	return meta, true
}

func sanitizeFallbacks(locale string, fallbacks []string) []string {
	if len(fallbacks) == 0 {
		return nil
	}

	seen := map[string]struct{}{
		normalizeLocale(locale): {},
	}

	result := make([]string, 0, len(fallbacks))
	for _, candidate := range fallbacks {
		normalized := normalizeLocale(candidate)
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
