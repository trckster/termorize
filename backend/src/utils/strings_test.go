package utils

import "testing"

func TestNormalizeWordCasingForLanguageLowercasesItalianArticlePhrase(t *testing.T) {
	result := NormalizeWordCasingForLanguage("Il Cane", "it")

	if result != "il cane" {
		t.Fatalf("expected italian article phrase to be lowercased, got %q", result)
	}
}

func TestNormalizeWordCasingForLanguageKeepsNonItalianPhraseCasing(t *testing.T) {
	result := NormalizeWordCasingForLanguage("Das Haus", "de")

	if result != "Das Haus" {
		t.Fatalf("expected non-italian phrase casing to be preserved, got %q", result)
	}
}

func TestNormalizeWordCasingForLanguageRequiresExactlyTwoWords(t *testing.T) {
	result := NormalizeWordCasingForLanguage("Il Cane Nero", "it")

	if result != "Il Cane Nero" {
		t.Fatalf("expected three-word italian phrase casing to be preserved, got %q", result)
	}
}

func TestNormalizeTranslationPairCasingLowercasesTranslationForItalianArticlePhrase(t *testing.T) {
	original, translation := NormalizeTranslationPairCasing("La Casa", "it", "The House", "en")

	if original != "la casa" {
		t.Fatalf("expected original to be lowercased, got %q", original)
	}
	if translation != "the house" {
		t.Fatalf("expected translation to be lowercased, got %q", translation)
	}
}

func TestNormalizeTranslationPairCasingLowercasesOriginalForItalianArticlePhraseTranslation(t *testing.T) {
	original, translation := NormalizeTranslationPairCasing("The House", "en", "La Casa", "it")

	if original != "the house" {
		t.Fatalf("expected original to be lowercased, got %q", original)
	}
	if translation != "la casa" {
		t.Fatalf("expected translation to be lowercased, got %q", translation)
	}
}
