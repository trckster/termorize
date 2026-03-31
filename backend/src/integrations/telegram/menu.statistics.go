package telegram

import (
	"fmt"
	"strings"
	"termorize/src/services"
)

func buildStatisticsMenuText(userID uint, t BotTexts) (string, error) {
	vocabularyStatistics, err := services.GetVocabularyStatistics(userID)
	if err != nil {
		return "", err
	}

	exerciseStatistics, err := services.GetExerciseStatistics(userID)
	if err != nil {
		return "", err
	}

	totalExercises := exerciseStatistics.InProgress + exerciseStatistics.Done + exerciseStatistics.Failed + exerciseStatistics.Ignored

	lines := []string{
		t.MenuStatisticsTitle,
		"",
		t.MenuStatisticsVocabulary,
		fmt.Sprintf(t.MenuStatisticsTotalFormat, vocabularyStatistics.Total),
		fmt.Sprintf(t.MenuStatisticsMasteredFormat, vocabularyStatistics.Mastered),
		fmt.Sprintf(t.MenuStatisticsInProgressFormat, vocabularyStatistics.InProgress),
		fmt.Sprintf(t.MenuStatisticsPendingFormat, vocabularyStatistics.Pending),
		"",
		t.MenuStatisticsExercises,
		fmt.Sprintf(t.MenuStatisticsTotalFormat, totalExercises),
		fmt.Sprintf(t.MenuStatisticsInProgressFormat, exerciseStatistics.InProgress),
		fmt.Sprintf(t.MenuStatisticsSuccessfulFormat, exerciseStatistics.Done),
		fmt.Sprintf(t.MenuStatisticsUnsuccessfulFormat, exerciseStatistics.Failed+exerciseStatistics.Ignored),
	}

	return strings.Join(lines, "\n"), nil
}
