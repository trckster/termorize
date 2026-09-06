package tests

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"termorize/src/config"
	"termorize/src/data/db"
	"termorize/src/enums"
	"termorize/src/integrations/openrouter"
	"termorize/src/models"
	"termorize/src/services"
	"termorize/src/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTelegramListenButtonsKeepOriginalKeyboardAndPlayTheirOwnLanguage(t *testing.T) {
	for _, test := range []struct {
		name, original, translated string
		source, target, system     enums.Language
		label                      string
		fallback, targetFirst      bool
	}{
		{name: "English to Spanish", original: "dog", translated: "perro", source: enums.LanguageEn, target: enums.LanguageEs, system: enums.LanguageEn, label: "Listen"},
		{name: "detected reverse direction and target first", original: "perro", translated: "dog", source: enums.LanguageEs, target: enums.LanguageEn, system: enums.LanguageEn, label: "Listen", targetFirst: true},
		{name: "language-specific fallback voices", original: "dog", translated: "perro", source: enums.LanguageEn, target: enums.LanguageEs, system: enums.LanguageEn, label: "Listen", fallback: true},
		{name: "same spelling and localized labels", original: "radio", translated: "radio", source: enums.LanguageEn, target: enums.LanguageEs, system: enums.LanguageRu, label: "Слушать"},
	} {
		t.Run(test.name, func(t *testing.T) {
			testkit.Truncate(t)
			tg := testkit.MockTelegramAPI(t)
			const telegramID int64 = 740001
			user := testkit.CreateUser(t, testkit.WithTelegramID(telegramID), testkit.WithSettings(models.UserSettings{
				SystemLanguage: test.system, TranslationSourceLanguage: enums.LanguageEn, TranslationTargetLanguage: enums.LanguageEs,
			}))
			testkit.MockGoogleTranslate(t, &testkit.FakeGoogleTranslate{
				DetectFunc: func(input string) (string, error) {
					assert.Equal(t, test.original, input)
					return string(test.source), nil
				},
				TranslateFunc: func(input, source, target string) (string, error) {
					assert.Equal(t, test.original, input)
					assert.Equal(t, string(test.source), source)
					assert.Equal(t, string(test.target), target)
					return test.translated, nil
				},
			})
			rec := telegramUpdate(t, telegramPrivateMessage(telegramID, test.original))
			testkit.RequireStatus(t, rec, http.StatusOK)
			require.Len(t, tg.RequestsFor("sendMessage"), 1)
			var sent telegramKeyboardRequest
			require.NoError(t, json.Unmarshal(tg.RequestsFor("sendMessage")[0].Body, &sent))
			keyboard := sent.ReplyMarkup.InlineKeyboard
			require.NotEmpty(t, keyboard)
			buttons := keyboard[len(keyboard)-1]
			require.Len(t, buttons, 2)
			assert.Equal(t, test.label+" "+test.source.Flag(), buttons[0].Text)
			assert.Equal(t, test.label+" "+test.target.Flag(), buttons[1].Text)
			var translation models.Translation
			require.NoError(t, db.DB.Preload("Original").Preload("Translation").First(&translation).Error)
			words := []*models.Word{translation.Original, translation.Translation}
			assert.Equal(t, test.original, words[0].Word)
			assert.Equal(t, test.translated, words[1].Word)
			assert.Equal(t, test.source, words[0].Language)
			assert.Equal(t, test.target, words[1].Language)
			if test.original != test.translated {
				assert.Equal(t, int64(1), vocabCountForUser(t, user.ID))
			}

			originalFactory := openrouter.NewSpeechClient
			t.Cleanup(func() { openrouter.NewSpeechClient = originalFactory })
			generated := 0
			var expectedWord *models.Word
			openrouter.NewSpeechClient = func(model, voice, format string) openrouter.SpeechClient {
				configs := config.GetOpenRouterTTSConfigs(string(expectedWord.Language))
				expectedConfig := configs[0]
				isFallback := model == configs[1].Model
				if isFallback {
					expectedConfig = configs[1]
				}
				assert.Equal(t, expectedConfig.Model, model)
				assert.Equal(t, expectedConfig.Voice, voice)
				assert.Equal(t, expectedConfig.ResponseFormat, format)
				return &testkit.FakeOpenRouterSpeech{GenerateFunc: func(input string) ([]byte, error) {
					generated++
					if isFallback {
						assert.Equal(t, expectedWord.Word, input)
					} else {
						assert.Equal(t, fmt.Sprintf("Synthesize speech in %s. Speak only the transcript exactly as written.\nTranscript: %q", expectedWord.Language.DisplayName(), expectedWord.Word), input)
						if test.fallback {
							return nil, errors.New("primary provider unavailable")
						}
					}
					return []byte(string(expectedWord.Language) + ":" + expectedWord.Word), nil
				}}
			}

			order := []int{0, 1}
			if test.targetFirst {
				order = []int{1, 0}
			}
			for press, i := range order {
				expectedWord = words[i]
				update := pronunciationCallbackUpdate(telegramID, translation.ID)
				callback := update["callback_query"].(map[string]any)
				callback["data"] = buttons[i].CallbackData
				callback["message"].(map[string]any)["reply_markup"] = map[string]any{"inline_keyboard": keyboard}
				rec := telegramUpdate(t, update)
				testkit.RequireStatus(t, rec, http.StatusOK)
				require.Len(t, tg.RequestsFor("sendAudio"), press+1)
				audioRequest := tg.RequestsFor("sendAudio")[press]
				assert.Equal(t, []byte(string(expectedWord.Language)+":"+expectedWord.Word), uploadedAudio(t, audioRequest))
				fields := uploadedAudioFields(t, audioRequest)
				assert.Equal(t, expectedWord.Word, fields["title"])
				assert.Equal(t, "740001", fields["chat_id"])
				stored := loadPronunciation(t, expectedWord.ID)
				configIndex := 0
				if test.fallback {
					configIndex = 1
				}
				expectedConfig := config.GetOpenRouterTTSConfigs(string(expectedWord.Language))[configIndex]
				assert.Equal(t, expectedConfig.Model, stored.Model)
				assert.Equal(t, expectedConfig.Voice, stored.Voice)
				require.NotNil(t, stored.TelegramFileID)
				require.NoError(t, services.SetWordPronunciationTelegramFileID(stored.ID, "cached-"+string(expectedWord.Language)))
				assert.Empty(t, tg.RequestsFor("editMessageReplyMarkup"), "both callbacks use the original keyboard, which playback must not edit")
				assert.Empty(t, tg.RequestsFor("editMessageText"))
			}
			generationCount := 2
			if test.fallback {
				generationCount = 4
			}
			require.Equal(t, generationCount, generated)
			for i, button := range buttons {
				expectedWord = words[i]
				update := pronunciationCallbackUpdate(telegramID, translation.ID)
				callback := update["callback_query"].(map[string]any)
				callback["data"] = button.CallbackData
				callback["message"].(map[string]any)["reply_markup"] = map[string]any{"inline_keyboard": keyboard}
				rec := telegramUpdate(t, update)
				testkit.RequireStatus(t, rec, http.StatusOK)
				require.Len(t, tg.RequestsFor("sendAudio"), 3+i)
				var audioRequest map[string]any
				require.NoError(t, json.Unmarshal(tg.RequestsFor("sendAudio")[2+i].Body, &audioRequest))
				assert.Equal(t, words[i].Word, audioRequest["title"])
				assert.Equal(t, "cached-"+string(words[i].Language), audioRequest["audio"])
			}
			assert.Equal(t, generationCount, generated, "both words reuse their cached audio")
			assert.Empty(t, tg.RequestsFor("editMessageReplyMarkup"), "cached playback must also keep both buttons available")
			assert.Empty(t, tg.RequestsFor("editMessageText"))
			require.Len(t, tg.RequestsFor("answerCallbackQuery"), 4)
			for _, request := range tg.RequestsFor("answerCallbackQuery") {
				var answer map[string]any
				require.NoError(t, json.Unmarshal(request.Body, &answer))
				assert.Equal(t, "pronunciation-callback", answer["callback_query_id"])
			}
		})
	}
}

