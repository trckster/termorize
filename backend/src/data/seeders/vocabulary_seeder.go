package seeders

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type VocabularySeedRequest struct {
	UserID *uint
}

type WordPair struct {
	en string
	ru string
	it string
	de string
}

var wordPairs = []WordPair{
	{"apple", "яблоко", "mela", "Apfel"},
	{"banana", "банан", "banana", "Banane"},
	{"cat", "кот", "gatto", "Katze"},
	{"dog", "собака", "cane", "Hund"},
	{"house", "дом", "casa", "Haus"},
	{"water", "вода", "acqua", "Wasser"},
	{"sun", "солнце", "sole", "Sonne"},
	{"moon", "луна", "luna", "Mond"},
	{"star", "звезда", "stella", "Stern"},
	{"tree", "дерево", "albero", "Baum"},
	{"flower", "цветок", "fiore", "Blume"},
	{"bird", "птица", "uccello", "Vogel"},
	{"fish", "рыба", "pesce", "Fisch"},
	{"book", "книга", "libro", "Buch"},
	{"table", "стол", "tavolo", "Tisch"},
	{"chair", "стул", "sedia", "Stuhl"},
	{"door", "дверь", "porta", "Tür"},
	{"window", "окно", "finestra", "Fenster"},
	{"car", "машина", "auto", "Auto"},
	{"road", "дорога", "strada", "Straße"},
	{"mountain", "гора", "montagna", "Berg"},
	{"river", "река", "fiume", "Fluss"},
	{"sea", "море", "mare", "Meer"},
	{"beach", "пляж", "spiaggia", "Strand"},
	{"city", "город", "città", "Stadt"},
	{"country", "страна", "paese", "Land"},
	{"school", "школа", "scuola", "Schule"},
	{"hospital", "больница", "ospedale", "Krankenhaus"},
	{"restaurant", "ресторан", "ristorante", "Restaurant"},
	{"shop", "магазин", "negozio", "Geschäft"},
	{"heart", "сердце", "cuore", "Herz"},
	{"hand", "рука", "mano", "Hand"},
	{"head", "голова", "testa", "Kopf"},
	{"foot", "нога", "piede", "Fuß"},
	{"eye", "глаз", "occhio", "Auge"},
	{"ear", "ухо", "orecchio", "Ohr"},
	{"nose", "нос", "naso", "Nase"},
	{"mouth", "рот", "bocca", "Mund"},
	{"face", "лицо", "faccia", "Gesicht"},
	{"hair", "волосы", "capelli", "Haar"},
	{"love", "любовь", "amore", "Liebe"},
	{"friend", "друг", "amico", "Freund"},
	{"family", "семья", "famiglia", "Familie"},
	{"mother", "мать", "madre", "Mutter"},
	{"father", "отец", "padre", "Vater"},
	{"brother", "брат", "fratello", "Bruder"},
	{"sister", "сестра", "sorella", "Schwester"},
	{"child", "ребенок", "bambino", "Kind"},
	{"man", "мужчина", "uomo", "Mann"},
	{"woman", "женщина", "donna", "Frau"},
	{"time", "время", "tempo", "Zeit"},
	{"day", "день", "giorno", "Tag"},
	{"night", "ночь", "notte", "Nacht"},
	{"morning", "утро", "mattina", "Morgen"},
	{"evening", "вечер", "sera", "Abend"},
	{"week", "неделя", "settimana", "Woche"},
	{"month", "месяц", "mese", "Monat"},
	{"year", "год", "anno", "Jahr"},
	{"number", "число", "numero", "Zahl"},
	{"one", "один", "uno", "eins"},
	{"two", "два", "due", "zwei"},
	{"three", "три", "tre", "drei"},
	{"four", "четыре", "quattro", "vier"},
	{"five", "пять", "cinque", "fünf"},
	{"six", "шесть", "sei", "sechs"},
	{"seven", "семь", "sette", "sieben"},
	{"eight", "восемь", "otto", "acht"},
	{"nine", "девять", "nove", "neun"},
	{"ten", "десять", "dieci", "zehn"},
	{"good", "хороший", "buono", "gut"},
	{"bad", "плохой", "cattivo", "schlecht"},
	{"happy", "счастливый", "felice", "glücklich"},
	{"sad", "грустный", "triste", "traurig"},
	{"big", "большой", "grande", "groß"},
	{"small", "маленький", "piccolo", "klein"},
	{"hot", "горячий", "caldo", "heiß"},
	{"cold", "холодный", "freddo", "kalt"},
	{"fast", "быстрый", "veloce", "schnell"},
	{"slow", "медленный", "lento", "langsam"},
	{"strong", "сильный", "forte", "stark"},
	{"weak", "слабый", "debole", "schwach"},
	{"easy", "легкий", "facile", "einfach"},
	{"difficult", "трудный", "difficile", "schwierig"},
	{"beautiful", "красивый", "bello", "schön"},
	{"ugly", "уродливый", "brutto", "hässlich"},
	{"red", "красный", "rosso", "rot"},
	{"blue", "синий", "blu", "blau"},
	{"green", "зеленый", "verde", "grün"},
	{"yellow", "желтый", "giallo", "gelb"},
	{"black", "черный", "nero", "schwarz"},
	{"white", "белый", "bianco", "weiß"},
	{"eat", "есть", "mangiare", "essen"},
	{"drink", "пить", "bere", "trinken"},
	{"sleep", "спать", "dormire", "schlafen"},
	{"run", "бегать", "correre", "laufen"},
	{"sit", "сидеть", "sedere", "sitzen"},
	{"stand", "стоять", "stare", "stehen"},
	{"jump", "прыгать", "saltare", "springen"},
	{"play", "играть", "giocare", "spielen"},
	{"work", "работать", "lavorare", "arbeiten"},
	{"study", "учиться", "studiare", "studieren"},
	{"write", "писать", "scrivere", "schreiben"},
	{"read", "читать", "leggere", "lesen"},
	{"speak", "говорить", "parlare", "sprechen"},
	{"listen", "слушать", "ascoltare", "zuhören"},
	{"see", "видеть", "vedere", "sehen"},
	{"hear", "слышать", "sentire", "hören"},
	{"smell", "пахнуть", "odorare", "riechen"},
	{"taste", "пробовать", "assaggiare", "schmecken"},
	{"touch", "касаться", "toccare", "berühren"},
	{"think", "думать", "pensare", "denken"},
	{"know", "знать", "sapere", "wissen"},
	{"understand", "понимать", "capire", "verstehen"},
	{"learn", "учить что-то", "imparare", "lernen"},
	{"teach", "учить", "insegnare", "lehren"},
	{"give", "давать", "dare", "geben"},
	{"take", "брать", "prendere", "nehmen"},
	{"buy", "покупать", "comprare", "kaufen"},
	{"sell", "продавать", "vendere", "verkaufen"},
	{"come", "приходить", "venire", "kommen"},
	{"go", "идти", "andare", "gehen"},
	{"stay", "оставаться", "rimanere", "bleiben"},
	{"leave", "уходить", "partire", "verlassen"},
	{"arrive", "прибывать", "arrivare", "ankommen"},
	{"begin", "начинать", "iniziare", "beginnen"},
	{"end", "заканчивать", "finire", "beenden"},
	{"find", "найти", "trovare", "finden"},
	{"lose", "потерять", "perdere", "verlieren"},
	{"win", "выиграть", "vincere", "gewinnen"},
	{"happen", "случаться", "accadere", "geschehen"},
	{"rain", "идти дождь", "piovere", "regnen"},
	{"snow", "идти снег", "nevicare", "schneien"},
	{"shine", "светить", "brillare", "scheinen"},
	{"laugh", "смеяться", "ridere", "lachen"},
	{"cry", "плакать", "piangere", "weinen"},
	{"smile", "улыбаться", "sorridere", "lächeln"},
	{"dance", "танцевать", "ballare", "tanzen"},
	{"sing", "петь", "cantare", "singen"},
	{"swim", "плавать", "nuotare", "schwimmen"},
	{"fly", "летать", "volare", "fliegen"},
	{"drive", "ездить", "guidare", "fahren"},
	{"ephemeral", "мимолетный", "effimero", "flüchtig"},
	{"serendipity", "случайная удача", "serendipità", "Serendipität"},
	{"ubiquitous", "всюдусущий", "onnipresente", "allgegenwärtig"},
	{"paradigm", "парадигма", "paradigma", "Paradigma"},
	{"idiosyncratic", "идиосинкратический", "idiosincratico", "eigenwillig"},
	{"juxtaposition", "сопоставление", "giustapposizione", "Gegenüberstellung"},
	{"quintessential", "квинтэссенция", "quintessenza", "Quintessenz"},
	{"conundrum", "головоломка", "dilemma", "Dilemma"},
	{"ambivalence", "амбивалентность", "ambivalenza", "Ambivalenz"},
	{"epiphany", "озарение", "epifania", "Erleuchtung"},
	{"solitude", "уединение", "solitudine", "Einsamkeit"},
	{"inevitable", "неизбежный", "inevitabile", "unvermeidlich"},
	{"perplexing", "озадачивающий", "sconcertante", "verwirrend"},
	{"resilience", "устойчивость", "resilienza", "Belastbarkeit"},
	{"metamorphosis", "метаморфоза", "metamorfosi", "Metamorphose"},
	{"perennial", "многолетний", "perenne", "mehrjährig"},
	{"ephemerality", "мимолетность", "effimerità", "Flüchtigkeit"},
	{"vicissitude", "превратность", "vicissitudine", "Wechselfälle"},
	{"surreptitious", "тайный", "clandestino", "heimlich"},
	{"ubiquity", "всеобщность", "onnipresenza", "Allgegenwart"},
	{"esoteric", "эзотерический", "esoterico", "esoterisch"},
	{"candid", "откровенный", "schietto", "offen"},
	{"indigenous", "коренной", "indigeno", "einheimisch"},
	{"prolific", "плодовитый", "prolifico", "fruchtbar"},
	{"versatile", "универсальный", "versatile", "vielseitig"},
	{"obstinate", "упрямый", "ostinato", "eigensinnig"},
	{"vulnerable", "уязвимый", "vulnerabile", "verletzlich"},
	{"autonomous", "автономный", "autonomo", "autonom"},
	{"magnificent", "величественный", "magnifico", "großartig"},
	{"superfluous", "излишний", "superfluo", "überflüssig"},
	{"perseverance", "настойчивость", "perseveranza", "Ausdauer"},
	{"hypothesis", "гипотеза", "ipotesi", "Hypothese"},
	{"synchronicity", "синхронность", "sincronicità", "Synchronizität"},
	{"paradox", "парадокс", "paradosso", "Paradoxon"},
	{"anachronism", "анахронизм", "anacronismo", "Anachronismus"},
	{"cacophony", "какофония", "cacofonia", "Kakophonie"},
	{"serene", "безмятежный", "sereno", "gelassen"},
	{"nostalgia", "ностальгия", "nostalgia", "Nostalgie"},
	{"pragmatic", "практичный", "pragmatico", "pragmatisch"},
	{"spontaneous", "спонтанный", "spontaneo", "spontan"},
	{"ambiguous", "двусмысленный", "ambiguo", "mehrdeutig"},
	{"benevolent", "доброжелательный", "benevolo", "wohlwollend"},
	{"diligent", "прилежный", "diligente", "fleißig"},
	{"empirical", "эмпирический", "empirico", "empirisch"},
	{"fluctuate", "колебаться", "fluttuare", "schwanken"},
	{"gratitude", "благодарность", "gratitudine", "Dankbarkeit"},
	{"hierarchy", "иерархия", "gerarchia", "Hierarchie"},
	{"impeccable", "безупречный", "impeccabile", "makellos"},
	{"juxtapose", "сопоставлять", "giustapporre", "gegenüberstellen"},
	{"keystone", "краеугольный камень", "chiave di volta", "Schlussstein"},
	{"lethargic", "вялый", "letargico", "träge"},
	{"meticulous", "скрупулезный", "meticoloso", "sorgfältig"},
	{"nefarious", "злонамеренный", "nefario", "bösartig"},
	{"obfuscate", "затуманивать", "oscurare", "vernebeln"},
	{"palimpsest", "палимпсест", "palinsesto", "Palimpsest"},
	{"quagmire", "трясина", "pantano", "Sumpf"},
	{"beast", "зверь", "bestia", "Tier"},
	{"ocean", "океан", "oceano", "Ozean"},
	{"oak", "дуб", "quercia", "Eiche"},
}

