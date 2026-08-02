package tests

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
	"time"

	"termorize/src/config"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/integrations/openrouter"
	"termorize/src/models"
	"termorize/src/services"
	"termorize/src/testkit"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTelegramTranslationsAlwaysIncludePronunciationWithoutGeneratingAudio(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		translated string
		prepare    func(t *testing.T, user models.User)
	}{
		{name: "short translation auto-added", text: "dog", translated: "Hund"},
		{
			name:       "existing vocabulary",
			text:       "dog",
			translated: "Hund",
			prepare: func(t *testing.T, user models.User) {
				translationID := vocabSeedTranslation(t, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
				_, err := services.CreateVocabularyByTranslation(user.ID, translationID)
				require.NoError(t, err)
			},
		},
		{name: "multi-word translation", text: "one two three four five", translated: "eins zwei drei vier fünf"},
		{name: "matching translation", text: "radio", translated: "radio"},
	}

	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testkit.Truncate(t)
			tg := testkit.MockTelegramAPI(t)
			generated := 0
			testkit.MockOpenRouterSpeech(t, &testkit.FakeOpenRouterSpeech{GenerateFunc: func(string) ([]byte, error) {
				generated++
				return []byte("unexpected"), nil
			}})
			testkit.MockGoogleTranslate(t, &testkit.FakeGoogleTranslate{
				TranslateFunc: func(string, string, string) (string, error) { return test.translated, nil },
				DetectFunc:    func(string) (string, error) { return "en", nil },
			})

			telegramID := int64(701000 + index)
			user := testkit.CreateUser(t,
				testkit.WithTelegramID(telegramID),
				testkit.WithSettings(models.UserSettings{
					SystemLanguage:            enums.LanguageEn,
					TranslationSourceLanguage: enums.LanguageEn,
					TranslationTargetLanguage: enums.LanguageDe,
				}),
			)
			if test.prepare != nil {
				test.prepare(t, user)
			}

			rec := telegramUpdate(t, telegramPrivateMessage(telegramID, test.text))
			testkit.RequireStatus(t, rec, http.StatusOK)
			require.Zero(t, generated, "translation alone must not call TTS")

			requests := tg.RequestsFor("sendMessage")
			require.Len(t, requests, 1)
			var sent telegramKeyboardRequest
			require.NoError(t, json.Unmarshal(requests[0].Body, &sent))
			button, ok := findCallbackButton(sent.ReplyMarkup.InlineKeyboard, "pronunciation:")
			require.True(t, ok)
			assert.Equal(t, "🔊 Pronunciation", button.Text)
			assert.Len(t, strings.TrimPrefix(button.CallbackData, "pronunciation:"), 22)
		})
	}
}

func TestTelegramPronunciationCacheMissGeneratesStoresAndUploadsMP3(t *testing.T) {
	testkit.Truncate(t)
	tg := testkit.MockTelegramAPI(t)
	translationID, target := seedPronunciationTranslation(t)
	audio := []byte{0xff, 0xfb, 0x01, 0x02, 0x03}
	generated := 0
	testkit.MockOpenRouterSpeech(t, &testkit.FakeOpenRouterSpeech{GenerateFunc: func(input string) ([]byte, error) {
		generated++
		assert.Equal(t, target.Word, input)
		return audio, nil
	}})

	rec := telegramUpdate(t, pronunciationCallbackUpdate(710001, translationID))
	testkit.RequireStatus(t, rec, http.StatusOK)
	require.Equal(t, 1, generated)
	require.Len(t, tg.Requests(), 2)
	assert.Equal(t, "answerCallbackQuery", tg.Requests()[0].Action)
	assert.Equal(t, "sendAudio", tg.Requests()[1].Action)
	assert.Equal(t, audio, uploadedAudio(t, tg.Requests()[1]))

	stored := loadPronunciation(t, target.ID)
	assert.Equal(t, audio, stored.Audio)
	assert.Equal(t, models.PronunciationMIMETypeMP3, stored.MIMEType)
	assert.Equal(t, config.GetOpenRouterTTSModel(), stored.Model)
	assert.Equal(t, config.GetOpenRouterTTSVoice(), stored.Voice)
	require.NotNil(t, stored.TelegramFileID)
	assert.Equal(t, "test-telegram-audio-file-id", *stored.TelegramFileID)
}

