package classification

import (
	"context"
	"errors"
	"termorize/src/enums"
	"termorize/src/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Store interface {
	Load(context.Context, uuid.UUID) (*models.Word, error)
	Save(context.Context, models.Word, enums.PartOfSpeech) error
	Pending(context.Context, uuid.UUID, int) ([]uuid.UUID, error)
}

type WordStore struct{ DB *gorm.DB }

func (s WordStore) Load(ctx context.Context, id uuid.UUID) (*models.Word, error) {
	var word models.Word
	err := s.DB.WithContext(ctx).First(&word, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &word, err
}

func (s WordStore) Save(ctx context.Context, word models.Word, category enums.PartOfSpeech) error {
	return s.DB.WithContext(ctx).Model(&models.Word{}).
		Where("id = ? AND part_of_speech IS NULL AND word = ? AND language = ?", word.ID, word.Word, word.Language).
		Update("part_of_speech", category).Error
}

func (s WordStore) Pending(ctx context.Context, after uuid.UUID, limit int) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, limit)
	err := s.DB.WithContext(ctx).Model(&models.Word{}).
		Where("part_of_speech IS NULL AND id > ?", after).Order("id").Limit(limit).Pluck("id", &ids).Error
	return ids, err
}
