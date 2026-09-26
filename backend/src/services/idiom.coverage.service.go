package services

import (
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"
)

type IdiomLanguageCoverage struct {
	Language   enums.Language `json:"language"`
	Name       string         `json:"name"`
	IdiomCount int64          `json:"idiom_count"`
}

func ListIdiomLanguageCoverage() ([]IdiomLanguageCoverage, error) {
	var counts []struct {
		Language   enums.Language
		IdiomCount int64
	}
	languages := enums.AllLanguageValues()
	if err := db.DB.Model(&models.Word{}).Select("language, COUNT(*) AS idiom_count").
		Where("type = ? AND language IN ?", enums.TypeIdiom, languages).Group("language").Scan(&counts).Error; err != nil {
		return nil, err
	}
	byLanguage := make(map[enums.Language]int64, len(counts))
	for _, count := range counts {
		byLanguage[count.Language] = count.IdiomCount
	}
	result := make([]IdiomLanguageCoverage, 0, len(languages))
	for _, language := range languages {
		result = append(result, IdiomLanguageCoverage{Language: language, Name: language.DisplayName(), IdiomCount: byLanguage[language]})
	}
	return result, nil
}
