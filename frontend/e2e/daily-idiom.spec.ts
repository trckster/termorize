import { test, expect, type Page } from '@playwright/test'

const idiomID = '00000000-0000-4000-8000-000000000001'
const translationID = '00000000-0000-4000-8000-000000000002'

async function setup(page: Page, source = 'en', target = 'ru', language = 'en', failFirst = false) {
    page.on('pageerror', (error) => console.error(error))
    const calls: { path: string; body: any }[] = []
    const user = {
        id: 1,
        name: 'Alex',
        username: 'alex',
        is_admin: false,
        guest_expires_at: null,
        settings: {
            system_language: 'en',
            main_learning_language: language,
            translation_source_language: source,
            translation_target_language: target,
            time_zone: 'Europe/Rome',
            ignored_audio_languages: [],
            ignored_description_languages: [],
            telegram: {
                bot_enabled: true,
                daily_idiom_enabled: false,
                daily_questions_enabled: false,
                daily_questions_count: 10,
                daily_questions_schedule: [],
            },
        },
    }
    const word = language === 'it' ? 'rompere il ghiaccio' : 'break the ice'
    let failed = false
    await page.route('http://127.0.0.1:4173/api/**', async (route) => {
        const path = new URL(route.request().url()).pathname
        const body = route.request().postDataJSON()
        calls.push({ path, body })
        let json: unknown = {}
        if (path === '/api/me') json = user
        else if (path === '/api/settings') {
            if (route.request().method() === 'PUT') {
                user.settings = body
                json = user
            } else json = { languages: ['en', 'ru', 'it', 'de', 'es', 'fr', 'pl', 'tr', 'pt', 'uk'] }
        } else if (path === '/api/daily-idiom') {
            json = {
                date: new Date().toISOString().slice(0, 10),
                language,
                idiom: { id: idiomID, word_id: idiomID, word },
            }
        } else if (path.endsWith('/description'))
            json = { description: 'to make people feel more relaxed in a new situation' }
        else if (path === `/api/daily-idiom/${idiomID}/translate`) {
            if (failFirst && !failed) {
                failed = true
                return route.fulfill({ status: 503, json: { error: 'could not translate idiom; retry' } })
            }
            json = {
                id: translationID,
                original_word_id: idiomID,
                translation_word_id: translationID,
                translation: 'растопить лёд',
                source: 'idiom_llm',
            }
        } else if (path === '/api/translate')
            json = {
                id: 'ordinary-id',
                original_word_id: idiomID,
                translation_word_id: translationID,
                translation: 'ordinary translation',
                source: 'google',
            }
        return route.fulfill({ json })
    })
    await page.goto('/translation')
    await expect(page.getByRole('button', { name: 'Translate', exact: true })).toBeVisible()
    return { calls, word }
}

for (const [language, source, target, expectedTarget] of [
    ['en', 'en', 'ru', 'ru'],
    ['it', 'en', 'ru', 'en'],
    ['en', 'ru', 'en', 'ru'],
]) {
    test(`idiom ${language}, previous ${source}/${target}: auto LLM and Ctrl+S`, async ({ page }) => {
        const { calls, word } = await setup(page, source, target, language)
        await page.getByRole('button', { name: 'Translate', exact: true }).click()
        await expect(page.locator('#source-text')).toHaveValue(word)
        await expect(page.locator('#target-text')).toHaveValue('растопить лёд')
        await page.keyboard.press('Control+s')
        await expect.poll(() => calls.filter((c) => c.path === '/api/vocabulary/translation').length).toBe(1)
        expect(calls.find((c) => c.path === '/api/vocabulary/translation')?.body).toEqual({
            translation_id: translationID,
        })
        expect(calls.find((c) => c.path.endsWith('/translate'))?.body).toEqual({ to_language: expectedTarget })
        await expect
            .poll(() => calls.filter((c) => c.path === '/api/settings' && c.body).length)
            .toBe(source === language ? 0 : 1)
        expect(calls.filter((c) => c.path === '/api/translate')).toHaveLength(0)
        // Existing edit-before-save workflow retains the translation ID when unchanged.
        await expect(page.getByText('Translation added to vocabulary.', { exact: true })).toBeVisible()
        await page.keyboard.press('Control+e')
        await expect(page.locator('#editable-vocabulary-original')).toHaveValue(word)
        await expect(page.locator('#editable-vocabulary-translation')).toHaveValue('растопить лёд')
        await page.locator('#editable-vocabulary-translation').press('Shift+Enter')
        await expect.poll(() => calls.filter((c) => c.path === '/api/vocabulary/translation').length).toBe(2)
        expect(calls.filter((c) => c.path === '/api/translate')).toHaveLength(0)
    })
}