func TestTelegramPronunciationDatabaseHitWithoutFileIDUploadsStoredAudio(t *testing.T) {
	testkit.Truncate(t)
	tg := testkit.MockTelegramAPI(t)
	translationID, target := seedPronunciationTranslation(t)
	audio := []byte("stored-mp3")
	pronunciation, err := services.StoreWordPronunciation(target.ID, config.GetOpenRouterTTSModel(), config.GetOpenRouterTTSVoice(), audio)
	require.NoError(t, err)
	generated := 0
	testkit.MockOpenRouterSpeech(t, &testkit.FakeOpenRouterSpeech{GenerateFunc: func(string) ([]byte, error) {
		generated++
		return nil, errors.New("unexpected TTS")
	}})

	rec := telegramUpdate(t, pronunciationCallbackUpdate(710002, translationID))
	testkit.RequireStatus(t, rec, http.StatusOK)
	require.Zero(t, generated)
	require.Len(t, tg.RequestsFor("sendAudio"), 1)
	assert.Equal(t, audio, uploadedAudio(t, tg.RequestsFor("sendAudio")[0]))

	var stored models.WordPronunciation
	require.NoError(t, db.DB.Where("id = ?", pronunciation.ID).First(&stored).Error)
	require.NotNil(t, stored.TelegramFileID)
}

func TestTelegramPronunciationFileIDHitSendsOnlyCachedID(t *testing.T) {
	testkit.Truncate(t)
	tg := testkit.MockTelegramAPI(t)
	translationID, target := seedPronunciationTranslation(t)
	pronunciation, err := services.StoreWordPronunciation(target.ID, config.GetOpenRouterTTSModel(), config.GetOpenRouterTTSVoice(), []byte("must-not-load"))
	require.NoError(t, err)
	require.NoError(t, services.SetWordPronunciationTelegramFileID(pronunciation.ID, "cached-file-id"))
	testkit.MockOpenRouterSpeech(t, &testkit.FakeOpenRouterSpeech{GenerateFunc: func(string) ([]byte, error) {
		return nil, errors.New("unexpected TTS")
	}})

	rec := telegramUpdate(t, pronunciationCallbackUpdate(710003, translationID))
	testkit.RequireStatus(t, rec, http.StatusOK)
	require.Len(t, tg.RequestsFor("sendAudio"), 1)
	request := tg.RequestsFor("sendAudio")[0]
	assert.Equal(t, "application/json", request.Header.Get("Content-Type"))
	var body map[string]any
	require.NoError(t, json.Unmarshal(request.Body, &body))
	assert.Equal(t, "cached-file-id", body["audio"])
	assert.Equal(t, target.Word, body["title"])

	metadata, err := services.FindWordPronunciationMetadata(target.ID, config.GetOpenRouterTTSModel(), config.GetOpenRouterTTSVoice())
	require.NoError(t, err)
	assert.Nil(t, metadata.Audio, "metadata lookup must not load the MP3")
}

func TestTelegramPronunciationSuccessRemovesOnlyPronunciationButton(t *testing.T) {
	testkit.Truncate(t)
	tg := testkit.MockTelegramAPI(t)
	translationID, target := seedPronunciationTranslation(t)
	pronunciation, err := services.StoreWordPronunciation(target.ID, config.GetOpenRouterTTSModel(), config.GetOpenRouterTTSVoice(), []byte("cached-mp3"))
	require.NoError(t, err)
	require.NoError(t, services.SetWordPronunciationTelegramFileID(pronunciation.ID, "cached-file-id"))

	update := pronunciationCallbackUpdate(710007, translationID)
	message := update["callback_query"].(map[string]any)["message"].(map[string]any)
	message["reply_markup"] = map[string]any{
		"inline_keyboard": [][]map[string]string{
			{{"text": "Add to vocabulary", "callback_data": "vocabulary:add:" + translationID.String()}},
			{{"text": "🔊 Pronunciation", "callback_data": "pronunciation:" + telegramCompactUUID(translationID)}},
		},
	}

	rec := telegramUpdate(t, update)
	testkit.RequireStatus(t, rec, http.StatusOK)
	require.Len(t, tg.RequestsFor("sendAudio"), 1)
	require.Len(t, tg.RequestsFor("editMessageReplyMarkup"), 1)

	var edited telegramKeyboardRequest
	require.NoError(t, json.Unmarshal(tg.RequestsFor("editMessageReplyMarkup")[0].Body, &edited))
	require.Len(t, edited.ReplyMarkup.InlineKeyboard, 1)
	require.Len(t, edited.ReplyMarkup.InlineKeyboard[0], 1)
	assert.Equal(t, "Add to vocabulary", edited.ReplyMarkup.InlineKeyboard[0][0].Text)
	assert.Equal(t, "vocabulary:add:"+translationID.String(), edited.ReplyMarkup.InlineKeyboard[0][0].CallbackData)
	_, pronunciationStillPresent := findCallbackButton(edited.ReplyMarkup.InlineKeyboard, "pronunciation:")
	assert.False(t, pronunciationStillPresent)
}

