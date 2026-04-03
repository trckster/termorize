<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import TelegramLogin from '@/components/TelegramLogin.vue'
import { getTelegramWebAppInitData, isTelegramWebApp } from '@/lib/telegram'
import { useAuthStore } from '@/stores/auth'
import { useI18n } from '@/composables/useI18n'

const router = useRouter()
const authStore = useAuthStore()
const { t } = useI18n()
const error = ref<string | null>(null)
const isLoading = ref(false)
const isInsideTelegram = isTelegramWebApp()

onMounted(() => {
    if (getTelegramWebAppInitData()) {
        void startTelegramLogin()
    }
})

const startTelegramLogin = async () => {
    if (isLoading.value) {
        return
    }

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
        error.value = getErrorMessage(err, t.value.loginStartError)
        isLoading.value = false
    }
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
                <CardTitle class="text-2xl font-bold text-foreground">{{ t.loginTitle }}</CardTitle>
                <CardDescription class="text-muted-foreground">
                    {{ t.loginDescription }}
                </CardDescription>
            </CardHeader>
            <CardContent class="flex flex-col items-center gap-4 pt-2">
                <TelegramLogin :loading="isLoading" :inside-telegram="isInsideTelegram" @start="startTelegramLogin" />
                <div v-if="error" class="text-center text-sm text-destructive">{{ error }}</div>
            </CardContent>
        </Card>
    </div>
</template>
