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

### Formatter Registry & Default Locales

- `NewFormatterRegistry` eagerly registers locale-aware providers for English (`en`) and Spanish (`es`) using `golang.org/x/text` and CLDR bundle data.
- The registry shares the same fallback resolver as the translator; regional locales such as `es-MX` automatically fall back to their parent (`es`) when a dedicated provider is missing.
- Formatter func maps are memoised per locale. Any call to `Register`, `RegisterLocale`, `RegisterProvider`, or `RegisterTypedProvider` invalidates the cache so new helpers become visible immediately.
- Template helpers wrap each formatter, defaulting to the registry’s primary locale when templates omit a locale argument. Advanced templates can access `formatter_funcs` to inspect the resolved helper map directly.

### Generating Locale Bundles

Locale-specific CLDR bundles are generated via the helper in `cmd/i18n-formatters`.

1. Install the CLDR archive (`taskfile cldr:install`) and export `CLDR_CORE_DIR`.
2. Run the generator with the full Go toolchain path:
   ```bash
   /Users/goliatone/.g/go/bin/go run ./cmd/i18n-formatters \
     -locale=en -locale=es -locale=el \
     -cldr "${CLDR_CORE_DIR}" \
     -out formatters_cldr_data.go
   ```
3. Check the generated file into version control so builds remain deterministic.
4. Add the new locale to `WithFormatterLocales(...)` (or `WithLocales(...)`) so the registry ensures provider coverage during configuration.

## Built-in Formatters

The package includes locale-aware formatters for common use cases. Defaults are sourced from CLDR snapshots bundled in `formatters_cldr_data.go` and `golang.org/x/text` primitives.

- `FormatDate(locale, time)` - Date formatting
- `FormatDateTime(locale, time)` - DateTime formatting
- `FormatTime(locale, time)` - Time formatting
- `FormatCurrency(locale, amount, currency)` - Currency formatting
- `FormatNumber(locale, value, decimals)` - Number formatting
- `FormatPercent(locale, value, decimals)` - Percentage formatting
- `FormatOrdinal(locale, value)` - Ordinal number formatting
- `FormatList(locale, items)` - List formatting with commas and conjunctions
- `FormatMeasurement(locale, value, unit)` - Measurement formatting
- `FormatPhone(locale, raw)` - Phone metadata formatting

Custom formatters can be registered per locale:

```go
registry := i18n.NewFormatterRegistry()
registry.Register("format_currency", func(locale string, value float64, currency string) string {
    if locale == "es" && currency == "EUR" {
        return fmt.Sprintf("%.2f EUR", value)
    }
    return i18n.FormatCurrency(locale, value, currency)
})
```

### Phone Dial Plans & Libphonenumber Adapter

- The registry ships curated dial plans for high-traffic locales (`en`, `es`) so `format_phone` formats inputs into `+<country> <groups>` without extra setup.
- Register additional metadata-driven plans with `i18n.RegisterPhoneDialPlan(locale, i18n.PhoneDialPlan{ ... })`, or provide a fully custom formatter via `i18n.RegisterPhoneFormatter`.
- Query built-in coverage using `i18n.DefaultPhoneDialPlan(locale)` when you need to introspect the bundled configuration.

Optional libphonenumber integration lives in `modules/libphonenumber` with its own `go.mod`. Consumers opt-in explicitly:

```go
import libphone "github.com/goliatone/go-i18n/modules/libphonenumber"

func init() {
    libphone.RegisterMany([]string{"en-US", "es", "fr"})
    // Or specialise behaviour:
    libphone.Register("mx", libphone.WithRegion("MX"))
}
```

The adapter pulls `github.com/nyaruka/phonenumbers`, offering Google’s phone parsing and formatting while keeping the core module lean.

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

## Culture Data & Formatting Rules

Applications can provide locale-specific business data and formatting rules through a single JSON file. The library includes embedded defaults for common locales (en, es, el) and automatically merges application-provided data.

