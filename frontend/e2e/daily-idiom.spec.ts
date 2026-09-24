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
