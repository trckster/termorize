package utils

import (
	"strings"
	"unicode"
)

const kebabSlugMaxLength = 80

func KebabSlug(value string) string {
	var builder strings.Builder
	runeCount := 0
	pendingSeparator := false

	for _, character := range strings.TrimSpace(value) {
		if unicode.IsLetter(character) || unicode.IsDigit(character) {
			if pendingSeparator && builder.Len() > 0 && runeCount < kebabSlugMaxLength {
				builder.WriteByte('-')
				runeCount++
			}

			if runeCount >= kebabSlugMaxLength {
				break
			}

			builder.WriteRune(unicode.ToLower(character))
			runeCount++
			pendingSeparator = false
			continue
		}

		pendingSeparator = builder.Len() > 0
	}

	result := strings.TrimSuffix(builder.String(), "-")
	if result == "" {
		return "collection"
	}

	return result
}

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

func DamerauLevenshteinDistance(left string, right string) int {
	leftRunes := []rune(left)
	rightRunes := []rune(right)

	if len(leftRunes) == 0 {
		return len(rightRunes)
	}

	if len(rightRunes) == 0 {
		return len(leftRunes)
	}

	maxDistance := len(leftRunes) + len(rightRunes)
	distances := make([][]int, len(leftRunes)+2)
	for index := range distances {
		distances[index] = make([]int, len(rightRunes)+2)
	}

	distances[0][0] = maxDistance
	for leftIndex := 0; leftIndex <= len(leftRunes); leftIndex++ {
		distances[leftIndex+1][0] = maxDistance
		distances[leftIndex+1][1] = leftIndex
	}
	for rightIndex := 0; rightIndex <= len(rightRunes); rightIndex++ {
		distances[0][rightIndex+1] = maxDistance
		distances[1][rightIndex+1] = rightIndex
	}

	lastRowByRune := make(map[rune]int)
	for leftIndex := 1; leftIndex <= len(leftRunes); leftIndex++ {
		lastMatchingColumn := 0

		for rightIndex := 1; rightIndex <= len(rightRunes); rightIndex++ {
			matchingRow := lastRowByRune[rightRunes[rightIndex-1]]
			matchingColumn := lastMatchingColumn

			substitutionCost := 1
			if leftRunes[leftIndex-1] == rightRunes[rightIndex-1] {
				substitutionCost = 0
				lastMatchingColumn = rightIndex
			}

			substitution := distances[leftIndex][rightIndex] + substitutionCost
			insertion := distances[leftIndex+1][rightIndex] + 1
			deletion := distances[leftIndex][rightIndex+1] + 1
			transposition := distances[matchingRow][matchingColumn] +
				(leftIndex - matchingRow - 1) + 1 +
				(rightIndex - matchingColumn - 1)

			distances[leftIndex+1][rightIndex+1] = minInt(
				substitution,
				insertion,
				deletion,
				transposition,
			)
		}

		lastRowByRune[leftRunes[leftIndex-1]] = leftIndex
	}

	return distances[len(leftRunes)+1][len(rightRunes)+1]
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
