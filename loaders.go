package i18n

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

var placeholderPattern = regexp.MustCompile(`\{([a-zA-Z0-9_]+)\}`)

type FileLoader struct {
	paths     []string
	rulePaths []string
}

func NewFileLoader(paths ...string) *FileLoader {
	return &FileLoader{paths: append([]string(nil), paths...)}
}

func (l *FileLoader) WithPluralRuleFiles(paths ...string) *FileLoader {
	if l == nil {
		return l
	}
	if len(paths) == 0 {
		return l
	}
	l.rulePaths = append(l.rulePaths, paths...)
	return l
}

// WithPluralRules satisfies the pluralRuleLoader contract used by config wiring.
func (l *FileLoader) WithPluralRules(paths ...string) Loader {
	return l.WithPluralRuleFiles(paths...)
}

func (l *FileLoader) Load() (Translations, error) {
	if l == nil || len(l.paths) == 0 {
		return nil, errors.New("i18n: no loader paths configured")
	}

	buckets := make(map[string]map[string]Message)

	for _, path := range l.paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("i18n: read %s: %w", path, err)
		}

		src, err := decodeTranslationFile(path, data)
		if err != nil {
			return nil, fmt.Errorf("i18n: decode %s: %w", path, err)
		}
		mergeMessageBuckets(buckets, src)
	}

	rules, err := l.loadPluralRules()
	if err != nil {
		return nil, err
	}

	catalogs := make(Translations, len(buckets))
	for locale, messages := range buckets {
		catalog := &LocaleCatalog{
			Locale: Locale{Code: locale},
		}
		if len(messages) > 0 {
			catalog.Messages = messages
		}
		if ruleSet, ok := rules[locale]; ok {
			catalog.CardinalRules = ruleSet
			if ruleSet.DisplayName != "" {
				catalog.Locale.Name = ruleSet.DisplayName
			}
			if ruleSet.Parent != "" {
				catalog.Locale.Parent = ruleSet.Parent
			}
		}
		catalogs[locale] = catalog
	}

	return catalogs, nil
}

func decodeTranslationFile(path string, data []byte) (map[string]map[string]Message, error) {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".json":
		return decodeTranslationsJSON(path, data)
	case ".yaml", ".yml":
		return decodeTranslationsYAML(path, string(data))
	default:
		return nil, fmt.Errorf("unsupported extension %s", ext)
	}
}

func decodeTranslationsJSON(path string, data []byte) (map[string]map[string]Message, error) {
	var raw map[string]map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	result := make(map[string]map[string]Message, len(raw))
	for locale, catalog := range raw {
		if locale == "" {
			return nil, fmt.Errorf("i18n: empty locale in %s", path)
		}
		normalized := make(map[string]Message, len(catalog))
		for key, rawMessage := range catalog {
			if key == "" {
				return nil, fmt.Errorf("i18n: empty key in %s/%s", locale, path)
			}
			message, err := buildMessageFromJSON(locale, key, rawMessage, path)
			if err != nil {
				return nil, fmt.Errorf("%s/%s: %w", locale, key, err)
			}
			normalized[key] = message
		}
		result[locale] = normalized
	}
	return result, nil
}

func buildMessageFromJSON(locale, key string, raw json.RawMessage, source string) (Message, error) {
	var singular string
	if err := json.Unmarshal(raw, &singular); err == nil {
		return buildMessageFromVariants(locale, key, map[PluralCategory]string{PluralOther: singular}, source)
	}

	var plural map[string]string
	if err := json.Unmarshal(raw, &plural); err == nil {
		variants := make(map[PluralCategory]string, len(plural))
		for category, template := range plural {
			cat, err := parsePluralCategory(category)
			if err != nil {
				return Message{}, err
			}
			variants[cat] = template
		}
		return buildMessageFromVariants(locale, key, variants, source)
	}

	return Message{}, fmt.Errorf("unsupported message payload")
}

func decodeTranslationsYAML(path, input string) (map[string]map[string]Message, error) {
	var raw map[string]map[string]interface{}
	if err := yaml.Unmarshal([]byte(input), &raw); err != nil {
		return nil, fmt.Errorf("yaml parse error: %w", err)
	}

	if len(raw) == 0 {
		return nil, errors.New("empty translations yaml")
	}

	catalogs := make(map[string]map[string]Message, len(raw))
	for locale, messages := range raw {
		if locale == "" {
			return nil, fmt.Errorf("empty locale in %s", path)
		}

		catalog := make(map[string]Message, len(messages))
		for key, value := range messages {
			if key == "" {
				return nil, fmt.Errorf("empty key in %s/%s", locale, path)
			}

			message, err := buildMessageFromYAMLValue(locale, key, value, path)
			if err != nil {
				return nil, fmt.Errorf("%s/%s: %w", locale, key, err)
			}
			catalog[key] = message
		}
		catalogs[locale] = catalog
	}

	return catalogs, nil
}

