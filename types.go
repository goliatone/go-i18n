package i18n

type TranslationCatalog struct {
	Locale        Locale
	Messages      map[string]Message
	CardinalRules *PluralRuleSet
}

type Translations map[string]*TranslationCatalog

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

type PluralRange struct {
	Start float64
	End   float64
}

type PluralConditionOperator string

const (
	OperatorEquals    PluralConditionOperator = "eq"
	OperatorNotEquals PluralConditionOperator = "neq"
	OperatorIn        PluralConditionOperator = "in"
	OperatorNotIn     PluralConditionOperator = "not_in"
	OperatorWithin    PluralConditionOperator = "within"
	OperatorNotWithin PluralConditionOperator = "not_within"
)

type PluralCondition struct {
	Operand  string
	Mod      int
	Operator PluralConditionOperator
	Values   []float64
	Ranges   []PluralRange
}

type PluralRule struct {
	Category PluralCategory
	Groups   [][]PluralCondition
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

// Clone returns a deep copy of the rule set
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
		for i, rule := range set.Rules {
			out.Rules[i] = clonePluralRule(rule)
		}
	}
	return out
}

func clonePluralRule(rule PluralRule) PluralRule {
	if len(rule.Groups) == 0 {
		return PluralRule{Category: rule.Category}
	}

	groups := make([][]PluralCondition, len(rule.Groups))
	for i, group := range rule.Groups {
		if len(group) == 0 {
			continue
		}

		cloned := make([]PluralCondition, len(group))
		for j, condition := range group {
			cloned[j] = clonePluralCondition(condition)
		}
		groups[i] = cloned
	}

	return PluralRule{
		Category: rule.Category,
		Groups:   groups,
	}
}

func clonePluralCondition(condition PluralCondition) PluralCondition {
	clone := condition
	if len(condition.Values) > 0 {
		clone.Values = append([]float64(nil), condition.Values...)
	}
	if len(condition.Ranges) > 0 {
		clone.Ranges = append([]PluralRange(nil), condition.Ranges...)
	}
	return clone
}
