package db

import (
	"termorize/src/utils"

	"gorm.io/gorm"
)

type wordCasingRow struct {
	ID       string
	Word     string
	Language string
}

// backfillWordCasing re-normalizes the casing of every stored word using the
// same rules as word creation (utils.NormalizeWordCasing). Registered as
// migration 0004_normalize_word_casing.
//
// The words table has a UNIQUE (word, language) constraint, so if normalizing a
// row would collide with an already-correctly-cased row, the duplicate's
// translations are repointed to the canonical row and the duplicate is removed.
// Such collisions are essentially impossible in practice because word lookup has
// always been case-insensitive (LOWER(word) = LOWER(?)); the handling is defensive.
func backfillWordCasing() error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var rows []wordCasingRow
		if err := tx.Raw("SELECT id, word, language FROM words").Scan(&rows).Error; err != nil {
			return err
		}

		for _, row := range rows {
			normalized := utils.NormalizeWordCasing(row.Word)
			if normalized == row.Word {
				continue
			}

			var canonicalID string
			if err := tx.Raw(
				"SELECT id FROM words WHERE word = ? AND language = ? AND id <> ?",
				normalized, row.Language, row.ID,
			).Scan(&canonicalID).Error; err != nil {
				return err
			}

			if canonicalID == "" {
				if err := tx.Exec(
					"UPDATE words SET word = ? WHERE id = ?", normalized, row.ID,
				).Error; err != nil {
					return err
				}
				continue
			}

			if err := tx.Exec(
				"UPDATE translations SET original_id = ? WHERE original_id = ?", canonicalID, row.ID,
			).Error; err != nil {
				return err
			}
			if err := tx.Exec(
				"UPDATE translations SET translation_id = ? WHERE translation_id = ?", canonicalID, row.ID,
			).Error; err != nil {
				return err
			}
			if err := tx.Exec("DELETE FROM words WHERE id = ?", row.ID).Error; err != nil {
				return err
			}
		}

		return nil
	})
}
