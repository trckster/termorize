const assert = require('node:assert/strict')
const { describe, it } = require('node:test')
const {
    ME_ENDPOINT,
    SETTINGS_ENDPOINT,
    TRANSLATE_SELECTION_ENDPOINT,
    VOCABULARY_ENDPOINT,
    commandAction,
    getSession,
    handleCommand,
    isValidVocabularyPayload,
    saveSelection,
    saveVocabulary,
    translateSelectedText,
    updateTargetLanguage,
} = require('./background.js')

const validPayload = {
    original: 'hello',
    translation: 'ciao',
    original_language: 'en',
    translation_language: 'it',
}

function chromeWithCookie(cookie) {
    return {
        cookies: {
            get(_details, callback) {
                callback(cookie)
            },
        },
        runtime: { lastError: null },
    }
}

describe('isValidVocabularyPayload', () => {
    it('accepts supported, distinct languages and bounded text', () => {
        assert.equal(isValidVocabularyPayload(validPayload), true)
    })

    it('rejects unsupported languages and oversized content', () => {
        assert.equal(isValidVocabularyPayload({ ...validPayload, translation_language: 'ja' }), false)
        assert.equal(isValidVocabularyPayload({ ...validPayload, original: 'x'.repeat(5001) }), false)
    })
})

describe('commandAction', () => {
    it('maps Google Translate save commands only on its HTTPS origin', () => {
        assert.equal(commandAction('save-with-editing', 'https://translate.google.com/?sl=en&tl=it'), 'edit')
        assert.equal(commandAction('save-without-editing', 'https://translate.google.com/'), 'save')
        assert.equal(commandAction('save-with-editing', 'https://example.com/'), null)
        assert.equal(commandAction('unknown', 'https://translate.google.com/'), null)
    })

    it('maps selected-text translation on any normal page', () => {
        assert.equal(commandAction('translate-selection', 'https://example.com/article'), 'translate')
        assert.equal(commandAction('translate-selection', undefined), 'translate')
    })
})

describe('handleCommand', () => {
    it('opens selected-text translation for the tab supplied by the browser', async () => {
        const tab = { id: 17, url: 'https://example.com/article' }
        let openedTab
        const chromeApi = { tabs: {} }

        const handled = await handleCommand('translate-selection', tab, chromeApi, {
            openSelectionOverlay: async (receivedTab) => {
                openedTab = receivedTab
            },
        })

        assert.equal(handled, true)
        assert.equal(openedTab, tab)
    })

    it('falls back to the active tab when the browser omits the command tab', async () => {
        const tab = { id: 23, url: 'https://example.com/selection' }
        let openedTab
        const chromeApi = {
            tabs: {
                query: async (query) => {
                    assert.deepEqual(query, { active: true, currentWindow: true })
                    return [tab]
                },
            },
        }

        const handled = await handleCommand('translate-selection', undefined, chromeApi, {
            openSelectionOverlay: async (receivedTab) => {
                openedTab = receivedTab
            },
        })

        assert.equal(handled, true)
        assert.equal(openedTab, tab)
    })

    it('ignores commands when no accessible active tab exists', async () => {
        const handled = await handleCommand(
            'translate-selection',
            undefined,
            { tabs: { query: async () => [] } },
            {
                openSelectionOverlay: async () => {
                    assert.fail('the overlay should not open')
                },
            }
        )

        assert.equal(handled, false)
    })
})

describe('saveVocabulary', () => {
    it('returns unauthorized before fetching when the website session is absent', async () => {
        let fetched = false
        const response = await saveVocabulary(validPayload, {
            chromeApi: chromeWithCookie(null),
            fetchApi: async () => {
                fetched = true
            },
        })

        assert.deepEqual(response, { ok: false, reason: 'unauthorized' })
        assert.equal(fetched, false)
    })

    it('posts the pair using a Bearer copy of the existing session', async () => {
        let request
        const response = await saveVocabulary(validPayload, {
            chromeApi: chromeWithCookie({ value: 'session-jwt' }),
            fetchApi: async (url, options) => {
                request = { url, options }
                return { ok: true, status: 201 }
            },
        })

        assert.deepEqual(response, { ok: true })
        assert.equal(request.url, VOCABULARY_ENDPOINT)
        assert.equal(request.options.headers.Authorization, 'Bearer session-jwt')
        assert.deepEqual(JSON.parse(request.options.body), validPayload)
    })

    it('maps duplicate and expired-session responses', async () => {
        const duplicate = await saveVocabulary(validPayload, {
            chromeApi: chromeWithCookie({ value: 'session-jwt' }),
            fetchApi: async () => ({ ok: false, status: 409 }),
        })
        const unauthorized = await saveVocabulary(validPayload, {
            chromeApi: chromeWithCookie({ value: 'session-jwt' }),
            fetchApi: async () => ({ ok: false, status: 401 }),
        })

        assert.deepEqual(duplicate, { ok: false, reason: 'duplicate' })
        assert.deepEqual(unauthorized, { ok: false, reason: 'unauthorized' })
    })
})

describe('getSession', () => {
    it('returns the account language settings without exposing the session token', async () => {
        let request
        const response = await getSession({
            chromeApi: chromeWithCookie({ value: 'session-jwt' }),
            fetchApi: async (url, options) => {
                request = { url, options }
                return {
                    ok: true,
                    status: 200,
                    json: async () => ({
                        name: 'Daniil',
                        settings: { translation_target_language: 'it' },
                    }),
                }
            },
        })

        assert.equal(request.url, ME_ENDPOINT)
        assert.equal(request.options.headers.Authorization, 'Bearer session-jwt')
        assert.deepEqual(response.user, {
            name: 'Daniil',
            settings: { translation_target_language: 'it' },
        })
        assert.equal(JSON.stringify(response).includes('session-jwt'), false)
    })
})