func TestTelegramListenRejectsInvalidCallbacksWithoutPlayingAudio(t *testing.T) {
	for _, test := range []struct {
		name, payload string
	}{
		{name: "unknown side", payload: "%s:other"},
		{name: "empty side", payload: "%s:"},
		{name: "extra payload", payload: "%s:source:extra"},
		{name: "invalid ID", payload: "invalid:source"},
		{name: "missing translation", payload: "00000000-0000-0000-0000-000000000000:source"},
	} {
		t.Run(test.name, func(t *testing.T) {
			testkit.Truncate(t)
			tg := testkit.MockTelegramAPI(t)
			translationID, _ := seedPronunciationTranslation(t)
			generated := 0
			testkit.MockOpenRouterSpeech(t, &testkit.FakeOpenRouterSpeech{GenerateFunc: func(string) ([]byte, error) {
				generated++
				return []byte("unexpected"), nil
			}})
			update := pronunciationCallbackUpdate(740002, translationID)
			payload := strings.ReplaceAll(test.payload, "%s", telegramCompactUUID(translationID))
			update["callback_query"].(map[string]any)["data"] = "pronunciation:" + payload
			rec := telegramUpdate(t, update)
			testkit.RequireStatus(t, rec, http.StatusOK)
			assert.Zero(t, generated, "invalid callbacks must not call TTS")
			require.Len(t, tg.Requests(), 1)
			assert.Equal(t, "answerCallbackQuery", tg.Requests()[0].Action)
		})
	}
}
