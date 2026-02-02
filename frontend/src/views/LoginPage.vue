<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import TelegramLogin from '@/components/TelegramLogin.vue'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const authStore = useAuthStore()
const error = ref<string | null>(null)

const handleTelegramAuth = async (authData: any) => {
    console.log(authData)
    try {
        error.value = null
        await authStore.login(authData)
        router.push('/')
    } catch (err) {
        error.value = err instanceof Error ? err.message : 'Login failed'
    }
}
</script>

<template>
    <div class="flex items-center justify-center min-h-screen p-4">
        <Card class="w-full max-w-md">
            <CardHeader class="space-y-1">
                <CardTitle class="text-2xl font-bold text-center">Login with Telegram</CardTitle>
            </CardHeader>
            <CardContent class="flex flex-col items-center gap-4 py-6">
                <TelegramLogin @auth="handleTelegramAuth" />
                <div v-if="error" class="text-red-500 text-sm text-center">{{ error }}</div>
            </CardContent>
        </Card>
    </div>
</template>
