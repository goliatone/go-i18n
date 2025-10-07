package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/text/language"
	cldr "golang.org/x/text/unicode/cldr"
)

type localeSpec struct {
	Locale    string
	Territory string
}

type generatorConfig struct {
	pkg      string
	out      string
	cldrPath string
	locales  []localeSpec
}

type bundlePayload struct {
	Locale      string
	List        listPatterns
	Ordinal     string
	Measurement map[string]string
	Phone       phoneMetadata
}

var emptyRegion language.Region

type listPatterns struct {
	Pair   string
	Start  string
	Middle string
	End    string
}

type phoneMetadata struct {
	CountryCode    string
	NationalPrefix string
	Groups         []int
}

var defaultMeasurementUnits = map[string]string{
	"km": "length-kilometer",
	"kg": "mass-kilogram",
	"m":  "length-meter",
	"mi": "length-mile",
	"lb": "mass-pound",
}

type localeFlag struct {
	items []string
}

func (f *localeFlag) String() string {
	return strings.Join(f.items, ",")
}

func (f *localeFlag) Set(value string) error {
	parts := strings.Split(value, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		f.items = append(f.items, part)
	}
	return nil
}

func main() {
	cfg, err := parseFlags()
	if err != nil {
		reportError(err)
	}

	if err := run(cfg); err != nil {
		reportError(err)
	}
}

func reportError(err error) {
	fmt.Fprintf(os.Stderr, "i18n-formatters: %v\n", err)
	os.Exit(1)
}

func parseFlags() (generatorConfig, error) {
	var cfg generatorConfig
	var localeList localeFlag

	flag.StringVar(&cfg.pkg, "pkg", "i18n", "package name for generated file")
	flag.StringVar(&cfg.out, "out", "formatters_cldr_data.go", "path to generated Go file")
	flag.StringVar(&cfg.cldrPath, "cldr", "", "path to CLDR core data directory (expects subdirectories like main/ and supplemental/)")
	flag.Var(&localeList, "locale", "locale to generate (optionally include territory using locale:REGION). Repeat flag to add more.")

	flag.Parse()

	if len(localeList.items) == 0 {
		return generatorConfig{}, errors.New("at least one -locale value is required")
	}

	for _, spec := range localeList.items {
		parsed, err := parseLocaleSpec(spec)
		if err != nil {
			return generatorConfig{}, err
		}
		cfg.locales = append(cfg.locales, parsed)
	}

	if cfg.cldrPath == "" {
		cfg.cldrPath = os.Getenv("CLDR_CORE_DIR")
	}

	if cfg.cldrPath == "" {
		return generatorConfig{}, errors.New("missing CLDR data directory (set -cldr or CLDR_CORE_DIR)")
	}

	return cfg, nil
}

func run(cfg generatorConfig) error {
	data, err := loadCLDR(cfg.cldrPath)
	if err != nil {
		return err
	}

	supplemental := data.Supplemental()
	var bundles []bundlePayload

	for _, spec := range cfg.locales {
		if err := normalizeLocaleSpec(&spec); err != nil {
			return err
		}

		payload, err := buildBundle(data, supplemental, spec)
		if err != nil {
			return fmt.Errorf("build bundle for %s: %w", spec.Locale, err)
		}
		bundles = append(bundles, payload)
	}

	sort.Slice(bundles, func(i, j int) bool {
		return bundles[i].Locale < bundles[j].Locale
	})

	source, err := renderSource(cfg.pkg, bundles)
	if err != nil {
		return err
	}

	if err := ensureDir(cfg.out); err != nil {
		return err
	}

	return os.WriteFile(cfg.out, source, 0o644)
}

func loadCLDR(path string) (*cldr.CLDR, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat CLDR directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("CLDR path %q is not a directory", path)
	}

	var decoder cldr.Decoder
	decoder.SetSectionFilter("main", "supplemental")

	data, err := decoder.DecodePath(path)
	if err != nil {
		return nil, fmt.Errorf("decode CLDR data: %w", err)
	}
	return data, nil
}

func parseLocaleSpec(input string) (localeSpec, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return localeSpec{}, errors.New("empty locale value")
	}

	spec := localeSpec{}
	if strings.Contains(input, ":") {
		parts := strings.SplitN(input, ":", 2)
		spec.Locale = strings.TrimSpace(parts[0])
		spec.Territory = strings.ToUpper(strings.TrimSpace(parts[1]))
	} else {
		spec.Locale = input
	}

	if spec.Locale == "" {
		return localeSpec{}, fmt.Errorf("invalid locale spec %q", input)
	}
	return spec, nil
}

