Basic Formatter + Translator Example
====================================

This document describes the runnable `examples/basic` program and the shipped formatter and locale-policy surfaces it exercises.

Scenario
--------
- Loads `en` and `es` locales from `examples/basic/locales/`.
- Builds a shared `Config` with:
  - `WithLocales("en", "es")`
  - `WithDefaultLocale("en")`
  - `EnablePluralization("testdata/cldr_cardinal.json")`
  - `WithFallback("es", "en")`
- Wraps a translator hook to log lookups and injects a custom formatter registry where Spanish currency output differs from the CLDR/X/Text defaults.

Formatter Touchpoints
---------------------
- `NewFormatterRegistry` seeds `en`/`es` providers. A custom call to `Register("format_currency", …)` demonstrates how global overrides layer on top of generated data.
- Template helpers are created via `TemplateHelpers(translator, HelperConfig{TemplateHelperKey: "t", Registry: registry})`. The helper map exports formatter functions that honour the fallback chain when templates omit locale arguments.

Running the Example
-------------------
Execute:
```
/Users/goliatone/.g/go/bin/go run ./examples/basic
```
Expect:
- Rendered template in Spanish with dates/number formatting resolved through the formatter registry.
- Logged translator lookups (via the hook) showing fallback behaviour.
- Demonstration of missing translation handling.

Extending the Example
---------------------
- Add `WithFormatterLocales("el")` after generating Greek bundles to opt in to additional locales.
- Use a custom registry setup to test regional fallbacks such as `es-MX -> es` when the host application wants formatter overrides beyond the defaults.
- Use `cfg.LocaleCatalog()`, `NormalizeLocale`, `Match`, `MatchAcceptLanguageWithOptions`, and `ResolveLocale` when the host application needs canonical locale selection outside the translator itself.
