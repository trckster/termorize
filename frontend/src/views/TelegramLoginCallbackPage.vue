<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const error = ref<string | null>(null)
const isLoading = ref(true)

onMounted(async () => {
    const code = getSingleQueryParam(route.query.code)
    const state = getSingleQueryParam(route.query.state)
    const telegramError = getSingleQueryParam(route.query.error)
    const telegramErrorDescription = getSingleQueryParam(route.query.error_description)

    if (telegramError) {
        error.value = telegramErrorDescription || telegramError
        isLoading.value = false
        return
    }

    if (!code || !state) {
        error.value = 'Telegram login response is incomplete'
        isLoading.value = false
        return
    }

    try {
        await authStore.completeTelegramLogin({ code, state })
        await router.replace('/')
    } catch (err) {
        error.value = getErrorMessage(err, 'Telegram login failed')
        isLoading.value = false
    }
})

function getSingleQueryParam(value: string | string[] | null | undefined): string | null {
    if (typeof value === 'string') {
        return value
    }

    if (Array.isArray(value) && value.length > 0) {
        return value[0]
    }

    return null
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

const goToLogin = () => {
    router.replace('/login')
}
</script>

<template>
    <div class="flex min-h-screen items-center justify-center px-4 py-10">
        <Card class="w-full max-w-md border-border/70 bg-card/95 shadow-xl backdrop-blur-sm">
            <CardHeader class="space-y-2 text-center">
                <CardTitle class="text-2xl font-bold text-foreground">
                    {{ isLoading ? 'Finishing Telegram login' : 'Telegram login' }}
                </CardTitle>
                <CardDescription class="text-muted-foreground">
                    {{ isLoading ? 'We are verifying your Telegram session.' : 'Something interrupted the login flow.' }}
                </CardDescription>
            </CardHeader>
            <CardContent class="flex flex-col items-center gap-4 py-6">
                <div v-if="isLoading" class="text-sm text-muted-foreground">Please wait...</div>
                <div v-else-if="error" class="text-center text-sm text-destructive">{{ error }}</div>
                <Button v-if="!isLoading" type="button" class="w-full" @click="goToLogin">Back to login</Button>
            </CardContent>
        </Card>
    </div>
</template>
