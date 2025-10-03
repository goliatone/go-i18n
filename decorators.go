package i18n

type TranslationHook interface {
	BeforeTranslate(ctx *TranslatorHookContext)
	AfterTranslate(ctx *TranslatorHookContext)
}

type TranslatorHookContext struct {
	Locale   string
	Key      string
	Args     []any
	Result   string
	Error    error
	Metadata map[string]any
}

func (ctx *TranslatorHookContext) ensureMetadata() {
	if ctx.Metadata == nil {
		ctx.Metadata = make(map[string]any)
	}
}

func (ctx *TranslatorHookContext) SetMetadata(key string, value any) {
	if ctx == nil || key == "" {
		return
	}
	ctx.ensureMetadata()
	ctx.Metadata[key] = value
}

func (ctx *TranslatorHookContext) MetadataValue(key string) (any, bool) {
	if ctx == nil || ctx.Metadata == nil {
		return nil, false
	}
	val, ok := ctx.Metadata[key]
	return val, ok
}

type TranslationHookFuncs struct {
	Before func(ctx *TranslatorHookContext)
	After  func(ctx *TranslatorHookContext)
}

func (h TranslationHookFuncs) BeforeTranslate(ctx *TranslatorHookContext) {
	if h.Before != nil {
		h.Before(ctx)
	}
}

func (h TranslationHookFuncs) AfterTranslate(ctx *TranslatorHookContext) {
	if h.After != nil {
		h.After(ctx)
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

	ctx := &TranslatorHookContext{
		Locale: locale,
		Key:    key,
		Args:   args,
	}

	for _, hook := range t.hooks {
		hook.BeforeTranslate(ctx)
	}

	result, err := t.next.Translate(ctx.Locale, ctx.Key, ctx.Args...)

	ctx.Result = result
	ctx.Error = err

	for _, hook := range t.hooks {
		hook.AfterTranslate(ctx)
	}

	result = ctx.Result
	err = ctx.Error

	return result, err
}
