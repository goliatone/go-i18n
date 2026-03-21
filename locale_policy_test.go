package i18n

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type localePolicyFixture struct {
	Normalization []struct {
		Name                 string `json:"name"`
		Input                string `json:"input"`
		CanonicalExpectation string `json:"canonical_expectation"`
	} `json:"normalization"`
	NormalizeLocales []struct {
		Name                 string   `json:"name"`
		Input                []string `json:"input"`
		CanonicalExpectation []string `json:"canonical_expectation"`
	} `json:"normalize_locales"`
	NormalizeAndSortLocales []struct {
		Name              string   `json:"name"`
		Input             []string `json:"input"`
		SortedExpectation []string `json:"sorted_expectation"`
	} `json:"normalize_and_sort_locales"`
	ParentChain []struct {
		Name              string   `json:"name"`
		Input             string   `json:"input"`
		ParentExpectation string   `json:"parent_expectation"`
		ChainExpectation  []string `json:"chain_expectation"`
	} `json:"parent_chain"`
	CatalogMatch []struct {
		Name           string   `json:"name"`
		CatalogLocales []string `json:"catalog_locales"`
		Inactive       []string `json:"inactive_locales"`
		Scope          string   `json:"scope"`
		Strategy       string   `json:"strategy"`
		Input          string   `json:"input"`
		Matched        string   `json:"matched_expectation"`
	} `json:"catalog_match"`
	AcceptLanguage []struct {
		Name           string   `json:"name"`
		CatalogLocales []string `json:"catalog_locales"`
		Inactive       []string `json:"inactive_locales"`
		Scope          string   `json:"scope"`
		Header         string   `json:"header"`
		Matched        string   `json:"matched_expectation"`
	} `json:"accept_language"`
	ResolveLocale []struct {
		Name              string              `json:"name"`
		Input             string              `json:"input"`
		CatalogLocales    []string            `json:"catalog_locales"`
		ResolverFallbacks map[string][]string `json:"resolver_fallbacks"`
		Options           struct {
			Strategy        string `json:"strategy"`
			Scope           string `json:"scope"`
			ExpandParents   bool   `json:"expand_parents"`
			ExpandFallbacks bool   `json:"expand_fallbacks"`
			IncludeDefault  bool   `json:"include_default"`
			DefaultLocale   string `json:"default_locale"`
		} `json:"options"`
		Expected struct {
			Canonical string   `json:"canonical"`
			Matched   string   `json:"matched"`
			Parents   []string `json:"parents"`
			Fallbacks []string `json:"fallbacks"`
			Default   string   `json:"default"`
			Chain     []string `json:"chain"`
		} `json:"expected"`
	} `json:"resolve_locale"`
	MetadataDecode []struct {
		Name     string         `json:"name"`
		Locale   string         `json:"locale"`
		Metadata map[string]any `json:"metadata"`
	} `json:"metadata_decode"`
}

func TestLocalePolicyFixture_Normalization(t *testing.T) {
	fixture := loadLocalePolicyFixture(t)

	for _, tc := range fixture.Normalization {
		t.Run(tc.Name, func(t *testing.T) {
			if got := NormalizeLocale(tc.Input); got != tc.CanonicalExpectation {
				t.Fatalf("NormalizeLocale(%q) = %q, want %q", tc.Input, got, tc.CanonicalExpectation)
			}
		})
	}

	for _, tc := range fixture.NormalizeLocales {
		t.Run(tc.Name, func(t *testing.T) {
			if got := NormalizeLocales(tc.Input); !reflect.DeepEqual(got, tc.CanonicalExpectation) {
				t.Fatalf("NormalizeLocales(%#v) = %#v, want %#v", tc.Input, got, tc.CanonicalExpectation)
			}
		})
	}

	for _, tc := range fixture.NormalizeAndSortLocales {
		t.Run(tc.Name, func(t *testing.T) {
			if got := NormalizeAndSortLocales(tc.Input); !reflect.DeepEqual(got, tc.SortedExpectation) {
				t.Fatalf("NormalizeAndSortLocales(%#v) = %#v, want %#v", tc.Input, got, tc.SortedExpectation)
			}
		})
	}

	for _, tc := range fixture.ParentChain {
		t.Run(tc.Name, func(t *testing.T) {
			if got := LocaleParent(tc.Input); got != tc.ParentExpectation {
				t.Fatalf("LocaleParent(%q) = %q, want %q", tc.Input, got, tc.ParentExpectation)
			}
			if got := LocaleParentChain(tc.Input); !reflect.DeepEqual(got, tc.ChainExpectation) {
				t.Fatalf("LocaleParentChain(%q) = %#v, want %#v", tc.Input, got, tc.ChainExpectation)
			}
		})
	}
}

