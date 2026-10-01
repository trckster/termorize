import { test, expect, type Page } from '@playwright/test'
import type { PartOfSpeech } from '../src/lib/partOfSpeech'

type Word = { id: string; word: string; language: string; part_of_speech: PartOfSpeech | null }
const word = (id: string, text: string, category: PartOfSpeech | null, language = 'en'): Word => ({
    id,
    word: text,
    language,
    part_of_speech: category,
})
const pair = (id: string, original: Word, translation: Word) => ({
    id,
    translation: { id: `translation-${id}`, original, translation, source: 'google' },
    progress: [],
    created_at: '2026-09-27T12:00:00Z',
    mastered_at: null,
})

async function setup(page: Page, locale = 'en', admin = true) {
    const shared = word('shared', 'sourire', 'unknown', 'fr')
    const noun = word('noun', 'smile', 'noun')
    const unknowns = [
        shared,
        ...Array.from({ length: 20 }, (_, i) =>
            word(`review-${i}`, `Review word ${i + 1}`, i === 19 ? null : 'unknown')
        ),
    ]
    const mismatches = [
        pair('record-one', shared, noun),
        pair('record-two', shared, noun),
        pair('record-three', word('noun-two', 'decision', 'noun'), word('verb', 'decidere', 'verb', 'it')),
    ]
    const vocabulary = [
        pair('same', word('a', 'book', 'noun'), word('b', 'libro', 'noun', 'it')),
        pair('different', word('c', 'decision', 'noun'), word('d', 'decidere', 'verb', 'it')),
        pair('pending', word('e', 'new word', null), word('f', 'parola nuova', null, 'it')),
        pair('unknown', word('g', 'gibberish', 'unknown'), word('h', 'nonsense', 'unknown')),
        pair('pending-unknown', word('i', 'unprocessed', null), word('j', 'uncertain', 'unknown')),
    ]
    const calls: { path: string; method: string; body: any }[] = []
    const allWords = Array.from(
        new Map(
            [...unknowns, ...mismatches.flatMap((p) => [p.translation.original, p.translation.translation])].map(
                (w) => [w.id, w]
            )
        ).values()
    )
    const state = {
        failList: false,
        failStats: false,
        failSave: false,
        restartStatus: 202,
        worker: { active: false, queued: 0, processed: 0, failed: 0, scan_failed: false },
    }
    await page.route('http://127.0.0.1:4173/api/**', async (route) => {
        const url = new URL(route.request().url())
        const path = url.pathname
        const method = route.request().method()
        const body = route.request().postDataJSON()
        calls.push({ path, method, body })
        const paginated = (data: unknown[]) => {
            const current = Number(url.searchParams.get('page') || 1)
            return {
                data: data.slice((current - 1) * 20, current * 20),
                pagination: {
                    page: current,
                    page_size: 20,
                    total: data.length,
                    total_pages: Math.ceil(data.length / 20),
                },
            }
        }
        let json: unknown = {}
        if (path === '/api/me')
            json = {
                id: 1,
                name: 'Test Admin',
                username: 'reviewer',
                is_admin: admin,
                guest_expires_at: null,
                settings: {
                    system_language: locale,
                    main_learning_language: 'en',
                    translation_source_language: 'en',
                    translation_target_language: 'it',
                    time_zone: 'UTC',
                    ignored_audio_languages: [],
                    ignored_description_languages: [],
                    telegram: {
                        bot_enabled: false,
                        daily_idiom_enabled: false,
                        daily_questions_enabled: false,
                        daily_questions_count: 10,
                        daily_questions_schedule: [],
                    },
                },
            }
        else if (path === '/api/settings') json = { languages: ['en', 'it', 'fr', 'ru'] }
        else if (path === '/api/vocabulary') json = paginated(vocabulary)
        else if (path === '/api/admin/categorization/stats') {
            if (state.failStats) return route.fulfill({ status: 503, json: { error: 'unavailable' } })
            json = {
                total: allWords.length,
                pending: allWords.filter((w) => w.part_of_speech === null).length,
                unknown: allWords.filter((w) => w.part_of_speech === 'unknown').length,
                categorized: allWords.filter((w) => w.part_of_speech !== null && w.part_of_speech !== 'unknown').length,
                worker: state.worker,
            }
        } else if (path === '/api/admin/categorization/restart') {
            if (state.restartStatus !== 202)
                return route.fulfill({ status: state.restartStatus, json: { error: 'cannot start' } })
            allWords
                .filter((w) => w.part_of_speech === 'unknown')
                .forEach((w) => {
                    w.part_of_speech = null
                })
            state.worker.active = true
            return route.fulfill({ status: 202, json: { status: 'started' } })
        } else if (path === '/api/admin/categorization/words') {
            const search = (url.searchParams.get('search') || '').toLowerCase()
            json = paginated(allWords.filter((w) => w.word.toLowerCase().includes(search)))
        } else if (path === '/api/admin/categorization/unknown') {
            if (state.failList) return route.fulfill({ status: 503, json: { error: 'unavailable' } })
            json = paginated(allWords.filter((w) => w.part_of_speech === 'unknown' || w.part_of_speech === null))
        } else if (path === '/api/admin/categorization/mismatches')
            json = paginated(
                mismatches.filter(
                    (p) => p.translation.original.part_of_speech !== p.translation.translation.part_of_speech
                )
            )
        else if (path.endsWith('/part-of-speech')) {
            if (state.failSave) return route.fulfill({ status: 503, json: { error: 'unavailable' } })
            const item = allWords.find((w) => path.includes(`/${w.id}/`))!
            item.part_of_speech = body.part_of_speech
            json = item
        }
        return route.fulfill({ json })
    })
    return { calls, vocabulary, shared, state, allWords }
}