test('LLM failure retries without Google; later ordinary input uses the normal translator', async ({ page }) => {
    const { calls } = await setup(page, 'en', 'ru', 'en', true)
    await page.getByRole('button', { name: 'Translate', exact: true }).click()
    await expect(page.getByRole('button', { name: 'Retry', exact: true })).toBeVisible()
    await page.keyboard.press('Control+s')
    expect(calls.filter((c) => c.path === '/api/vocabulary/translation')).toHaveLength(0)
    await page.getByRole('button', { name: 'Retry', exact: true }).click()
    await expect(page.locator('#target-text')).toHaveValue('растопить лёд')
    expect(calls.filter((c) => c.path === '/api/translate')).toHaveLength(0)
    await page.locator('#source-text').fill('hello')
    await expect(page.locator('#target-text')).toHaveValue('ordinary translation')
    expect(calls.filter((c) => c.path === '/api/translate')).toHaveLength(1)
})

test('swapping an idiom preserves the displayed pair and saved translation', async ({ page }) => {
    const { calls } = await setup(page)
    await page.getByRole('button', { name: 'Translate', exact: true }).click()
    await expect(page.locator('#target-text')).toHaveValue('растопить лёд')
    await page.keyboard.press('Control+Shift+s')
    await expect(page.locator('#target-text')).toHaveValue('break the ice')
    await page.keyboard.press('Control+s')
    await expect.poll(() => calls.filter((c) => c.path === '/api/vocabulary/translation').length).toBe(1)
    expect(calls.filter((c) => c.path === '/api/translate')).toHaveLength(0)
    expect(calls.find((c) => c.path === '/api/vocabulary/translation')?.body.translation_id).toBe(translationID)
})

test('changing target language keeps the idiom on the LLM route', async ({ page }) => {
    const { calls } = await setup(page)
    await page.getByRole('button', { name: 'Translate', exact: true }).click()
    await expect(page.locator('#target-text')).toHaveValue('растопить лёд')
    await page.getByRole('combobox', { name: 'Target language' }).click()
    await page.getByRole('option', { name: '🇮🇹 Italian' }).click()
    await expect
        .poll(() => calls.filter((c) => c.path.endsWith('/translate') && c.body?.to_language === 'it').length)
        .toBe(1)
    expect(calls.filter((c) => c.path === '/api/translate')).toHaveLength(0)
    await expect(page.locator('#source-text')).toHaveValue('break the ice')
})

test('Telegram daily idiom setting is off by default and saves both states', async ({ page }) => {
    const { calls } = await setup(page)
    await page.goto('/settings')
    const toggle = page.getByRole('switch', { name: 'Daily idiom in Telegram' })
    await expect(toggle).toHaveAttribute('aria-checked', 'false')
    await expect(page.getByText('Daily idiom in Telegram', { exact: true })).toBeVisible()
    for (const enabled of [true, false]) {
        await toggle.click()
        await page.getByRole('button', { name: 'Save', exact: true }).click()
        await expect
            .poll(
                () =>
                    calls.filter((c) => c.path === '/api/settings' && c.body).at(-1)?.body.telegram.daily_idiom_enabled
            )
            .toBe(enabled)
        await expect(toggle).toHaveAttribute('aria-checked', String(enabled))
        await expect(page.getByRole('button', { name: 'Save', exact: true })).toHaveCount(0)
    }
})

