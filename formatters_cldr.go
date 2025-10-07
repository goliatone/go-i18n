package i18n

import (
	"strconv"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

// RegisterCLDRFormatters wires locale-specific helpers sourced from CLDR bundles.
func RegisterCLDRFormatters(registry *FormatterRegistry, locales ...string) {
	if registry == nil {
		return
	}

	for _, locale := range locales {
		trimmed := strings.TrimSpace(locale)
		if trimmed == "" {
			continue
		}

		if bundle, ok := cldrBundles[trimmed]; ok {
			registry.RegisterTypedProvider(trimmed, newCLDRProvider(trimmed, bundle))
		}
	}
}

type cldrProvider struct {
	locale  string
	bundle  cldrBundle
	tag     language.Tag
	printer *message.Printer
	funcs   map[string]any
}

func newCLDRProvider(locale string, bundle cldrBundle) *cldrProvider {
	tag := language.Make(locale)
	p := &cldrProvider{
		locale:  locale,
		bundle:  bundle,
		tag:     tag,
		printer: message.NewPrinter(tag),
	}

	p.funcs = map[string]any{
		"format_list":        p.formatList,
		"format_ordinal":     p.formatOrdinal,
		"format_measurement": p.formatMeasurement,
		"format_phone":       p.formatPhone,
	}

	return p
}

func (p *cldrProvider) Capabilities() FormatterCapabilities {
	return FormatterCapabilities{
		List:        true,
		Ordinal:     true,
		Measurement: true,
		Phone:       true,
	}
}

func (p *cldrProvider) Formatter(name string) (any, bool) {
	if p == nil {
		return nil, false
	}
	fn, ok := p.funcs[name]
	return fn, ok
}

func (p *cldrProvider) FuncMap() map[string]any {
	if p == nil {
		return nil
	}
	return cloneFuncMap(p.funcs)
}

func (p *cldrProvider) formatList(_ string, items []string) string {
	pattern := p.bundle.List
	pair := pattern.Pair
	start := pattern.Start
	middle := pattern.Middle
	end := pattern.End
	if end == "" {
		end = pair
	}

	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return applyListPattern(pair, items[0], items[1])
	default:
		if start == "" || middle == "" {
			head := strings.Join(items[:len(items)-1], ", ")
			return applyListPattern(end, head, items[len(items)-1])
		}
		result := applyListPattern(start, items[0], items[1])
		for i := 2; i < len(items)-1; i++ {
			result = applyListPattern(middle, result, items[i])
		}
		return applyListPattern(end, result, items[len(items)-1])
	}
}

func (p *cldrProvider) formatOrdinal(_ string, value int) string {
	switch p.bundle.Ordinal.System {
	case "spanish":
		return formatOrdinalWithSuffix(value, "º")
	default:
		return formatOrdinalISO(p.locale, value)
	}
}

func (p *cldrProvider) formatMeasurement(_ string, value float64, unit string) string {
	trimmedUnit := strings.TrimSpace(unit)
	formatted := p.printer.Sprintf("%v", number.Decimal(value))
	if trimmedUnit == "" {
		return formatted
	}

	lowered := strings.ToLower(trimmedUnit)
	if localized, ok := p.bundle.Measurement.Units[lowered]; ok && localized != "" {
		trimmedUnit = localized
	}

	return formatted + " " + trimmedUnit
}

func (p *cldrProvider) formatPhone(_ string, raw string) string {
	meta := p.bundle.Phone
	if meta.CountryCode == "" || len(meta.Groups) == 0 {
		return strings.TrimSpace(raw)
	}

	trimmed := strings.TrimSpace(raw)
	digits := extractDigits(trimmed)
	if len(digits) == 0 {
		return trimmed
	}

	total := 0
	for _, g := range meta.Groups {
		total += g
	}

	var national string
	switch {
	case strings.HasPrefix(digits, meta.CountryCode) && len(digits) >= len(meta.CountryCode)+total:
		national = digits[len(meta.CountryCode):]
	case len(digits) == total:
		national = digits
	default:
		return trimmed
	}

	if len(national) < total {
		return trimmed
	}

	var builder strings.Builder
	builder.WriteString("+")
	builder.WriteString(meta.CountryCode)
	builder.WriteString(" ")

	pos := 0
	for i, group := range meta.Groups {
		if group <= 0 || pos >= len(national) {
			break
		}
		upper := pos + group
		if upper > len(national) {
			upper = len(national)
		}
		if i > 0 {
			builder.WriteString(" ")
		}
		builder.WriteString(national[pos:upper])
		pos = upper
	}

	if pos < len(national) {
		builder.WriteString(" ")
		builder.WriteString(national[pos:])
	}

	return builder.String()
}

func applyListPattern(pattern, head, tail string) string {
	result := strings.ReplaceAll(pattern, "{0}", head)
	return strings.ReplaceAll(result, "{1}", tail)
}

func formatOrdinalWithSuffix(value int, suffix string) string {
	return strconv.Itoa(value) + suffix
}

func extractDigits(input string) string {
	var builder strings.Builder
	for _, r := range input {
		if r >= '0' && r <= '9' {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}
