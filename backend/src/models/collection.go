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
	// No gorm default tag: a `default` tag makes GORM omit the zero value from INSERTs, which
	// would silently turn unpublished drafts into published. Every Create path sets it explicitly.
	IsPublished bool       `json:"is_published"`
	InviteToken string     `json:"-"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"-"`
	DeletedAt   *time.Time `json:"-"`
}

func (c *Collection) TableName() string {
	return "collections"
}

func (c *Collection) IsOwnedBy(userID uint) bool {
	return c.OwnerID != nil && *c.OwnerID == userID
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

type CollectionTranslation struct {
	CollectionID  uuid.UUID `gorm:"primaryKey"`
	TranslationID uuid.UUID `gorm:"primaryKey"`
	Position      int
}

func (CollectionTranslation) TableName() string {
	return "collection_translations"
}

type CollectionMember struct {
	CollectionID uuid.UUID `gorm:"primaryKey"`
	UserID       uint      `gorm:"primaryKey"`
	CreatedAt    time.Time
}

func (CollectionMember) TableName() string {
	return "collection_members"
}

type CollectionUserAdd struct {
	CollectionID uuid.UUID `gorm:"primaryKey"`
	UserID       uint      `gorm:"primaryKey"`
	CreatedAt    time.Time
}

func (CollectionUserAdd) TableName() string {
	return "collection_user_adds"
}
