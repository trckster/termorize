package services

import (
	"context"
	"database/sql"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DailyIdiomWord struct {
	ID     uuid.UUID `json:"id"`
	WordID uuid.UUID `json:"word_id"`
	Word   string    `json:"word"`
}

type DailyIdiomResponse struct {
	Date     string          `json:"date"`
	Language enums.Language  `json:"language"`
	Idiom    *DailyIdiomWord `json:"idiom"`
}

func dailyIdiomDate(now time.Time, timezone string) string {
	location, err := time.LoadLocation(timezone)
	if timezone == "" || err != nil {
		location = time.UTC
	}
	return now.In(location).Format(time.DateOnly)
}

// GetDailyIdiom shares assignments by local date and language, independent of the user.
// now is the request time, captured before waiting for the selection lock.
func GetDailyIdiom(ctx context.Context, userID uint, language enums.Language, now time.Time) (*DailyIdiomResponse, error) {
	var user models.User
	if err := db.DB.WithContext(ctx).Select("settings").First(&user, userID).Error; err != nil {
		return nil, err
	}
	result := &DailyIdiomResponse{Date: dailyIdiomDate(now, user.Settings.TimeZone), Language: language}
	err := db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Serialize all dates in this language so adjacent local dates cannot both
		// choose from stale usage counts. The next statement gets a fresh snapshot.
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", "daily-idiom:"+string(language)).Error; err != nil {
			return err
		}

		var existing DailyIdiomWord
		query := tx.Raw(`
			SELECT d.id, d.word_id, w.word
			FROM daily_idioms d JOIN words w ON w.id = d.word_id
			WHERE d.date = ? AND d.language = ?
		`, result.Date, language).Scan(&existing)
		if query.Error != nil {
			return query.Error
		}
		if query.RowsAffected > 0 {
			result.Idiom = &existing
			return nil
		}

		var word models.Word
		query = tx.Raw(`
			WITH usage AS (
				SELECT w.id, w.word, COUNT(d.id) AS appearances
				FROM words w
				LEFT JOIN daily_idioms d ON d.word_id = w.id AND d.language = w.language
				WHERE w.language = ? AND w.type = 'idiom'
				GROUP BY w.id
			), candidates AS (
				SELECT *, MIN(appearances) OVER () AS lowest, MAX(appearances) OVER () AS highest
				FROM usage
			)
			SELECT id, word FROM candidates
			WHERE lowest = highest OR appearances < highest
			ORDER BY RANDOM() LIMIT 1
		`, language).Scan(&word)
		if query.Error != nil || query.RowsAffected == 0 {
			return query.Error
		}

		date, err := time.Parse(time.DateOnly, result.Date)
		if err != nil {
			return err
		}
		selection := models.DailyIdiom{Date: date, Language: language, WordID: word.ID}
		if err := tx.Create(&selection).Error; err != nil {
			return err
		}
		result.Idiom = &DailyIdiomWord{ID: selection.ID, WordID: word.ID, Word: word.Word}
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}
	return result, nil
}
