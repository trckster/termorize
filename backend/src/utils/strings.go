package utils

import (
	"strings"
	"unicode"
)

// NormalizeWordCasing decides the stored casing of a vocabulary word. It is
// purely string-based — casing does not depend on the word's language.
//
// Rules, applied in order (first match wins):
//  1. Trim surrounding whitespace.
//  2. Multi-token phrases keep their casing as-is. This protects German nouns,
//     which are entered with their article (e.g. "das Haus"), and other phrases
//     ("New York").
//  3. Acronyms (all-uppercase, 2+ letters) keep their casing: USB, NATO.
//  4. Words with internal capitals keep their casing: iPhone, eBay.
//  5. Otherwise lowercase. A single-token German word has no article, so it is
//     not a noun and is correctly lowercased ("gehen", "schön").
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

// isAllUppercaseWord reports whether every letter is uppercase and there are at
// least two letters (so single letters like "I" fall through to other rules).
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

// hasInternalUppercase reports whether any rune after the first is uppercase
// (e.g. iPhone, eBay, McDonald), signalling intentional brand/proper-noun casing.
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
