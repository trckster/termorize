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
        ...Array.from({ length: 20 }, (_, i) => word(`review-${i}`, `Review word ${i + 1}`, 'unknown')),
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
    const state = { conflict: false, failList: false }
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
        else if (path === '/api/admin/categorization/unknown') {
            if (state.failList) return route.fulfill({ status: 503, json: { error: 'unavailable' } })
            json = paginated(unknowns.filter((w) => w.part_of_speech === 'unknown'))
        } else if (path === '/api/admin/categorization/mismatches')
            json = paginated(
                mismatches.filter(
                    (p) => p.translation.original.part_of_speech !== p.translation.translation.part_of_speech
                )
            )
        else if (path.endsWith('/part-of-speech')) {
            const item = unknowns.find((w) => path.includes(`/${w.id}/`))!
            item.part_of_speech = state.conflict ? 'noun' : body.part_of_speech
            if (state.conflict) return route.fulfill({ status: 409, json: { error: 'permanent category' } })
            json = item
        }
        return route.fulfill({ json })
    })
    return { calls, vocabulary, shared, state }
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
    await expect(page.getByText('Concrete categories are permanent.', { exact: false })).toBeVisible()
    await expect(page.getByRole('combobox')).toHaveCount(20)
    await page.getByRole('button', { name: 'Next', exact: true }).click()
    await expect(page.getByRole('combobox')).toHaveCount(1)
    await expect(page.getByText('Page 2 of 2')).toBeVisible()
    await page.getByRole('button', { name: 'Mismatched vocabulary pairs', exact: false }).click()
    await expect(page.locator('main li')).toHaveCount(3)
    await expect(page.getByRole('combobox')).toHaveCount(2)
    await expect(page.locator('main li').last().getByRole('combobox')).toHaveCount(0)
    await page.getByRole('combobox').first().selectOption('noun')
    await page.getByRole('button', { name: 'Save category' }).first().click()
    await expect(page.getByRole('status')).toHaveText('Category saved for every shared use of this word.')
    await expect(page.locator('main li')).toHaveCount(1)
    await expect(page.getByRole('combobox')).toHaveCount(0)
    expect(calls.find((c) => c.method === 'PUT')?.body).toEqual({ part_of_speech: 'noun' })
    await page.getByRole('button', { name: 'Unknown words', exact: false }).click()
    await expect(page.getByRole('combobox')).toHaveCount(20)
    await expect(page.getByText('sourire', { exact: true })).toHaveCount(0)
    await expect(page.getByText('Page 2 of 2')).toHaveCount(0)
})

test('admin stale saves refresh permanent labels and a failed list can be retried', async ({ page }) => {
    const { state } = await setup(page)
    state.failList = true
    await page.goto('/admin/categorization')
    await expect(page.getByRole('alert')).toContainText('Could not load categories')
    state.failList = false
    await page.getByRole('button', { name: 'Retry', exact: true }).click()
    await expect(page.getByRole('combobox')).toHaveCount(20)
    state.conflict = true
    await page.getByRole('combobox').first().selectOption('verb')
    await page.getByRole('button', { name: 'Save category' }).first().click()
    await expect(page.getByRole('status')).toContainText('already has a permanent category')
    await expect(page.getByText('sourire', { exact: true })).toHaveCount(0)
})

test('non-admins cannot enter categorization', async ({ page }) => {
    const { calls } = await setup(page, 'en', false)
    await page.goto('/admin/categorization')
    await expect(page).not.toHaveURL(/admin/)
    expect(calls.filter((c) => c.path.includes('/admin/'))).toHaveLength(0)
})
