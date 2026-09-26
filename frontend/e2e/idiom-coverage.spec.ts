import { test, expect, type Page } from '@playwright/test'

async function setup(page: Page, failFirst = false) {
    const names = [
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
    const codes = ['en', 'ru', 'it', 'de', 'es', 'fr', 'pl', 'tr', 'pt', 'uk']
    let coverageRequests = 0
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
        if (path === '/api/settings') return route.fulfill({ json: { languages: codes } })
        if (path === '/api/admin/dictionaries/coverage') {
            coverageRequests++
            if (failFirst && coverageRequests === 1)
                return route.fulfill({ status: 500, json: { error: 'unavailable' } })
            return route.fulfill({
                json: codes.map((language, i) => ({
                    language,
                    name: names[i],
                    idiom_count: i === 9 ? 0 : coverageRequests === 1 ? 3 : 8,
                })),
            })
        }
        if (path === '/api/admin/dictionaries')
            return route.fulfill({
                json: [
                    {
                        id: 'source-en',
                        name: 'English Wiktionary',
                        edition: 'enwiktionary',
                        url: 'https://en.wiktionary.org',
                        license: 'CC BY-SA',
                        attribution: 'Wiktionary contributors',
                        download_url: '',
                    },
                ],
            })
        if (path.endsWith('/imports'))
            return route.fulfill({
                json: { data: [], pagination: { page: 1, total: 0, total_pages: 0, page_size: 20 } },
            })
        return route.fulfill({ json: {} })
    })
    await page.goto('/admin/dictionaries')
    return names
}

test('all languages and zero counts are visible separately from import sources, with refresh', async ({ page }) => {
    const names = await setup(page)
    const table = page.getByRole('table', { name: 'Idioms by language' })
    await expect(table.getByRole('row')).toHaveCount(11)
    for (const name of names) await expect(table.getByRole('rowheader', { name, exact: true })).toBeVisible()
    await expect(table.getByRole('row', { name: 'Ukrainian 0 · No idioms yet' })).toBeVisible()
    await expect(page.getByRole('heading', { name: 'Import sources', exact: true })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Import', exact: true })).toBeVisible()
    await page.getByRole('button', { name: 'Refresh counts' }).click()
    await expect(table.getByRole('row', { name: 'German 8', exact: true })).toBeVisible()
    await page.setViewportSize({ width: 390, height: 844 })
    await expect(table.getByRole('rowheader', { name: 'Portuguese' })).toBeVisible()
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
})

test('coverage failure can be retried without hiding the import controls', async ({ page }) => {
    await setup(page, true)
    await expect(page.getByRole('alert')).toContainText('Could not load language coverage')
    await expect(page.getByRole('button', { name: 'Import', exact: true })).toBeVisible()
    await page.getByRole('button', { name: 'Refresh counts' }).click()
    await expect(page.getByRole('table').getByRole('rowheader', { name: 'Ukrainian' })).toBeVisible()
    await expect(page.getByRole('alert')).toHaveCount(0)
})
