import { expect, test, type Page } from '@playwright/test'
import type { VocabularyItem } from '../src/api/vocabulary'

const user = {
    id: 1,
    username: 'learner',
    guest_expires_at: null,
    settings: {
        system_language: 'en',
        translation_source_language: 'en',
        translation_target_language: 'it',
        ignored_audio_languages: [],
        ignored_description_languages: [],
        telegram: { daily_questions_schedule: [] },
    },
}

const words: VocabularyItem[] = Array.from({ length: 1002 }, (_, index) => {
    const id = String(index + 1)
    const createdAt = '2026-10-27T12:00:00Z'
    return {
        id,
        translation_id: id,
        translation: {
            id,
            source: 'user',
            user_id: user.id,
            original: { id: `en-${id}`, word: `word ${id}`, language: 'en', part_of_speech: null },
            translation: {
                id: `it-${id}`,
                word: index === 0 ? 'una traduzione molto lunga '.repeat(4).trim() : `traduzione ${id}`,
                language: 'it',
                part_of_speech: null,
            },
            created_at: createdAt,
        },
        progress: [],
        created_at: createdAt,
        mastered_at: null,
    }
})

const mockVocabulary = async (page: Page, shouldFail = (_page: number, _pageSize: number) => false) => {
    const requests: { page: number; pageSize: number; search: string }[] = []

    await page.route('/api/**', async (route) => {
        const url = new URL(route.request().url())
        if (url.pathname === '/api/me') return route.fulfill({ json: user })
        if (url.pathname === '/api/settings') return route.fulfill({ json: { languages: ['en', 'it'] } })
        if (url.pathname !== '/api/vocabulary') return route.fulfill({ status: 404 })

        const page = Number(url.searchParams.get('page'))
        const pageSize = Number(url.searchParams.get('page_size'))
        const search = url.searchParams.get('search') ?? ''
        requests.push({ page, pageSize, search })
        if (shouldFail(page, pageSize)) return route.fulfill({ status: 503, json: {} })

        const matches = words.filter(
            ({ translation }) =>
                translation.original.word.includes(search) || translation.translation.word.includes(search)
        )
        return route.fulfill({
            json: {
                data: matches.slice((page - 1) * pageSize, page * pageSize),
                pagination: {
                    page,
                    page_size: pageSize,
                    total: matches.length,
                    total_pages: Math.ceil(matches.length / pageSize),
                },
            },
        })
    })

    await page.goto('/vocabulary')
    return requests
}

test('show all words across multiple batches in one-line rows on mobile', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 900 })
    const requests = await mockVocabulary(page)
    const pagination = page.locator('[data-slot="pagination"]')
    const showAll = page.getByRole('checkbox', { name: 'Show all words' })
    await expect(showAll).not.toBeChecked()
    await pagination.getByRole('button', { name: 'Page 2', exact: true }).click()
    await expect(page.getByText('word 21', { exact: true })).toBeVisible()

    await showAll.check()
    await expect(pagination).toHaveCount(0)
    const compactList = page.getByRole('region', { name: 'Saved Words' })
    const rows = compactList.getByRole('listitem')
    await expect(rows).toHaveCount(words.length)
    await expect(rows.last()).toContainText('word 1002')
    expect(requests.slice(-2)).toEqual([
        { page: 1, pageSize: 1000, search: '' },
        { page: 2, pageSize: 1000, search: '' },
    ])

    const longPair = rows.first()
    await expect(longPair).toHaveCSS('white-space', 'nowrap')
    await expect(longPair).toContainText(words[0].translation.translation.word)
    expect((await longPair.boundingBox())!.height).toBeLessThanOrEqual(25)
    expect(await compactList.evaluate((list) => list.scrollWidth > list.clientWidth)).toBe(true)
    expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBe(390)

    await showAll.uncheck()
    await expect(compactList).toHaveCount(0)
    await expect(pagination).toBeVisible()
    await expect(page.getByText('word 21', { exact: true })).toBeVisible()
    expect(requests.at(-1)).toEqual({ page: 2, pageSize: 20, search: '' })
})

test('retry an incomplete load and keep search active in show-all mode', async ({ page }) => {
    let failSecondBatch = true
    const requests = await mockVocabulary(page, (page, pageSize) => failSecondBatch && page === 2 && pageSize === 1000)
    const showAll = page.getByRole('checkbox', { name: 'Show all words' })
    await showAll.check()
    await expect(page.getByRole('alert')).toHaveText('Could not load your vocabulary. Please try again.')
    const rows = page.getByRole('region', { name: 'Saved Words' }).getByRole('listitem')
    await expect(rows).toHaveCount(0)

    failSecondBatch = false
    await page.getByRole('button', { name: 'Retry' }).click()
    await expect(rows).toHaveCount(words.length)

    await page.getByRole('textbox', { name: 'Search vocabulary...' }).fill('traduzione 100')
    await expect(rows).toHaveCount(4)
    await expect(rows.last()).toContainText('word 1002')
    await expect(showAll).toBeChecked()
    expect(requests.at(-1)).toEqual({ page: 1, pageSize: 1000, search: 'traduzione 100' })
})
