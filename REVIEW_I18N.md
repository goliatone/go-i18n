# go-i18n Package Review

## Summary
- The package covers the core translator contract, in-memory store, JSON/YAML loaders, decorator hooks, and go-template helper wiring we need.
- Several behaviors diverge from the CMS design goals (opaque locale codes, opt-in fallbacks, richer locale metadata), so adopting `go-i18n` as-is would require follow-up changes.
- Formatter helpers exist but currently emit English-centric output regardless of locale, which undercuts the "locale aware" expectation in our docs.

## Strengths
- Translator API matches the CMS contract and exposes the same signature (`translator.go:167`).
- File loader and static store provide immutable snapshots with cloning for safe sharing (`loaders.go:37`, `store.go:34`).
- Template helper map exposes translation helpers plus formatter lookup/override hooks (`template.go:20-112`).
- Decorator hooks let us layer logging/metrics without modifying the core translator (`decorators.go:1-118`).

## Gaps & Risks
- `SimpleTranslator.lookupLocales` always collapses `en-US`→`en` and appends the default locale even when no fallback is configured, violating our "opaque locale codes" and "no implicit fallback" requirements (`translator.go:224-254`).
- `Config.seedResolverFallbacks` auto-derives fallback chains whenever pluralization is enabled, layering another implicit fallback path that we cannot disable (`config.go:212-258`).
- Configuration only tracks locale codes; there is no `AddLocale` helper, locale metadata, or functional options for fallback chains/groups as described in our CMS docs (`config.go:8-19`, `types.go:7-17`).
- Advanced fallback scenarios (locale groups / named chains) are not supported; `StaticFallbackResolver` only stores per-locale arrays with no grouping API (`fallback.go:6-41`).
- Formatter helpers ignore the `locale` argument and produce English-specific output (e.g., ordinal suffixes, list conjunctions), meaning we cannot deliver locale-aware formatting out of the box (`formatters.go:10-75`).
- `FormatPhone` is currently a passthrough stub, so phone formatting is effectively unimplemented (`formatters.go:64-65`).

## Recommendations
1. Make locale fallback opt-in: remove automatic parent/default fallbacks or gate them behind configuration so simple deployments stay isolated.
2. Extend configuration helpers to mirror the CMS design (`AddLocale`, `LocaleOption`, metadata fields, locale groups) or supply an adapter layer in `go-cms`.
3. Flesh out formatter implementations (or wire `golang.org/x/text`) so helpers respect locale conventions; at minimum document the current limitations.
4. Implement phone formatting or expose an interface so applications can provide their own implementation without monkey-patching the registry.
