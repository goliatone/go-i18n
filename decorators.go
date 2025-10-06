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

// PluralMetadata returns plural-specific hook metadata if present.
func (ctx *TranslatorHookContext) PluralMetadata() (PluralHookMetadata, bool) {
	if ctx == nil || len(ctx.Metadata) == 0 {
		return PluralHookMetadata{}, false
	}

	meta := PluralHookMetadata{}
	seen := false

	if value, ok := ctx.Metadata[metadataPluralCategory]; ok {
		if category, okCast := asPluralCategory(value); okCast {
			meta.Category = category
			seen = true
		}
	}

	if value, ok := ctx.Metadata[metadataPluralCount]; ok {
		meta.Count = value
		seen = true
	}

	if value, ok := ctx.Metadata[metadataPluralMessage]; ok {
		if message, okCast := value.(string); okCast {
			meta.Message = message
			seen = true
		}
	}

	if value, ok := ctx.Metadata[metadataPluralMissing]; ok {
		if missing := parsePluralMissingEvent(value); missing != nil {
			meta.Missing = missing
			seen = true
		}
	}

	return meta, seen
}

func asPluralCategory(value any) (PluralCategory, bool) {
	switch v := value.(type) {
	case PluralCategory:
		return v, true
	case string:
		return PluralCategory(v), true
	default:
		return "", false
	}
}

func parsePluralMissingEvent(value any) *PluralMissingEvent {
	var (
		req PluralCategory
		fb  PluralCategory
	)

	switch data := value.(type) {
	case map[string]any:
		req, _ = asPluralCategory(data["requested"])
		fb, _ = asPluralCategory(data["fallback"])
	case map[string]PluralCategory:
		req = data["requested"]
		fb = data["fallback"]
	case PluralMissingEvent:
		return &data
	case *PluralMissingEvent:
		if data == nil {
			return nil
		}
		return &PluralMissingEvent{
			Requested: data.Requested,
			Fallback:  data.Fallback,
		}
	default:
		return nil
	}

	if req == "" && fb == "" {
		return nil
	}

	return &PluralMissingEvent{Requested: req, Fallback: fb}
}

type PluralHookMetadata struct {
	Category PluralCategory
	Count    any
	Message  string
	Missing  *PluralMissingEvent
}

type PluralMissingEvent struct {
	Requested PluralCategory
	Fallback  PluralCategory
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

	var (
		result   string
		err      error
		metadata map[string]any
	)

	if mt, ok := t.next.(metadataTranslator); ok {
		result, metadata, err = mt.TranslateWithMetadata(ctx.Locale, ctx.Key, ctx.Args...)
		if len(metadata) > 0 {
			for key, value := range metadata {
				ctx.SetMetadata(key, value)
			}
		}
	} else {
		result, err = t.next.Translate(ctx.Locale, ctx.Key, ctx.Args...)
	}

	ctx.Result = result
	ctx.Error = err

	for _, hook := range t.hooks {
		hook.AfterTranslate(ctx)
	}

	result = ctx.Result
	err = ctx.Error

	return result, err
}
