# Examples

The `examples` directory hosts runnable snippets that demonstrate how the `i18n` toolkit can be wired inside an application.

## Available scenarios

- `basic`: loads file-based translations, builds a translator through `Config`, wires template helpers, and shows how to override formatters.
- `plural`: demonstrates plural translations in a nested example module with its own `go.mod`.
- `web`: serves template-backed translations with culture data and formatting helpers.

The package also exposes shared locale-policy helpers for non-template consumers:

- `NormalizeLocale` and `NormalizeLocales` for canonical request handling
- `LocaleCatalog.Match` for route params and other direct supported-locale checks
- `MatchAcceptLanguage` for convenience header selection across all configured locales
- `MatchAcceptLanguageWithOptions(..., MatchOptions{Scope: ScopeActiveOnly})` for active-only admin or request-binding flows
- `ResolveLocale` for explicit locale-chain planning when convenience matching is not enough

## Running the demo

Use the provided command entrypoint:

```bash
go run ./cmd/example
```

This renders a sample invoice in Spanish, prints translation hook output, and showcases fallback + missing key handling.

For the shipped locale-policy contract and compatibility notes, see:

- [`docs/locale_policy_v1.md`](/Users/goliatone/Development/GO/src/github.com/goliatone/go-i18n/docs/locale_policy_v1.md)
- [`docs/locale_policy_compatibility_report.md`](/Users/goliatone/Development/GO/src/github.com/goliatone/go-i18n/docs/locale_policy_compatibility_report.md)
