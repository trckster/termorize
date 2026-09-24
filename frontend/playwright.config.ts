import { defineConfig } from '@playwright/test'

export default defineConfig({
    testDir: './e2e',
    use: { baseURL: 'http://127.0.0.1:4173', viewport: { width: 1440, height: 1000 } },
    webServer: {
        command: 'pnpm exec vite --host 127.0.0.1 --port 4173 --strictPort',
        url: 'http://127.0.0.1:4173',
        env: { VITE_API_URL: '/api', VITE_BOT_USERNAME: 'termorize_test_bot' },
    },
})
