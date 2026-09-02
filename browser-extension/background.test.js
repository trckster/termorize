const assert = require('node:assert/strict')
const { describe, it } = require('node:test')
const {
    ME_ENDPOINT,
    TARGET_LANGUAGE_ENDPOINT,
    TRANSLATE_SELECTION_ENDPOINT,
    VOCABULARY_ENDPOINT,
    captureTabSelection,
    commandAction,
    getSession,
    handleCommand,
    handleContextMenu,
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

describe('handleContextMenu', () => {
    it('opens a top-frame overlay for text selected inside an inaccessible frame', async () => {
        const tab = { id: 41, url: 'https://example.com/with-embedded-content' }
        let received

        const handled = await handleContextMenu(
            { menuItemId: 'translate-selection', selectionText: ' iframe selection ' },
            tab,
            {},
            {
                showSelectionOverlay: async (...args) => {
                    received = args
                },
            }
        )

        assert.equal(handled, true)
        assert.deepEqual(received.slice(0, 3), [
            tab,
            { ok: true, text: 'iframe selection', rect: null, frameId: 0 },
            0,
        ])
    })
})

describe('captureTabSelection', () => {
    it('follows the active frame chain after the popup takes document focus', async () => {
        const response = await captureTabSelection(29, {
            scripting: {
                executeScript: async () => [
                    {
                        frameId: 0,
                        result: {
                            text: 'stale parent text',
                            rect: null,
                            focused: false,
                            focusDelegatedToFrame: true,
                            activeFrameChain: true,
                        },
                    },
                    {
                        frameId: 7,
                        result: {
                            text: 'current child text',
                            rect: { left: 1, top: 2, right: 3, bottom: 4 },
                            focused: false,
                            focusDelegatedToFrame: false,
                            activeFrameChain: true,
                        },
                    },
                ],
            },
        })

        assert.deepEqual(response, {
            ok: true,
            text: 'current child text',
            rect: { left: 1, top: 2, right: 3, bottom: 4 },
            frameId: 7,
        })
    })

    it('keeps the current top-frame selection ahead of a stale child selection', async () => {
        const response = await captureTabSelection(31, {
            scripting: {
                executeScript: async () => [
                    {
                        frameId: 0,
                        result: {
                            text: 'current parent text',
                            rect: null,
                            focused: false,
                            focusDelegatedToFrame: false,
                            activeFrameChain: true,
                        },
                    },
                    {
                        frameId: 9,
                        result: {
                            text: 'stale child text',
                            rect: null,
                            focused: false,
                            focusDelegatedToFrame: false,
                            activeFrameChain: false,
                        },
                    },
                ],
            },
        })

        assert.equal(response.text, 'current parent text')
        assert.equal(response.frameId, 0)
    })

    it('does not reuse stale parent text when the active child selection is empty', async () => {
        const response = await captureTabSelection(37, {
            scripting: {
                executeScript: async () => [
                    {
                        frameId: 0,
                        result: {
                            text: 'stale parent text',
                            rect: null,
                            focused: false,
                            focusDelegatedToFrame: true,
                            activeFrameChain: true,
                        },
                    },
                    {
                        frameId: 12,
                        result: {
                            text: '',
                            rect: null,
                            focused: false,
                            focusDelegatedToFrame: false,
                            activeFrameChain: true,
                        },
                    },
                ],
            },
        })

        assert.deepEqual(response, { ok: false, reason: 'empty' })
    })

    it('directs inaccessible active-frame selections to the context-menu fallback', async () => {
        const response = await captureTabSelection(43, {
            scripting: {
                executeScript: async () => [
                    {
                        frameId: 0,
                        result: {
                            text: 'stale parent text',
                            rect: null,
                            focused: false,
                            focusDelegatedToFrame: true,
                            activeFrameChain: true,
                        },
                    },
                ],
            },
        })

        assert.deepEqual(response, { ok: false, reason: 'frame-unavailable' })
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
    it('patches only the requested account setting', async () => {
        let request
        const response = await updateTargetLanguage('de', {
            chromeApi: chromeWithCookie({ value: 'session-jwt' }),
            fetchApi: async (url, options) => {
                request = { url, options }
                return {
                    ok: true,
                    status: 200,
                    json: async () => ({
                        settings: { translation_target_language: 'de' },
                    }),
                }
            },
        })

        assert.equal(request.url, TARGET_LANGUAGE_ENDPOINT)
        assert.equal(request.options.method, 'PATCH')
        assert.deepEqual(JSON.parse(request.options.body), { translation_target_language: 'de' })
        assert.equal(response.settings.translation_target_language, 'de')
    })

    it('serializes rapid changes so the last choice is persisted last', async () => {
        const requests = []
        let releaseFirst
        const firstResponse = new Promise((resolve) => {
            releaseFirst = resolve
        })
        const dependencies = {
            chromeApi: chromeWithCookie({ value: 'session-jwt' }),
            fetchApi: async (url, options) => {
                requests.push({ url, options })
                if (requests.length === 1) return firstResponse
                return {
                    ok: true,
                    status: 200,
                    json: async () => ({ settings: { translation_target_language: 'it' } }),
                }
            },
        }

        const first = updateTargetLanguage('de', dependencies)
        const second = updateTargetLanguage('it', dependencies)
        await new Promise((resolve) => setImmediate(resolve))

        assert.equal(requests.length, 1)
        releaseFirst({
            ok: true,
            status: 200,
            json: async () => ({ settings: { translation_target_language: 'de' } }),
        })
        await Promise.all([first, second])

        assert.deepEqual(
            requests.map((request) => JSON.parse(request.options.body).translation_target_language),
            ['de', 'it']
        )
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
