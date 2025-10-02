package i18n

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func FormatDate(locale string, t time.Time) string {
	return t.Format("2006-01-02")
}

func FormatDateTime(locale string, t time.Time) string {
	return t.Format(time.RFC3339)
}

func FormatTime(locale string, t time.Time) string {
	return t.Format("15:04")
}

func FormatCurrency(locale string, amount float64, currency string) string {
	formatted := FormatNumber(locale, amount, 2)
	currency = strings.TrimSpace(currency)
	if currency == "" {
		return formatted
	}
	return currency + " " + formatted
}

func FormatNumber(locale string, value float64, decimals int) string {
	prec := decimals
	if prec < 0 {
		prec = -1
	}
	return strconv.FormatFloat(value, 'f', prec, 64)
}

func FormatPercent(locale string, value float64, decimals int) string {
	formatted := FormatNumber(locale, value*100, decimals)
	return formatted + "%"
}

func FormatOrdinal(locale string, value int) string {
	suffix := ordinalSuffic(value)
	return fmt.Sprintf("%d%s", value, suffix)
}

func FormatList(locale string, items []string) string {
	count := len(items)
	switch count {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return items[0] + " and " + items[1]
	default:
		head := strings.Join(items[:count-1], ", ")
		return head + ", and " + items[count-1]
	}
}

func FormatPhone(locale, raw string) string {
	return raw
}

func FormatMeasurement(locale string, value float64, unit string) string {
	formatted := FormatNumber(locale, value, -1)
	unit = strings.TrimSpace(unit)
	if unit == "" {
		return formatted
	}
	return formatted + " " + unit
}

func ordinalSuffic(value int) string {
	abs := value
	if abs < 0 {
		abs = -abs
	}
	mod100 := abs % 100
	if mod100 >= 11 && mod100 <= 13 {
		return "th"
	}
	switch abs % 10 {
	case 1:
		return "st"
	case 2:
		return "nd"
	case 3:
		return "rd"
	default:
		return "th"
	}
}
