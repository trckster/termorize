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

func TestDamerauLevenshteinDistance(t *testing.T) {
	tests := []struct {
		name     string
		left     string
		right    string
		expected int
	}{
		{
			name:     "identical",
			left:     "peach",
			right:    "peach",
			expected: 0,
		},
		{
			name:     "insertion",
			left:     "peach",
			right:    "preach",
			expected: 1,
		},
		{
			name:     "deletion",
			left:     "peach",
			right:    "each",
			expected: 1,
		},
		{
			name:     "substitution",
			left:     "peach",
			right:    "poach",
			expected: 1,
		},
		{
			name:     "adjacent transposition",
			left:     "peahc",
			right:    "peach",
			expected: 1,
		},
		{
			name:     "overlapping edits use full damerau distance",
			left:     "CA",
			right:    "ABC",
			expected: 2,
		},
		{
			name:     "unicode transposition",
			left:     "caéf",
			right:    "café",
			expected: 1,
		},
		{
			name:     "empty left",
			left:     "",
			right:    "word",
			expected: 4,
		},
		{
			name:     "empty right",
			left:     "word",
			right:    "",
			expected: 4,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := DamerauLevenshteinDistance(test.left, test.right)

			if result != test.expected {
				t.Fatalf("expected distance %d, got %d", test.expected, result)
			}
		})
	}
}
