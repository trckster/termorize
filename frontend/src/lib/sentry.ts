import * as Sentry from '@sentry/vue'
import type { App } from 'vue'
import type { Router } from 'vue-router'

const TRACES_SAMPLE_RATE = 0

export function initSentry(app: App, router: Router): void {
    const dsn = import.meta.env.VITE_SENTRY_DSN

    if (!dsn) {
        return
    }

    Sentry.init({
        app,
        dsn,
        environment: import.meta.env.MODE,
        integrations: [Sentry.browserTracingIntegration({ router })],
        tracesSampleRate: TRACES_SAMPLE_RATE,
        dataCollection: {
            userInfo: false,
            cookies: false,
            httpHeaders: {
                request: { deny: ['forwarded', '-ip', 'remote-', 'via', '-user'] },
                response: { deny: ['forwarded', '-ip', 'remote-', 'via', '-user'] },
            },
            httpBodies: [],
            urlQueryParams: { deny: ['forwarded', '-ip', 'remote-', 'via', '-user'] },
            genAI: { inputs: false, outputs: false },
            databaseQueryData: false,
            graphQL: { document: false, variables: false },
        },
    })
}
