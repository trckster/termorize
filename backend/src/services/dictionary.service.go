package services

import (
	"errors"
	"termorize/src/data/db"
	"termorize/src/integrations/kaikki"
	"termorize/src/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrDictionaryImportActive = errors.New("an import is already active for this dictionary")
var ErrDictionaryEdition = errors.New("unsupported dictionary edition")

func ListDictionaries() ([]models.Dictionary, error) {
	dictionaries := []models.Dictionary{}
	err := db.DB.Order("name").Find(&dictionaries).Error
	return dictionaries, err
}

func StartDictionaryImport(id uuid.UUID) (*models.DictionaryImportJob, error) {
	var dictionary models.Dictionary
	if err := db.DB.First(&dictionary, "id = ?", id).Error; err != nil {
		return nil, err
	}
	if !kaikki.SupportedEdition(dictionary.Edition) {
		return nil, ErrDictionaryEdition
	}
	job := models.DictionaryImportJob{
		DictionaryID: id, SourceName: dictionary.Name, Edition: dictionary.Edition,
		DownloadURL: dictionary.DownloadURL, TargetLanguage: dictionary.TargetLanguage, Status: "queued", RecordErrors: []string{},
	}
	if err := db.DB.Create(&job).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrDictionaryImportActive
		}
		return nil, err
	}
	return &job, nil
}

type DictionaryImportHistory struct {
	Data       []models.DictionaryImportJob `json:"data"`
	Pagination Pagination                   `json:"pagination"`
}

func ListDictionaryImports(id uuid.UUID, page int) (*DictionaryImportHistory, error) {
	if page < 1 || page > 1000000 {
		return nil, ErrInvalidPage
	}
	var dictionary models.Dictionary
	if err := db.DB.First(&dictionary, "id = ?", id).Error; err != nil {
		return nil, err
	}
	result := DictionaryImportHistory{Data: []models.DictionaryImportJob{}, Pagination: Pagination{Page: page, PageSize: 20}}
	query := db.DB.Model(&models.DictionaryImportJob{}).Where("dictionary_id = ?", id)
	if err := query.Count(&result.Pagination.Total).Error; err != nil {
		return nil, err
	}
	result.Pagination.TotalPages = int((result.Pagination.Total + 19) / 20)
	err := query.Order("created_at DESC, id DESC").Offset((page - 1) * 20).Limit(20).Find(&result.Data).Error
	return &result, err
}
