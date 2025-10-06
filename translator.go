package i18n

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Translator resolves a string for a given locale and message key.
type Translator interface {
	Translate(locale, key string, args ...any) (string, error)
}

// Formatter formats a template string with positional arguments
type Formatter interface {
	Format(template string, args ...any) (string, error)
}

// FormatterFunc adapts plain functions into Formatter
type FormatterFunc func(string, ...any) (string, error)

// Format impelements Fromatter for FormatterFunc
func (fn FormatterFunc) Format(template string, args ...any) (string, error) {
	if fn == nil {
		return template, nil
	}
	return fn(template, args...)
}

// SimpleTranslatorOption configures SimpleTranslator
type SimpleTranslatorOption func(*SimpleTranslator)

// SimpleTranslator performs in memory lookups backed by a Store
type SimpleTranslator struct {
	store         Store
	defaultLocale string
	formatter     Formatter
	resolver      FallbackResolver
}

type metadataTranslator interface {
	TranslateWithMetadata(locale, key string, args ...any) (string, map[string]any, error)
}

const (
	metadataPluralCount    = "plural.count"
	metadataPluralCategory = "plural.category"
	metadataPluralMessage  = "plural.message"
	metadataPluralMissing  = "plural.missing"
)

type translateOption interface {
	applyOption(*translateRuntime)
}

// TranslateOption configures translation behaviour (e.g. counts for plurals).
type TranslateOption interface {
	translateOption
}

type translateOptionFunc func(*translateRuntime)

func (fn translateOptionFunc) applyOption(rt *translateRuntime) {
	if fn == nil || rt == nil {
		return
	}
	fn(rt)
}

// WithCount annotates a translation call with a quantity used for plural selection.
func WithCount(value any) TranslateOption {
	return translateOptionFunc(func(rt *translateRuntime) {
		rt.setCount(value)
	})
}

type translateRuntime struct {
	formatArgs    []any
	hasCount      bool
	countValue    pluralOperands
	countLiteral  string
	countOriginal any
}

func newTranslateRuntime(args []any) translateRuntime {
	rt := translateRuntime{}
	for _, arg := range args {
		if opt, ok := arg.(translateOption); ok {
			opt.applyOption(&rt)
			continue
		}
		rt.formatArgs = append(rt.formatArgs, arg)
	}
	return rt
}

func (rt *translateRuntime) setCount(value any) {
	op, literal, ok := toPluralOperands(value)
	if !ok {
		rt.hasCount = false
		rt.countLiteral = ""
		rt.countOriginal = nil
		return
	}
	rt.hasCount = true
	rt.countValue = op
	rt.countLiteral = literal
	rt.countOriginal = value
}

type pluralOperands struct {
	n float64
	i int64
	v int
	w int
	f int64
	t int64
}

func NewSimpleTranslator(store Store, opts ...SimpleTranslatorOption) (*SimpleTranslator, error) {
	st := &SimpleTranslator{
		store:     store,
		formatter: FormatterFunc(sprintfFormatter),
		resolver:  NewStaticFallbackResolver(),
	}

	if st.store == nil {
		st.store = NewStaticStore(nil)
	}

	for _, opt := range opts {
		if opt != nil {
			opt(st)
		}
	}

	if st.formatter == nil {
		st.formatter = FormatterFunc(sprintfFormatter)
	}

	if st.resolver == nil {
		st.resolver = NewStaticFallbackResolver()
	}

	return st, nil
}

func WithTranslatorDefaultLocale(locale string) SimpleTranslatorOption {
	return func(st *SimpleTranslator) {
		st.defaultLocale = locale
	}
}

func WithTranslatorFormatter(formatter Formatter) SimpleTranslatorOption {
	return func(st *SimpleTranslator) {
		st.formatter = formatter
	}
}

func WithTranslatorFallbackResolver(resolver FallbackResolver) SimpleTranslatorOption {
	return func(st *SimpleTranslator) {
		st.resolver = resolver
	}
}

