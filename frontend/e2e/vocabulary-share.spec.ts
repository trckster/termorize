import { test, expect } from '@playwright/test'

const item = {
    id: 'item-one',
    translation: {
        id: 'translation-one',
        original: { id: 'word-one', word: 'forest', language: 'en', part_of_speech: 'noun' },
        translation: { id: 'word-two', word: 'foresta', language: 'it', part_of_speech: 'noun' },
    },
    progress: [],
    created_at: '2026-10-27T12:00:00Z',
    mastered_at: null,
}
const user = {
    id: 1,
    username: 'owner',
    settings: {
        system_language: 'en',
        translation_source_language: 'en',
        translation_target_language: 'it',
        ignored_audio_languages: [],
        ignored_description_languages: [],
        telegram: { daily_questions_schedule: [] },
    },
}

for (const width of [390, 1440]) {
    test(`share a word directly at ${width}px`, async ({ page }) => {
        await page.setViewportSize({ width, height: 900 })
        let fail = true
        let revoked = false
        const calls: string[] = []
        await page.route('http://127.0.0.1:4173/api/**', async (route) => {
            const path = new URL(route.request().url()).pathname
            calls.push(`${route.request().method()} ${path}`)
            if (path === '/api/me') return route.fulfill({ json: user })
            if (path === '/api/settings') return route.fulfill({ json: { languages: ['en', 'it'] } })
            if (path === '/api/vocabulary/item-one/share') {
                if (route.request().method() === 'DELETE') {
                    revoked = true
                    return route.fulfill({ status: 204 })
                }
                if (fail) return route.fulfill({ status: 503, json: {} })
                return route.fulfill({ json: { translation: item.translation, share_path: '/words/token' } })
            }
            if (path === '/api/vocabulary')
                return route.fulfill({
                    json: { data: [item], pagination: { page: 1, page_size: 20, total: 1, total_pages: 1 } },
                })
            return route.fulfill({ json: {} })
        })
        await page.goto('/vocabulary')
        await expect(page.getByRole('checkbox')).toHaveCount(0)
        await page.getByRole('button', { name: 'Share word forest' }).click()
        await expect(page.getByRole('alert')).toContainText('Could not create a link')
        fail = false
        await page.getByRole('button', { name: 'Retry' }).click()
        await expect(page.getByLabel('Share link')).toHaveValue('http://127.0.0.1:4173/words/token')
        await page.evaluate(() =>
            Object.defineProperty(navigator, 'clipboard', {
                configurable: true,
                value: {
                    writeText: async () => {
                        throw new Error('denied')
                    },
                },
            })
        )
        await page.getByRole('button', { name: 'Copy link' }).click()
        await expect(page.getByRole('alert')).toContainText('Select and copy it')
        await page.screenshot({ path: `/tmp/termorize-word-share-${width}.png`, fullPage: true })
        await page.getByRole('button', { name: 'Stop sharing', exact: true }).click()
        await expect(page.getByRole('dialog')).toHaveCount(0)
        expect(revoked).toBe(true)
        expect(calls.some((call) => call.includes('/collections'))).toBe(false)
    })

    test(`public word preview and guest save at ${width}px`, async ({ page }) => {
        await page.setViewportSize({ width, height: 900 })
        let authenticated = false
        let saved = false
        await page.route('http://127.0.0.1:4173/api/**', async (route) => {
            const path = new URL(route.request().url()).pathname
            if (path === '/api/me')
                return route.fulfill({ status: authenticated ? 200 : 401, json: authenticated ? user : {} })
            if (path === '/api/settings') return route.fulfill({ json: { languages: ['en', 'it'] } })
            if (path === '/api/public/vocabulary/token')
                return route.fulfill({ json: { translation: item.translation, share_path: '/words/token' } })
            if (path === '/api/guest/login') {
                authenticated = true
                return route.fulfill({ json: user })
            }
            if (path === '/api/vocabulary/translation') {
                expect(route.request().postDataJSON()).toEqual({ translation_id: 'translation-one' })
                saved = true
                return route.fulfill({ status: 201, json: item })
            }
            return route.fulfill({ status: 404, json: {} })
        })
        await page.goto('/words/token')
        await expect(page.getByText('forest', { exact: true })).toBeVisible()
        await expect(page.getByText('foresta', { exact: true })).toBeVisible()
        await page.screenshot({ path: `/tmp/termorize-public-word-${width}.png`, fullPage: true })
        expect(authenticated).toBe(false)
        await page.getByRole('button', { name: 'Save to my vocabulary' }).click()
        await expect(page.getByRole('status')).toContainText('This word is in your vocabulary')
        expect(saved).toBe(true)
    })
}

test('invalid share links show recovery', async ({ page }) => {
    await page.route('http://127.0.0.1:4173/api/**', (route) => route.fulfill({ status: 404, json: {} }))
    await page.goto('/words/invalid')
    await expect(page.getByRole('heading', { name: 'Word unavailable' })).toBeVisible()
    await expect(page.getByRole('alert')).toContainText('Ask your friend for a new link')
})