func buildMessageFromYAMLValue(locale, key string, value interface{}, source string) (Message, error) {
	switch v := value.(type) {
	case string:
		return buildMessageFromVariants(locale, key, map[PluralCategory]string{PluralOther: v}, source)
	case map[string]interface{}:
		variants := make(map[PluralCategory]string, len(v))
		for category, template := range v {
			cat, err := parsePluralCategory(category)
			if err != nil {
				return Message{}, err
			}
			templateStr, ok := template.(string)
			if !ok {
				return Message{}, fmt.Errorf("plural variant %s must be a string, got %T", category, template)
			}
			variants[cat] = templateStr
		}
		return buildMessageFromVariants(locale, key, variants, source)
	default:
		return Message{}, fmt.Errorf("unsupported message value type: %T", value)
	}
}

func buildMessageFromVariants(locale, key string, variants map[PluralCategory]string, source string) (Message, error) {
	if len(variants) == 0 {
		return Message{}, fmt.Errorf("no variants defined for %s", key)
	}

	if _, ok := variants[PluralOther]; !ok {
		if len(variants) == 1 {
			for category, template := range variants {
				variants[PluralOther] = template
				delete(variants, category)
				break
			}
		} else {
			return Message{}, fmt.Errorf("missing 'other' plural form for %s", key)
		}
	}

	message := Message{
		MessageMetadata: MessageMetadata{
			ID:     key,
			Domain: inferDomain(key),
			Locale: locale,
		},
		Variants: make(map[PluralCategory]MessageVariant, len(variants)),
	}

	for category, template := range variants {
		message.SetVariant(category, buildVariant(template, source))
	}

	return message, nil
}

func buildVariant(template, source string) MessageVariant {
	variant := MessageVariant{
		Template: template,
		Source:   source,
		Checksum: checksum(template),
	}

	if strings.Contains(template, "{count}") {
		variant.UsesCount = true
	}

	if args := extractFormatArgs(template); len(args) > 0 {
		variant.FormatArgs = args
	}

	return variant
}

func extractFormatArgs(template string) []string {
	matches := placeholderPattern.FindAllStringSubmatch(template, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(matches))
	args := make([]string, 0, len(matches))

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		name := strings.TrimSpace(match[1])
		if name == "" {
			continue
		}
		if strings.EqualFold(name, "count") {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		args = append(args, name)
	}

	if len(args) == 0 {
		return nil
	}

	sort.Strings(args)
	return args
}

func mergeMessageBuckets(dst, src map[string]map[string]Message) {
	for locale, catalog := range src {
		target := dst[locale]
		if target == nil {
			target = make(map[string]Message, len(catalog))
			dst[locale] = target
		}
		for key, message := range catalog {
			if existing, ok := target[key]; ok {
				if existing.Variants == nil {
					existing.Variants = make(map[PluralCategory]MessageVariant)
				}
				for category, variant := range message.Variants {
					existing.Variants[category] = variant
				}
				existing.MessageMetadata = message.MessageMetadata
				target[key] = existing
			} else {
				target[key] = message
			}
		}
	}
}

func (l *FileLoader) loadPluralRules() (map[string]*PluralRuleSet, error) {
	if len(l.rulePaths) == 0 {
		return nil, nil
	}

	rules := make(map[string]*PluralRuleSet)
	for _, path := range l.rulePaths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("i18n: read plural rules %s: %w", path, err)
		}
		parsed, err := decodePluralRules(path, data)
		if err != nil {
			return nil, fmt.Errorf("i18n: decode plural rules %s: %w", path, err)
		}
		mergeRuleSets(rules, parsed)
	}

	return rules, nil
}

type rawPluralRulesFile struct {
	Locales map[string]rawLocaleRules `json:"locales"`
}

type rawLocaleRules struct {
	Name     string                         `json:"name"`
	Parent   string                         `json:"parent"`
	Cardinal map[string][]rawConditionGroup `json:"cardinal"`
}

type rawConditionGroup []rawCondition

type rawCondition struct {
	Operand  string     `json:"operand"`
	Mod      *int       `json:"mod,omitempty"`
	Operator string     `json:"operator"`
	Values   []float64  `json:"values,omitempty"`
	Ranges   []rawRange `json:"ranges,omitempty"`
}

type rawRange struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

