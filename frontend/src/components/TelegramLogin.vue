<script setup lang="ts">
import { onMounted, ref } from 'vue'

const telegramWidgetContainer = ref<HTMLElement | null>(null)

type TelegramAuthResult = {
    id: number
    auth_date: number
    username: string
    first_name: string
    last_name: string
    photo_url: string
    hash: string
}

const emit = defineEmits<{
    (e: 'auth', data: TelegramAuthResult): void
}>()

window.onTelegramAuth = (authData: TelegramAuthResult) => {
    emit('auth', authData)
}

declare global {
    interface Window {
        onTelegramAuth: (user: any) => void
    }
}

onMounted(() => {
    if (telegramWidgetContainer.value) {
        const script = document.createElement('script')
        script.src = 'https://telegram.org/js/telegram-widget.js?22'
        script.setAttribute('data-telegram-login', import.meta.env.VITE_BOT_USERNAME)
        script.setAttribute('data-size', 'large')
        script.setAttribute('data-onauth', 'onTelegramAuth(user)')
        script.setAttribute('data-request-access', 'write')
        script.async = true
        telegramWidgetContainer.value.appendChild(script)
    }
})
</script>

<template>
    <div ref="telegramWidgetContainer"></div>
</template>