func TestLocalePolicyFixture_CatalogMatch(t *testing.T) {
	fixture := loadLocalePolicyFixture(t)

	for _, tc := range fixture.CatalogMatch {
		t.Run(tc.Name, func(t *testing.T) {
			catalog := mustBuildCatalog(t, tc.CatalogLocales, tc.Inactive, nil)
			meta, ok := catalog.matchWithOptions(tc.Input, parseMatchStrategy(tc.Strategy), parseLocaleScope(tc.Scope))
			if tc.Matched == "" {
				if ok {
					t.Fatalf("matchWithOptions(%q) = %+v, expected no match", tc.Input, meta)
				}
				return
			}
			if !ok || meta.Code != tc.Matched {
				t.Fatalf("matchWithOptions(%q) = %+v,%v, want code %q", tc.Input, meta, ok, tc.Matched)
			}
		})
	}
}

func TestLocaleCatalogMatch_DefaultsToExactOrParentAcrossAllLocales(t *testing.T) {
	catalog := mustBuildCatalog(t, []string{"en", "es"}, []string{"es"}, nil)

	meta, ok := catalog.Match("es-MX")
	if !ok || meta.Code != "es" {
		t.Fatalf("Match(%q) = %+v,%v, want code %q", "es-MX", meta, ok, "es")
	}
}

func TestLocaleParentFollowsBCP47NonPrefixParents(t *testing.T) {
	testCases := []struct {
		locale string
		parent string
		chain  []string
	}{
		{locale: "es-MX", parent: "es-419", chain: []string{"es-419", "es"}},
		{locale: "zh-TW", parent: "zh-Hant", chain: []string{"zh-Hant", "zh"}},
	}

	for _, tc := range testCases {
		t.Run(tc.locale, func(t *testing.T) {
			if got := LocaleParent(tc.locale); got != tc.parent {
				t.Fatalf("LocaleParent(%q) = %q, want %q", tc.locale, got, tc.parent)
			}
			if got := LocaleParentChain(tc.locale); !reflect.DeepEqual(got, tc.chain) {
				t.Fatalf("LocaleParentChain(%q) = %#v, want %#v", tc.locale, got, tc.chain)
			}
		})
	}
}

func TestLocaleCatalogMatchPrefersClosestSupportedBCP47Parent(t *testing.T) {
	catalog := mustBuildCatalog(t, []string{"es", "es-419"}, nil, nil)

	meta, ok := catalog.Match("es-MX")
	if !ok || meta.Code != "es-419" {
		t.Fatalf("Match(%q) = %+v,%v, want code %q", "es-MX", meta, ok, "es-419")
	}
}

func TestLocalePolicyFixture_AcceptLanguage(t *testing.T) {
	fixture := loadLocalePolicyFixture(t)

	for _, tc := range fixture.AcceptLanguage {
		t.Run(tc.Name, func(t *testing.T) {
			catalog := mustBuildCatalog(t, tc.CatalogLocales, tc.Inactive, nil)

			var (
				meta LocaleMetadata
				ok   bool
			)
			if tc.Scope == "" || tc.Scope == "all" {
				meta, ok = catalog.MatchAcceptLanguage(tc.Header)
			} else {
				meta, ok = catalog.MatchAcceptLanguageWithOptions(tc.Header, MatchOptions{
					Scope: parseLocaleScope(tc.Scope),
				})
			}

			if tc.Matched == "" {
				if ok {
					t.Fatalf("Accept-Language match(%q) = %+v, expected no match", tc.Header, meta)
				}
				return
			}
			if !ok || meta.Code != tc.Matched {
				t.Fatalf("Accept-Language match(%q) = %+v,%v, want code %q", tc.Header, meta, ok, tc.Matched)
			}
		})
	}
}

