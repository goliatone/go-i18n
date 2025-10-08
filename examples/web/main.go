package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/goliatone/go-i18n"
)

var (
	translator  i18n.Translator
	tmpl        *template.Template
	registry    *i18n.FormatterRegistry
	helperFuncs map[string]any
)

type PageData struct {
	Locale     string
	Title      string
	UserName   string
	ItemCount  int
	OrderDate  time.Time
	CartTotal  float64
	Completion float64
	CartWeight float64
}

func main() {
	if err := setup(); err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", homeHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	fmt.Println("Server starting on http://localhost:8080")
	fmt.Println("Available languages: English (?lang=en), Spanish (?lang=es), Greek (?lang=el), Arabic (?lang=ar)")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func setup() error {
	loader := i18n.NewFileLoader(
		filepath.Join("locales", "messages.json"),
	)

	cfg, err := i18n.NewConfig(
		i18n.WithLocales("en", "es", "es-MX", "el", "ar"),
		i18n.WithDefaultLocale("en"),
		i18n.WithLoader(loader),
		i18n.EnablePluralization(filepath.Join("..", "..", "testdata", "cldr_cardinal.json")),
		i18n.WithFallback("es", "en"),
		i18n.WithFallback("es-MX", "es"),
		i18n.WithFallback("el", "en"),
		i18n.WithFallback("ar", "en"),
		i18n.WithFormatterLocales("en", "es", "es-MX", "el", "ar"),
		i18n.WithCultureData(filepath.Join("locales", "culture_data.json")),
	)
	if err != nil {
		return err
	}

	translator, err = cfg.BuildTranslator()
	if err != nil {
		return err
	}

	registry = cfg.FormatterRegistry()

	helperFuncs = cfg.TemplateHelpers(translator, i18n.HelperConfig{
		TemplateHelperKey: "t",
		Registry:          registry,
		LocaleKey:         "Locale",
		OnMissing: func(locale, key string, args []any, err error) string {
			return fmt.Sprintf("[missing:%s]", key)
		},
	})

	tmpl = template.Must(template.New("index.html").Funcs(helperFuncs).ParseFiles(
		filepath.Join("templates", "index.html"),
	))

	return nil
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	locale := r.URL.Query().Get("lang")
	if locale == "" {
		locale = "en"
	}

	// Validate locale
	validLocales := map[string]bool{"en": true, "es": true, "es-MX": true, "el": true, "ar": true}
	if !validLocales[locale] {
		locale = "en"
	}

	title, _ := translator.Translate(locale, "site.title")

	orderDate := time.Now()
	cartTotal := 129.95
	cartWeight := 2.75

	data := PageData{
		Locale:     locale,
		Title:      title,
		UserName:   "Guest",
		ItemCount:  3,
		OrderDate:  orderDate,
		CartTotal:  cartTotal,
		Completion: 0.42,
		CartWeight: cartWeight,
	}

	log.Printf("render locale=%s", locale)
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("Template error: %v", err)
	}
}