func SeedVocabulary(req VocabularySeedRequest) error {
	userID := req.UserID

	if userID == nil {
		selectedUser, err := getDefaultUser()
		if err != nil {
			return err
		}
		userID = &selectedUser.ID
	}

	user := &models.User{}
	if err := db.DB.First(user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("user with id %d not found", *userID)
		}
		return fmt.Errorf("failed to query user: %w", err)
	}

	words, err := getOrCreateWords()
	if err != nil {
		return fmt.Errorf("failed to get or create words: %w", err)
	}

	translations, err := generateTranslation(words)
	if err != nil {
		return fmt.Errorf("failed to generate vocabulary and translations: %w", err)
	}

	for i := range translations {
		existingTranslation, err := getExistingTranslation(translations[i].Word1ID, translations[i].Word2ID)
		if err != nil {
			return fmt.Errorf("failed to check existing translation: %w", err)
		}
		if existingTranslation != nil {
			translations[i].ID = existingTranslation.ID
		} else {
			if err := db.DB.Create(&translations[i]).Error; err != nil {
				return fmt.Errorf("failed to create translation: %w", err)
			}
		}

		progressJSON, _ := json.Marshal([]models.ProgressEntry{{
			Knowledge: rand.Intn(101),
			Type:      enums.KnowledgeTypeTranslation,
		}})

		vocabulary := models.Vocabulary{
			UserID:        *userID,
			TranslationID: translations[i].ID,
			Progress:      progressJSON,
		}
		if err := db.DB.FirstOrCreate(&vocabulary, models.Vocabulary{
			UserID:        vocabulary.UserID,
			TranslationID: vocabulary.TranslationID,
		}).Error; err != nil {
			return fmt.Errorf("failed to create vocabulary item: %w", err)
		}
	}

	fmt.Printf("Successfully seeded vocabulary items for user ID %d\n", *userID)
	return nil
}