func (t *SimpleTranslator) Translate(locale, key string, args ...any) (string, error) {
	result, _, err := t.TranslateWithMetadata(locale, key, args...)
	return result, err
}

func (t *SimpleTranslator) TranslateWithMetadata(locale, key string, args ...any) (string, map[string]any, error) {
	if t == nil {
		return "", nil, ErrMissingTranslation
	}

	if key == "" {
		return "", nil, ErrMissingTranslation
	}

	runtime := newTranslateRuntime(args)

	primary := locale
	if primary == "" {
		primary = t.defaultLocale
	}

	if primary == "" {
		return "", nil, ErrMissingTranslation
	}

	for _, candidate := range t.lookupLocales(primary) {
		message, ok := t.store.Message(candidate, key)
		if !ok {
			continue
		}

		variant, category, missing := t.selectVariant(candidate, message, runtime)
		text, err := t.renderVariant(variant, runtime)
		if err != nil {
			return "", nil, err
		}

		metadata := map[string]any{
			metadataPluralMessage: variant.Template,
		}
		if runtime.hasCount {
			metadata[metadataPluralCount] = runtime.countOriginal
			metadata[metadataPluralCategory] = category
			if missing {
				metadata[metadataPluralMissing] = map[string]any{
					"requested": category,
					"fallback":  PluralOther,
				}
			}
		}

		return text, metadata, nil
	}

	return "", nil, ErrMissingTranslation
}

func (t *SimpleTranslator) lookupLocales(primary string) []string {
	order := make([]string, 0, 4)
	seen := make(map[string]struct{}, 4)

	appendLocale := func(locale string) {
		if locale == "" {
			return
		}

		if _, ok := seen[locale]; ok {
			return
		}
		seen[locale] = struct{}{}
		order = append(order, locale)
	}

	appendLocale(primary)

	for parent := parentLocaleTag(primary); parent != ""; parent = parentLocaleTag(parent) {
		appendLocale(parent)
	}

	if t.resolver != nil {
		for _, fb := range t.resolver.Resolve(primary) {
			appendLocale(fb)
		}
	}

	appendLocale(t.defaultLocale)

	return order
}

func sprintfFormatter(template string, args ...any) (string, error) {
	return fmt.Sprintf(template, args...), nil
}

func (t *SimpleTranslator) selectVariant(locale string, message Message, runtime translateRuntime) (MessageVariant, PluralCategory, bool) {
	variant, ok := message.Variant(PluralOther)
	if !ok {
		variant = MessageVariant{}
	}
	category := PluralOther
	missing := false

	if runtime.hasCount {
		resolved := t.resolvePluralCategory(locale, message, runtime.countValue)
		if resolved == "" {
			resolved = PluralOther
		}

		category = resolved

		hasExact := false
		if message.Variants != nil {
			_, hasExact = message.Variants[resolved]
		}

		if selected, ok := message.Variant(resolved); ok {
			variant = selected
			if !hasExact && resolved != PluralOther {
				missing = true
			}
		} else if resolved != PluralOther {
			missing = true
		}
	}

	return variant, category, missing
}

func (t *SimpleTranslator) renderVariant(variant MessageVariant, runtime translateRuntime) (string, error) {
	text := variant.Template
	if runtime.hasCount {
		text = strings.ReplaceAll(text, "{count}", runtime.countLiteral)
	}

	if len(runtime.formatArgs) == 0 || t.formatter == nil {
		return text, nil
	}

	return t.formatter.Format(text, runtime.formatArgs...)
}

func (t *SimpleTranslator) resolvePluralCategory(locale string, message Message, operands pluralOperands) PluralCategory {
	rules := t.ruleSetFor(locale)
	if rules != nil {
		if category := selectPluralCategory(rules, operands); category != "" {
			return category
		}
	}

	if message.Locale != "" && !strings.EqualFold(message.Locale, locale) {
		if rules = t.ruleSetFor(message.Locale); rules != nil {
			if category := selectPluralCategory(rules, operands); category != "" {
				return category
			}
		}
	}

	return PluralOther
}

