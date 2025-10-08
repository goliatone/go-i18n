package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/goliatone/go-i18n"
)

var (
	translator i18n.Translator
	tmpl       *template.Template
	registry   *i18n.FormatterRegistry
)

type PageData struct {
	Locale         string
	Title          string
	UserName       string
	ItemCount      int
	OrderDate      time.Time
	FormattedDate  string
	CartTotal      float64
	Currency       string
	Trending       []string
	Completion     float64
	CartWeight     float64
	CartWeightUnit string
	SupportLine    string
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
	)
	if err != nil {
		return err
	}

	translator, err = cfg.BuildTranslator()
	if err != nil {
		return err
	}

	registry = cfg.FormatterRegistry()
	trailingCurrency := func(locale string, value float64, currency string) string {
		currency = strings.TrimSpace(strings.ToUpper(currency))
		number := i18n.FormatNumber(locale, value, 2)
		if fnAny, ok := registry.Formatter("format_number", locale); ok {
			if fn, ok := fnAny.(func(string, float64, int) string); ok {
				number = fn(locale, value, 2)
			}
		}
		if currency == "" {
			return number
		}
		return fmt.Sprintf("%s %s", number, currency)
	}
	registry.RegisterLocale("es", "format_currency", trailingCurrency)
	registry.RegisterLocale("es-MX", "format_currency", trailingCurrency)

	helpers := cfg.TemplateHelpers(translator, i18n.HelperConfig{
		TemplateHelperKey: "t",
		Registry:          registry,
		OnMissing: func(locale, key string, args []any, err error) string {
			return fmt.Sprintf("[missing:%s]", key)
		},
	})

	tmpl = template.Must(template.New("index.html").Funcs(helpers).ParseFiles(
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

	// Resolve localized formatters via registry (falls back to ISO defaults when missing).
	formatDateFnAny, ok := registry.Formatter("format_date", locale)
	if !ok {
		formatDateFnAny = i18n.FormatDate
	}
	formatDateFn, _ := formatDateFnAny.(func(string, time.Time) string)

	currency := map[string]string{
		"en":    "USD",
		"es":    "EUR",
		"es-MX": "MXN",
		"el":    "EUR",
		"ar":    "AED",
	}[locale]
	if currency == "" {
		currency = "USD"
	}

	trendingByLocale := map[string][]string{
		"en":    {"coffee", "tea", "cake"},
		"es":    {"café", "té", "pastel"},
		"es-MX": {"café", "pan dulce", "chocolate"},
		"el":    {"καφές", "τσάι", "κέικ"},
		"ar":    {"قهوة", "شاي", "كعكة"},
	}
	trending := trendingByLocale[locale]
	if len(trending) == 0 {
		trending = trendingByLocale["en"]
	}

	supportLines := map[string]string{
		"en":    "+1 555 010 4242",
		"es":    "+34 900 123 456",
		"es-MX": "+52 55 1234 5678",
		"el":    "+30 210 123 4567",
		"ar":    "+971 4 123 4567",
	}
	support := supportLines[locale]
	if support == "" {
		support = supportLines["en"]
	}

	orderDate := time.Now()

	data := PageData{
		Locale:         locale,
		Title:          title,
		UserName:       "Guest",
		ItemCount:      3,
		OrderDate:      orderDate,
		FormattedDate:  formatDateFn(locale, orderDate),
		CartTotal:      129.95,
		Currency:       currency,
		Trending:       trending,
		Completion:     0.42,
		CartWeight:     2.75,
		CartWeightUnit: "kg",
		SupportLine:    support,
	}

	if locale == "en" {
		data.CartWeight = data.CartWeight * 2.20462
		data.CartWeightUnit = "lb"
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("Template error: %v", err)
	}
}
