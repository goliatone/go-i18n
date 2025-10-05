package i18n

import (
	"sort"
)

// Store exposes a read only access to translated mesasge templates
type Store interface {
	// Get returns the message template for locale/key and ok=false if missing
	Get(locale, key string) (string, bool)
	// Message returns the full message payload for locale/key
	Message(locale, key string) (Message, bool)
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

var _ Store = &StaticStore{}

// NewStaticStore builds an immutable snapthot from the given translations
func NewStaticStore(data Translations) *StaticStore {
	if len(data) == 0 {
		return &StaticStore{translations: make(Translations)}
	}

	translations := make(Translations, len(data))
	locales := make([]string, 0, len(data))

	for locale, catalog := range data {
		if catalog == nil {
			continue
		}
		clone := &LocaleCatalog{
			Locale: catalog.Locale,
		}
		if clone.Locale.Code == "" {
			clone.Locale.Code = locale
		}

		if len(catalog.Messages) > 0 {
			clone.Messages = make(map[string]Message, len(catalog.Messages))
			for key, message := range catalog.Messages {
				clone.Messages[key] = message.Clone()
			}
		}

		if catalog.CardinalRules != nil {
			clone.CardinalRules = catalog.CardinalRules.Clone()
			if clone.Locale.Name == "" {
				clone.Locale.Name = clone.CardinalRules.DisplayName
			}
			if clone.Locale.Parent == "" {
				clone.Locale.Parent = clone.CardinalRules.Parent
			}
		}

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

func (s *StaticStore) Message(locale, key string) (Message, bool) {
	if s == nil {
		return Message{}, false
	}

	catalog, ok := s.translations[locale]
	if !ok || catalog == nil {
		return Message{}, false
	}

	if catalog.Messages == nil {
		return Message{}, false
	}

	msg, ok := catalog.Messages[key]
	if !ok {
		return Message{}, false
	}

	return msg.Clone(), ok
}

// Get returns the the message template for locale/key
func (s *StaticStore) Get(locale, key string) (string, bool) {
	msg, ok := s.Message(locale, key)
	if !ok {
		return "", false
	}
	return msg.Content(), ok
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