func TestTelegramPronunciationStaleFileIDReuploadsWithoutTTS(t *testing.T) {
	testkit.Truncate(t)
	tg := testkit.MockTelegramAPI(t)
	tg.FailNext("sendAudio")
	translationID, target := seedPronunciationTranslation(t)
	audio := []byte("canonical-mp3")
	pronunciation, err := services.StoreWordPronunciation(target.ID, config.GetOpenRouterTTSModel(), config.GetOpenRouterTTSVoice(), audio)
	require.NoError(t, err)
	createdAt := pronunciation.CreatedAt
	require.NoError(t, services.SetWordPronunciationTelegramFileID(pronunciation.ID, "stale-file-id"))
	generated := 0
	testkit.MockOpenRouterSpeech(t, &testkit.FakeOpenRouterSpeech{GenerateFunc: func(string) ([]byte, error) {
		generated++
		return nil, errors.New("unexpected TTS")
	}})

	rec := telegramUpdate(t, pronunciationCallbackUpdate(710004, translationID))
	testkit.RequireStatus(t, rec, http.StatusOK)
	require.Zero(t, generated)
	requests := tg.RequestsFor("sendAudio")
	require.Len(t, requests, 2)
	var cachedRequest map[string]any
	require.NoError(t, json.Unmarshal(requests[0].Body, &cachedRequest))
	assert.Equal(t, "stale-file-id", cachedRequest["audio"])
	assert.Equal(t, audio, uploadedAudio(t, requests[1]))

	var stored models.WordPronunciation
	require.NoError(t, db.DB.Where("id = ?", pronunciation.ID).First(&stored).Error)
	require.NotNil(t, stored.TelegramFileID)
	assert.Equal(t, "test-telegram-audio-file-id", *stored.TelegramFileID)
	assert.WithinDuration(t, createdAt, stored.CreatedAt, time.Microsecond)
}

func TestTelegramPronunciationCacheIsSharedAcrossUsers(t *testing.T) {
	testkit.Truncate(t)
	tg := testkit.MockTelegramAPI(t)
	translationID, _ := seedPronunciationTranslation(t)
	generated := 0
	testkit.MockOpenRouterSpeech(t, &testkit.FakeOpenRouterSpeech{GenerateFunc: func(string) ([]byte, error) {
		generated++
		return []byte("shared-mp3"), nil
	}})

	first := telegramUpdate(t, pronunciationCallbackUpdate(710005, translationID))
	second := telegramUpdate(t, pronunciationCallbackUpdate(710006, translationID))
	testkit.RequireStatus(t, first, http.StatusOK)
	testkit.RequireStatus(t, second, http.StatusOK)
	assert.Equal(t, 1, generated)
	require.Len(t, tg.RequestsFor("sendAudio"), 2)
	assert.Contains(t, tg.RequestsFor("sendAudio")[0].Header.Get("Content-Type"), "multipart/form-data")
	assert.Equal(t, "application/json", tg.RequestsFor("sendAudio")[1].Header.Get("Content-Type"))
}

func TestTelegramPronunciationFailuresStaySilentAndReturnOK(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(t *testing.T, tg *testkit.FakeTelegramServer)
	}{
		{
			name: "provider failure",
			prepare: func(t *testing.T, _ *testkit.FakeTelegramServer) {
				testkit.MockOpenRouterSpeech(t, &testkit.FakeOpenRouterSpeech{GenerateFunc: func(string) ([]byte, error) {
					return nil, openrouter.ErrNotConfigured
				}})
			},
		},
		{
			name: "telegram failure",
			prepare: func(t *testing.T, tg *testkit.FakeTelegramServer) {
				tg.Fail("sendAudio")
				testkit.MockOpenRouterSpeech(t, nil)
			},
		},
		{
			name: "callback answer failure",
			prepare: func(t *testing.T, tg *testkit.FakeTelegramServer) {
				tg.Fail("answerCallbackQuery")
				testkit.MockOpenRouterSpeech(t, nil)
			},
		},
	}

	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testkit.Truncate(t)
			tg := testkit.MockTelegramAPI(t)
			translationID, _ := seedPronunciationTranslation(t)
			test.prepare(t, tg)

			rec := telegramUpdate(t, pronunciationCallbackUpdate(int64(720000+index), translationID))
			testkit.RequireStatus(t, rec, http.StatusOK)
			assert.Empty(t, tg.RequestsFor("sendMessage"), "pronunciation errors must not send warnings")
		})
	}
}

