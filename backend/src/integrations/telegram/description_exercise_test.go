package telegram

import (
	"testing"

	"termorize/src/enums"

	"github.com/stretchr/testify/assert"
)

func TestBuildDescriptionExerciseQuestionEscapesGeneratedText(t *testing.T) {
	question := BuildDescriptionExerciseQuestion("Used for *writing* [daily].", enums.LanguageEn, botTextsEn)

	assert.Contains(t, question, "Guess the word described below in English")
	assert.Contains(t, question, `Used for \*writing\*`)
	assert.Contains(t, question, `\[daily].`)
	assert.NotContains(t, question, "*writing*")
}
