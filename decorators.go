package i18n

type TranslationHook interface {
	BeforeTranslate(locale, key string, args []any)
	AfterTranslate(locale, key string, args []any, result string, err error)
}

type TranslationHookFuncs struct {
	Before func(locale, key string, args []any)
	After  func(locale, key string, args []any, result string, err error)
}

func (h TranslationHookFuncs) BeforeTranslate(locale, key string, args []any) {
	if h.Before != nil {
		h.Before(locale, key, args)
	}
}

func (h TranslationHookFuncs) AfterTranslate(locale, key string, args []any, result string, err error) {
	if h.After != nil {
		h.After(locale, key, args, result, err)
	}
}

var _ Translator = &HookedTranslator{}

type HookedTranslator struct {
	next  Translator
	hooks []TranslationHook
}

func WrapTranslatorWithHooks(next Translator, hooks ...TranslationHook) Translator {
	if next == nil || len(hooks) == 0 {
		return next
	}

	filtered := hooks[:0]
	for _, hook := range hooks {
		if hook == nil {
			continue
		}

		filtered = append(filtered, hook)
	}

	if len(filtered) == 0 {
		return next
	}

	return &HookedTranslator{next: next, hooks: filtered}
}

func (t *HookedTranslator) Translate(locale, key string, args ...any) (string, error) {
	if t == nil || t.next == nil {
		return "", ErrMissingTranslation
	}

	for _, hook := range t.hooks {
		hook.BeforeTranslate(locale, key, args)
	}

	result, err := t.next.Translate(locale, key, args...)

	for _, hook := range t.hooks {
		hook.AfterTranslate(locale, key, args, result, err)
	}

	return result, err
}
