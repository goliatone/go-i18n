package i18n

import "testing"

func TestSimpleTranslatorTranslate(t *testing.T) {
	store := NewStaticStore(Translations{
		"en": {
			"home.title":    "Welcome",
			"home.greeting": "Hello %s",
		},
		"es": {
			"home.title": "Bienvenido",
		},
	})

	translator, err := NewSimpleTranslator(store, WithTranslatorDefaultLocale("en"))
	if err != nil {
		t.Fatalf("NewSimpleTranslator: %v", err)
	}

	tests := []struct {
		name    string
		locale  string
		key     string
		args    []any
		want    string
		wantErr error
	}{
		{
			name:   "explicit locale",
			locale: "es",
			key:    "home.title",
			want:   "Bienvenido",
		},
		{
			name: "default locale",
			key:  "home.title",
			want: "Welcome",
		},
		{
			name:   "format args",
			locale: "en",
			key:    "home.greeting",
			args:   []any{"Alice"},
			want:   "Hello Alice",
		},
		{
			name:    "missing key",
			locale:  "en",
			key:     "missing",
			wantErr: ErrMissingTranslation,
		},
		{
			name:    "missing locale",
			locale:  "",
			key:     "spanish",
			wantErr: ErrMissingTranslation,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := translator.Translate(tc.locale, tc.key, tc.args...)
			if tc.wantErr != nil {
				if err != tc.wantErr {
					t.Fatalf("expected err %v, got %v", tc.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}

			if got != tc.want {
				t.Fatalf("Translate() = %q want %q", got, tc.want)
			}
		})
	}
}

func TestSimpleTranslatorCustomFormatter(t *testing.T) {
	store := NewStaticStore(Translations{
		"en": {
			"home.greeting": "Hello %s",
		},
	})

	rack := false
	formatter := FormatterFunc(func(template string, args ...any) (string, error) {
		rack = true
		return "custom", nil
	})

	translator, err := NewSimpleTranslator(store,
		WithTranslatorDefaultLocale("en"),
		WithTranslatorFormatter(formatter),
	)
	if err != nil {
		t.Fatalf("NewSimpleTranslator: %v", err)
	}

	got, err := translator.Translate("", "home.greeting", "bob")
	if err != nil {
		t.Fatalf("Translate: %v", err)
	}

	if got != "custom" {
		t.Fatalf("Translate() = %q want custom", got)
	}

	if !rack {
		t.Fatal("expected formatter to be invoked")
	}
}
