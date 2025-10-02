package i18n

import "testing"

type recordingHook struct {
	beforeCalls int
	afterCalls  int
	lastErr     error
	lastResult  string
}

func (h *recordingHook) BeforeTranslate(locale, key string, args []any) {
	h.beforeCalls++
}

func (h *recordingHook) AfterTranslate(locale, key string, args []any, result string, err error) {
	h.afterCalls++
	h.lastErr = err
	h.lastResult = result
}

func TestWrapTranslatorWithHooks(t *testing.T) {
	store := NewStaticStore(Translations{
		"en": {"home.title": "Welcome"},
	})

	base, err := NewSimpleTranslator(store, WithTranslatorDefaultLocale("en"))
	if err != nil {
		t.Fatalf("NewSimpleTranslator: %v", err)
	}

	recorder := &recordingHook{}
	translator := WrapTranslatorWithHooks(base, recorder)

	got, err := translator.Translate("en", "home.title")
	if err != nil {
		t.Fatalf("Translate: %v", err)
	}

	if got != "Welcome" {
		t.Fatalf("Translate() = %q want Welcome", got)
	}

	if recorder.beforeCalls != 1 || recorder.afterCalls != 1 {
		t.Fatalf("unexpected hook counts before=%d after=%d", recorder.beforeCalls, recorder.afterCalls)
	}

	if recorder.lastErr != nil {
		t.Fatalf("expected nil error in hook, got %v", recorder.lastErr)
	}

	if recorder.lastResult != "Welcome" {
		t.Fatalf("expected hook result Welcome, got %q", recorder.lastResult)
	}
}

func TestWrapTranslatorWithHooksError(t *testing.T) {
	store := NewStaticStore(nil)
	base, err := NewSimpleTranslator(store, WithTranslatorDefaultLocale("en"))
	if err != nil {
		t.Fatalf("NewSimpleTranslator: %v", err)
	}

	recorder := &recordingHook{}
	translator := WrapTranslatorWithHooks(base, recorder)

	if _, err := translator.Translate("en", "missing"); err != ErrMissingTranslation {
		t.Fatalf("expected ErrMissingTranslation, got %v", err)
	}

	if recorder.lastErr != ErrMissingTranslation {
		t.Fatalf("hook saw err %v, want %v", recorder.lastErr, ErrMissingTranslation)
	}
}
