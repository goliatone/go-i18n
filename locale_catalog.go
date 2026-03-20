package i18n

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	"golang.org/x/text/language"
)

// LocaleCatalog is an immutable snapshot of locale metadata loaded from culture data.
type LocaleCatalog struct {
	defaultLocale string
	locales       map[string]localeEntry
	allCodes      []string
	activeCodes   []string
	allMatcher    localeMatchState
	activeMatcher localeMatchState
}

type localeEntry struct {
	displayName string
	active      bool
	fallbacks   []string
	metadata    map[string]any
}

type localeMatchState struct {
	codes   []string
	matcher language.Matcher
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
		allMatcher:    newLocaleMatchState(allCodes),
		activeMatcher: newLocaleMatchState(activeCodes),
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

// Match resolves a requested locale to a supported catalog locale.
// The default public behavior is exact-or-parent matching across all locales.
func (c *LocaleCatalog) Match(locale string) (LocaleMetadata, bool) {
	return c.matchWithOptions(locale, MatchExactOrParent, ScopeAll)
}

// MatchAcceptLanguage resolves an Accept-Language header to the best supported locale
// across all configured locales.
func (c *LocaleCatalog) MatchAcceptLanguage(header string) (LocaleMetadata, bool) {
	return c.matchAcceptLanguageWithOptions(header, ScopeAll)
}

// MatchAcceptLanguageWithOptions resolves an Accept-Language header to the best
// supported locale using explicit matching scope controls.
func (c *LocaleCatalog) MatchAcceptLanguageWithOptions(header string, opts MatchOptions) (LocaleMetadata, bool) {
	return c.matchAcceptLanguageWithOptions(header, opts.Scope)
}

// DecodeMetadata decodes locale metadata into a caller-owned output struct.
func (c *LocaleCatalog) DecodeMetadata(locale string, out any) error {
	if c == nil {
		return fmt.Errorf("locale catalog: unavailable")
	}
	if out == nil {
		return fmt.Errorf("locale catalog: output must be a non-nil pointer")
	}
	value := reflect.ValueOf(out)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return fmt.Errorf("locale catalog: output must be a non-nil pointer")
	}

	normalizedLocale := NormalizeLocale(locale)
	metadata, ok := c.locales[normalizedLocale]
	if !ok {
		return fmt.Errorf("locale catalog: locale %q not found", normalizedLocale)
	}
	if len(metadata.metadata) == 0 {
		return fmt.Errorf("locale catalog: locale %q has no metadata", normalizedLocale)
	}

	raw, err := json.Marshal(metadata.metadata)
	if err != nil {
		return fmt.Errorf("locale catalog: encode metadata for %q: %w", normalizedLocale, err)
	}
	decoded := reflect.New(value.Elem().Type())
	if err := json.Unmarshal(raw, decoded.Interface()); err != nil {
		return fmt.Errorf("locale catalog: decode metadata for %q: %w", normalizedLocale, err)
	}
	value.Elem().Set(decoded.Elem())

	return nil
}

func (c *LocaleCatalog) matchWithOptions(locale string, strategy MatchStrategy, scope LocaleScope) (LocaleMetadata, bool) {
	if c == nil {
		return LocaleMetadata{}, false
	}

	normalized := NormalizeLocale(locale)
	if normalized == "" {
		return LocaleMetadata{}, false
	}

	if meta, ok := c.lookupScopeLocale(normalized, scope); ok {
		return meta, true
	}

	switch strategy {
	case MatchExact:
		return LocaleMetadata{}, false
	case MatchBestFit:
		if meta, ok := c.matchBestFit(normalized, scope); ok {
			return meta, true
		}
		fallthrough
	case MatchExactOrParent:
		for _, parent := range LocaleParentChain(normalized) {
			if meta, ok := c.lookupScopeLocale(parent, scope); ok {
				return meta, true
			}
		}
	default:
		return LocaleMetadata{}, false
	}

	return LocaleMetadata{}, false
}

func (c *LocaleCatalog) matchAcceptLanguageWithOptions(header string, scope LocaleScope) (LocaleMetadata, bool) {
	if c == nil {
		return LocaleMetadata{}, false
	}

	tags, _, err := language.ParseAcceptLanguage(header)
	if err != nil || len(tags) == 0 {
		return LocaleMetadata{}, false
	}

	state := c.matchState(scope)
	if state.matcher == nil || len(state.codes) == 0 {
		return LocaleMetadata{}, false
	}

	_, index, _ := state.matcher.Match(tags...)
	if index < 0 || index >= len(state.codes) {
		return LocaleMetadata{}, false
	}

	return c.Locale(state.codes[index])
}

func (c *LocaleCatalog) matchBestFit(locale string, scope LocaleScope) (LocaleMetadata, bool) {
	state := c.matchState(scope)
	if state.matcher == nil || len(state.codes) == 0 {
		return LocaleMetadata{}, false
	}

	tag, err := language.Parse(locale)
	if err != nil {
		return LocaleMetadata{}, false
	}

	_, index, _ := state.matcher.Match(tag)
	if index < 0 || index >= len(state.codes) {
		return LocaleMetadata{}, false
	}

	return c.Locale(state.codes[index])
}

func (c *LocaleCatalog) lookupScopeLocale(locale string, scope LocaleScope) (LocaleMetadata, bool) {
	if c == nil {
		return LocaleMetadata{}, false
	}

	meta, ok := c.Locale(locale)
	if !ok {
		return LocaleMetadata{}, false
	}
	if scope == ScopeActiveOnly && !meta.Active {
		return LocaleMetadata{}, false
	}

	return meta, true
}

func (c *LocaleCatalog) matchState(scope LocaleScope) localeMatchState {
	if c == nil {
		return localeMatchState{}
	}
	if scope == ScopeActiveOnly {
		return c.activeMatcher
	}
	return c.allMatcher
}

func newLocaleMatchState(codes []string) localeMatchState {
	if len(codes) == 0 {
		return localeMatchState{}
	}

	tags := make([]language.Tag, 0, len(codes))
	normalizedCodes := make([]string, 0, len(codes))
	for _, code := range codes {
		tag, err := language.Parse(code)
		if err != nil {
			continue
		}
		tags = append(tags, tag)
		normalizedCodes = append(normalizedCodes, code)
	}

	if len(tags) == 0 {
		return localeMatchState{}
	}

	return localeMatchState{
		codes:   normalizedCodes,
		matcher: language.NewMatcher(tags),
	}
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
