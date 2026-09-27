package models

import (
	"github.com/google/uuid"
	"time"
)

type Dictionary struct {
	ID          uuid.UUID `json:"id" gorm:"default:gen_random_uuid()"`
	Name        string    `json:"name"`
	Edition     string    `json:"edition"`
	URL         string    `json:"url"`
	License     string    `json:"license"`
	Attribution string    `json:"attribution"`
	DownloadURL string    `json:"download_url"`
}

type DictionaryImportJob struct {
	ID              uuid.UUID  `json:"id" gorm:"default:gen_random_uuid()"`
	DictionaryID    uuid.UUID  `json:"dictionary_id"`
	SourceName      string     `json:"source_name"`
	Edition         string     `json:"edition"`
	DownloadURL     string     `json:"download_url"`
	Status          string     `json:"status"`
	Processed       int64      `json:"processed"`
	Inserted        int64      `json:"inserted"`
	Classified      int64      `json:"classified"`
	Skipped         int64      `json:"skipped"`
	Failed          int64      `json:"failed"`
	DownloadedBytes int64      `json:"downloaded_bytes"`
	TotalBytes      *int64     `json:"total_bytes"`
	Error           string     `json:"error"`
	RecordErrors    []string   `json:"record_errors" gorm:"serializer:json"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	StartedAt       *time.Time `json:"started_at"`
	FinishedAt      *time.Time `json:"finished_at"`
}
