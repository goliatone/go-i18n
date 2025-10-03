# go-i18n

Internationalization library for Go applications.

## Overview

go-i18n provides translation management through a simple `Translate(locale, key, args...)` API. It supports multiple translation file formats, locale fallback chains, custom formatters, and integration with Go templates.

## Installation

```bash
go get github.com/goliatone/go-i18n
```

## Core Concepts

### Store

The `Store` interface provides read only access to translation templates indexed by locale and key. The package includes `StaticStore`, an immutable in-memory implementation.

### Loader

The `Loader` interface retrieves translations from external sources. `FileLoader` supports JSON and YAML files. Multiple files can be loaded and merged.

### Translator

The `Translator` interface exposes a single method: `Translate(locale, key string, args ...any) (string, error)`. The `SimpleTranslator` implementation handles locale fallback and template formatting.

### Formatter

The `Formatter` interface formats translation templates with arguments. Default implementation uses `fmt.Sprintf`. Custom formatters can be injected via configuration.

### FallbackResolver

The `FallbackResolver` interface returns fallback locale chains. When a translation is missing in the requested locale, the translator checks each fallback in order.

## Basic Usage

```go
// Load translations from files
loader := i18n.NewFileLoader(
    "locales/en.json",
    "locales/es.json",
)

// Build configuration
cfg, err := i18n.NewConfig(
    i18n.WithLocales("en", "es"),
    i18n.WithDefaultLocale("en"),
    i18n.WithLoader(loader),
    i18n.WithFallback("es", "en"),
)
if err != nil {
    return err
}

// Create translator
translator, err := cfg.BuildTranslator()
if err != nil {
    return err
}

// Translate messages
msg, err := translator.Translate("es", "home.greeting", "Alice")
if err != nil {
    return err
}
```

## Translation Files

### JSON Format

```json
{
  "en": {
    "home.title": "Welcome",
    "home.greeting": "Hello %s"
  },
  "es": {
    "home.title": "Bienvenido",
    "home.greeting": "Hola %s"
  }
}
```

### YAML Format

```yaml
en:
  home.title: Welcome
  home.greeting: Hello %s
es:
  home.title: Bienvenido
  home.greeting: Hola %s
```

## Template Integration

The package provides helpers for Go templates including translation and formatting functions.

```go
registry := i18n.NewFormatterRegistry()

helpers := i18n.TemplateHelpers(translator, i18n.HelperConfig{
    TemplateHelperKey: "t",
    Registry:          registry,
    OnMissing: func(locale, key string, args []any, err error) string {
        return fmt.Sprintf("[missing:%s]", key)
    },
})

tmpl := template.New("page").Funcs(helpers)
```

Template usage:

```
{{t .Locale "home.greeting" .Name}}
{{format_date .Locale .Timestamp}}
{{format_currency .Locale .Amount "USD"}}
```

## Built-in Formatters

The package includes formatters for common use cases:

- `FormatDate(locale, time)` - Date formatting
- `FormatDateTime(locale, time)` - DateTime formatting
- `FormatTime(locale, time)` - Time formatting
- `FormatCurrency(locale, amount, currency)` - Currency formatting
- `FormatNumber(locale, value, decimals)` - Number formatting
- `FormatPercent(locale, value, decimals)` - Percentage formatting
- `FormatOrdinal(locale, value)` - Ordinal number formatting
- `FormatList(locale, items)` - List formatting with commas and conjunctions
- `FormatMeasurement(locale, value, unit)` - Measurement formatting

Custom formatters can be registered per locale:

```go
registry := i18n.NewFormatterRegistry()
registry.Register("format_currency", func(locale string, value float64, currency string) string {
    if locale == "es" && currency == "EUR" {
        return fmt.Sprintf("%.2f �", value)
    }
    return i18n.FormatCurrency(locale, value, currency)
})
```

## Translation Hooks

Hooks allow interception of translation calls for logging, metrics, or debugging:

```go
cfg, err := i18n.NewConfig(
    i18n.WithTranslatorHooks(i18n.TranslationHookFuncs{
        Before: func(ctx *i18n.TranslatorHookContext) {
            // Called before translation lookup
        },
        After: func(ctx *i18n.TranslatorHookContext) {
            // Called after translation lookup
            // ctx.Result and ctx.Error available
        },
    }),
)
```

## Configuration Options

- `WithDefaultLocale(locale)` - Set default locale
- `WithLocales(...locales)` - Register supported locales
- `WithLoader(loader)` - Set translation loader
- `WithStore(store)` - Set custom store implementation
- `WithFallbackResolver(resolver)` - Set custom fallback resolver
- `WithFallback(locale, ...fallbacks)` - Configure fallback chain for a locale
- `WithFormatter(formatter)` - Set custom formatter
- `WithTranslatorHooks(...hooks)` - Add translation hooks

## Error Handling

The package defines standard errors:

- `ErrMissingTranslation` - Translation not found in any locale including fallbacks
- `ErrNotImplemented` - Feature not implemented

## License

MIT
