import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path'
import { designPreview } from './dev/design-preview'

export default ({ mode }: { mode: string }) => {
    const env = loadEnv(mode, process.cwd(), '')
    const appVersion = `${Date.now()}`

    return defineConfig({
        plugins: [
            vue(),
            ...(env.TERMORIZE_DESIGN_PREVIEW === '1' ? [designPreview()] : []),
            {
                name: 'app-version',
                transformIndexHtml(html) {
                    return html.replace('<head>', `<head>\n    <meta name="app-version" content="${appVersion}">`)
                },
            },
        ],
        define: {
            __APP_VERSION__: JSON.stringify(appVersion),
        },
        resolve: {
            alias: {
                '@': path.resolve(__dirname, './src'),
            },
        },
        server: {
            host: env.HOST,
            port: +(env.PORT || 8080),
        },
    })
}
