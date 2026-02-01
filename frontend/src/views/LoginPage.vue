<script setup lang="ts">
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import TelegramLogin from '@/components/TelegramLogin.vue'

const handleTelegramAuth = (authData: any) => {
    console.log(authData)
    // Here you would typically call auth.login(authData)
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
            </CardContent>
        </Card>
    </div>
</template>
