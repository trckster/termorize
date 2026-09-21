export function getTelegramWebAppInitData(): string | null {
    const initData = window.Telegram?.WebApp?.initData?.trim()
    return initData ? initData : null
}

export function isTelegramWebApp(): boolean {
    return getTelegramWebAppInitData() !== null
}

export function bindTelegramSettingsButton(openSettings: () => void): () => void {
    const webApp = window.Telegram?.WebApp
    if (!isTelegramWebApp() || !webApp?.onEvent || !webApp.offEvent) {
        return () => {}
    }

    webApp.onEvent('settingsButtonClicked', openSettings)
    webApp.SettingsButton?.show()

    return () => {
        webApp.offEvent?.('settingsButtonClicked', openSettings)
        webApp.SettingsButton?.hide()
    }
}
