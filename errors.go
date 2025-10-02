package i18n

import "errors"

// ErrMissingTranslation indicates that no translation was found for locale/key.
var ErrMissingTranslation = errors.New("i18n: missing translation")

// ErrNotImplemented marks APIs that are intentionally stubbed during bootstrapping
var ErrNotImplemented = errors.New("i18n: not implemented")
