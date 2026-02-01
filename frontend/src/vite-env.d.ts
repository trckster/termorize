/// <reference types="vite/client" />

interface ImportMetaEnv {
    readonly HOST: string
    readonly PORT: string

    readonly VITE_API_URL: string
    readonly VITE_BOT_USERNAME: string
}

interface ImportMeta {
    readonly env: ImportMetaEnv
}

declare module '*.vue' {
    import type { DefineComponent } from 'vue'
    const component: DefineComponent<{}, {}, any>
    export default component
}
