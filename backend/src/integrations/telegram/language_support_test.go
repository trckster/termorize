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
