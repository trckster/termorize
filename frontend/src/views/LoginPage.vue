<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import TelegramLogin from '@/components/TelegramLogin.vue'
import { getTelegramWebAppInitData, isTelegramWebApp } from '@/lib/telegram'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const authStore = useAuthStore()
const error = ref<string | null>(null)
const isLoading = ref(false)
const isInsideTelegram = isTelegramWebApp()
const debugOutput = ref<string | null>(null)

const startTelegramLogin = async () => {
    try {
        error.value = null
        isLoading.value = true

        const initData = getTelegramWebAppInitData()
        if (initData) {
            await authStore.completeTelegramLogin({ init_data: initData })
            await router.replace('/')
            return
        }

        const authUrl = await authStore.startTelegramLogin()
        window.location.assign(authUrl)
    } catch (err) {
        error.value = getErrorMessage(err, 'Unable to start Telegram login')
        isLoading.value = false
    }
}

const toggleDebug = () => {
    debugOutput.value = debugOutput.value === null ? formatTelegramDebugData() : null
}

function formatTelegramDebugData(): string {
    return JSON.stringify(buildDebugSnapshot(), null, 2)
}

function buildDebugSnapshot() {
    return {
        timestamp: new Date().toISOString(),
        location: {
            href: window.location.href,
            origin: window.location.origin,
            pathname: window.location.pathname,
            search: window.location.search,
            hash: window.location.hash,
        },
        document: {
            referrer: document.referrer,
            readyState: document.readyState,
        },
        navigator: {
            userAgent: window.navigator.userAgent,
            language: window.navigator.language,
            languages: window.navigator.languages,
            platform: window.navigator.platform,
            onLine: window.navigator.onLine,
            cookieEnabled: window.navigator.cookieEnabled,
        },
        telegramScriptLoaded: hasTelegramSdkScript(),
        telegram: serializeTelegramValue(window.Telegram) ?? null,
        telegramWebAppInitData: getTelegramWebAppInitData(),
        urlParams: Object.fromEntries(new URLSearchParams(window.location.search).entries()),
    }
}

function hasTelegramSdkScript(): boolean {
    return Array.from(document.scripts).some((script) => script.src.includes('telegram-web-app.js'))
}

function serializeTelegramValue(value: unknown, seen = new WeakSet<object>()): unknown {
    if (value === null || value === undefined) {
        return value
    }

    if (typeof value === 'function') {
        return '[function]'
    }

    if (typeof value !== 'object') {
        return value
    }

    if (value instanceof Date) {
        return value.toISOString()
    }

    if (Array.isArray(value)) {
        return value.map((item) => serializeTelegramValue(item, seen))
    }

    if (seen.has(value)) {
        return '[circular]'
    }

    seen.add(value)

    const entries = Object.entries(value as Record<string, unknown>).map(([key, entryValue]) => [
        key,
        serializeTelegramValue(entryValue, seen),
    ])

    return Object.fromEntries(entries)
}

function getErrorMessage(err: unknown, fallback: string): string {
    if (err instanceof Error) {
        return err.message
    }

    if (typeof err === 'object' && err !== null && 'body' in err) {
        const body = (err as { body?: { error?: string; details?: string; message?: string } }).body
        return body?.details || body?.error || body?.message || fallback
    }

    return fallback
}
</script>

<template>
    <div class="flex min-h-screen items-center justify-center px-4 py-10">
        <Card class="w-full max-w-md border-border/70 bg-card/95 shadow-xl backdrop-blur-sm">
            <CardHeader class="space-y-2 text-center">
                <CardTitle class="text-2xl font-bold text-foreground">Login with Telegram</CardTitle>
                <CardDescription class="text-muted-foreground">
                    Login in Termorize to translate, check vocabulary, exercises, statistics and app settings.
                </CardDescription>
            </CardHeader>
            <CardContent class="flex flex-col items-center gap-4 pt-2">
                <TelegramLogin :loading="isLoading" :inside-telegram="isInsideTelegram" @start="startTelegramLogin" />
                <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    class="border-border/60 bg-muted text-muted-foreground hover:bg-muted/80"
                    @click="toggleDebug"
                >
                    {{ debugOutput === null ? 'Debug' : 'Hide debug' }}
                </Button>
                <div v-if="error" class="text-center text-sm text-destructive">{{ error }}</div>
                <pre
                    v-if="debugOutput"
                    class="max-h-80 w-full overflow-auto rounded-md border border-border/60 bg-muted/30 p-3 text-left text-xs text-muted-foreground"
                >{{ debugOutput }}</pre>
            </CardContent>
        </Card>
    </div>
</template>