func normalizeLocaleSpec(spec *localeSpec) error {
	if spec == nil {
		return errors.New("nil locale spec")
	}

	spec.Locale = strings.ReplaceAll(strings.TrimSpace(spec.Locale), "_", "-")
	if spec.Locale == "" {
		return errors.New("empty locale identifier")
	}

	if spec.Territory != "" {
		spec.Territory = strings.ToUpper(spec.Territory)
		return nil
	}

	// Attempt to derive territory from locale region.
	if tag, err := language.Parse(spec.Locale); err == nil {
		if region, _ := tag.Region(); region != emptyRegion {
			spec.Territory = strings.ToUpper(region.String())
			return nil
		}
	}

	spec.Territory = ""
	return nil
}

func buildBundle(data *cldr.CLDR, supplemental *cldr.SupplementalData, spec localeSpec) (bundlePayload, error) {
	var payload bundlePayload
	payload.Locale = spec.Locale

	ldml := findLDML(data, spec.Locale)
	if ldml == nil {
		return payload, fmt.Errorf("missing LDML data")
	}

	payload.List = extractListPatterns(ldml)
	payload.Ordinal = detectOrdinalSystem(spec.Locale)
	payload.Measurement = extractMeasurementUnits(ldml)
	payload.Phone = extractPhoneMetadata(supplemental, spec)

	return payload, nil
}

func findLDML(data *cldr.CLDR, locale string) *cldr.LDML {
	if data == nil {
		return nil
	}
	candidate := strings.ReplaceAll(locale, "-", "_")
	for {
		if candidate == "" {
			break
		}
		if ldml := data.RawLDML(candidate); ldml != nil {
			return ldml
		}
		if idx := strings.LastIndex(candidate, "_"); idx >= 0 {
			candidate = candidate[:idx]
			continue
		}
		break
	}
	return data.RawLDML("root")
}

func extractListPatterns(ldml *cldr.LDML) listPatterns {
	var patterns listPatterns
	if ldml == nil || ldml.ListPatterns == nil {
		return patterns
	}

	for _, pattern := range ldml.ListPatterns.ListPattern {
		common := pattern.GetCommon()
		if common != nil && common.Type != "" && common.Type != "standard" {
			continue
		}

		for _, part := range pattern.ListPatternPart {
			if part == nil {
				continue
			}
			switch strings.ToLower(part.Type) {
			case "2":
				patterns.Pair = part.Data()
			case "start":
				patterns.Start = part.Data()
			case "middle":
				patterns.Middle = part.Data()
			case "end":
				patterns.End = part.Data()
			}
		}

		if patterns.Pair != "" {
			break
		}
	}

	return patterns
}

func detectOrdinalSystem(locale string) string {
	locale = strings.ToLower(locale)
	switch {
	case strings.HasPrefix(locale, "es"):
		return "spanish"
	default:
		return "english"
	}
}

func extractMeasurementUnits(ldml *cldr.LDML) map[string]string {
	result := make(map[string]string, len(defaultMeasurementUnits))
	if ldml == nil || ldml.Units == nil {
		return result
	}

	for _, length := range ldml.Units.UnitLength {
		if length == nil {
			continue
		}
		common := length.GetCommon()
		if common != nil && common.Type != "" && common.Type != "long" {
			continue
		}

		for _, unit := range length.Unit {
			if unit == nil {
				continue
			}
			unitType := ""
			if c := unit.GetCommon(); c != nil {
				unitType = c.Type
			}
			if unitType == "" {
				continue
			}

			for key, canonical := range defaultMeasurementUnits {
				if unitType != canonical {
					continue
				}
				if _, exists := result[key]; exists {
					continue
				}
				result[key] = selectUnitDisplayName(unit.DisplayName)
			}
		}
	}

	return result
}

func selectUnitDisplayName(list []*struct {
	cldr.Common
	Count string `xml:"count,attr"`
}) string {
	var fallback string
	for _, entry := range list {
		if entry == nil {
			continue
		}
		if entry.Count == "" || strings.EqualFold(entry.Count, "other") {
			return entry.Data()
		}
		if fallback == "" {
			fallback = entry.Data()
		}
	}
	return fallback
}

func extractPhoneMetadata(supplemental *cldr.SupplementalData, spec localeSpec) phoneMetadata {
	var meta phoneMetadata
	if supplemental == nil || supplemental.TelephoneCodeData == nil {
		return meta
	}

	territory := spec.Territory
	if territory == "" {
		if tag, err := language.Parse(spec.Locale); err == nil {
			if region, _ := tag.Region(); region != emptyRegion {
				territory = region.String()
			}
		}
	}
	if territory == "" {
		return meta
	}

	for _, entry := range supplemental.TelephoneCodeData.CodesByTerritory {
		if entry == nil {
			continue
		}
		if !strings.EqualFold(entry.Territory, territory) {
			continue
		}
		if len(entry.TelephoneCountryCode) == 0 {
			break
		}
		code := entry.TelephoneCountryCode[0]
		meta.CountryCode = strings.TrimSpace(code.Code)
		meta.NationalPrefix = ""

		if length := nationalNumberLength(code); length > 0 {
			meta.Groups = []int{length}
		}
		break
	}

	return meta
}

