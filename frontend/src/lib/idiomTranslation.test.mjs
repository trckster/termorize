import { test } from 'node:test'
import assert from 'node:assert/strict'
import { idiomLanguages, matchesIdiom } from './idiomTranslation.ts'

test('daily idiom keeps the previous target only when its source language matches', () => {
    assert.deepEqual(idiomLanguages('en', 'en', 'ru'), { source: 'en', target: 'ru' })
    assert.deepEqual(idiomLanguages('it', 'en', 'ru'), { source: 'it', target: 'en' })
    assert.deepEqual(idiomLanguages('ru', 'en', 'ru'), { source: 'ru', target: 'en' })
})

test('idiom translation mode follows the original text and language', () => {
    const idiom = { id: 'daily-id', word: 'break the ice', language: 'en' }
    assert.ok(matchesIdiom(idiom, 'break the ice', 'en'))
    assert.ok(!matchesIdiom(idiom, 'break the ice', 'it'))
    assert.ok(!matchesIdiom(idiom, 'new text', 'en'))
})
