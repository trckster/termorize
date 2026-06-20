package testkit

import (
	"testing"

	"termorize/src/integrations/google"
	"termorize/src/integrations/openrouter"
	"termorize/src/integrations/telegram"
)

// ---------------------------------------------------------------------------
// Google Translate
// ---------------------------------------------------------------------------

// FakeGoogleTranslate is a configurable, network-free implementation of
// google.TranslateClient. Set the function fields to control behavior; leave
// them nil to use the canned defaults.
type FakeGoogleTranslate struct {
	// TranslateFunc, if set, fully controls Translate. Otherwise the default
	// returns "translated:" + text.
	TranslateFunc func(text, sourceLang, targetLang string) (string, error)
	// DetectFunc, if set, fully controls DetectLanguage. Otherwise the default
	// returns "en".
	DetectFunc func(text string) (string, error)
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

// MockGoogleTranslate swaps the google.NewTranslateClient constructor so it
// returns the provided fake for the duration of the test, restoring the original
// automatically via t.Cleanup. If fake is nil, a default canned fake is used.
//
//	testkit.MockGoogleTranslate(t, &testkit.FakeGoogleTranslate{
//	    TranslateFunc: func(text, src, dst string) (string, error) { return "hola", nil },
//	})
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

// ---------------------------------------------------------------------------
// OpenRouter
// ---------------------------------------------------------------------------

// FakeOpenRouter is a configurable, network-free implementation of
// openrouter.Client. Set GenerateFunc to control behavior; leave it nil for the
// canned default (an empty-but-valid collection titled "Test Collection").
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

// MockOpenRouter swaps the openrouter.NewClient constructor so it returns the
// provided fake for the duration of the test, restoring the original
// automatically via t.Cleanup. If fake is nil, a default canned fake is used.
//
//	testkit.MockOpenRouter(t, &testkit.FakeOpenRouter{
//	    GenerateFunc: func(prompt string, langs []string) (*openrouter.GeneratedCollection, error) {
//	        return &openrouter.GeneratedCollection{Title: "Animals"}, nil
//	    },
//	})
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

// installDefaultExternalFakes points the external client constructors at
// network-free fakes during Setup, so that no test can accidentally reach the
// real Google/OpenRouter APIs unless a test explicitly re-mocks them. Tests that
// call MockGoogleTranslate/MockOpenRouter override these for their duration and
// restore them (back to these fakes) afterwards.
func installDefaultExternalFakes() {
	google.NewTranslateClient = func() google.TranslateClient { return &FakeGoogleTranslate{} }
	openrouter.NewClient = func() openrouter.Client { return &FakeOpenRouter{} }

	// Point the Telegram API at an unroutable local address so a stray outbound
	// call (from a test that forgot MockTelegramAPI) fails fast instead of hitting
	// the real Telegram API. Tests that exercise the webhook override this via
	// MockTelegramAPI and restore it afterwards.
	telegram.SetAPIBaseURLForTest("http://127.0.0.1:0")
}
