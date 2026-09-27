import { test, expect, type Page } from '@playwright/test'

const languages = [
    'English',
    'Russian',
    'Italian',
    'German',
    'Spanish',
    'French',
    'Polish',
    'Turkish',
    'Portuguese',
    'Ukrainian',
]
async function setup(page: Page, failFirst = false) {
    let lists = 0
    const started = new Set<string>()
    await page.route('http://127.0.0.1:4173/api/**', async (route) => {
        const path = new URL(route.request().url()).pathname
        if (path === '/api/me')
            return route.fulfill({
                json: {
                    id: 1,
                    name: 'Admin',
                    username: 'admin',
                    is_admin: true,
                    guest_expires_at: null,
                    settings: {
                        system_language: 'en',
                        main_learning_language: 'en',
                        translation_source_language: 'en',
                        translation_target_language: 'ru',
                        time_zone: 'UTC',
                        telegram: { bot_enabled: true },
                    },
                },
            })
        if (path === '/api/settings')
            return route.fulfill({ json: { languages: ['en', 'ru', 'it', 'de', 'es', 'fr', 'pl', 'tr', 'pt', 'uk'] } })
        if (path === '/api/admin/dictionaries') {
            if (failFirst && ++lists === 1) return route.fulfill({ status: 500, json: { error: 'unavailable' } })
            return route.fulfill({
                json: languages.map((name) => ({
                    id: name,
                    name: `${name} idioms`,
                    edition: `${name.toLowerCase()}wiktionary`,
                    url: 'https://en.wiktionary.org',
                    license: 'CC BY-SA',
                    attribution: 'Wiktionary contributors',
                    download_url: '',
                })),
            })
        }
        if (path.endsWith('/imports')) {
            const language = path.split('/').at(-2)!
            const job = {
                id: `job-${language}`,
                status: 'queued',
                processed: 0,
                inserted: 0,
                classified: 0,
                skipped: 0,
                failed: 0,
                downloaded_bytes: 0,
                total_bytes: null,
                record_errors: [],
                created_at: '2026-09-27T10:00:00Z',
            }
            if (route.request().method() === 'POST') {
                started.add(language)
                return route.fulfill({ status: 202, json: job })
            }
            return route.fulfill({
                json: {
                    data: started.has(language) ? [job] : [],
                    pagination: { page: 1, total: started.has(language) ? 1 : 0, total_pages: 1, page_size: 20 },
                },
            })
        }
        return route.fulfill({ json: {} })
    })
    await page.goto('/admin/dictionaries')
}

test('all ten sources retain working import controls on desktop and mobile', async ({ page }) => {
    await setup(page)
    await expect(page.getByRole('button', { name: 'Import', exact: true })).toHaveCount(10)
    for (const language of languages) {
        const source = page.getByRole('region', { name: `${language} idioms`, exact: true })
        await expect(source.getByRole('heading')).toBeVisible()
        await source.getByRole('button', { name: 'Import', exact: true }).click()
        await expect(source.getByRole('button', { name: 'Import in progress' })).toBeDisabled()
        await expect(source.getByRole('status')).toHaveText('Queued')
    }
    await page.setViewportSize({ width: 390, height: 844 })
    await expect(page.getByRole('heading', { name: 'Portuguese idioms' })).toBeVisible()
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
})

test('source loading failures can be retried', async ({ page }) => {
    await setup(page, true)
    await expect(page.getByRole('alert')).toBeVisible()
    await page.getByRole('button', { name: 'Refresh', exact: true }).click()
    await expect(page.getByRole('button', { name: 'Import', exact: true })).toHaveCount(10)
    await expect(page.getByRole('alert')).toHaveCount(0)
})
