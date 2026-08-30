const assert = require('node:assert/strict')
const { describe, it } = require('node:test')
const { extractGoogleTranslation, shortcutAction, validatePair } = require('./content.js')

function element({ value = '', textContent = '', lang = '' } = {}) {
    return {
        value,
        textContent,
        getAttribute(name) {
            return name === 'lang' ? lang : null
        },
    }
}

function fixture({ selectors = {}, selectorLists = {} }) {
    return {
        querySelector(selector) {
            if (selector.includes(',')) {
                for (const part of selector.split(',').map((value) => value.trim())) {
                    if (selectors[part]) return selectors[part]
                }
            }
            return selectors[selector] || null
        },
        querySelectorAll(selector) {
            return selectorLists[selector] || []
        },
    }
}

describe('extractGoogleTranslation', () => {
    it('reads the current semantic Google Translate fields and URL languages', () => {
        const documentApi = fixture({
            selectors: {
                'textarea[aria-label="Source text"]': element({ value: ' hello ' }),
                'textarea[jsname="YPqjbf"][lang]': element({ value: ' ciao ', lang: 'it' }),
            },
        })

        assert.deepEqual(
            extractGoogleTranslation(documentApi, 'https://translate.google.com/?sl=en&tl=it&text=hello'),
            {
                original: 'hello',
                translation: 'ciao',
                original_language: 'en',
                translation_language: 'it',
            }
        )
    })

    it('falls back to result spans and the result language attribute', () => {
        const result = element({ lang: 'de' })
        const documentApi = fixture({
            selectors: {
                'textarea[jsname="BJE2fc"]': element({ value: 'good morning' }),
                'textarea[jsname="YPqjbf"][lang]': result,
            },
            selectorLists: {
                '[data-language-for-alternatives] span.ryNqvb': [element({ textContent: 'guten' }), element({ textContent: 'Morgen' })],
            },
        })

        assert.deepEqual(extractGoogleTranslation(documentApi, 'https://translate.google.com/?sl=en'), {
            original: 'good morning',
            translation: 'guten Morgen',
            original_language: 'en',
            translation_language: 'de',
        })
    })
})

describe('shortcutAction', () => {
    const event = {
        isTrusted: true,
        altKey: false,
        ctrlKey: true,
        metaKey: false,
        shiftKey: false,
        code: 'KeyE',
    }

    it('maps Ctrl+E to editing and Ctrl+S to direct save', () => {
        assert.equal(shortcutAction(event), 'edit')
        assert.equal(shortcutAction({ ...event, code: 'KeyS' }), 'save')
    })

    it('ignores synthetic and additionally modified shortcuts', () => {
        assert.equal(shortcutAction({ ...event, isTrusted: false }), null)
        assert.equal(shortcutAction({ ...event, altKey: true }), null)
        assert.equal(shortcutAction({ ...event, shiftKey: true }), null)
    })
})

describe('validatePair', () => {
    const validPair = {
        original: 'hello',
        translation: 'ciao',
        original_language: 'en',
        translation_language: 'it',
    }

    it('accepts a complete supported language pair', () => {
        assert.deepEqual(validatePair(validPair), { ok: true })
    })

    it('requires an explicit source language and a result', () => {
        assert.equal(validatePair({ ...validPair, original_language: 'auto' }).ok, false)
        assert.equal(validatePair({ ...validPair, translation: '' }).ok, false)
    })

    it('rejects unsupported or identical languages', () => {
        assert.equal(validatePair({ ...validPair, translation_language: 'ja' }).ok, false)
        assert.equal(validatePair({ ...validPair, translation_language: 'en' }).ok, false)
    })
})
