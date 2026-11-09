import assert from 'node:assert/strict'
import { test } from 'node:test'
import { rememberPostAuthPath, takePostAuthPath, clearPostAuthPath } from './postAuthRedirect.ts'

test('login returns to shared words and existing collection links', () => {
    const storage = new Map()
    globalThis.window = {
        location: { origin: 'https://termorize.example' },
        sessionStorage: {
            getItem: (key) => storage.get(key),
            setItem: (key, value) => storage.set(key, value),
            removeItem: (key) => storage.delete(key),
        },
    }
    for (const path of ['/words/token', '/collections/join/example?source=friend']) {
        rememberPostAuthPath(path)
        assert.equal(takePostAuthPath(), path)
        assert.equal(takePostAuthPath(), null)
    }
    for (const path of [
        'https://evil.example/words/token',
        '//evil.example/words/token',
        '/words/../../admin',
        '/login',
    ]) {
        clearPostAuthPath()
        rememberPostAuthPath(path)
        assert.equal(takePostAuthPath(), null)
    }
    delete globalThis.window
})
