import assert from 'node:assert/strict'
import { afterEach, describe, it } from 'node:test'
import { createMemoryHistory, createRouter } from 'vue-router'
import { bindTelegramSettingsButton } from './telegram.ts'

afterEach(() => {
    delete globalThis.window
})

function mockWebApp({ withSettingsButton = true } = {}) {
    const handlers = new Map()
    const webApp = {
        initData: 'signed-telegram-init-data',
        onEvent(event, callback) {
            if (!handlers.has(event)) handlers.set(event, new Set())
            handlers.get(event).add(callback)
        },
        offEvent(event, callback) {
            handlers.get(event)?.delete(callback)
        },
    }
    if (withSettingsButton) {
        webApp.SettingsButton = {
            isVisible: false,
            show() {
                this.isVisible = true
            },
            hide() {
                this.isVisible = false
            },
        }
    }
    globalThis.window = { Telegram: { WebApp: webApp } }
    return {
        webApp,
        click: () => handlers.get('settingsButtonClicked')?.forEach((handler) => handler()),
    }
}

describe('Telegram settings button', () => {
    it('opens settings through the router and removes the handler when unbound', async () => {
        const router = createRouter({
            history: createMemoryHistory(),
            routes: [
                { path: '/', component: {} },
                { path: '/settings', name: 'settings', component: {} },
            ],
        })
        await router.push('/')
        const { webApp, click } = mockWebApp()
        let navigation
        let calls = 0
        const openSettings = () => {
            calls++
            navigation = router.push({ name: 'settings' })
        }
        const unbind = bindTelegramSettingsButton(openSettings)
        assert.equal(webApp.SettingsButton.isVisible, true)

        click()
        await navigation
        assert.equal(router.currentRoute.value.path, '/settings')

        unbind()
        assert.equal(webApp.SettingsButton.isVisible, false)
        click()
        assert.equal(calls, 1)

        const unbindAgain = bindTelegramSettingsButton(openSettings)
        click()
        await navigation
        assert.equal(calls, 2)
        unbindAgain()
    })

    it('handles the legacy settings event without the SettingsButton object', () => {
        const { click } = mockWebApp({ withSettingsButton: false })
        let calls = 0
        const unbind = bindTelegramSettingsButton(() => calls++)
        click()
        assert.equal(calls, 1)
        unbind()
        click()
        assert.equal(calls, 1)
    })

    it('does nothing outside Telegram or without event APIs', () => {
        for (const window of [{}, { Telegram: { WebApp: { initData: 'data' } } }]) {
            globalThis.window = window
            const unbind = bindTelegramSettingsButton(() => assert.fail('Unexpected settings click'))
            unbind()
        }

        const { webApp, click } = mockWebApp()
        webApp.initData = ' '
        bindTelegramSettingsButton(() => assert.fail('Unexpected settings click'))()
        click()
        assert.equal(webApp.SettingsButton.isVisible, false)
    })
})
