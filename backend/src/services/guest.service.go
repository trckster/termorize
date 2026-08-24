package services

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"
	"time"

	"gorm.io/gorm"
)

const GuestAccountLifetime = 7 * 24 * time.Hour

const guestVocabularySeedLock = "termorize_guest_vocabulary_seed"

type guestVocabularyPair struct {
	Original    string
	Translation string
}

var guestNameAdjectives = []string{
	"Ambiguous", "Brave", "Calm", "Clever", "Curious", "Daring", "Eager", "Gentle",
	"Kind", "Lively", "Mellow", "Nimble", "Patient", "Quiet", "Radiant", "Steady",
	"Thoughtful", "Vivid", "Witty", "Young",
}

var guestNameAnimals = []string{
	"Badger", "Dolphin", "Falcon", "Fox", "Gazelle", "Heron", "Koala", "Lynx", "Otter",
	"Owl", "Panda", "Panther", "Penguin", "Raven", "Seal", "Sparrow", "Tiger", "Turtle",
	"Wolf", "Wombat",
}

var guestVocabularySeed = []guestVocabularyPair{
	{Original: "apple", Translation: "яблоко"},
	{Original: "book", Translation: "книга"},
	{Original: "water", Translation: "вода"},
	{Original: "house", Translation: "дом"},
	{Original: "friend", Translation: "друг"},
	{Original: "family", Translation: "семья"},
	{Original: "school", Translation: "школа"},
	{Original: "work", Translation: "работа"},
	{Original: "time", Translation: "время"},
	{Original: "day", Translation: "день"},
	{Original: "night", Translation: "ночь"},
	{Original: "morning", Translation: "утро"},
	{Original: "city", Translation: "город"},
	{Original: "country", Translation: "страна"},
	{Original: "road", Translation: "дорога"},
	{Original: "car", Translation: "машина"},
	{Original: "train", Translation: "поезд"},
	{Original: "food", Translation: "еда"},
	{Original: "coffee", Translation: "кофе"},
	{Original: "tea", Translation: "чай"},
	{Original: "music", Translation: "музыка"},
	{Original: "movie", Translation: "фильм"},
	{Original: "language", Translation: "язык"},
	{Original: "word", Translation: "слово"},
	{Original: "question", Translation: "вопрос"},
	{Original: "answer", Translation: "ответ"},
	{Original: "happy", Translation: "счастливый"},
	{Original: "sad", Translation: "грустный"},
	{Original: "big", Translation: "большой"},
	{Original: "small", Translation: "маленький"},
	{Original: "fast", Translation: "быстрый"},
	{Original: "slow", Translation: "медленный"},
	{Original: "new", Translation: "новый"},
	{Original: "old", Translation: "старый"},
	{Original: "good", Translation: "хороший"},
	{Original: "bad", Translation: "плохой"},
	{Original: "beautiful", Translation: "красивый"},
	{Original: "important", Translation: "важный"},
	{Original: "easy", Translation: "лёгкий"},
	{Original: "difficult", Translation: "трудный"},
	{Original: "learn", Translation: "учиться"},
	{Original: "read", Translation: "читать"},
	{Original: "write", Translation: "писать"},
	{Original: "speak", Translation: "говорить"},
	{Original: "listen", Translation: "слушать"},
	{Original: "remember", Translation: "помнить"},
	{Original: "understand", Translation: "понимать"},
	{Original: "travel", Translation: "путешествовать"},
	{Original: "begin", Translation: "начинать"},
	{Original: "finish", Translation: "заканчивать"},
}

func CreateGuestUser(timezone string, systemLanguage enums.Language) (*models.User, error) {
	username, err := randomGuestUsername()
	if err != nil {
		return nil, err
	}

	name, err := randomGuestName()
	if err != nil {
		return nil, err
	}

	settings := defaultUserSettings(timezone, false)
	settings.SystemLanguage = systemLanguage
	settings.Telegram.DailyQuestionsEnabled = false
	expiresAt := time.Now().UTC().Add(GuestAccountLifetime)
	user := models.User{
		Username:       username,
		Name:           name,
		Settings:       settings,
		GuestExpiresAt: &expiresAt,
	}

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit("TelegramID").Create(&user).Error; err != nil {
			return err
		}

		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", guestVocabularySeedLock).Error; err != nil {
			return err
		}

		return seedGuestVocabulary(tx, user.ID)
	})
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func DeleteExpiredGuestUsers(expiredBefore time.Time) (int64, error) {
	result := db.DB.
		Where("guest_expires_at IS NOT NULL").
		Where("guest_expires_at <= ?", expiredBefore.UTC()).
		Delete(&models.User{})

	return result.RowsAffected, result.Error
}

func seedGuestVocabulary(tx *gorm.DB, userID uint) error {
	for _, pair := range guestVocabularySeed {
		original, err := GetOrCreateWord(tx, pair.Original, enums.LanguageEn)
		if err != nil {
			return err
		}

		translated, err := GetOrCreateWord(tx, pair.Translation, enums.LanguageRu)
		if err != nil {
			return err
		}

		translation, err := getOrCreateGuestSeedTranslation(tx, original, translated)
		if err != nil {
			return err
		}

		vocabulary := models.Vocabulary{
			UserID:        userID,
			TranslationID: translation.ID,
			Progress:      models.BuildDefaultProgress(),
		}
		if err := tx.Create(&vocabulary).Error; err != nil {
			return err
		}
	}

	return nil
}

func getOrCreateGuestSeedTranslation(tx *gorm.DB, original, translated *models.Word) (*models.Translation, error) {
	var translation models.Translation
	result := tx.
		Where("original_id = ?", original.ID).
		Where("translation_id = ?", translated.ID).
		Where("source = ?", enums.TranslationSourceDictionary).
		Where("user_id IS NULL").
		First(&translation)

	if result.Error == nil {
		return &translation, nil
	}
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, result.Error
	}

	translation = models.Translation{
		OriginalID:    original.ID,
		TranslationID: translated.ID,
		Source:        enums.TranslationSourceDictionary,
	}
	if err := tx.Create(&translation).Error; err != nil {
		return nil, err
	}

	return &translation, nil
}

func randomGuestUsername() (string, error) {
	number, err := secureRandomIndex(1_000_000)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("absolutely_random_user_%06d", number), nil
}

func randomGuestName() (string, error) {
	adjectiveIndex, err := secureRandomIndex(len(guestNameAdjectives))
	if err != nil {
		return "", err
	}

	animalIndex, err := secureRandomIndex(len(guestNameAnimals))
	if err != nil {
		return "", err
	}

	return guestNameAdjectives[adjectiveIndex] + " " + guestNameAnimals[animalIndex], nil
}

func secureRandomIndex(limit int) (int, error) {
	value, err := rand.Int(rand.Reader, big.NewInt(int64(limit)))
	if err != nil {
		return 0, err
	}

	return int(value.Int64()), nil
}
