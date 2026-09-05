// Run with Playwright available to Node; see ../README.md. No live account or API is used.
const { chromium } = require('playwright')
const { mkdir } = require('node:fs/promises')
const path = require('node:path')
const { pathToFileURL } = require('node:url')

const extension = path.resolve(__dirname, '../..')
const output = path.resolve(extension, '../frontend/public/images/termoclip')

function installDemoMessaging() {
    window.__demoListeners = []
    window.chrome = {
        runtime: {
            id: 'termoclip-screenshot-demo',
            onMessage: {
                addListener(listener) {
                    window.__demoListeners.push(listener)
                },
            },
            sendMessage(message, callback) {
                const translation = {
                    id: 1,
                    original: 'passeggiata',
                    translated: 'stroll',
                    originalLanguage: 'it',
                    targetLanguage: 'en',
                }
                const responses = {
                    GET_SESSION: {
                        ok: true,
                        user: { name: 'Alex', settings: { translation_target_language: 'en' } },
                        languages: ['en', 'ru', 'it', 'de', 'es', 'fr', 'pl', 'tr', 'pt', 'uk'],
                    },
                    GET_ACTIVE_SELECTION: { ok: true, text: 'passeggiata' },
                    TRANSLATE_SELECTION: { ok: true, translation },
                    SAVE_SELECTION: { ok: true },
                }
                if (!responses[message.type]) throw new Error(`Unexpected demo message: ${message.type}`)
                callback(responses[message.type])
            },
        },
    }
}

async function main() {
    await mkdir(output, { recursive: true })
    const browser = await chromium.launch({
        headless: true,
        ...(process.env.CHROME_PATH ? { executablePath: process.env.CHROME_PATH } : {}),
    })
    try {
        const context = await browser.newContext({ colorScheme: 'light', reducedMotion: 'reduce' })
        await context.addInitScript(installDemoMessaging)
        const page = await context.newPage()
        page.on('pageerror', (error) => {
            throw error
        })
        await page.setViewportSize({ width: 1120, height: 800 })
        await page.goto(pathToFileURL(path.join(__dirname, 'reading-demo.html')).href)
        await page.addScriptTag({ path: path.join(extension, 'selection-overlay.js') })
        await page.evaluate(() =>
            window.__demoListeners[0](
                {
                    type: 'OPEN_SELECTION_OVERLAY',
                    selection: { ok: true, text: 'passeggiata', rect: { left: 680, top: 100, bottom: 110 } },
                },
                { id: chrome.runtime.id },
                () => {}
            )
        )
        // The production overlay deliberately has a closed shadow root. Inspect it through CDP
        // to wait for the real translation state without modifying its DOM or implementation.
        const cdp = await context.newCDPSession(page)
        for (let attempt = 0; ; attempt++) {
            const snapshot = await cdp.send('DOMSnapshot.captureSnapshot', { computedStyles: [] })
            if (snapshot.strings.includes('Ready to save. You can edit either field first.')) break
            if (attempt === 49) throw new Error('Translation did not become ready')
            await page.waitForTimeout(100)
        }
        await page.screenshot({ path: path.join(output, 'selection.png') })
        await context.close()

        const popupContext = await browser.newContext({
            viewport: { width: 390, height: 640 },
            deviceScaleFactor: 2,
            colorScheme: 'light',
            reducedMotion: 'reduce',
        })
        await popupContext.addInitScript(installDemoMessaging)
        const popup = await popupContext.newPage()
        popup.on('pageerror', (error) => {
            throw error
        })
        await popup.goto(pathToFileURL(path.join(extension, 'popup.html')).href)
        await popup.locator('#save-translation').click()
        await popup.getByText('Saved to your Termorize vocabulary.', { exact: true }).waitFor()
        await popup.screenshot({ path: path.join(output, 'popup-saved.png') })
        console.log(`Captured TermoClip screenshots in ${output}`)
        await popupContext.close()
    } finally {
        await browser.close()
    }
}

main().catch((error) => {
    console.error(error)
    process.exitCode = 1
})
