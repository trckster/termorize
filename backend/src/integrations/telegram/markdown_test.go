package telegram

import (
	"testing"

	"github.com/stretchr/testify/require"

	"termorize/src/enums"
	"termorize/src/services"
)

func TestEscapeTelegramMarkdown(t *testing.T) {
	require.Equal(t, `plain \_ \* \`+"`"+` \[ \\`, escapeTelegramMarkdown("plain _ * ` [ \\"))
}

func TestBuildTranslateQuestionTextEscapesDynamicValues(t *testing.T) {
	texts := BotTexts{QuestionTranslateReplyFormat: "Translate *%s* into %s"}

	result := buildTranslateQuestionText("_word_[x]", "lang_*", enums.ExerciseTypeBasicDirect, texts)

	require.Equal(t, `Translate *\_word\_\[x]* into lang\_\*`, result)
}

func TestBuildExerciseAnswerPairTextEscapesVocabulary(t *testing.T) {
	texts := BotTexts{ExerciseAnswerPairFormat: "%s *%s* — *%s* %s"}

	result := buildExerciseAnswerPairText(
		"`one`",
		"[two]*",
		enums.LanguageEn,
		enums.LanguageIt,
		texts,
	)

	require.Contains(t, result, `*\`+"`"+`one\`+"`"+`*`)
	require.Contains(t, result, `*\[two]\**`)
}

func TestBuildCharacterBoardTextEscapesSelectedCharacters(t *testing.T) {
	board := &services.CharacterBoardState{
		Characters: []string{"_", "*", "`", "[", "\\"},
		Chosen:     []int{0, 1, 2, 3, 4},
	}

	require.Equal(t, "Question\n\n\\_ \\* \\` \\[ \\\\", buildCharacterBoardText("Question", board))
}
