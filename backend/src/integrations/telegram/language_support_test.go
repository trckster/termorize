package telegram

import (
	"testing"

	"termorize/src/enums"

	"github.com/stretchr/testify/assert"
)

func TestPortugueseAndUkrainianLocalizedLanguageNames(t *testing.T) {
	assert.Equal(t, "Portuguese", botTextsEn.LanguageNames[enums.LanguagePt])
	assert.Equal(t, "Ukrainian", botTextsEn.LanguageNames[enums.LanguageUk])
	assert.Equal(t, "Португальский", botTextsRu.LanguageNames[enums.LanguagePt])
	assert.Equal(t, "Украинский", botTextsRu.LanguageNames[enums.LanguageUk])
}

func TestLanguageSelectionIncludesPortugueseAndUkrainian(t *testing.T) {
	rows := buildLanguageSelectionKeyboard(enums.LanguageEn, enums.LanguageRu, true, botTextsEn)
	labels := make([]string, 0, len(rows))
	for _, row := range rows {
		for _, button := range row {
			labels = append(labels, button.Text)
		}
	}

	assert.Contains(t, labels, enums.LanguagePt.DisplayNameWithFlag())
	assert.Contains(t, labels, enums.LanguageUk.DisplayNameWithFlag())
}

func TestMenuShowsOpenAppBeforeLocalizedTranslationPair(t *testing.T) {
	keyboard := getMenuKeyboard(enums.LanguageEn, enums.LanguageRu, botTextsRu)

	assert.Equal(t, "Открыть приложение 🌐", keyboard[0][0].Text)
	assert.Equal(t, "🇬🇧 Английский → 🇷🇺 Русский", keyboard[1][0].Text)
	assert.Equal(t, "menu:translation_pair", keyboard[1][0].CallbackData)
}

func TestTranslationPairEditorUsesDedicatedActions(t *testing.T) {
	keyboard := buildTranslationPairKeyboard(enums.LanguageEn, enums.LanguageRu, botTextsEn)

	assert.Equal(t, "menu:change_pair_source_lang", keyboard[0][0].CallbackData)
	assert.Equal(t, "menu:change_pair_target_lang", keyboard[0][1].CallbackData)
	assert.Equal(t, "menu:swap_translation_pair", keyboard[1][0].CallbackData)
	assert.Equal(t, "menu:back", keyboard[2][0].CallbackData)
}

func TestTranslationPairLanguageSelectionReturnsToEditor(t *testing.T) {
	rows := buildTranslationPairLanguageSelectionKeyboard(enums.LanguageEn, enums.LanguageRu, true, botTextsEn)

	assert.Equal(t, "menu:set_pair_source_lang:it", rows[0][0].CallbackData)
	assert.Equal(t, "menu:translation_pair", rows[len(rows)-1][0].CallbackData)
}
