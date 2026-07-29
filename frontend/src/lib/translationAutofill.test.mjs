import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import { createTranslationAutofill } from './translationAutofill.ts'

const request = {
    text: ' hello ',
    sourceLanguage: 'en',
    targetLanguage: 'it',
}

const waitForTimer = () => new Promise((resolve) => setTimeout(resolve, 10))

const deferred = () => {
    let resolve
    const promise = new Promise((resolvePromise) => {
        resolve = resolvePromise
    })
    return { promise, resolve }
}

describe('createTranslationAutofill', () => {
    it('debounces the request and fills the suggestion using the selected languages', async () => {
        const requests = []
        const suggestions = []
        const loadingChanges = []
        const autofill = createTranslationAutofill({
            debounceMs: 0,
            translate: async (translationRequest) => {
                requests.push(translationRequest)
                return 'ciao'
            },
            onSuggestion: (suggestion) => suggestions.push(suggestion),
            onLoadingChange: (isLoading) => loadingChanges.push(isLoading),
        })

        autofill.activate()
        autofill.queue(request)

        assert.deepEqual(suggestions, [''])
        assert.deepEqual(requests, [])

        await waitForTimer()

        assert.deepEqual(requests, [
            {
                text: 'hello',
                sourceLanguage: 'en',
                targetLanguage: 'it',
            },
        ])
        assert.deepEqual(suggestions, ['', 'ciao'])
        assert.deepEqual(loadingChanges, [true, false])
    })

    it('stops making suggestions after the target field is edited', async () => {
        let translateCalls = 0
        const suggestions = []
        const autofill = createTranslationAutofill({
            debounceMs: 0,
            translate: async () => {
                translateCalls += 1
                return 'ciao'
            },
            onSuggestion: (suggestion) => suggestions.push(suggestion),
        })

        autofill.activate()
        autofill.queue(request)
        autofill.markTargetEdited()
        autofill.queue({ ...request, text: 'goodbye' })
        await waitForTimer()

        assert.equal(translateCalls, 0)
        assert.deepEqual(suggestions, [''])
    })

    it('does not overwrite a user edit with an in-flight response', async () => {
        const pendingTranslation = deferred()
        const suggestions = []
        const loadingChanges = []
        const autofill = createTranslationAutofill({
            debounceMs: 0,
            translate: () => pendingTranslation.promise,
            onSuggestion: (suggestion) => suggestions.push(suggestion),
            onLoadingChange: (isLoading) => loadingChanges.push(isLoading),
        })

        autofill.activate()
        autofill.queue(request)
        await waitForTimer()

        autofill.markTargetEdited()
        pendingTranslation.resolve('ciao')
        await Promise.resolve()

        assert.deepEqual(suggestions, [''])
        assert.deepEqual(loadingChanges, [true, false])
    })

    it('discards stale responses when the source text changes', async () => {
        const firstTranslation = deferred()
        let translateCalls = 0
        const suggestions = []
        const autofill = createTranslationAutofill({
            debounceMs: 0,
            translate: () => {
                translateCalls += 1
                return translateCalls === 1 ? firstTranslation.promise : Promise.resolve('arrivederci')
            },
            onSuggestion: (suggestion) => suggestions.push(suggestion),
        })

        autofill.activate()
        autofill.queue(request)
        await waitForTimer()

        autofill.queue({ ...request, text: 'goodbye' })
        firstTranslation.resolve('ciao')
        await waitForTimer()

        assert.deepEqual(suggestions, ['', '', 'arrivederci'])
    })

    it('allows suggestions again for a fresh form entry', async () => {
        let translateCalls = 0
        const suggestions = []
        const autofill = createTranslationAutofill({
            debounceMs: 0,
            translate: async () => {
                translateCalls += 1
                return 'ciao'
            },
            onSuggestion: (suggestion) => suggestions.push(suggestion),
        })

        autofill.activate()
        autofill.markTargetEdited()
        autofill.queue(request)
        autofill.activate()
        autofill.queue(request)
        await waitForTimer()

        assert.equal(translateCalls, 1)
        assert.equal(suggestions.at(-1), 'ciao')
    })
})