const hintName = 'Change in Settings'
const showIdiomCard = (page: Page) =>
    page.locator('#daily-idiom-title').evaluate((title) => title.closest('section')!.scrollIntoView({ block: 'center' }))

async function expectBesideIdiom(page: Page) {
    const note = (await page.getByRole('dialog', { name: hintName }).boundingBox())!
    const word = (await page.getByText('break the ice', { exact: true }).boundingBox())!
    const viewport = page.viewportSize()!
    const overlaps =
        note.x < word.x + word.width &&
        word.x < note.x + note.width &&
        note.y < word.y + word.height &&
        word.y < note.y + note.height
    expect(overlaps).toBe(false)
    expect(note.x).toBeGreaterThanOrEqual(0)
    expect(note.x + note.width).toBeLessThanOrEqual(viewport.width)
}

test('daily idiom hint stays dismissed after navigation and reload', async ({ page }) => {
    await setup(page)
    const hint = page.getByRole('dialog', { name: hintName })
    await showIdiomCard(page)
    await expect(hint).toBeVisible()
    await expectBesideIdiom(page)
    await expect(hint).toContainText('Idioms are shown in your main learning language. You can change it in Settings.')
    await expect(hint).toContainText('You can also enable daily idioms in the Telegram bot there.')
    await hint.getByRole('link', { name: 'Settings', exact: true }).click()
    await expect(page).toHaveURL(/\/settings$/)
    await page.goBack()
    await showIdiomCard(page)
    await hint.getByRole('button', { name: 'Got it' }).click()
    await expect(hint).toHaveCount(0)
    expect(await page.evaluate(() => localStorage.getItem('termorize:daily-idiom-hint-dismissed'))).toBe('true')
    await page.getByRole('link', { name: 'Vocabulary', exact: true }).click()
    await expect(page).toHaveURL(/\/vocabulary$/)
    await page.getByRole('link', { name: 'Home', exact: true }).click()
    await showIdiomCard(page)
    await page.waitForTimeout(800)
    await expect(hint).toHaveCount(0)
    await page.reload()
    await showIdiomCard(page)
    await expect(page.getByText('break the ice', { exact: true })).toBeVisible()
    await page.waitForTimeout(800)
    await expect(hint).toHaveCount(0)
    await expect(page.getByText(hintName)).toHaveCount(0)
})

test('daily idiom hint is absent without an idiom or when it fails to load', async ({ page }) => {
    await setup(page)
    let fail = false
    await page.route('**/api/daily-idiom', (route) =>
        fail
            ? route.fulfill({ status: 503, json: { message: 'Temporary failure' } })
            : route.fulfill({
                  json: { date: new Date().toISOString().slice(0, 10), language: 'en', idiom: null },
              })
    )
    const empty = page.waitForResponse('**/api/daily-idiom')
    await page.reload()
    await empty
    await page.waitForTimeout(800)
    await expect(page.getByRole('dialog', { name: hintName })).toHaveCount(0)
    fail = true
    await page.reload()
    await expect(page.getByText('Could not load the idiom and its meaning. Please retry.')).toBeVisible()
    await showIdiomCard(page)
    await page.waitForTimeout(800)
    await expect(page.getByRole('dialog', { name: hintName })).toHaveCount(0)
})

