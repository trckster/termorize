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

for (const width of [390, 1440]) {
    test(`share a word directly at ${width}px`, async ({ page }) => {
        await page.setViewportSize({ width, height: 900 })
        let fail = true
        let holdShare = false
        let releaseShare: (() => void) | undefined
        const calls: string[] = []
        await page.route('http://127.0.0.1:4173/api/**', async (route) => {
            const path = new URL(route.request().url()).pathname
            calls.push(`${route.request().method()} ${path}`)
            if (path === '/api/me') return route.fulfill({ json: user })
            if (path === '/api/settings') return route.fulfill({ json: { languages: ['en', 'it'] } })
            if (path === '/api/vocabulary/item-one/share') {
                if (holdShare)
                    await new Promise<void>((resolve) => {
                        releaseShare = resolve
                    })
                if (fail) return route.fulfill({ status: 503, json: {} })
                return route.fulfill({ json: { translation: item.translation, share_path: '/words/token' } })
            }
            if (path === '/api/vocabulary')
                return route.fulfill({
                    json: { data: [item], pagination: { page: 1, page_size: 20, total: 1, total_pages: 1 } },
                })
            return route.fulfill({ json: {} })
        })
        await page.addInitScript(() =>
            Object.defineProperty(navigator, 'clipboard', {
                configurable: true,
                value: {
                    writeText: async (text: string) => {
                        ;(window as any).copiedShareLink = text
                    },
                },
            })
        )
        await page.goto('/vocabulary')
        const shareButton = page.getByRole('button', { name: 'Share word forest' })
        await shareButton.click()
        await expect(
            page.getByText('Could not create a link for this word. Please try again.', { exact: true })
        ).toBeVisible()
        await expect(page.getByRole('dialog')).toHaveCount(0)
        await page.getByRole('button', { name: 'Close', exact: true }).click()
        await expect(page.locator('.toast-notification[data-state=closed]')).toHaveCount(0)
        fail = false
        await shareButton.click()
        await expect(page.getByText('Link copied', { exact: true })).toBeVisible()
        await expect(shareButton).toHaveAttribute('data-state', 'copied')
        await expect(shareButton.locator('[data-share-icon="loading"]')).toHaveCount(0)
        await expect(shareButton.locator('[data-share-icon="copied"]')).toBeVisible()
        expect(await page.evaluate(() => (window as any).copiedShareLink)).toBe('http://127.0.0.1:4173/words/token')
        await expect(page.getByRole('dialog')).toHaveCount(0)
        await page.getByText('Link copied', { exact: true }).evaluate(async (element) => {
            const toast = element.closest('li')
            if (toast) await Promise.all(toast.getAnimations().map((animation) => animation.finished))
        })
        await page.screenshot({ path: `/tmp/termorize-direct-copy-${width}.png`, fullPage: true })
        await page.getByRole('button', { name: 'Close', exact: true }).click()
        await expect(page.locator('.toast-notification[data-state=closed]')).toHaveCount(0)
        await page.evaluate(() => {
            navigator.clipboard.writeText = async () => {
                throw new Error('Denied')
            }
            document.execCommand = () => false
        })
        await shareButton.click()
        await expect(page.getByText('Unable to copy the link. Please try again.', { exact: true })).toBeVisible()
        await expect(shareButton).toHaveAttribute('data-state', 'idle')
        await expect(page.getByText('Link copied', { exact: true })).toHaveCount(0)
        await page.getByRole('button', { name: 'Close', exact: true }).click()
        await expect(page.locator('.toast-notification[data-state=closed]')).toHaveCount(0)
        await page.evaluate(() => {
            document.execCommand = () => {
                ;(window as any).copiedShareLink = (document.activeElement as HTMLTextAreaElement).value
                return true
            }
        })
        await shareButton.click()
        await expect(page.getByText('Link copied', { exact: true })).toBeVisible()
        expect(await page.evaluate(() => (window as any).copiedShareLink)).toBe('http://127.0.0.1:4173/words/token')
        await expect(shareButton).toBeFocused()
        await expect(page.locator('textarea')).toHaveCount(0)
        await expect(page.getByText('Link copied', { exact: true })).toHaveCount(0, { timeout: 6000 })
        await expect(shareButton).toHaveAttribute('data-state', 'idle')
        await page.emulateMedia({ reducedMotion: 'reduce' })
        await shareButton.click()
        const reducedToast = page.locator('.toast-notification').filter({ hasText: 'Link copied' })
        await expect(reducedToast).toBeVisible()
        await expect(shareButton).toHaveAttribute('data-state', 'copied')
        await expect(reducedToast).toHaveCSS('transform', 'none')
        await reducedToast.evaluate((element) =>
            element.addEventListener('animationstart', () => {
                if (element.getAttribute('data-state') === 'closed')
                    element.getAnimations().forEach((animation) => animation.pause())
            })
        )
        await page.getByRole('button', { name: 'Close', exact: true }).click()
        await expect(reducedToast).toHaveAttribute('data-state', 'closed')
        await expect(reducedToast).toBeVisible()
        await reducedToast.evaluate((element) => element.getAnimations().forEach((animation) => animation.finish()))
        await expect(reducedToast).toHaveCount(0)
        holdShare = true
        await shareButton.click()
        await expect(shareButton).toHaveAttribute('aria-busy', 'true')
        await expect(shareButton.locator('[data-share-icon="share"]')).toBeVisible()
        releaseShare?.()
        await expect(shareButton).toHaveAttribute('data-state', 'copied')
        await expect(shareButton.locator('[data-share-icon="loading"]')).toHaveCount(0)
        expect(calls.some((call) => call.includes('/collections'))).toBe(false)
    })

    for (const account of ['visitor', 'guest', 'permanent']) {
        test(`shared word saves only to permanent accounts: ${account} at ${width}px`, async ({ page }) => {
            await page.setViewportSize({ width, height: 900 })
            let signedIn = account === 'permanent'
            let saves = 0
            let guestLogins = 0
            await page.route('http://127.0.0.1:4173/api/**', async (route) => {
                const path = new URL(route.request().url()).pathname
                if (path === '/api/me') {
                    if (signedIn) return route.fulfill({ json: user })
                    if (account === 'guest')
                        return route.fulfill({ json: { ...user, guest_expires_at: '2099-01-01T00:00:00Z' } })
                    return route.fulfill({ status: 401, json: {} })
                }
                if (path === '/api/settings') return route.fulfill({ json: { languages: ['en', 'it'] } })
                if (path === '/api/public/vocabulary/token')
                    return route.fulfill({ json: { translation: item.translation, share_path: '/words/token' } })
                if (path === '/api/guest/login') {
                    guestLogins++
                    return route.fulfill({ status: 500, json: {} })
                }
                if (path === '/api/telegram/login/start')
                    return route.fulfill({
                        json: { auth_url: 'http://127.0.0.1:4173/login/telegram/callback?code=demo&state=demo' },
                    })
                if (path === '/api/telegram/login/callback') {
                    signedIn = true
                    return route.fulfill({ json: user })
                }
                if (path === '/api/vocabulary-shares/token/save') {
                    expect(signedIn).toBe(true)
                    saves++
                    return route.fulfill({ status: 201, json: item })
                }
                return route.fulfill({ status: 404, json: {} })
            })
            await page.goto('/words/token')
            await expect(page.getByText('forest', { exact: true })).toBeVisible()
            await expect(page.getByText('foresta', { exact: true })).toBeVisible()
            if (account !== 'permanent') {
                await expect(page.getByRole('button', { name: 'Save to my vocabulary' })).toHaveCount(0)
                await expect(
                    page.getByText('Sign in with Telegram to save this word to your vocabulary.')
                ).toBeVisible()
                expect(saves).toBe(0)
                await page.screenshot({ path: `/tmp/termorize-public-word-${account}-${width}.png`, fullPage: true })
                await page.getByRole('button', { name: 'Continue via Telegram' }).click()
                await expect(page).toHaveURL('http://127.0.0.1:4173/words/token')
            }
            await page.getByRole('button', { name: 'Save to my vocabulary' }).click()
            await expect(page.getByRole('status')).toContainText('This word is in your vocabulary')
            expect(saves).toBe(1)
            expect(guestLogins).toBe(0)
        })
    }
}

test('invalid share links show recovery', async ({ page }) => {
    await page.route('http://127.0.0.1:4173/api/**', (route) => route.fulfill({ status: 404, json: {} }))
    await page.goto('/words/invalid')
    await expect(page.getByRole('heading', { name: 'Word unavailable' })).toBeVisible()
    await expect(page.getByRole('alert')).toContainText('Ask your friend for a new link')
})