func decodePluralRules(path string, data []byte) (map[string]*PluralRuleSet, error) {
	wrapper := rawPluralRulesFile{}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		var direct map[string]rawLocaleRules
		if errDirect := json.Unmarshal(data, &direct); errDirect != nil {
			return nil, err
		}
		wrapper.Locales = direct
	}

	if len(wrapper.Locales) == 0 {
		return nil, fmt.Errorf("i18n: plural rule file %s has no locales", path)
	}

	result := make(map[string]*PluralRuleSet, len(wrapper.Locales))
	for locale, rawRules := range wrapper.Locales {
		ruleSet, err := buildRuleSet(locale, rawRules)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", locale, err)
		}
		result[locale] = ruleSet
	}

	return result, nil
}

func buildRuleSet(locale string, raw rawLocaleRules) (*PluralRuleSet, error) {
	if len(raw.Cardinal) == 0 {
		return nil, fmt.Errorf("missing cardinal rules")
	}

	entries := make([]PluralRule, 0, len(raw.Cardinal))
	categories := make([]string, 0, len(raw.Cardinal))
	for category := range raw.Cardinal {
		categories = append(categories, category)
	}
	sort.Strings(categories)

	for _, category := range categories {
		cat, err := parsePluralCategory(category)
		if err != nil {
			return nil, err
		}

		rawGroups := raw.Cardinal[category]
		groups := make([][]PluralCondition, 0, len(rawGroups))
		for _, rawGroup := range rawGroups {
			if len(rawGroup) == 0 {
				continue
			}
			conditions := make([]PluralCondition, 0, len(rawGroup))
			for _, rawCondition := range rawGroup {
				operator, err := parseConditionOperator(rawCondition.Operator)
				if err != nil {
					return nil, fmt.Errorf("%s: %w", category, err)
				}
				cond := PluralCondition{
					Operand:  rawCondition.Operand,
					Operator: operator,
				}
				if rawCondition.Mod != nil {
					cond.Mod = *rawCondition.Mod
				}
				if len(rawCondition.Values) > 0 {
					cond.Values = append([]float64(nil), rawCondition.Values...)
				}
				if len(rawCondition.Ranges) > 0 {
					cond.Ranges = make([]PluralRange, 0, len(rawCondition.Ranges))
					for _, r := range rawCondition.Ranges {
						cond.Ranges = append(cond.Ranges, PluralRange{Start: r.Start, End: r.End})
					}
				}
				conditions = append(conditions, cond)
			}
			if len(conditions) > 0 {
				groups = append(groups, conditions)
			}
		}

		entries = append(entries, PluralRule{Category: cat, Groups: groups})
	}

	sort.SliceStable(entries, func(i, j int) bool {
		return pluralCategoryOrder(entries[i].Category) < pluralCategoryOrder(entries[j].Category)
	})

	hasOther := false
	for _, entry := range entries {
		if entry.Category == PluralOther {
			hasOther = true
			break
		}
	}
	if !hasOther {
		entries = append(entries, PluralRule{Category: PluralOther})
	}

	return &PluralRuleSet{
		Locale:      locale,
		DisplayName: raw.Name,
		Parent:      raw.Parent,
		Rules:       entries,
	}, nil
}

func mergeRuleSets(dst, src map[string]*PluralRuleSet) {
	for locale, set := range src {
		dst[locale] = set
	}
}

func parsePluralCategory(raw string) (PluralCategory, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "zero":
		return PluralZero, nil
	case "one":
		return PluralOne, nil
	case "two":
		return PluralTwo, nil
	case "few":
		return PluralFew, nil
	case "many":
		return PluralMany, nil
	case "other":
		return PluralOther, nil
	default:
		return "", fmt.Errorf("unknown plural category %q", raw)
	}
}

func parseConditionOperator(raw string) (PluralConditionOperator, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(OperatorEquals), "=":
		return OperatorEquals, nil
	case string(OperatorNotEquals), "!=":
		return OperatorNotEquals, nil
	case string(OperatorIn):
		return OperatorIn, nil
	case string(OperatorNotIn):
		return OperatorNotIn, nil
	case string(OperatorWithin):
		return OperatorWithin, nil
	case string(OperatorNotWithin):
		return OperatorNotWithin, nil
	default:
		return "", fmt.Errorf("unknown condition operator %q", raw)
	}
}

func pluralCategoryOrder(category PluralCategory) int {
	switch category {
	case PluralZero:
		return 0
	case PluralOne:
		return 1
	case PluralTwo:
		return 2
	case PluralFew:
		return 3
	case PluralMany:
		return 4
	case PluralOther:
		return 5
	default:
		return 99
	}
}

func inferDomain(key string) string {
	if idx := strings.Index(key, "."); idx > 0 {
		return key[:idx]
	}
	return "default"
}

func checksum(input string) string {
	sum := sha1.Sum([]byte(input))
	return hex.EncodeToString(sum[:])
}