func getDefaultUser() (*models.User, error) {
	var count int64
	if err := db.DB.Model(&models.User{}).Count(&count).Error; err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}

	if count == 0 {
		return nil, fmt.Errorf("no users found in db")
	}

	if count > 1 {
		return nil, fmt.Errorf("db has more than one user (%d users). please specify user ID explicitly", count)
	}

	user := &models.User{}
	if err := db.DB.First(user).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch first user: %w", err)
	}

	return user, nil
}

func getOrCreateWords() (map[string]uuid.UUID, error) {
	wordsByLangAndValue := make(map[string]uuid.UUID)

	languages := []struct {
		lang enums.Language
		idx  int
	}{
		{enums.LanguageEn, 0},
		{enums.LanguageRu, 1},
		{enums.LanguageIt, 2},
		{enums.LanguageDe, 3},
	}

	for _, pair := range wordPairs {
		wordValues := [4]string{pair.en, pair.ru, pair.it, pair.de}
		for i, lang := range languages {
			key := fmt.Sprintf("%s_%s", wordValues[i], lang.lang)

			var existingWord models.Word
			result := db.DB.Where("word = ? AND language = ?", wordValues[i], lang.lang).First(&existingWord)
			if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("failed to query existing word: %w", result.Error)
			}

			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				newWord := models.Word{
					Word:     wordValues[i],
					Language: lang.lang,
				}
				if err := db.DB.Create(&newWord).Error; err != nil {
					return nil, fmt.Errorf("failed to create word: %w", err)
				}
				wordsByLangAndValue[key] = newWord.ID
			} else {
				wordsByLangAndValue[key] = existingWord.ID
			}
		}
	}

	return wordsByLangAndValue, nil
}

