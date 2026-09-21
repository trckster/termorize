/// <reference types="vite/client" />

declare const __APP_VERSION__: string

interface ImportMetaEnv {
    readonly HOST: string
    readonly PORT: string

    readonly VITE_API_URL: string
    readonly VITE_BOT_USERNAME: string
    readonly VITE_SENTRY_DSN?: string
}

interface ImportMeta {
    readonly env: ImportMetaEnv
}

interface TelegramWebApp {
    initData?: string
    onEvent?(event: 'settingsButtonClicked', callback: () => void): void
    offEvent?(event: 'settingsButtonClicked', callback: () => void): void
    SettingsButton?: {
        show(): void
        hide(): void
    }
}

interface TelegramGlobal {
    WebApp?: TelegramWebApp
}

interface Window {
    Telegram?: TelegramGlobal
}

declare module '*.vue' {
    import type { DefineComponent } from 'vue'
    const component: DefineComponent<{}, {}, any>
    export default component
}