func TestLocaleCatalogMatchAcceptLanguageMalformedHeader(t *testing.T) {
	catalog := mustBuildCatalog(t, []string{"en", "es"}, nil, nil)

	if meta, ok := catalog.MatchAcceptLanguage("!!!"); ok {
		t.Fatalf("MatchAcceptLanguage returned unexpected match: %+v", meta)
	}
}

func TestLocaleCatalogMatchAcceptLanguageNormalizesUnderscoreTags(t *testing.T) {
	catalog := mustBuildCatalog(t, []string{"en", "es-MX"}, nil, nil)

	meta, ok := catalog.MatchAcceptLanguageWithOptions(" ES_mx ; q=0.9, en;q=0.8 ", MatchOptions{
		Scope: ScopeActiveOnly,
	})
	if !ok || meta.Code != "es-MX" {
		t.Fatalf("MatchAcceptLanguageWithOptions returned %+v,%v, want code %q", meta, ok, "es-MX")
	}
}

func TestNewLocaleCatalogFromLocalesBuildsActiveScopedCatalog(t *testing.T) {
	catalog, err := NewLocaleCatalogFromLocales("en", []string{"en", "ES_mx"})
	if err != nil {
		t.Fatalf("NewLocaleCatalogFromLocales returned error: %v", err)
	}
	if catalog == nil {
		t.Fatalf("expected non-nil catalog")
	}

	meta, ok := catalog.MatchAcceptLanguageWithOptions("ES_mx,es;q=0.9", MatchOptions{
		Scope: ScopeActiveOnly,
	})
	if !ok || meta.Code != "es-MX" {
		t.Fatalf("MatchAcceptLanguageWithOptions returned %+v,%v, want code %q", meta, ok, "es-MX")
	}
}

func TestLocaleCatalogMatchAcceptLanguageWithOptionsDefaultsToAllLocales(t *testing.T) {
	catalog := mustBuildCatalog(t, []string{"en", "fr"}, []string{"fr"}, nil)

	defaultMeta, defaultOK := catalog.MatchAcceptLanguage("fr-CA,fr;q=0.9,en;q=0.8")
	scopedMeta, scopedOK := catalog.MatchAcceptLanguageWithOptions("fr-CA,fr;q=0.9,en;q=0.8", MatchOptions{})

	if defaultOK != scopedOK || defaultMeta.Code != scopedMeta.Code {
		t.Fatalf("MatchAcceptLanguageWithOptions zero value = %+v,%v, want %+v,%v", scopedMeta, scopedOK, defaultMeta, defaultOK)
	}
}

func TestLocaleCatalogMatchAcceptLanguageDeterministicAcrossCatalogOrder(t *testing.T) {
	header := "en"
	first := mustBuildCatalog(t, []string{"en-US", "en-GB"}, nil, nil)
	second := mustBuildCatalog(t, []string{"en-GB", "en-US"}, nil, nil)

	firstMeta, firstOK := first.MatchAcceptLanguage(header)
	secondMeta, secondOK := second.MatchAcceptLanguage(header)
	if !firstOK || !secondOK {
		t.Fatalf("MatchAcceptLanguage(%q) did not return matches for both catalogs", header)
	}
	if firstMeta.Code != secondMeta.Code {
		t.Fatalf("MatchAcceptLanguage(%q) unstable across catalog order: %q vs %q", header, firstMeta.Code, secondMeta.Code)
	}
}

