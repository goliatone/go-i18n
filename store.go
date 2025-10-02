package i18n

import (
	"maps"
	"sort"
)

// Translations groups message templates by locale and token
type Translations map[string]map[string]string

// Store exposes a read only access to translated mesasge templates
type Store interface {
	// Get returns the message template for locale/key and ok=false if missing
	Get(locale, key string) (string, bool)
	// Locales returns the list of locales known to the store
	Locales() []string
}

// Loader retrieves the translations used to seed a Store
type Loader interface {
	Load() (Translations, error)
}

// LoaderFunc adapters allow bare functions to implement Loader interface
type LoaderFunc func() (Translations, error)

// Load implements Loader for LoaderFunc
func (fn LoaderFunc) Load() (Translations, error) {
	return fn()
}

// StaticStore is an in memory store, read only after cosntruction
type StaticStore struct {
	translations Translations
	locales      []string
}

// NewStaticStore builds an immutable snapthot from the given translations
func NewStaticStore(data Translations) *StaticStore {
	if len(data) == 0 {
		return &StaticStore{translations: make(Translations)}
	}

	translations := make(Translations, len(data))
	locales := make([]string, 0, len(data))

	for locale, catalog := range data {
		clone := make(map[string]string, len(catalog))
		maps.Copy(clone, catalog)
		translations[locale] = clone
		locales = append(locales, locale)
	}

	// make locales deterministic
	sort.Strings(locales)

	return &StaticStore{
		translations: translations,
		locales:      locales,
	}
}

// NewStaticStoreFromLoader hydrates a StaticStore using the provided loader
func NewStaticStoreFromLoader(loader Loader) (*StaticStore, error) {
	if loader == nil {
		return NewStaticStore(nil), nil
	}

	translations, err := loader.Load()
	if err != nil {
		return nil, err
	}

	return NewStaticStore(translations), nil
}

// Get returns the the message template for locale/key
func (s *StaticStore) Get(locale, key string) (string, bool) {
	if s == nil {
		return "", false
	}
	catalog, ok := s.translations[locale]
	if !ok {
		return "", false
	}
	msg, ok := catalog[key]
	return msg, ok
}

// Locales returns a slice with all locale codes
func (s *StaticStore) Locales() []string {
	if s == nil || len(s.locales) == 0 {
		return nil
	}
	out := make([]string, len(s.locales))
	copy(out, s.locales)
	return out
}
