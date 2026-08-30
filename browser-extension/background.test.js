const assert = require('node:assert/strict')
const { describe, it } = require('node:test')
const { VOCABULARY_ENDPOINT, commandAction, isValidVocabularyPayload, saveVocabulary } = require('./background.js')

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
    it('maps both commands only on the Google Translate HTTPS origin', () => {
        assert.equal(commandAction('save-with-editing', 'https://translate.google.com/?sl=en&tl=it'), 'edit')
        assert.equal(commandAction('save-without-editing', 'https://translate.google.com/'), 'save')
        assert.equal(commandAction('save-with-editing', 'https://example.com/'), null)
        assert.equal(commandAction('unknown', 'https://translate.google.com/'), null)
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