func TestLocalePolicyFixture_ResolveLocale(t *testing.T) {
	fixture := loadLocalePolicyFixture(t)

	for _, tc := range fixture.ResolveLocale {
		t.Run(tc.Name, func(t *testing.T) {
			catalog := mustBuildCatalog(t, tc.CatalogLocales, nil, nil)
			resolver := NewStaticFallbackResolver()
			for locale, chain := range tc.ResolverFallbacks {
				resolver.Set(locale, chain...)
			}

			resolution := ResolveLocale(tc.Input, ResolveLocaleOptions{
				Catalog:         catalog,
				Resolver:        resolver,
				DefaultLocale:   tc.Options.DefaultLocale,
				MatchStrategy:   parseMatchStrategy(tc.Options.Strategy),
				Scope:           parseLocaleScope(tc.Options.Scope),
				ExpandParents:   tc.Options.ExpandParents,
				ExpandFallbacks: tc.Options.ExpandFallbacks,
				IncludeDefault:  tc.Options.IncludeDefault,
			})

			if resolution.Canonical != tc.Expected.Canonical {
				t.Fatalf("Canonical = %q, want %q", resolution.Canonical, tc.Expected.Canonical)
			}
			if resolution.Matched != tc.Expected.Matched {
				t.Fatalf("Matched = %q, want %q", resolution.Matched, tc.Expected.Matched)
			}
			if !reflect.DeepEqual(resolution.Parents, tc.Expected.Parents) {
				t.Fatalf("Parents = %#v, want %#v", resolution.Parents, tc.Expected.Parents)
			}
			if !reflect.DeepEqual(resolution.Fallbacks, tc.Expected.Fallbacks) {
				t.Fatalf("Fallbacks = %#v, want %#v", resolution.Fallbacks, tc.Expected.Fallbacks)
			}
			if resolution.Default != tc.Expected.Default {
				t.Fatalf("Default = %q, want %q", resolution.Default, tc.Expected.Default)
			}
			if !reflect.DeepEqual(resolution.Chain, tc.Expected.Chain) {
				t.Fatalf("Chain = %#v, want %#v", resolution.Chain, tc.Expected.Chain)
			}
		})
	}
}

func TestLocalePolicyFixture_DecodeMetadata(t *testing.T) {
	fixture := loadLocalePolicyFixture(t)

	type localeSearchMetadata struct {
		SearchEnabled *bool  `json:"search_enabled,omitempty"`
		Analyzer      string `json:"analyzer,omitempty"`
		SemanticModel string `json:"semantic_model,omitempty"`
	}

	for _, tc := range fixture.MetadataDecode {
		t.Run(tc.Name, func(t *testing.T) {
			catalog := mustBuildCatalog(t, []string{tc.Locale}, nil, map[string]map[string]any{
				tc.Locale: tc.Metadata,
			})

			var out localeSearchMetadata
			if err := catalog.DecodeMetadata(tc.Locale, &out); err != nil {
				t.Fatalf("DecodeMetadata(%q): %v", tc.Locale, err)
			}
			if out.SearchEnabled == nil || !*out.SearchEnabled {
				t.Fatalf("SearchEnabled = %#v, want true", out.SearchEnabled)
			}
			if out.Analyzer != "spanish" {
				t.Fatalf("Analyzer = %q, want %q", out.Analyzer, "spanish")
			}
			if out.SemanticModel != "multilingual-v1" {
				t.Fatalf("SemanticModel = %q, want %q", out.SemanticModel, "multilingual-v1")
			}
		})
	}
}

func TestStaticFallbackResolver_NormalizesLocaleKeys(t *testing.T) {
	resolver := NewStaticFallbackResolver()
	resolver.Set(" ES_mx ", " es ", " EN_us ", "en-US")

	got := resolver.Resolve("es-MX")
	want := []string{"es", "en-US"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Resolve(%q) = %#v, want %#v", "es-MX", got, want)
	}
}

func TestStaticStore_NormalizesLocaleKeys(t *testing.T) {
	store := NewStaticStore(Translations{
		"EN_us": newStringCatalog(" EN_us ", map[string]string{"home.title": "Welcome"}),
	})

	if got, ok := store.Get("en-US", "home.title"); !ok || got != "Welcome" {
		t.Fatalf("Get(%q) = %q,%v, want %q,true", "en-US", got, ok, "Welcome")
	}

	locales := store.Locales()
	if !reflect.DeepEqual(locales, []string{"en-US"}) {
		t.Fatalf("Locales() = %#v, want %#v", locales, []string{"en-US"})
	}
}

func TestFormatterRegistry_NormalizesLocaleRegistration(t *testing.T) {
	registry := NewFormatterRegistry()
	registry.RegisterLocale("EN_us", "format_list", func(_ string, items []string) string {
		return "custom"
	})

	fn, ok := registry.Formatter("format_list", "en-US")
	if !ok {
		t.Fatal("expected normalized formatter lookup to succeed")
	}

	listFormatter, ok := fn.(func(string, []string) string)
	if !ok {
		t.Fatalf("unexpected formatter type %T", fn)
	}
	if got := listFormatter("en-US", []string{"a", "b"}); got != "custom" {
		t.Fatalf("custom formatter returned %q", got)
	}
}

