package models

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Collection struct {
	ID      uuid.UUID `json:"id" gorm:"default:gen_random_uuid()"`
	Title   string    `json:"title"`
	OwnerID *uint     `json:"-"`
	IsAdmin bool      `json:"is_admin"`
	// No gorm default tag: GORM omits zero-valued fields that carry a `default` tag from
	// INSERTs, which would make Published:false (drafts) silently become true. Every Create
	// path sets Published explicitly; the DB column keeps DEFAULT true for the backfill.
	IsPublished   bool       `json:"is_published"`
	InviteToken string     `json:"-"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"-"`
	DeletedAt   *time.Time `json:"-"`
}

func (c *Collection) TableName() string {
	return "collections"
}

func (c *Collection) BeforeCreate(_ *gorm.DB) error {
	if strings.TrimSpace(c.Title) == "" {
		return errors.New("collection title can't be empty")
	}

	if !c.IsAdmin && c.OwnerID == nil {
		return errors.New("non-admin collection must have an owner")
	}

	return nil
}

// CollectionTranslation is the join row linking a collection to a translation.
// Position is the manual sort order within the collection (ascending); a newly added
// translation gets max(position)+1 so it lands at the end of the list.
type CollectionTranslation struct {
	CollectionID  uuid.UUID `gorm:"primaryKey"`
	TranslationID uuid.UUID `gorm:"primaryKey"`
	Position      int
}

func (CollectionTranslation) TableName() string {
	return "collection_translations"
}

// CollectionMember is the join row granting a user access to a shared collection
// (added via the collection's invite link).
type CollectionMember struct {
	CollectionID uuid.UUID `gorm:"primaryKey"`
	UserID       uint      `gorm:"primaryKey"`
	CreatedAt    time.Time
}

func (CollectionMember) TableName() string {
	return "collection_members"
}

// CollectionUserAdd tracks which users have added a collection to their vocabulary.
// Each user can only count once per collection.
type CollectionUserAdd struct {
	CollectionID uuid.UUID `gorm:"primaryKey"`
	UserID       uint      `gorm:"primaryKey"`
	CreatedAt    time.Time
}

func (CollectionUserAdd) TableName() string {
	return "collection_user_adds"
}
