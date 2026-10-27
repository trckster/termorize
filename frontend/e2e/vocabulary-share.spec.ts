import { test, expect } from '@playwright/test'

for (const width of [390, 1440]) {
    test(`share selected vocabulary at ${width}px`, async ({ page }) => {
        await page.setViewportSize({ width, height: 900 })
        let fail = true
        const shares: unknown[] = []
        await page.route('http://127.0.0.1:4173/api/**', async (route) => {
            const url = new URL(route.request().url())
            if (url.pathname === '/api/me')
                return route.fulfill({
                    json: {
                        id: 1,
                        username: 'owner',
                        settings: {
                            system_language: 'en',
                            translation_source_language: 'en',
                            translation_target_language: 'de',
                            ignored_audio_languages: [],
                            ignored_description_languages: [],
                            telegram: { daily_questions_schedule: [] },
                        },
                    },
                })
            if (url.pathname === '/api/settings') return route.fulfill({ json: { languages: ['en', 'de'] } })
            if (url.pathname === '/api/vocabulary/share') {
                shares.push(route.request().postDataJSON())
                if (fail) return route.fulfill({ status: 400, json: { error: 'unavailable' } })
                return route.fulfill({
                    status: 201,
                    json: { id: 'shared', share_path: '/collections/join/forest-token' },
                })
            }
            if (url.pathname === '/api/vocabulary') {
                const searching = !!url.searchParams.get('search')
                const id = searching ? 'second' : 'first'
                return route.fulfill({
                    json: {
                        data: [
                            {
                                id,
                                translation: {
                                    original: { id: 'word1', word: searching ? 'river' : 'forest', language: 'en' },
                                    translation: { id: 'word2', word: searching ? 'Fluss' : 'Wald', language: 'de' },
                                },
                                progress: [],
                                created_at: '2026-10-27T12:00:00Z',
                                mastered_at: null,
                            },
                        ],
                        pagination: { page: 1, page_size: 20, total: 1, total_pages: 1 },
                    },
                })
            }
            return route.fulfill({ json: {} })
        })
        await page.goto('/vocabulary')
        await page.getByRole('button', { name: 'Share words' }).click()
        await page.getByRole('checkbox', { name: 'Select forest' }).check()
        await page.getByRole('textbox', { name: 'Search' }).fill('river')
        await page.getByRole('checkbox', { name: 'Select river' }).check()
        await expect(page.getByRole('status')).toContainText('2 selected')
        await page.getByRole('button', { name: 'Create share link' }).click()
        await page.getByLabel('Collection title').fill('Outdoor words')
        await page.getByRole('button', { name: 'Create share link (2)' }).click()
        await expect(page.getByRole('alert')).toContainText('Could not share')
        fail = false
        await page.getByRole('button', { name: 'Create share link (2)' }).click()
        await expect(page.getByLabel('Share link', { exact: true })).toHaveValue(
            'http://127.0.0.1:4173/collections/join/forest-token'
        )
        expect(shares).toEqual(Array(2).fill({ title: 'Outdoor words', vocabulary_ids: ['first', 'second'] }))
        await expect(page.getByRole('link', { name: 'Manage collection' })).toHaveAttribute(
            'href',
            '/collections/shared'
        )
        await page.screenshot({ path: `/tmp/termorize-vocabulary-share-${width}.png`, fullPage: true })
    })
}