test('centered Save pill combines timezone, language, and Telegram edits in one request', async ({ page }) => {
    const { calls } = await setup(page)
    await page.goto('/settings')
    const pill = page.getByRole('region', { name: 'Unsaved changes' })
    await expect(pill).toHaveCount(0)
    const timezone = page.locator('input[name="time-zone"]')
    await timezone.fill('Europe/Paris')
    await page.getByRole('option', { name: 'Europe/Paris', exact: true }).click()
    await expect(page.locator('#settings-telegram')).toContainText('Europe/Paris')
    await expect(page.locator('#settings-telegram')).not.toContainText('Europe/Rome')
    await page.locator('input[name="main-learning-language"]').click()
    await page.getByRole('option', { name: '🇮🇹 Italian', exact: true }).click()
    await page.getByRole('checkbox').first().check()
    await page.getByRole('switch', { name: 'Daily idiom in Telegram' }).click()
    await page.evaluate(() => window.scrollTo(0, 0))
    await expect(pill).toBeInViewport()
    await pill.getByRole('button', { name: 'Save', exact: true }).click()
    await expect(pill).toHaveCount(0)
    const saves = calls.filter((call) => call.path === '/api/settings' && call.body)
    expect(saves).toHaveLength(1)
    expect(saves[0]?.body).toMatchObject({
        time_zone: 'Europe/Paris',
        main_learning_language: 'it',
        ignored_audio_languages: ['en'],
        translation_source_language: 'en',
        translation_target_language: 'ru',
        telegram: { daily_idiom_enabled: true, bot_enabled: true },
    })
    await page.reload()
    await expect(timezone).toHaveValue('Europe/Paris')
    await expect(page.locator('input[name="main-learning-language"]')).toHaveValue('🇮🇹 Italian')
    await expect(page.getByRole('switch', { name: 'Daily idiom in Telegram' })).toHaveAttribute('aria-checked', 'true')
    await expect(pill).toHaveCount(0)
})

test('Save pill preserves changes after a failed request and clears when edits are reverted', async ({ page }) => {
    await setup(page)
    await page.goto('/settings')
    const pill = page.getByRole('region', { name: 'Unsaved changes' })
    const toggle = page.getByRole('switch', { name: 'Daily idiom in Telegram' })
    await toggle.click()
    await toggle.click()
    await expect(pill).toHaveCount(0)
    await toggle.click()
    let fail = true
    await page.route('**/api/settings', async (route) => {
        if (route.request().method() === 'PUT' && fail) {
            fail = false
            return route.fulfill({ status: 503, json: { message: 'Temporary failure' } })
        }
        return route.fallback()
    })
    await pill.getByRole('button', { name: 'Save', exact: true }).click()
    await expect(page.getByText('Failed to save settings. Please try again.', { exact: true })).toBeVisible()
    await expect(toggle).toHaveAttribute('aria-checked', 'true')
    await expect(pill.getByRole('button', { name: 'Save', exact: true })).toBeEnabled()
    await pill.getByRole('button', { name: 'Save', exact: true }).click()
    await expect(pill).toHaveCount(0)
})

test('Save pill reviews invalid Telegram schedules and stays above mobile navigation', async ({ page }) => {
    const { calls } = await setup(page)
    await page.setViewportSize({ width: 390, height: 844 })
    await page.goto('/settings')
    await page.getByRole('switch', { name: 'Send daily exercises' }).click()
    await page.getByRole('button', { name: '+ Interval', exact: true }).click()
    await page.locator('#schedule-from-0').fill('11:00')
    await page.getByRole('switch', { name: 'Send daily exercises' }).click()
    await expect(page.locator('#schedule-from-0')).toBeEnabled()
    await page.evaluate(() => window.scrollTo(0, 0))
    const pill = page.getByRole('region', { name: 'Unsaved changes' })
    await expect(pill).toBeInViewport()
    const pillBox = await pill.boundingBox()
    const navBox = await page.getByRole('navigation', { name: 'Main navigation' }).boundingBox()
    expect(pillBox!.y + pillBox!.height).toBeLessThan(navBox!.y)
    await pill.getByRole('button', { name: 'Review', exact: true }).click()
    await expect(page.locator('#schedule-from-0')).toBeFocused()
    expect(calls.filter((call) => call.path === '/api/settings' && call.body)).toHaveLength(0)
    await page.locator('#schedule-from-0').fill('09:00')
    await pill.getByRole('button', { name: 'Save', exact: true }).click()
    await expect(pill).toHaveCount(0)
})

test('phone hint waits for the idiom card, keeps focus in place, and stays gone after Escape', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 })
    await setup(page)
    const hint = page.getByRole('dialog', { name: hintName })
    await expect(hint).toHaveCount(0)
    await showIdiomCard(page)
    await expect(hint).toBeVisible()
    await expectBesideIdiom(page)
    const note = (await hint.boundingBox())!
    const card = (await page.locator('section[aria-labelledby="daily-idiom-title"]').boundingBox())!
    expect(note.y + note.height).toBeLessThan(card.y)
    await expect(page.locator('#source-text')).not.toBeFocused()
    expect(await hint.evaluate((element) => element.contains(document.activeElement))).toBe(false)
    await page.keyboard.press('Escape')
    await expect(hint).toHaveCount(0)
    await page.reload()
    await showIdiomCard(page)
    await page.waitForTimeout(800)
    await expect(hint).toHaveCount(0)
})

