package telegram

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"termorize/src/services"
)

func TestChoiceExerciseKeyboardUsesSingleColumnForLongLabels(t *testing.T) {
	exerciseID := uuid.MustParse("52fdfc07-2182-454f-963f-5f0f9a621d72")

	tests := []struct {
		name      string
		longLabel string
		rowCount  int
		rowWidth  int
	}{
		{
			name:      "keeps two columns at the limit",
			longLabel: strings.Repeat("я", exerciseCompactButtonMaxRunes),
			rowCount:  2,
			rowWidth:  2,
		},
		{
			name:      "uses one column above the limit",
			longLabel: strings.Repeat("я", exerciseCompactButtonMaxRunes+1),
			rowCount:  4,
			rowWidth:  1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			options := []services.ExerciseOption{
				{VocabularyID: uuid.New(), Label: "one"},
				{VocabularyID: uuid.New(), Label: test.longLabel},
				{VocabularyID: uuid.New(), Label: "three"},
				{VocabularyID: uuid.New(), Label: "four"},
			}

			keyboard := buildExerciseKeyboard(exerciseID, options)

			require.Len(t, keyboard, test.rowCount)
			for _, row := range keyboard {
				require.Len(t, row, test.rowWidth)
			}
			require.Equal(t, "one", keyboard[0][0].Text)
			require.Contains(t, keyboard[0][0].CallbackData, compactCallbackUUID(options[0].VocabularyID))
			secondButton := keyboard[1/test.rowWidth][1%test.rowWidth]
			require.Equal(t, test.longLabel, secondButton.Text)
			require.Contains(
				t,
				secondButton.CallbackData,
				compactCallbackUUID(options[1].VocabularyID),
			)
		})
	}
}

func TestMatchExerciseKeyboardUsesSingleColumnForLongWords(t *testing.T) {
	exerciseID := uuid.MustParse("52fdfc07-2182-454f-963f-5f0f9a621d72")
	tests := []struct {
		name     string
		word     string
		rowCount int
		rowWidth int
	}{
		{
			name:     "keeps two columns at the limit despite status icons",
			word:     strings.Repeat("я", exerciseCompactButtonMaxRunes),
			rowCount: 2,
			rowWidth: 2,
		},
		{
			name:     "uses one column above the limit",
			word:     strings.Repeat("я", exerciseCompactButtonMaxRunes+1),
			rowCount: 4,
			rowWidth: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			vocabularyID := uuid.New()
			board := &services.MatchBoardState{
				Order: []int{0, 1, 2, 3},
				Cards: []services.ExerciseMatchCard{
					{ID: "card-1", VocabularyID: vocabularyID, Word: "one"},
					{ID: "card-2", VocabularyID: uuid.New(), Word: test.word},
					{ID: "card-3", VocabularyID: uuid.New(), Word: "three"},
					{ID: "card-4", VocabularyID: uuid.New(), Word: "four"},
				},
				Pending:  0,
				Resolved: map[uuid.UUID]string{vocabularyID: services.ExerciseVocabularyResultCorrect},
				CardWrong: map[string]int{
					"card-2": 1,
				},
			}

			keyboard := buildMatchKeyboard(exerciseID, board)

			require.Len(t, keyboard, test.rowCount)
			for _, row := range keyboard {
				require.Len(t, row, test.rowWidth)
			}
			require.Equal(t, "✅ one", keyboard[0][0].Text)
			secondButton := keyboard[1/test.rowWidth][1%test.rowWidth]
			require.Equal(t, "⚠️ "+test.word, secondButton.Text)
		})
	}
}
