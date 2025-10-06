package basic

import (
	"bytes"
	"fmt"
	"path/filepath"
	"text/template"
	"time"

	"github.com/goliatone/go-i18n"
)

// Run constructs a translator, wires template helpers, and renders a sample invoice view.
func Run() error {
	loader := i18n.NewFileLoader(
		filepath.Join("examples", "basic", "locales", "en.json"),
		filepath.Join("examples", "basic", "locales", "es.json"),
	)

	cfg, err := i18n.NewConfig(
		i18n.WithLocales("en", "es"),
		i18n.WithDefaultLocale("en"),
		i18n.WithLoader(loader),
		i18n.EnablePluralization(filepath.Join("testdata", "cldr_cardinal.json")),
		i18n.WithFallback("es", "en"),
		i18n.WithTranslatorHooks(i18n.TranslationHookFuncs{
			After: func(ctx *i18n.TranslatorHookContext) {
				fmt.Printf("lookup[%s] %s => %q (err=%v)\n", ctx.Locale, ctx.Key, ctx.Result, ctx.Error)
			},
		}),
	)
	if err != nil {
		return err
	}

	translator, err := cfg.BuildTranslator()
	if err != nil {
		return err
	}

	registry := i18n.NewFormatterRegistry()
	baseCurrency := i18n.FormatCurrency
	registry.Register("format_currency", func(locale string, value float64, currency string) string {
		if locale == "es" && currency == "EUR" {
			return fmt.Sprintf("%.2f €", value)
		}
		return baseCurrency(locale, value, currency)
	})

	helpers := i18n.TemplateHelpers(translator, i18n.HelperConfig{
		TemplateHelperKey: "t",
		Registry:          registry,
		OnMissing: func(locale, key string, args []any, err error) string {
			return fmt.Sprintf("[missing:%s]", key)
		},
	})

	tmpl := template.Must(template.New("invoice").Funcs(helpers).Parse(invoiceTemplate))

	data := invoiceData{
		Locale:   "es",
		Customer: "Laura",
		OrderID:  "INV-42",
		Total:    42.54,
		PlacedAt: time.Date(2024, 5, 18, 10, 30, 0, 0, time.UTC),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return err
	}

	fmt.Println("Rendered template:")
	fmt.Println(buf.String())

	farewell, err := translator.Translate("es", "invoice.farewell")
	if err != nil {
		return err
	}
	fmt.Printf("Fallback example: %s\n", farewell)

	if _, err := translator.Translate("es", "does.not.exist"); err != nil {
		fmt.Printf("Missing translation lookup => %v\n", err)
	}

	return nil
}

type invoiceData struct {
	Locale   string
	Customer string
	OrderID  string
	Total    float64
	PlacedAt time.Time
}

const invoiceTemplate = `{{$locale := .Locale}}
{{t $locale "invoice.subtitle" .OrderID}}
{{t $locale "invoice.greeting" .Customer (format_currency $locale .Total "EUR")}}
Fecha: {{format_date $locale .PlacedAt}}
{{t $locale "marketing.banner"}}
`