test('keyboard users can dismiss the hint and land on the idiom action', async ({ page }) => {
    await setup(page)
    const hint = page.getByRole('dialog', { name: hintName })
    await showIdiomCard(page)
    await hint.getByRole('link', { name: 'Settings', exact: true }).focus()
    await page.keyboard.press('Tab')
    await expect(hint.getByRole('button', { name: 'Got it' })).toBeFocused()
    await page.keyboard.press('Enter')
    await expect(hint).toHaveCount(0)
    await expect(page.getByRole('button', { name: 'Translate', exact: true })).toBeFocused()
    expect(await page.evaluate(() => localStorage.getItem('termorize:daily-idiom-hint-dismissed'))).toBe('true')
})

test('capture current idiom interface for the landing', async ({ page }) => {
    test.skip(!process.env.CAPTURE_IDIOM_PREVIEW, 'Asset capture is opt-in')
    await page.setViewportSize({ width: 1280, height: 1000 })
    await page.addInitScript(() => {
        localStorage.setItem('theme', 'dark')
        localStorage.setItem('palette', 'emerald')
    })
    await setup(page)
    await page.getByRole('button', { name: 'Translate', exact: true }).click()
    await expect(page.locator('#target-text')).toHaveValue('растопить лёд')
    await page.locator('#daily-idiom-title').click()
    await page.evaluate(() => {
        window.scrollTo(0, 0)
        ;(document.activeElement as HTMLElement)?.blur()
    })
    await page.screenshot({ path: 'public/images/daily-idiom-translation.png', fullPage: true, animations: 'disabled' })
    await page.setViewportSize({ width: 390, height: 844 })
    await page.screenshot({ path: '../tmp/idiom-review/app-mobile.png', fullPage: true, animations: 'disabled' })
    await page.goto('/settings')
    await expect(page.getByRole('switch', { name: 'Daily idiom in Telegram' })).toBeVisible()
    const switchControl = page.getByRole('switch', { name: 'Daily idiom in Telegram' })
    await switchControl.focus()
    await switchControl
        .locator('..')
        .screenshot({ path: '../tmp/idiom-review/settings-mobile.png', animations: 'disabled' })
})

test('landing explains daily idioms and shows the current app screenshot', async ({ page }) => {
    await page.route('http://127.0.0.1:4173/api/**', (route) =>
        route.fulfill({
            status: route.request().url().endsWith('/me') ? 401 : 200,
            json: { languages: ['en', 'ru'] },
        })
    )
    await page.goto('/')
    await expect(page.getByText(/Discover a new idiom each day/)).toBeVisible()
    await expect(page.getByText(/at 11:00 in your timezone/)).toBeVisible()
    const screenshot = page.locator('img[src="/images/daily-idiom-translation.png"]')
    await expect(screenshot).toBeVisible()
    await expect.poll(() => screenshot.evaluate((img: HTMLImageElement) => img.naturalWidth)).toBe(1280)
    if (process.env.CAPTURE_IDIOM_PREVIEW) {
        await page.setViewportSize({ width: 1440, height: 1000 })
        await page.screenshot({
            path: '../tmp/idiom-review/landing-desktop.png',
            fullPage: true,
            animations: 'disabled',
        })
        await page
            .locator('#telegram')
            .screenshot({ path: '../tmp/idiom-review/landing-telegram.png', animations: 'disabled' })
        await page.setViewportSize({ width: 390, height: 844 })
        await page.evaluate(() => window.scrollTo(0, 0))
        await page.screenshot({
            path: '../tmp/idiom-review/landing-mobile.png',
            fullPage: true,
            animations: 'disabled',
        })
    }
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
})