func (t *SimpleTranslator) ruleSetFor(locale string) *PluralRuleSet {
	if t == nil || locale == "" {
		return nil
	}

	visited := make(map[string]struct{}, 4)
	current := locale
	for current != "" {
		if _, seen := visited[current]; seen {
			break
		}
		visited[current] = struct{}{}
		if rules, ok := t.store.Rules(current); ok {
			return rules
		}
		base := parentLocaleTag(current)
		if base == current {
			break
		}
		current = base
	}

	if t.defaultLocale != "" && !strings.EqualFold(locale, t.defaultLocale) {
		if rules, ok := t.store.Rules(t.defaultLocale); ok {
			return rules
		}
	}

	return nil
}

func parentLocaleTag(locale string) string {
	if idx := strings.LastIndex(locale, "-"); idx > 0 {
		return locale[:idx]
	}
	return ""
}

func selectPluralCategory(rules *PluralRuleSet, operands pluralOperands) PluralCategory {
	if rules == nil {
		return PluralOther
	}

	var other PluralCategory = PluralOther
	for _, rule := range rules.Rules {
		if rule.Category == PluralOther {
			other = PluralOther
			continue
		}
		if matchPluralRule(rule, operands) {
			return rule.Category
		}
	}
	return other
}

func matchPluralRule(rule PluralRule, operands pluralOperands) bool {
	if len(rule.Groups) == 0 {
		return true
	}
	for _, group := range rule.Groups {
		if len(group) == 0 {
			continue
		}
		matched := true
		for _, condition := range group {
			if !matchPluralCondition(condition, operands) {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

const floatEqualityEpsilon = 1e-9

func matchPluralCondition(condition PluralCondition, operands pluralOperands) bool {
	value, ok := operandValue(condition, operands)
	if !ok {
		return false
	}

	switch condition.Operator {
	case OperatorEquals:
		if len(condition.Values) == 0 {
			return false
		}
		return numbersEqual(value, condition.Values[0])
	case OperatorNotEquals:
		if len(condition.Values) == 0 {
			return true
		}
		return !numbersEqual(value, condition.Values[0])
	case OperatorIn:
		return membershipMatch(value, condition, false)
	case OperatorNotIn:
		return !membershipMatch(value, condition, false)
	case OperatorWithin:
		return membershipMatch(value, condition, true)
	case OperatorNotWithin:
		return !membershipMatch(value, condition, true)
	default:
		return false
	}
}

func operandValue(condition PluralCondition, operands pluralOperands) (float64, bool) {
	operand := strings.ToLower(condition.Operand)
	var value float64
	switch operand {
	case "n":
		value = operands.n
	case "i":
		value = float64(operands.i)
	case "v":
		value = float64(operands.v)
	case "w":
		value = float64(operands.w)
	case "f":
		value = float64(operands.f)
	case "t":
		value = float64(operands.t)
	default:
		return 0, false
	}

	if condition.Mod > 0 {
		mod := float64(condition.Mod)
		if operand == "n" {
			value = math.Mod(value, mod)
		} else {
			intValue := math.Round(value)
			value = math.Mod(math.Abs(intValue), mod)
		}
	}

	return value, true
}

func membershipMatch(value float64, condition PluralCondition, allowFraction bool) bool {
	if len(condition.Values) > 0 {
		for _, candidate := range condition.Values {
			if allowFraction {
				if numbersEqual(value, candidate) {
					return true
				}
				continue
			}
			if isInteger(value) && isInteger(candidate) {
				if int64(math.Round(value)) == int64(math.Round(candidate)) {
					return true
				}
			}
		}
	}

	for _, r := range condition.Ranges {
		if allowFraction {
			if value >= r.Start && value <= r.End {
				return true
			}
			continue
		}
		if !isInteger(value) {
			continue
		}
		start := math.Round(r.Start)
		end := math.Round(r.End)
		intValue := math.Round(value)
		if intValue >= start && intValue <= end {
			return true
		}
	}

	return false
}

func numbersEqual(a, b float64) bool {
	return math.Abs(a-b) < floatEqualityEpsilon
}

func isInteger(value float64) bool {
	return numbersEqual(value, math.Round(value))
}

func toPluralOperands(value any) (pluralOperands, string, bool) {
	switch v := value.(type) {
	case int:
		return buildOperandsFromLiteral(strconv.FormatInt(int64(v), 10))
	case int8:
		return buildOperandsFromLiteral(strconv.FormatInt(int64(v), 10))
	case int16:
		return buildOperandsFromLiteral(strconv.FormatInt(int64(v), 10))
	case int32:
		return buildOperandsFromLiteral(strconv.FormatInt(int64(v), 10))
	case int64:
		return buildOperandsFromLiteral(strconv.FormatInt(v, 10))
	case uint:
		return buildOperandsFromLiteral(strconv.FormatUint(uint64(v), 10))
	case uint8:
		return buildOperandsFromLiteral(strconv.FormatUint(uint64(v), 10))
	case uint16:
		return buildOperandsFromLiteral(strconv.FormatUint(uint64(v), 10))
	case uint32:
		return buildOperandsFromLiteral(strconv.FormatUint(uint64(v), 10))
	case uint64:
		return buildOperandsFromLiteral(strconv.FormatUint(v, 10))
	case float32:
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return pluralOperands{}, "", false
		}
		return buildOperandsFromLiteral(strconv.FormatFloat(float64(v), 'f', -1, 32))
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return pluralOperands{}, "", false
		}
		return buildOperandsFromLiteral(strconv.FormatFloat(v, 'f', -1, 64))
	case string:
		return buildOperandsFromLiteral(v)
	case fmt.Stringer:
		return buildOperandsFromLiteral(v.String())
	default:
		return pluralOperands{}, "", false
	}
}

func buildOperandsFromLiteral(raw string) (pluralOperands, string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return pluralOperands{}, "", false
	}

	literal := trimmed
	sign := 1
	if strings.HasPrefix(trimmed, "+") {
		trimmed = trimmed[1:]
		literal = strings.TrimSpace(trimmed)
	} else if strings.HasPrefix(trimmed, "-") {
		sign = -1
		trimmed = trimmed[1:]
	}

	if trimmed == "" {
		return pluralOperands{}, "", false
	}

	parts := strings.SplitN(trimmed, ".", 2)
	intPart := parts[0]
	fracPart := ""
	if len(parts) == 2 {
		fracPart = parts[1]
	}

	if intPart == "" {
		intPart = "0"
	}

	if !digitsOnly(intPart) || (fracPart != "" && !digitsOnly(fracPart)) {
		return pluralOperands{}, "", false
	}

	intValue, err := strconv.ParseInt(intPart, 10, 64)
	if err != nil {
		return pluralOperands{}, "", false
	}

	floatInput := trimmed
	if sign < 0 {
		floatInput = "-" + trimmed
	}

	floatValue, err := strconv.ParseFloat(floatInput, 64)
	if err != nil {
		return pluralOperands{}, "", false
	}

	op := pluralOperands{}
	op.n = math.Abs(floatValue)
	op.i = intValue
	if sign < 0 {
		op.i = int64(math.Abs(float64(intValue)))
	}
	op.v = len(fracPart)
	if op.v > 0 {
		fracValue, err := strconv.ParseInt(fracPart, 10, 64)
		if err != nil {
			return pluralOperands{}, "", false
		}
		op.f = fracValue
		trimmedFrac := strings.TrimRight(fracPart, "0")
		op.w = len(trimmedFrac)
		if op.w > 0 {
			tValue, err := strconv.ParseInt(trimmedFrac, 10, 64)
			if err != nil {
				return pluralOperands{}, "", false
			}
			op.t = tValue
		}
	} else {
		op.f = 0
		op.w = 0
		op.t = 0
	}

	if sign < 0 {
		literal = "-" + trimmed
	}

	return op, literal, true
}

func digitsOnly(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
