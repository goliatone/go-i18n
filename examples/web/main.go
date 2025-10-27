package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/goliatone/go-i18n"
)

var (
	translator    i18n.Translator
	tmpl          *template.Template
	registry      *i18n.FormatterRegistry
	helperFuncs   map[string]any
	localeCatalog *i18n.LocaleCatalog
	localeOptions []LocaleOption
	defaultLocale string
)

type LocaleOption struct {
	Code  string
	Label string
	Beta  bool
}

type PageData struct {
	Locale        string
	LocaleName    string
	Locales       []LocaleOption
	Title         string
	UserName      string
	ItemCount     int
	OrderDate     time.Time
	CartTotal     float64
	Completion    float64
	CartWeight    float64
	IsRTL         bool
	FallbackChain string
}

func main() {
	if err := setup(); err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", homeHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	fmt.Println("Server starting on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func setup() error {
	messageLoader := i18n.NewFileLoader(
		filepath.Join("locales", "messages.json"),
	)

	culturePath := filepath.Join("locales", "culture_data.json")
	cultureLoader := i18n.NewCultureDataLoader(culturePath)
	data, err := cultureLoader.Load()
	if err != nil {
		return fmt.Errorf("load culture data: %w", err)
	}

	formatterLocales := collectActiveLocaleCodes(data)

	cfg, err := i18n.NewConfig(
		i18n.WithLoader(messageLoader),
		i18n.EnablePluralization(filepath.Join("..", "..", "testdata", "cldr_cardinal.json")),
		i18n.WithFormatterLocales(formatterLocales...),
		i18n.WithCultureData(culturePath),
	)
	if err != nil {
		return err
	}

	translator, err = cfg.BuildTranslator()
	if err != nil {
		return err
	}

	registry = cfg.FormatterRegistry()
	localeCatalog = cfg.LocaleCatalog()
	defaultLocale = cfg.DefaultLocale
	if defaultLocale == "" && localeCatalog != nil {
		if codes := localeCatalog.ActiveLocaleCodes(); len(codes) > 0 {
			defaultLocale = codes[0]
		}
	}
	localeOptions = buildLocaleOptions(localeCatalog)

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

	printAvailableLocales()

	return nil
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	locale := resolveLocale(r.URL.Query().Get("lang"))
	meta := i18n.LocaleMetadata{Code: locale}
	if localeCatalog != nil {
		if m, ok := localeCatalog.Locale(locale); ok {
			meta = m
		}
	}

	title, _ := translator.Translate(locale, "site.title")

	orderDate := time.Now()
	cartTotal := 129.95
	cartWeight := 2.75
	isRTL := metadataBool(meta.Metadata, "rtl")
	fallbackChain := ""
	if localeCatalog != nil {
		if chain := localeCatalog.Fallbacks(locale); len(chain) > 0 {
			fallbackChain = strings.Join(chain, " → ")
		}
	}

	data := PageData{
		Locale:        locale,
		LocaleName:    displayName(meta),
		Locales:       localeOptions,
		Title:         title,
		UserName:      "Guest",
		ItemCount:     3,
		OrderDate:     orderDate,
		CartTotal:     cartTotal,
		Completion:    0.42,
		CartWeight:    cartWeight,
		IsRTL:         isRTL,
		FallbackChain: fallbackChain,
	}

	log.Printf("render locale=%s", locale)
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("Template error: %v", err)
	}
}

func resolveLocale(requested string) string {
	if localeCatalog == nil {
		if requested == "" {
			return defaultLocale
		}
		return requested
	}

	code := normalizeLocaleCode(requested)
	if code != "" && localeCatalog.IsActive(code) {
		return code
	}

	if defaultLocale != "" {
		return defaultLocale
	}

	if codes := localeCatalog.ActiveLocaleCodes(); len(codes) > 0 {
		return codes[0]
	}

	return code
}

func buildLocaleOptions(catalog *i18n.LocaleCatalog) []LocaleOption {
	if catalog == nil {
		return nil
	}

	var options []LocaleOption
	for _, code := range catalog.ActiveLocaleCodes() {
		meta, ok := catalog.Locale(code)
		if !ok {
			continue
		}
		options = append(options, LocaleOption{
			Code:  code,
			Label: displayName(meta),
			Beta:  metadataBool(meta.Metadata, "beta"),
		})
	}
	return options
}

func collectActiveLocaleCodes(data *i18n.CultureData) []string {
	if data == nil || len(data.Locales) == 0 {
		return nil
	}

	codes := make([]string, 0, len(data.Locales))
	for code, definition := range data.Locales {
		if definition.Active != nil && !*definition.Active {
			continue
		}
		codes = append(codes, normalizeLocaleCode(code))
	}
	sort.Strings(codes)
	return codes
}

func normalizeLocaleCode(locale string) string {
	if locale == "" {
		return ""
	}
	return strings.ReplaceAll(strings.TrimSpace(locale), "_", "-")
}

func metadataBool(metadata map[string]any, key string) bool {
	if metadata == nil {
		return false
	}
	if value, ok := metadata[key]; ok {
		switch v := value.(type) {
		case bool:
			return v
		case string:
			return strings.EqualFold(v, "true")
		}
	}
	return false
}

func displayName(meta i18n.LocaleMetadata) string {
	if meta.DisplayName != "" {
		return meta.DisplayName
	}
	if meta.Code != "" {
		return meta.Code
	}
	return "unknown"
}

func printAvailableLocales() {
	if len(localeOptions) == 0 {
		return
	}
	fmt.Println("Available languages:")
	for _, option := range localeOptions {
		label := option.Label
		if option.Beta {
			label = fmt.Sprintf("%s (beta)", label)
		}
		fmt.Printf(" - %s (?lang=%s)\n", label, option.Code)
	}
}
