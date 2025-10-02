package i18n

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
)

type FileLoader struct {
	paths []string
}

func NewFileLoader(paths ...string) *FileLoader {
	return &FileLoader{paths: append([]string(nil), paths...)}
}

func (l *FileLoader) Load() (Translations, error) {
	if l == nil || len(l.paths) == 0 {
		return nil, errors.New("i18n: no loader paths configured")
	}

	result := make(Translations)

	for _, path := range l.paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("i18n: read %s: %w", path, err)
		}

		src, err := decodeTranslationFile(path, data)
		if err != nil {
			return nil, fmt.Errorf("i18n: decode %s: %w", path, err)
		}
		mergeTranslations(result, src)
	}

	return result, nil
}

func decodeTranslationFile(path string, data []byte) (Translations, error) {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".json":
		var parsed Translations
		if err := json.Unmarshal(data, &parsed); err != nil {
			return nil, err
		}
		return parsed, nil
	case ".yaml", ".yml":
		return decodeTranslationsYAML(string(data))
	default:
		return nil, fmt.Errorf("unsupported extension %s", ext)
	}
}

func decodeTranslationsYAML(input string) (Translations, error) {
	translations := make(Translations)

	var currentLocale string

	lines := strings.Split(input, "\n")
	for idx, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if !strings.HasPrefix(raw, " ") && !strings.HasPrefix(raw, "\t") {
			if !strings.HasSuffix(line, ":") {
				return nil, fmt.Errorf("invalid yaml locale in line %d", idx+1)
			}
			locale := strings.TrimSuffix(line, ":")
			if locale == "" {
				return nil, fmt.Errorf("empty locale on line %d", idx+1)
			}

			currentLocale = locale
			if translations[currentLocale] == nil {
				translations[currentLocale] = make(map[string]string)
			}

			continue
		}

		if currentLocale == "" {
			return nil, fmt.Errorf("entry before locale definition on line %d", idx+1)
		}

		entry := strings.TrimSpace(raw)
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid key/value on line %d", idx+1)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.TrimPrefix(value, "|")
		value = strings.TrimSpace(value)

		if key == "" {
			return nil, fmt.Errorf("emtpy key on line %d", idx+1)
		}

		if n := len(value); n >= 2 {
			switch {
			case strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\""):
				value = strings.Trim(value, "\"")
			case strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'"):
				value = strings.Trim(value, "'")
			}
		}
		translations[currentLocale][key] = value
	}

	if len(translations) == 0 {
		return nil, errors.New("empty translations yaml")
	}

	return translations, nil
}

func mergeTranslations(dst, src Translations) {
	for locale, catalog := range src {
		if catalog == nil {
			continue
		}

		target := dst[locale]
		if target == nil {
			target = make(map[string]string, len(catalog))
			dst[locale] = target
		}
		maps.Copy(target, catalog)
	}
}