func nationalNumberLength(code *struct {
	cldr.Common
	Code string `xml:"code,attr"`
	From string `xml:"from,attr"`
	To   string `xml:"to,attr"`
}) int {
	if code == nil {
		return 0
	}
	if digits := extractDigits(code.From); digits != "" {
		return len(digits)
	}
	if digits := extractDigits(code.To); digits != "" {
		return len(digits)
	}
	return 0
}

func extractDigits(input string) string {
	var b strings.Builder
	for _, r := range input {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func renderSource(pkg string, bundles []bundlePayload) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("// Code generated by i18n-formatters. DO NOT EDIT.\n\n")
	fmt.Fprintf(&buf, "package %s\n\n", pkg)

	buf.WriteString("type cldrListPatterns struct {\n")
	buf.WriteString("\tPair   string\n")
	buf.WriteString("\tStart  string\n")
	buf.WriteString("\tMiddle string\n")
	buf.WriteString("\tEnd    string\n")
	buf.WriteString("}\n\n")

	buf.WriteString("type cldrOrdinalRules struct {\n")
	buf.WriteString("\tSystem string\n")
	buf.WriteString("}\n\n")

	buf.WriteString("type cldrMeasurementData struct {\n")
	buf.WriteString("\tUnits map[string]string\n")
	buf.WriteString("}\n\n")

	buf.WriteString("type cldrPhoneMetadata struct {\n")
	buf.WriteString("\tCountryCode    string\n")
	buf.WriteString("\tNationalPrefix string\n")
	buf.WriteString("\tGroups         []int\n")
	buf.WriteString("}\n\n")

	buf.WriteString("type cldrBundle struct {\n")
	buf.WriteString("\tList        cldrListPatterns\n")
	buf.WriteString("\tOrdinal     cldrOrdinalRules\n")
	buf.WriteString("\tMeasurement cldrMeasurementData\n")
	buf.WriteString("\tPhone       cldrPhoneMetadata\n")
	buf.WriteString("}\n\n")

	buf.WriteString("var cldrBundles = map[string]cldrBundle{\n")
	for _, bundle := range bundles {
		fmt.Fprintf(&buf, "\t%q: {\n", bundle.Locale)
		buf.WriteString("\t\tList: cldrListPatterns{\n")
		fmt.Fprintf(&buf, "\t\t\tPair: %q,\n", bundle.List.Pair)
		fmt.Fprintf(&buf, "\t\t\tStart: %q,\n", bundle.List.Start)
		fmt.Fprintf(&buf, "\t\t\tMiddle: %q,\n", bundle.List.Middle)
		fmt.Fprintf(&buf, "\t\t\tEnd: %q,\n", bundle.List.End)
		buf.WriteString("\t\t},\n")

		buf.WriteString("\t\tOrdinal: cldrOrdinalRules{\n")
		fmt.Fprintf(&buf, "\t\t\tSystem: %q,\n", bundle.Ordinal)
		buf.WriteString("\t\t},\n")

		buf.WriteString("\t\tMeasurement: cldrMeasurementData{\n")
		buf.WriteString("\t\t\tUnits: map[string]string{\n")
		var unitKeys []string
		for key := range bundle.Measurement {
			unitKeys = append(unitKeys, key)
		}
		sort.Strings(unitKeys)
		for _, key := range unitKeys {
			fmt.Fprintf(&buf, "\t\t\t\t%q: %q,\n", key, bundle.Measurement[key])
		}
		buf.WriteString("\t\t\t},\n")
		buf.WriteString("\t\t},\n")

		buf.WriteString("\t\tPhone: cldrPhoneMetadata{\n")
		fmt.Fprintf(&buf, "\t\t\tCountryCode: %q,\n", bundle.Phone.CountryCode)
		fmt.Fprintf(&buf, "\t\t\tNationalPrefix: %q,\n", bundle.Phone.NationalPrefix)
		buf.WriteString("\t\t\tGroups: []int{")
		for i, g := range bundle.Phone.Groups {
			if i > 0 {
				buf.WriteString(", ")
			}
			fmt.Fprintf(&buf, "%d", g)
		}
		buf.WriteString("},\n")
		buf.WriteString("\t\t},\n")

		buf.WriteString("\t},\n")
	}
	buf.WriteString("}\n\n")

	buf.WriteString("var generatedCLDRLocales = []string{\n")
	for _, bundle := range bundles {
		fmt.Fprintf(&buf, "\t%q,\n", bundle.Locale)
	}
	buf.WriteString("}\n\n")

	buf.WriteString("func RegisterGeneratedCLDRFormatters(registry *FormatterRegistry) {\n")
	buf.WriteString("\tRegisterCLDRFormatters(registry, generatedCLDRLocales...)\n")
	buf.WriteString("}\n\n")

	buf.WriteString("func GeneratedCLDRLocales() []string {\n")
	buf.WriteString("\treturn append([]string{}, generatedCLDRLocales...)\n")
	buf.WriteString("}\n")

	return format.Source(buf.Bytes())
}

func ensureDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}