func TestDecodeMetadataErrors(t *testing.T) {
	catalog := mustBuildCatalog(t, []string{"en-US"}, nil, map[string]map[string]any{
		"en-US": {
			"enabled": true,
		},
	})

	var nonPointer struct {
		Enabled bool `json:"enabled"`
	}
	if err := catalog.DecodeMetadata("missing", &nonPointer); err == nil {
		t.Fatal("expected error for missing locale")
	}
	if err := catalog.DecodeMetadata("en-US", nil); err == nil {
		t.Fatal("expected error for nil output")
	}
	if err := catalog.DecodeMetadata("en-US", nonPointer); err == nil {
		t.Fatal("expected error for non-pointer output")
	}

	type wrongShape struct {
		Enabled string `json:"enabled"`
	}
	var out wrongShape
	if err := catalog.DecodeMetadata("en-US", &out); err == nil {
		t.Fatal("expected error for incompatible target field type")
	}
}

func TestDecodeMetadataReplacesExistingOutputValues(t *testing.T) {
	catalog := mustBuildCatalog(t, []string{"en", "es"}, nil, map[string]map[string]any{
		"en": {
			"enabled":  true,
			"analyzer": "english",
		},
		"es": {
			"enabled": true,
		},
	})

	type decodeTarget struct {
		Enabled  bool   `json:"enabled"`
		Analyzer string `json:"analyzer"`
	}

	var out decodeTarget
	if err := catalog.DecodeMetadata("en", &out); err != nil {
		t.Fatalf("DecodeMetadata(%q): %v", "en", err)
	}
	if out.Analyzer != "english" {
		t.Fatalf("Analyzer after en decode = %q, want %q", out.Analyzer, "english")
	}

	if err := catalog.DecodeMetadata("es", &out); err != nil {
		t.Fatalf("DecodeMetadata(%q): %v", "es", err)
	}
	if out.Analyzer != "" {
		t.Fatalf("Analyzer after es decode = %q, want empty string", out.Analyzer)
	}
}

func BenchmarkNormalizeLocale(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NormalizeLocale(" ES_mx ")
		_ = NormalizeLocale("en_US")
		_ = NormalizeLocale("bad_locale_value")
	}
}

func loadLocalePolicyFixture(t *testing.T) localePolicyFixture {
	t.Helper()

	path := filepath.Join("testdata", "locale_policy_matrix.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read locale policy fixture: %v", err)
	}

	var fixture localePolicyFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatalf("decode locale policy fixture: %v", err)
	}

	return fixture
}

func mustBuildCatalog(t *testing.T, locales []string, inactive []string, metadata map[string]map[string]any) *LocaleCatalog {
	t.Helper()

	inactiveSet := make(map[string]struct{}, len(inactive))
	for _, locale := range inactive {
		inactiveSet[NormalizeLocale(locale)] = struct{}{}
	}

	definitions := make(map[string]LocaleDefinition, len(locales))
	for i, locale := range locales {
		normalized := NormalizeLocale(locale)
		active := true
		if _, exists := inactiveSet[normalized]; exists {
			active = false
		}

		definition := LocaleDefinition{
			DisplayName: normalized,
			Active:      &active,
		}
		if metadata != nil {
			if value, ok := metadata[locale]; ok {
				definition.Metadata = value
			} else if value, ok := metadata[normalized]; ok {
				definition.Metadata = value
			}
		}
		definitions[locale] = definition

		if i == 0 {
			// Keep the first configured locale as default for deterministic fixtures.
		}
	}

	defaultLocale := ""
	if len(locales) > 0 {
		defaultLocale = locales[0]
	}

	catalog, err := newLocaleCatalog(defaultLocale, definitions)
	if err != nil {
		t.Fatalf("newLocaleCatalog: %v", err)
	}

	return catalog
}

func parseMatchStrategy(value string) MatchStrategy {
	switch value {
	case "exact_or_parent":
		return MatchExactOrParent
	case "best_fit":
		return MatchBestFit
	default:
		return MatchExact
	}
}

func parseLocaleScope(value string) LocaleScope {
	if value == "active_only" {
		return ScopeActiveOnly
	}
	return ScopeAll
}
