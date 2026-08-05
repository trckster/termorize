import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import { createProgrammaticChangeGuard, getLanguageChangeDirection } from './translationPageState.ts'

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
