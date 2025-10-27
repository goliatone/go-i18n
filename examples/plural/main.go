package main

import (
	"bytes"
	"fmt"
	"log"
	"path/filepath"
	"text/template"

	"github.com/goliatone/go-i18n"
)

func main() {
	if err := Run(); err != nil {
		log.Fatal(err)
	}
}

// metadataTranslator mirrors the staged translator interface that exposes metadata.
type metadataTranslator interface {
	TranslateWithMetadata(locale, key string, args ...any) (string, map[string]any, error)
}

// Run demonstrates plural-aware translation flows while staging under .tmp.
func Run() error {
	loader := i18n.NewFileLoader(
		filepath.Join("locales", "en.json"),
		filepath.Join("locales", "es.json"),
		filepath.Join("locales", "el.json"),
	)

	cfg, err := i18n.NewConfig(
		i18n.WithLoader(loader),
		i18n.WithLocales("en", "es", "el", "el-CY"),
		i18n.WithDefaultLocale("en"),
		i18n.EnablePluralization(filepath.Join("..", "..", "testdata", "cldr_cardinal.json")),
		i18n.WithFallback("el-CY", "el"),
	)
	if err != nil {
		return err
	}

	translator, err := cfg.BuildTranslator()
	if err != nil {
		return err
	}

	metaTranslator, ok := translator.(metadataTranslator)
	if !ok {
		return fmt.Errorf("translator does not expose metadata")
	}

	helpers := i18n.TemplateHelpers(translator, i18n.HelperConfig{
		TemplateHelperKey: "t",
		OnMissing: func(locale, key string, args []any, missingErr error) string {
			return fmt.Sprintf("[missing:%s]", key)
		},
	})

	tmpl := template.Must(template.New("summary").Funcs(helpers).Parse(summaryTemplate))

	model := viewModel{
		Locale:         "es",
		Customer:       "Laura",
		CartItems:      3,
		PendingAlerts:  1,
		WarehouseStock: 27,
	}

	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, model); err != nil {
		return err
	}

	fmt.Println("Template output:\n" + rendered.String())

	fmt.Println("\nDirect translator lookups (Greek rules):")
	for _, count := range []int{1, 2, 11, 21} {
		text, metadata, err := metaTranslator.TranslateWithMetadata("el", "cart.items", i18n.WithCount(count))
		if err != nil {
			return err
		}
		fmt.Printf("el cart.items(%d) => %q category=%v\n", count, text, metadata["plural.category"])
	}

	_, missingMeta, err := metaTranslator.TranslateWithMetadata("el", "alerts.pending", i18n.WithCount(1))
	if err != nil {
		return err
	}
	if payload, ok := missingMeta["plural.missing"]; ok {
		fmt.Printf("missing plural payload => %v\n", payload)
	}

	return nil
}

type viewModel struct {
	Locale         string
	Customer       string
	CartItems      int
	PendingAlerts  int
	WarehouseStock int
}

const summaryTemplate = `{{$locale := .Locale}}
{{t $locale "order.heading" .Customer}}
{{$cart := translate_count $locale "cart.items" .CartItems}}
{{$alerts := translate_count $locale "alerts.pending" .PendingAlerts}}
{{$cart.text}}
{{$alerts.text}}{{with $alerts.plural.missing}} (fallback to {{.fallback}}){{end}}
{{t $locale "inventory.message" .WarehouseStock}}
`