for (const locale of ['en', 'ru']) {
    test(`vocabulary shows shared/separate and pending/unknown labels in ${locale}, on refresh only`, async ({
        page,
    }) => {
        const { calls, vocabulary } = await setup(page, locale)
        await page.goto('/vocabulary')
        const cards = page.locator('div.group.grid')
        await expect(cards).toHaveCount(5)
        const labels =
            locale === 'ru'
                ? ['Существительное', 'Глагол', 'Ожидает определения', 'Не определено']
                : ['Noun', 'Verb', 'Pending', 'Unknown']
        await expect(cards.nth(0).getByText(labels[0], { exact: true })).toHaveCount(1)
        await expect(cards.nth(1).getByText(labels[0], { exact: true })).toHaveCount(1)
        await expect(cards.nth(1).getByText(labels[1], { exact: true })).toHaveCount(1)
        await expect(cards.nth(1).locator('h3 > span').first()).toContainText(labels[0])
        await expect(cards.nth(1).locator('h3 > span').last()).toContainText(labels[1])
        await expect(cards.nth(2).getByText(labels[2], { exact: true })).toHaveCount(1)
        await expect(cards.nth(3).getByText(labels[3], { exact: true })).toHaveCount(1)
        await expect(cards.nth(4).getByText(labels[2], { exact: true })).toHaveCount(1)
        await expect(cards.nth(4).getByText(labels[3], { exact: true })).toHaveCount(1)
        vocabulary[2].translation.original.part_of_speech = 'phrase'
        vocabulary[2].translation.translation.part_of_speech = 'phrase'
        const initialCalls = calls.filter((c) => c.path === '/api/vocabulary').length
        await page.clock.install()
        await page.clock.fastForward(3_600_000)
        expect(calls.filter((c) => c.path === '/api/vocabulary')).toHaveLength(initialCalls)
        await expect(cards.nth(2)).toContainText(labels[2])
        await page.reload()
        await expect(cards.nth(2)).toContainText(locale === 'ru' ? 'Фраза' : 'Phrase')
    })
}

test('admin keeps per-record mismatches and refreshes all shared usages after a save', async ({ page }) => {
    const { calls } = await setup(page)
    await page.goto('/admin/categorization')
    await expect(page.getByRole('heading', { name: 'Categorization' })).toBeVisible()
    await expect(page.getByText('Any category can be edited.', { exact: false })).toBeVisible()
    await expect(page.getByRole('combobox')).toHaveCount(20)
    await page.getByRole('button', { name: 'Next', exact: true }).click()
    await expect(page.getByRole('combobox')).toHaveCount(1)
    await expect(page.getByText('Page 2 of 2')).toBeVisible()
    await page.getByRole('button', { name: 'Mismatched vocabulary pairs', exact: false }).click()
    await expect(page.locator('main li')).toHaveCount(3)
    await expect(page.getByRole('combobox')).toHaveCount(6)
    await expect(page.locator('main li').last().getByRole('combobox')).toHaveCount(2)
    await page.getByRole('combobox').first().selectOption('noun')
    await page.getByRole('button', { name: 'Save category' }).first().click()
    await expect(page.getByRole('status')).toHaveText('Category saved for every shared use of this word.')
    await expect(page.locator('main li')).toHaveCount(1)
    await expect(page.getByRole('combobox')).toHaveCount(2)
    expect(calls.find((c) => c.method === 'PUT')?.body).toEqual({ part_of_speech: 'noun' })
    await page.getByRole('button', { name: /^Unknown and pending/ }).click()
    await expect(page.getByRole('combobox')).toHaveCount(20)
    await expect(page.getByText('sourire', { exact: true })).toHaveCount(0)
    await expect(page.getByText('Page 2 of 2')).toHaveCount(0)
})

