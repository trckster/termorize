package testkit

import (
	"testing"

	"termorize/src/integrations/google"
	"termorize/src/integrations/openrouter"
	"termorize/src/integrations/telegram"
)

type FakeGoogleTranslate struct {
	TranslateFunc func(text, sourceLang, targetLang string) (string, error)
	DetectFunc    func(text string) (string, error)
}

func (f *FakeGoogleTranslate) Translate(text, sourceLang, targetLang string) (string, error) {
	if f.TranslateFunc != nil {
		return f.TranslateFunc(text, sourceLang, targetLang)
	}
	return "translated:" + text, nil
}

func (f *FakeGoogleTranslate) DetectLanguage(text string) (string, error) {
	if f.DetectFunc != nil {
		return f.DetectFunc(text)
	}
	return "en", nil
}

func MockGoogleTranslate(t *testing.T, fake *FakeGoogleTranslate) *FakeGoogleTranslate {
	t.Helper()
	if fake == nil {
		fake = &FakeGoogleTranslate{}
	}

	original := google.NewTranslateClient
	google.NewTranslateClient = func() google.TranslateClient { return fake }
	t.Cleanup(func() { google.NewTranslateClient = original })

	return fake
}

type FakeOpenRouter struct {
	GenerateFunc func(prompt string, allowedLanguages []string) (*openrouter.GeneratedCollection, error)
}

func (f *FakeOpenRouter) GenerateCollection(prompt string, allowedLanguages []string) (*openrouter.GeneratedCollection, error) {
	if f.GenerateFunc != nil {
		return f.GenerateFunc(prompt, allowedLanguages)
	}
	return &openrouter.GeneratedCollection{
		Title:        "Test Collection",
		Translations: []openrouter.GeneratedTranslation{},
	}, nil
}

func MockOpenRouter(t *testing.T, fake *FakeOpenRouter) *FakeOpenRouter {
	t.Helper()
	if fake == nil {
		fake = &FakeOpenRouter{}
	}

	original := openrouter.NewClient
	openrouter.NewClient = func() openrouter.Client { return fake }
	t.Cleanup(func() { openrouter.NewClient = original })

	return fake
}

func installDefaultExternalFakes() {
	google.NewTranslateClient = func() google.TranslateClient { return &FakeGoogleTranslate{} }
	openrouter.NewClient = func() openrouter.Client { return &FakeOpenRouter{} }

	telegram.SetAPIBaseURLForTest("http://127.0.0.1:0")
}