### Culture Data File Format

```json
{
  "currency_codes": {
    "en": "USD",
    "es": "EUR",
    "ar": "AED"
  },
  "support_numbers": {
    "en": "+1 555 010 4242",
    "es": "+34 900 123 456"
  },
  "lists": {
    "trending_products": {
      "en": ["coffee", "tea", "cake"],
      "es": ["café", "té", "pastel"]
    }
  },
  "measurement_preferences": {
    "default": {
      "weight": {"unit": "kg"}
    },
    "en": {
      "weight": {
        "unit": "lb",
        "conversion_from": {"kg": 2.20462}
      }
    }
  },
  "formatting_rules": {
    "ar": {
      "locale": "ar",
      "date_patterns": {
        "pattern": "{day}/{month}/{year}",
        "day_first": true,
        "month_style": "number"
      },
      "currency_rules": {
        "pattern": "{amount} {symbol}",
        "symbol_position": "after",
        "decimal_separator": ".",
        "thousand_separator": ",",
        "decimals": 2
      },
      "month_names": [
        "يناير", "فبراير", "مارس", "أبريل", "مايو", "يونيو",
        "يوليو", "أغسطس", "سبتمبر", "أكتوبر", "نوفمبر", "ديسمبر"
      ],
      "time_format": {
        "use_24_hour": false,
        "pattern": "3:04 PM"
      }
    }
  }
}
```

### Using Culture Data

```go
cfg, err := i18n.NewConfig(
    i18n.WithLocales("en", "es", "ar"),
    i18n.WithCultureData("locales/culture_data.json"),
)

// Access culture service
cultureService := cfg.CultureService()
currencyCode, _ := cultureService.GetCurrencyCode("ar")
supportNumber, _ := cultureService.GetSupportNumber("es")
products, _ := cultureService.GetList("en", "trending_products")

// Access in templates
helpers := cfg.TemplateHelpers(translator, i18n.HelperConfig{
    LocaleKey: "Locale",
})
// Available helpers: currency_code, support_number, list, measurement_pref
```

### Culture Data Features

- **Embedded Defaults**: Library includes formatting rules for en, es, el
- **Application Override**: Provide custom rules that merge with or override defaults
- **Locale Fallback**: Uses same fallback chain as translations (e.g., ar-SA → ar → en)
- **Per-Locale Overrides**: Use `WithCultureOverride(locale, path)` for locale-specific files

### Formatting Rules

The `formatting_rules` section allows applications to customize how dates, times, currencies, and numbers are formatted for each locale:

- **date_patterns**: Date format patterns with placeholders `{day}`, `{month}`, `{year}`
- **currency_rules**: Currency symbol placement and separators
- **month_names**: Localized month names
- **time_format**: 12/24-hour clock preference

Custom formatting rules automatically integrate with the formatter registry and are used by `format_date`, `format_time`, `format_currency` template helpers.

## Configuration Options

- `WithDefaultLocale(locale)` - Set default locale
- `WithLocales(...locales)` - Register supported locales
- `WithLoader(loader)` - Set translation loader
- `WithStore(store)` - Set custom store implementation
- `WithFallbackResolver(resolver)` - Set custom fallback resolver
- `WithFallback(locale, ...fallbacks)` - Configure fallback chain for a locale
- `WithFormatter(formatter)` - Set custom formatter
- `WithFormatterLocales(...locales)` - Configure formatter provider coverage and fallback scaffolding
- `WithFormatterProvider(locale, provider)` - Inject custom formatter providers per locale
- `WithTranslatorHooks(...hooks)` - Add translation hooks
- `WithCultureData(path)` - Load culture data and formatting rules from JSON file
- `WithCultureOverride(locale, path)` - Add locale-specific culture data override

## Error Handling

The package defines standard errors:

- `ErrMissingTranslation` - Translation not found in any locale including fallbacks
- `ErrNotImplemented` - Feature not implemented

## License

MIT