func TestTelegramVocabularyDeleteRetainsPronunciationButton(t *testing.T) {
	testkit.Truncate(t)
	tg := testkit.MockTelegramAPI(t)
	const telegramID int64 = 730001
	user := testkit.CreateUser(t, testkit.WithTelegramID(telegramID), testkit.WithSettings(models.UserSettings{SystemLanguage: enums.LanguageEn}))
	translationID := vocabSeedTranslation(t, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	vocabulary, err := services.CreateVocabularyByTranslation(user.ID, translationID)
	require.NoError(t, err)

	update := map[string]any{
		"update_id": 1,
		"callback_query": map[string]any{
			"id": "delete-vocabulary", "data": "vocabulary:delete:" + vocabulary.ID.String(),
			"from": map[string]any{"id": telegramID, "is_bot": false},
			"message": map[string]any{
				"message_id": 90, "text": "dog — Hund\n\nIt was added to your vocabulary",
				"chat": map[string]any{"id": telegramID, "type": "private"},
			},
		},
	}

	rec := telegramUpdate(t, update)
	testkit.RequireStatus(t, rec, http.StatusOK)
	require.Len(t, tg.RequestsFor("editMessageText"), 1)
	var edited telegramKeyboardRequest
	require.NoError(t, json.Unmarshal(tg.RequestsFor("editMessageText")[0].Body, &edited))
	button, ok := findCallbackButton(edited.ReplyMarkup.InlineKeyboard, "pronunciation:")
	require.True(t, ok)
	assert.Equal(t, "pronunciation:"+telegramCompactUUID(translationID), button.CallbackData)
}

func seedPronunciationTranslation(t *testing.T) (uuid.UUID, *models.Word) {
	t.Helper()
	translationID := vocabSeedTranslation(t, "dog", "Hund", enums.LanguageEn, enums.LanguageDe)
	target, err := services.GetTranslationTargetWord(translationID)
	require.NoError(t, err)
	return translationID, target
}

func pronunciationCallbackUpdate(telegramID int64, translationID uuid.UUID) map[string]any {
	return map[string]any{
		"update_id": 1,
		"callback_query": map[string]any{
			"id":   "pronunciation-callback",
			"data": "pronunciation:" + telegramCompactUUID(translationID),
			"from": map[string]any{"id": telegramID, "is_bot": false},
			"message": map[string]any{
				"message_id": 80,
				"chat":       map[string]any{"id": telegramID, "type": "private"},
			},
		},
	}
}

func loadPronunciation(t *testing.T, wordID uuid.UUID) models.WordPronunciation {
	t.Helper()
	var pronunciation models.WordPronunciation
	require.NoError(t, db.DB.Where("word_id = ?", wordID).First(&pronunciation).Error)
	return pronunciation
}

func uploadedAudio(t *testing.T, request testkit.TelegramRequest) []byte {
	t.Helper()
	mediaType, params, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	require.NoError(t, err)
	require.Equal(t, "multipart/form-data", mediaType)
	reader := multipart.NewReader(bytes.NewReader(request.Body), params["boundary"])
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		if part.FormName() == "audio" {
			body, err := io.ReadAll(part)
			require.NoError(t, err)
			assert.Equal(t, "pronunciation.mp3", part.FileName())
			assert.Equal(t, "audio/mpeg", part.Header.Get("Content-Type"))
			return body
		}
	}
	t.Fatal("audio multipart field not found")
	return nil
}

type telegramKeyboardRequest struct {
	ReplyMarkup struct {
		InlineKeyboard [][]telegramKeyboardButton `json:"inline_keyboard"`
	} `json:"reply_markup"`
}

type telegramKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

func findCallbackButton(keyboard [][]telegramKeyboardButton, prefix string) (telegramKeyboardButton, bool) {
	for _, row := range keyboard {
		for _, button := range row {
			if strings.HasPrefix(button.CallbackData, prefix) {
				return button, true
			}
		}
	}
	return telegramKeyboardButton{}, false
}
