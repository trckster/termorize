package telegram

import (
	"fmt"
	"strings"
	"termorize/src/models"
	"termorize/src/services"
)

const vocabularyPreviewLimit = 5

func buildVocabularyMenuText(userID uint, t BotTexts) (string, error) {
	vocabulary, err := services.GetVocabulary(userID, 1, vocabularyPreviewLimit, "")
	if err != nil {
		return "", err
	}

	total := int(vocabulary.Pagination.Total)
	if total == 0 {
		return t.MenuVocabularyEmpty, nil
	}

	items := buildVocabularyMenuItems(vocabulary.Data)
	if len(items) == 0 {
		return t.MenuVocabularyEmpty, nil
	}

	lines := make([]string, 0, len(items)+3)
	lines = append(lines, fmt.Sprintf(t.MenuVocabularyLatestFormat, len(items)))
	lines = append(lines, "")
	lines = append(lines, items...)

	remaining := total - len(items)
	if remaining > 0 {
		lines = append(lines, "", fmt.Sprintf(t.MenuVocabularyMoreFormat, remaining))
	}

	return strings.Join(lines, "\n"), nil
}

func buildVocabularyMenuItems(vocabulary []models.Vocabulary) []string {
	items := make([]string, 0, len(vocabulary))
	for _, item := range vocabulary {
		if item.Translation == nil || item.Translation.Original == nil || item.Translation.Translation == nil {
			continue
		}

		items = append(items, buildVocabularyTranslationText(
			item.Translation.Original.Language,
			item.Translation.Original.Word,
			item.Translation.Translation.Word,
			item.Translation.Translation.Language,
		))
	}

	return items
}
