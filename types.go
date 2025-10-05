package i18n

type LocaleCatalog struct {
	Locale        Locale
	Messages      map[string]Message
	CardinalRules *PluralRuleSet
}

type Translations map[string]*LocaleCatalog

// Locale metadata placeholder pending richer implementation
type Locale struct {
	Code   string
	Name   string
	Parent string
}

// TranslationKey models identifier metadata
type TranslationKey struct {
	ID          string
	Domain      string
	Description string
}

type PluralCategory string

const (
	PluralZero  PluralCategory = "zero"
	PluralOne   PluralCategory = "one"
	PluralTwo   PluralCategory = "two"
	PluralFew   PluralCategory = "few"
	PluralMany  PluralCategory = "many"
	PluralOther PluralCategory = "other"
)

type MessageMetadata struct {
	ID          string
	Domain      string
	Locale      string
	Description string
}

type MessageVariant struct {
	Template   string
	FormatArgs []string
	UsesCount  bool
	Source     string
	Checksum   string
}

type Message struct {
	MessageMetadata
	Variants map[PluralCategory]MessageVariant
}

func (m Message) Variant(category PluralCategory) (MessageVariant, bool) {
	if m.Variants == nil {
		return MessageVariant{}, false
	}

	if variant, ok := m.Variants[category]; ok {
		return variant, true
	}

	variant, ok := m.Variants[PluralOther]
	return variant, ok
}

func (m *Message) SetVariant(category PluralCategory, vairant MessageVariant) {
	if m.Variants == nil {
		m.Variants = make(map[PluralCategory]MessageVariant)
	}
	m.Variants[category] = vairant
}

func (m Message) Content() string {
	if variant, ok := m.Variant(PluralOther); ok {
		return variant.Template
	}
	return ""
}

func (m *Message) SetContent(content string) {
	m.SetVariant(PluralOther, MessageVariant{Template: content})
}

func (m Message) Clone() Message {
	out := Message{MessageMetadata: m.MessageMetadata}
	if len(m.Variants) == 0 {
		return out
	}

	out.Variants = make(map[PluralCategory]MessageVariant, len(m.Variants))
	for category, variant := range m.Variants {
		out.Variants[category] = variant.clone()
	}
	return out
}

func (v MessageVariant) clone() MessageVariant {
	copy := v
	if len(v.FormatArgs) > 0 {
		copy.FormatArgs = append([]string(nil), v.FormatArgs...)
	}
	return copy
}

type PluralRule struct {
	Category  PluralCategory
	Condition string
}

type PluralRuleSet struct {
	Locale      string
	DisplayName string
	Parent      string
	Rules       []PluralRule
}

func (set *PluralRuleSet) Categories() []PluralCategory {
	if set == nil || len(set.Rules) == 0 {
		return nil
	}

	categories := make([]PluralCategory, 0, len(set.Rules))
	seen := make(map[PluralCategory]struct{}, len(set.Rules))
	for _, rule := range set.Rules {
		if rule.Category == "" {
			continue
		}
		if _, ok := seen[rule.Category]; ok {
			continue
		}

		seen[rule.Category] = struct{}{}
		categories = append(categories, rule.Category)
	}
	return categories
}

func (set *PluralRuleSet) Clone() *PluralRuleSet {
	if set == nil {
		return nil
	}

	out := &PluralRuleSet{
		Locale:      set.Locale,
		DisplayName: set.DisplayName,
		Parent:      set.Parent,
	}
	if len(set.Rules) > 0 {
		out.Rules = make([]PluralRule, len(set.Rules))
		copy(out.Rules, set.Rules)
	}
	return out
}
