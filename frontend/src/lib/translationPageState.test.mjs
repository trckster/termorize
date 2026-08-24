import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import {
    createProgrammaticChangeGuard,
    getLanguageChangeDirection,
    hasMeaningfulVocabularyEdits,
    isEditableVocabularyShortcut,
} from './translationPageState.ts'

describe('getLanguageChangeDirection', () => {
    it('updates the text on the side whose language changed', () => {
        assert.equal(getLanguageChangeDirection('source'), 'target-to-source')
        assert.equal(getLanguageChangeDirection('target'), 'source-to-target')
    })
})

describe('createProgrammaticChangeGuard', () => {
    it('distinguishes a translated result from a user edit', () => {
        const guard = createProgrammaticChangeGuard()

        guard.mark('target', 'ciao')

        assert.equal(guard.consume('target', 'ciao'), true)
        assert.equal(guard.consume('target', 'arrivederci'), false)
    })

    it('does not suppress a user edit that replaces a pending result', () => {
        const guard = createProgrammaticChangeGuard()

        guard.mark('target', 'ciao')

        assert.equal(guard.consume('target', 'user text'), false)
        assert.equal(guard.consume('target', 'ciao'), false)
    })

    it('tracks the two translation fields independently', () => {
        const guard = createProgrammaticChangeGuard()

        guard.mark('source', 'hello')
        guard.mark('target', 'ciao')

        assert.equal(guard.consume('target', 'ciao'), true)
        assert.equal(guard.consume('source', 'hello'), true)
    })
})

describe('hasMeaningfulVocabularyEdits', () => {
    const initialPair = {
        original: 'hello',
        translation: 'ciao',
    }

    it('keeps an unchanged translation on its existing source', () => {
        assert.equal(hasMeaningfulVocabularyEdits(initialPair, { ...initialPair }), false)
    })

    it('marks an edited translation as custom', () => {
        assert.equal(
            hasMeaningfulVocabularyEdits(initialPair, {
                ...initialPair,
                translation: 'salve',
            }),
            true
        )
    })

    it('marks an edited original as custom', () => {
        assert.equal(
            hasMeaningfulVocabularyEdits(initialPair, {
                ...initialPair,
                original: 'hi',
            }),
            true
        )
    })

    it('ignores whitespace removed before saving', () => {
        assert.equal(
            hasMeaningfulVocabularyEdits(initialPair, {
                original: ' hello ',
                translation: 'ciao\n',
            }),
            false
        )
    })
})

describe('isEditableVocabularyShortcut', () => {
    const shortcut = {
        altKey: false,
        code: 'KeyE',
        ctrlKey: true,
        metaKey: false,
        shiftKey: false,
    }

    it('matches Ctrl+E exactly', () => {
        assert.equal(isEditableVocabularyShortcut(shortcut), true)
    })

    it('does not intercept AltGr or modified Ctrl+E input', () => {
        assert.equal(isEditableVocabularyShortcut({ ...shortcut, altKey: true }), false)
        assert.equal(isEditableVocabularyShortcut({ ...shortcut, metaKey: true }), false)
        assert.equal(isEditableVocabularyShortcut({ ...shortcut, shiftKey: true }), false)
    })
})