describe('translateSelectedText', () => {
    it('uses automatic source detection through the selection endpoint', async () => {
        let request
        const response = await translateSelectedText(' buongiorno ', 'en', {
            chromeApi: chromeWithCookie({ value: 'session-jwt' }),
            fetchApi: async (url, options) => {
                request = { url, options }
                return {
                    ok: true,
                    status: 200,
                    json: async () => ({
                        id: 'translation-id',
                        original_language: 'it',
                        translation: 'good morning',
                        source: 'google',
                    }),
                }
            },
        })

        assert.equal(request.url, TRANSLATE_SELECTION_ENDPOINT)
        assert.deepEqual(JSON.parse(request.options.body), {
            from_word: 'buongiorno',
            to_language: 'en',
        })
        assert.deepEqual(response.translation, {
            id: 'translation-id',
            original: 'buongiorno',
            translated: 'good morning',
            originalLanguage: 'it',
            targetLanguage: 'en',
            source: 'google',
        })
    })

    it('maps same-language and unsupported-language responses', async () => {
        const sameLanguage = await translateSelectedText('ciao', 'it', {
            chromeApi: chromeWithCookie({ value: 'session-jwt' }),
            fetchApi: async () => ({
                ok: false,
                status: 422,
                json: async () => ({
                    error: 'source language matches target language',
                    detected_language: 'it',
                }),
            }),
        })
        const unsupported = await translateSelectedText('おはよう', 'en', {
            chromeApi: chromeWithCookie({ value: 'session-jwt' }),
            fetchApi: async () => ({
                ok: false,
                status: 422,
                json: async () => ({
                    error: 'unsupported source language',
                    detected_language: 'ja',
                }),
            }),
        })

        assert.deepEqual(sameLanguage, {
            ok: false,
            reason: 'same-language',
            detectedLanguage: 'it',
        })
        assert.deepEqual(unsupported, {
            ok: false,
            reason: 'unsupported-language',
            detectedLanguage: 'ja',
        })
    })
})

describe('updateTargetLanguage', () => {
    it('preserves the rest of the account settings', async () => {
        const requests = []
        const response = await updateTargetLanguage('de', {
            chromeApi: chromeWithCookie({ value: 'session-jwt' }),
            fetchApi: async (url, options) => {
                requests.push({ url, options })
                if (url === ME_ENDPOINT) {
                    return {
                        ok: true,
                        status: 200,
                        json: async () => ({
                            name: 'Daniil',
                            settings: {
                                system_language: 'en',
                                main_learning_language: 'it',
                                translation_source_language: 'en',
                                translation_target_language: 'ru',
                                ignored_audio_languages: [],
                                time_zone: 'Europe/Rome',
                                telegram: {
                                    daily_questions_enabled: false,
                                    daily_questions_count: 10,
                                    daily_questions_schedule: [],
                                },
                            },
                        }),
                    }
                }
                return {
                    ok: true,
                    status: 200,
                    json: async () => ({
                        settings: { translation_target_language: 'de' },
                    }),
                }
            },
        })

        assert.equal(requests[1].url, SETTINGS_ENDPOINT)
        const payload = JSON.parse(requests[1].options.body)
        assert.equal(payload.system_language, 'en')
        assert.equal(payload.main_learning_language, 'it')
        assert.equal(payload.translation_target_language, 'de')
        assert.equal(response.settings.translation_target_language, 'de')
    })

    it('swaps the source to the previous target when the new target would match it', async () => {
        const requests = []
        const response = await updateTargetLanguage('en', {
            chromeApi: chromeWithCookie({ value: 'session-jwt' }),
            fetchApi: async (url, options) => {
                requests.push({ url, options })
                if (url === ME_ENDPOINT) {
                    return {
                        ok: true,
                        status: 200,
                        json: async () => ({
                            name: 'Daniil',
                            settings: {
                                system_language: 'en',
                                main_learning_language: 'it',
                                translation_source_language: 'en',
                                translation_target_language: 'ru',
                                ignored_audio_languages: [],
                                time_zone: 'Europe/Rome',
                                telegram: {
                                    daily_questions_enabled: false,
                                    daily_questions_count: 10,
                                    daily_questions_schedule: [],
                                },
                            },
                        }),
                    }
                }
                return { ok: true, status: 200, json: async () => null }
            },
        })

        const payload = JSON.parse(requests[1].options.body)
        assert.equal(payload.translation_source_language, 'ru')
        assert.equal(payload.translation_target_language, 'en')
        assert.equal(response.settings.translation_source_language, 'ru')
        assert.equal(response.settings.translation_target_language, 'en')
    })
})

describe('saveSelection', () => {
    it('keeps an unchanged API translation and creates a custom pair after editing', async () => {
        const urls = []
        const dependencies = {
            chromeApi: chromeWithCookie({ value: 'session-jwt' }),
            fetchApi: async (url) => {
                urls.push(url)
                return { ok: true, status: 201 }
            },
        }

        await saveSelection({ translation_id: 'translation-id', edited: false }, dependencies)
        await saveSelection({ ...validPayload, translation_id: 'translation-id', edited: true }, dependencies)

        assert.deepEqual(urls, [`${VOCABULARY_ENDPOINT}/translation`, VOCABULARY_ENDPOINT])
    })
})
