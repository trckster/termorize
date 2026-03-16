<script setup lang="ts">
import { ref } from 'vue'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import TelegramLogin from '@/components/TelegramLogin.vue'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()
const error = ref<string | null>(null)
const isLoading = ref(false)

const startTelegramLogin = async () => {
    try {
        error.value = null
        isLoading.value = true
        const authUrl = await authStore.startTelegramLogin()
        window.location.assign(authUrl)
    } catch (err) {
        error.value = getErrorMessage(err, 'Unable to start Telegram login')
        isLoading.value = false
    }
}

function getErrorMessage(err: unknown, fallback: string): string {
    if (err instanceof Error) {
        return err.message
    }

    if (typeof err === 'object' && err !== null && 'body' in err) {
        const body = (err as { body?: { error?: string; message?: string } }).body
        return body?.error || body?.message || fallback
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
                <TelegramLogin :loading="isLoading" @start="startTelegramLogin" />
                <div v-if="error" class="text-center text-sm text-destructive">{{ error }}</div>
            </CardContent>
        </Card>
    </div>
</template>
