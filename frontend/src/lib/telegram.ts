export function getTelegramWebAppInitData(): string | null {
    const initData = window.Telegram?.WebApp?.initData?.trim()
    return initData ? initData : null
}

export function isTelegramWebApp(): boolean {
    return getTelegramWebAppInitData() !== null
}