func generateTranslation(wordsByLangAndValue map[string]uuid.UUID) ([]models.Translation, error) {
	translations := make([]models.Translation, 0, 444)
	usedPairIndices := make(map[string]bool)

	languages := []struct {
		lang enums.Language
		idx  int
	}{
		{enums.LanguageEn, 0},
		{enums.LanguageRu, 1},
		{enums.LanguageIt, 2},
		{enums.LanguageDe, 3},
	}

	translationCount := 0
	for pairIdx := 0; pairIdx < len(wordPairs) && translationCount < 444; pairIdx++ {
		pair := wordPairs[pairIdx]

		for i, lang1 := range languages {
			if translationCount >= 444 {
				break
			}

			for j, lang2 := range languages {
				if translationCount >= 444 {
					break
				}

				if lang1.lang == lang2.lang || i >= j {
					continue
				}

				pairKey := fmt.Sprintf("%d_%d_%d", pairIdx, lang1.idx, lang2.idx)
				if usedPairIndices[pairKey] {
					continue
				}
				usedPairIndices[pairKey] = true

				wordValues := [4]string{pair.en, pair.ru, pair.it, pair.de}
				word1Key := fmt.Sprintf("%s_%s", wordValues[lang1.idx], lang1.lang)
				word2Key := fmt.Sprintf("%s_%s", wordValues[lang2.idx], lang2.lang)

				word1ID, ok1 := wordsByLangAndValue[word1Key]
				word2ID, ok2 := wordsByLangAndValue[word2Key]

				if !ok1 || !ok2 {
					continue
				}

				tID, _ := uuid.NewRandom()
				translation := models.Translation{
					ID:      tID,
					Word1ID: word1ID,
					Word2ID: word2ID,
					Source:  enums.TranslationSourceDictionary,
				}
				translations = append(translations, translation)

				translationCount++
			}
		}
	}

	return translations, nil
}

func getExistingTranslation(word1ID, word2ID uuid.UUID) (*models.Translation, error) {
	var existingTranslation models.Translation
	result := db.DB.Where(
		"(word_1_id = ? AND word_2_id = ?) OR (word_1_id = ? AND word_2_id = ?)",
		word1ID, word2ID, word2ID, word1ID,
	).First(&existingTranslation)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return &existingTranslation, nil
}
