package i18n

// FormattingRules contains all locale-specific formatting patterns
type FormattingRules struct {
	Locale        string              `json:"locale"`
	DatePatterns  DatePatternRules    `json:"date_patterns"`
	CurrencyRules CurrencyFormatRules `json:"currency_rules"`
	MonthNames    []string            `json:"month_names"`
	TimeFormat    TimeFormatRules     `json:"time_format"`
}

// DatePatternRules defines how dates are formatted
type DatePatternRules struct {
	// Pattern uses placeholders: {day}, {month}, {year}
	Pattern    string `json:"pattern"`
	DayFirst   bool   `json:"day_first"`
	MonthStyle string `json:"month_style"` // "name", "number", "short"
}

// CurrencyFormatRules defines currency formatting
type CurrencyFormatRules struct {
	// Pattern: {symbol}, {amount}
	Pattern        string `json:"pattern"`
	SymbolPosition string `json:"symbol_position"` // "before", "after"
	DecimalSep     string `json:"decimal_separator"`
	ThousandSep    string `json:"thousand_separator"`
	Decimals       int    `json:"decimals"`
}

// TimeFormatRules defines time formatting
type TimeFormatRules struct {
	Use24Hour bool   `json:"use_24_hour"`
	Pattern   string `json:"pattern"`
}

// FormattingDataLoader loads formatting rules from embedded data
type FormattingDataLoader interface {
	Load(locale string) (*FormattingRules, error)
	Available() []string
}
