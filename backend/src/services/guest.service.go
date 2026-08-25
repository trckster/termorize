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

const (
	guestEasyVocabularyCount   = 15
	guestMediumVocabularyCount = 20
	guestHardVocabularyCount   = 15
)

type guestVocabularyDifficulty uint8

const (
	guestVocabularyDifficultyEasy guestVocabularyDifficulty = iota
	guestVocabularyDifficultyMedium
	guestVocabularyDifficultyHard
)

type guestVocabularyPair struct {
	Original    string
	Translation string
	Difficulty  guestVocabularyDifficulty
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
	{Original: "apple", Translation: "яблоко", Difficulty: guestVocabularyDifficultyEasy},
	{Original: "book", Translation: "книга", Difficulty: guestVocabularyDifficultyEasy},
	{Original: "water", Translation: "вода", Difficulty: guestVocabularyDifficultyEasy},
	{Original: "house", Translation: "дом", Difficulty: guestVocabularyDifficultyEasy},
	{Original: "friend", Translation: "друг", Difficulty: guestVocabularyDifficultyEasy},
	{Original: "family", Translation: "семья", Difficulty: guestVocabularyDifficultyEasy},
	{Original: "school", Translation: "школа", Difficulty: guestVocabularyDifficultyEasy},
	{Original: "day", Translation: "день", Difficulty: guestVocabularyDifficultyEasy},
	{Original: "night", Translation: "ночь", Difficulty: guestVocabularyDifficultyEasy},
	{Original: "city", Translation: "город", Difficulty: guestVocabularyDifficultyEasy},
	{Original: "road", Translation: "дорога", Difficulty: guestVocabularyDifficultyEasy},
	{Original: "car", Translation: "автомобиль", Difficulty: guestVocabularyDifficultyEasy},
	{Original: "train", Translation: "поезд", Difficulty: guestVocabularyDifficultyEasy},
	{Original: "food", Translation: "еда", Difficulty: guestVocabularyDifficultyEasy},
	{Original: "coffee", Translation: "кофе", Difficulty: guestVocabularyDifficultyEasy},
	{Original: "journey", Translation: "путешествие", Difficulty: guestVocabularyDifficultyMedium},
	{Original: "decision", Translation: "решение", Difficulty: guestVocabularyDifficultyMedium},
	{Original: "achievement", Translation: "достижение", Difficulty: guestVocabularyDifficultyMedium},
	{Original: "responsibility", Translation: "ответственность", Difficulty: guestVocabularyDifficultyMedium},
	{Original: "opportunity", Translation: "возможность", Difficulty: guestVocabularyDifficultyMedium},
	{Original: "obstacle", Translation: "препятствие", Difficulty: guestVocabularyDifficultyMedium},
	{Original: "evidence", Translation: "доказательство", Difficulty: guestVocabularyDifficultyMedium},
	{Original: "consequence", Translation: "последствие", Difficulty: guestVocabularyDifficultyMedium},
	{Original: "uncertainty", Translation: "неопределённость", Difficulty: guestVocabularyDifficultyMedium},
	{Original: "awareness", Translation: "осведомлённость", Difficulty: guestVocabularyDifficultyMedium},
	{Original: "ambiguity", Translation: "неоднозначность", Difficulty: guestVocabularyDifficultyMedium},
	{Original: "feasibility", Translation: "осуществимость", Difficulty: guestVocabularyDifficultyMedium},
	{Original: "incompatibility", Translation: "несовместимость", Difficulty: guestVocabularyDifficultyMedium},
	{Original: "longevity", Translation: "долголетие", Difficulty: guestVocabularyDifficultyMedium},
	{Original: "vigilance", Translation: "бдительность", Difficulty: guestVocabularyDifficultyMedium},
	{Original: "vulnerability", Translation: "уязвимость", Difficulty: guestVocabularyDifficultyMedium},
	{Original: "concise", Translation: "лаконичный", Difficulty: guestVocabularyDifficultyMedium},
	{Original: "impartial", Translation: "беспристрастный", Difficulty: guestVocabularyDifficultyMedium},
	{Original: "inevitable", Translation: "неизбежный", Difficulty: guestVocabularyDifficultyMedium},
	{Original: "obsolete", Translation: "устаревший", Difficulty: guestVocabularyDifficultyMedium},
	{Original: "candor", Translation: "откровенность", Difficulty: guestVocabularyDifficultyHard},
	{Original: "equanimity", Translation: "невозмутимость", Difficulty: guestVocabularyDifficultyHard},
	{Original: "futility", Translation: "тщетность", Difficulty: guestVocabularyDifficultyHard},
	{Original: "ingenuity", Translation: "изобретательность", Difficulty: guestVocabularyDifficultyHard},
	{Original: "magnanimity", Translation: "великодушие", Difficulty: guestVocabularyDifficultyHard},
	{Original: "unanimity", Translation: "единодушие", Difficulty: guestVocabularyDifficultyHard},
	{Original: "impeccable", Translation: "безупречный", Difficulty: guestVocabularyDifficultyHard},
	{Original: "inadvertent", Translation: "непреднамеренный", Difficulty: guestVocabularyDifficultyHard},
	{Original: "meticulous", Translation: "скрупулёзный", Difficulty: guestVocabularyDifficultyHard},
	{Original: "paramount", Translation: "первостепенный", Difficulty: guestVocabularyDifficultyHard},
	{Original: "plausible", Translation: "правдоподобный", Difficulty: guestVocabularyDifficultyHard},
	{Original: "ubiquitous", Translation: "вездесущий", Difficulty: guestVocabularyDifficultyHard},
	{Original: "exacerbate", Translation: "усугублять", Difficulty: guestVocabularyDifficultyHard},
	{Original: "elucidate", Translation: "разъяснять", Difficulty: guestVocabularyDifficultyHard},
	{Original: "impede", Translation: "препятствовать", Difficulty: guestVocabularyDifficultyHard},
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
	seed, err := randomizedGuestVocabularySeed()
	if err != nil {
		return err
	}

	for _, pair := range seed {
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

		knowledge, err := randomGuestVocabularyKnowledge(pair.Difficulty)
		if err != nil {
			return err
		}

		vocabulary := models.Vocabulary{
			UserID:        userID,
			TranslationID: translation.ID,
			Progress: models.ProgressEntries{{
				Knowledge: knowledge,
				Type:      enums.KnowledgeTypeTranslation,
			}},
		}
		if err := tx.Create(&vocabulary).Error; err != nil {
			return err
		}
	}

	return nil
}

