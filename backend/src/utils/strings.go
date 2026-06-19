package utils

import (
	"strings"
	"unicode"
)

var italianArticles = map[string]bool{
	"il":  true,
	"lo":  true,
	"i":   true,
	"gli": true,
	"la":  true,
	"le":  true,
	"un":  true,
	"uno": true,
	"una": true,
}

func NormalizeWordCasing(word string) string {
	trimmed := strings.TrimSpace(word)

	if len(strings.Fields(trimmed)) > 1 {
		return trimmed
	}

	if isAllUppercaseWord(trimmed) {
		return trimmed
	}

	if hasInternalUppercase(trimmed) {
		return trimmed
	}

	return strings.ToLower(trimmed)
}

func NormalizeWordCasingForLanguage(word string, language string) string {
	trimmed := strings.TrimSpace(word)
	if IsItalianArticlePhrase(trimmed, language) {
		return strings.ToLower(trimmed)
	}

	return NormalizeWordCasing(trimmed)
}

func NormalizeTranslationPairCasing(
	original string,
	originalLanguage string,
	translation string,
	translationLanguage string,
) (string, string) {
	normalizedOriginal := NormalizeWordCasingForLanguage(original, originalLanguage)
	normalizedTranslation := NormalizeWordCasingForLanguage(translation, translationLanguage)

	if IsItalianArticlePhrase(original, originalLanguage) {
		normalizedTranslation = strings.ToLower(strings.TrimSpace(translation))
	}
	if IsItalianArticlePhrase(translation, translationLanguage) {
		normalizedOriginal = strings.ToLower(strings.TrimSpace(original))
	}

	return normalizedOriginal, normalizedTranslation
}

func IsItalianArticlePhrase(word string, language string) bool {
	if strings.ToLower(strings.TrimSpace(language)) != "it" {
		return false
	}

	parts := strings.Fields(strings.TrimSpace(word))
	if len(parts) != 2 {
		return false
	}

	return italianArticles[strings.ToLower(parts[0])]
}

func isAllUppercaseWord(value string) bool {
	letters := 0
	for _, r := range value {
		if !unicode.IsLetter(r) {
			continue
		}
		if !unicode.IsUpper(r) {
			return false
		}
		letters++
	}

	return letters >= 2
}

func hasInternalUppercase(value string) bool {
	for index, r := range []rune(value) {
		if index == 0 {
			continue
		}
		if unicode.IsUpper(r) {
			return true
		}
	}

	return false
}

func LevenshteinDistance(left string, right string) int {
	leftRunes := []rune(left)
	rightRunes := []rune(right)

	if len(leftRunes) == 0 {
		return len(rightRunes)
	}

	if len(rightRunes) == 0 {
		return len(leftRunes)
	}

	previous := make([]int, len(rightRunes)+1)
	current := make([]int, len(rightRunes)+1)

	for index := range previous {
		previous[index] = index
	}

	for leftIndex, leftRune := range leftRunes {
		current[0] = leftIndex + 1

		for rightIndex, rightRune := range rightRunes {
			cost := 0
			if leftRune != rightRune {
				cost = 1
			}

			deletion := previous[rightIndex+1] + 1
			insertion := current[rightIndex] + 1
			substitution := previous[rightIndex] + cost
			current[rightIndex+1] = minInt(deletion, insertion, substitution)
		}

		previous, current = current, previous
	}

	return previous[len(rightRunes)]
}

func minInt(values ...int) int {
	result := values[0]
	for _, value := range values[1:] {
		if value < result {
			result = value
		}
	}

	return result
}