test('admin can retry list and save failures without losing the selected category', async ({ page }) => {
    const { state } = await setup(page)
    state.failList = true
    await page.goto('/admin/categorization')
    await expect(page.getByRole('alert')).toContainText('Could not load categories')
    state.failList = false
    await page.getByRole('button', { name: 'Retry', exact: true }).click()
    await expect(page.getByRole('combobox')).toHaveCount(20)
    state.failSave = true
    await page.getByRole('combobox').first().selectOption('verb')
    await page.getByRole('button', { name: 'Save category' }).first().click()
    await expect(page.getByRole('status')).toContainText('Could not save the category')
    await expect(page.getByRole('combobox').first()).toHaveValue('verb')
    state.failSave = false
    await page.getByRole('button', { name: 'Save category' }).first().click()
    await expect(page.getByRole('status')).toHaveText('Category saved for every shared use of this word.')
})

test('non-admins cannot enter categorization', async ({ page }) => {
    const { calls } = await setup(page, 'en', false)
    await page.goto('/admin/categorization')
    await expect(page).not.toHaveURL(/admin/)
    expect(calls.filter((c) => c.path.includes('/admin/'))).toHaveLength(0)
})

test('admin can find and change an already categorized word', async ({ page }) => {
    const { calls } = await setup(page)
    await page.goto('/admin/categorization')
    await page.getByRole('button', { name: 'All words', exact: false }).click()
    await page.getByRole('searchbox', { name: 'Search words or phrases' }).fill('decision')
    await expect(page.getByRole('combobox')).toHaveCount(1)
    await expect(page.getByRole('combobox')).toHaveValue('noun')
    await expect(page.getByRole('button', { name: 'Save category' })).toBeDisabled()
    await page.getByRole('combobox').selectOption('verb')
    await page.getByRole('button', { name: 'Save category' }).click()
    await expect(page.getByRole('status')).toContainText('Category saved')
    await expect(page.getByRole('combobox')).toHaveValue('verb')
    expect(calls.find((c) => c.method === 'PUT')?.path).toContain('/noun-two/part-of-speech')
})

test('retry shows live progress, survives reload, and polling preserves edits', async ({ page }) => {
    const { calls, allWords, state } = await setup(page)
    await page.goto('/admin/categorization')
    const retry = page.getByRole('button', { name: 'Retry unknown and pending', exact: true })
    await expect(retry).toBeEnabled()
    await retry.click()
    await expect(page.getByText('Categorization in progress', { exact: true })).toBeVisible()
    await expect(retry).toBeDisabled()
    await page.reload()
    await expect(page.getByText('Categorization in progress', { exact: true })).toBeVisible()
    await page.getByRole('combobox').first().selectOption('adverb')
    const initialLists = calls.filter((c) => c.path === '/api/admin/categorization/unknown').length
    allWords[0].part_of_speech = 'phrase'
    state.worker.processed = 1
    await expect(page.getByText('1 processed since last retry', { exact: true })).toBeVisible()
    await expect(page.getByRole('combobox').first()).toHaveValue('adverb')
    expect(calls.filter((c) => c.path === '/api/admin/categorization/unknown')).toHaveLength(initialLists)
    state.worker.active = false
    state.worker.failed = 1
    await expect(page.getByText('Categorization idle', { exact: true })).toBeVisible()
    await expect(page.getByText('1 failed', { exact: true })).toBeVisible()
    await expect(retry).toBeEnabled()
    await page.goto('/vocabulary')
    const stoppedCalls = calls.filter((c) => c.path === '/api/admin/categorization/stats').length
    await page.clock.install()
    await page.clock.fastForward(30_000)
    expect(calls.filter((c) => c.path === '/api/admin/categorization/stats')).toHaveLength(stoppedCalls)
})

for (const status of [409, 503]) {
    test(`failed retry (${status}) can recover by refreshing status`, async ({ page }) => {
        const { state } = await setup(page)
        state.restartStatus = status
        await page.goto('/admin/categorization')
        await page.getByRole('button', { name: 'Retry unknown and pending', exact: true }).click()
        await expect(page.getByRole('status')).toContainText(
            status === 409 ? 'already in progress' : 'Could not confirm the retry'
        )
        state.restartStatus = 202
        await page.getByRole('button', { name: 'Retry unknown and pending', exact: true }).click()
        await expect(page.getByText('Categorization in progress', { exact: true })).toBeVisible()
    })
}

test('unavailable progress disables retry and can recover', async ({ page }) => {
    const { state } = await setup(page)
    state.failStats = true
    await page.goto('/admin/categorization')
    await expect(page.getByRole('alert')).toContainText('Could not refresh progress')
    await expect(page.getByRole('button', { name: 'Retry unknown and pending', exact: true })).toBeDisabled()
    state.failStats = false
    await page.getByRole('button', { name: 'Refresh', exact: true }).click()
    await expect(page.getByRole('button', { name: 'Retry unknown and pending', exact: true })).toBeEnabled()
})