func randomizedGuestVocabularySeed() ([]guestVocabularyPair, error) {
	easy := make([]guestVocabularyPair, 0, guestEasyVocabularyCount)
	medium := make([]guestVocabularyPair, 0, guestMediumVocabularyCount)
	hard := make([]guestVocabularyPair, 0, guestHardVocabularyCount)

	for _, pair := range guestVocabularySeed {
		switch pair.Difficulty {
		case guestVocabularyDifficultyEasy:
			easy = append(easy, pair)
		case guestVocabularyDifficultyMedium:
			medium = append(medium, pair)
		case guestVocabularyDifficultyHard:
			hard = append(hard, pair)
		default:
			return nil, fmt.Errorf("unknown guest vocabulary difficulty: %d", pair.Difficulty)
		}
	}

	if len(easy) != guestEasyVocabularyCount || len(medium) != guestMediumVocabularyCount || len(hard) != guestHardVocabularyCount {
		return nil, fmt.Errorf("invalid guest vocabulary distribution: easy=%d medium=%d hard=%d", len(easy), len(medium), len(hard))
	}

	for _, group := range [][]guestVocabularyPair{easy, medium, hard} {
		if err := shuffleGuestVocabularyPairs(group); err != nil {
			return nil, err
		}
	}

	mediumExtraCount := len(medium) - len(easy)
	seed := make([]guestVocabularyPair, 0, len(guestVocabularySeed))
	seed = append(seed, medium[:mediumExtraCount]...)
	for index := range easy {
		batch := []guestVocabularyPair{easy[index], medium[mediumExtraCount+index], hard[index]}
		if err := shuffleGuestVocabularyPairs(batch); err != nil {
			return nil, err
		}
		seed = append(seed, batch...)
	}

	return seed, nil
}

func shuffleGuestVocabularyPairs(pairs []guestVocabularyPair) error {
	for index := len(pairs) - 1; index > 0; index-- {
		swapIndex, err := secureRandomIndex(index + 1)
		if err != nil {
			return err
		}
		pairs[index], pairs[swapIndex] = pairs[swapIndex], pairs[index]
	}

	return nil
}

func randomGuestVocabularyKnowledge(difficulty guestVocabularyDifficulty) (int, error) {
	var minimum, maximum int
	switch difficulty {
	case guestVocabularyDifficultyEasy:
		minimum, maximum = 70, 90
	case guestVocabularyDifficultyMedium:
		minimum, maximum = 30, 60
	case guestVocabularyDifficultyHard:
		minimum, maximum = 0, 10
	default:
		return 0, fmt.Errorf("unknown guest vocabulary difficulty: %d", difficulty)
	}

	offset, err := secureRandomIndex(maximum - minimum + 1)
	if err != nil {
		return 0, err
	}

	return minimum + offset, nil
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
