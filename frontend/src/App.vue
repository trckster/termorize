<script setup lang="ts">
import { onMounted, onBeforeUnmount, watch } from 'vue'
import { useRouter } from 'vue-router'
import { TooltipProvider } from '@/components/ui/tooltip'
import ToastProvider from '@/components/ToastProvider.vue'
import { getLocaleDirection, useI18n } from '@/composables/useI18n'
import { useTheme } from '@/composables/useTheme'
import { bindTelegramSettingsButton } from '@/lib/telegram'

const { locale } = useI18n()
const { syncSystemTheme } = useTheme()
const router = useRouter()
let unbindTelegramSettingsButton = () => {}

const systemThemeQuery = window.matchMedia('(prefers-color-scheme: dark)')

const handleSystemThemeChange = () => syncSystemTheme()

onMounted(() => {
    systemThemeQuery.addEventListener('change', handleSystemThemeChange)
    unbindTelegramSettingsButton = bindTelegramSettingsButton(() => {
        router.push({ name: 'settings' }).catch(console.error)
    })
})

watch(
    locale,
    (nextLocale) => {
        document.documentElement.lang = nextLocale
        document.documentElement.dir = getLocaleDirection(nextLocale)
    },
    { immediate: true }
)

onBeforeUnmount(() => {
    systemThemeQuery.removeEventListener('change', handleSystemThemeChange)
    unbindTelegramSettingsButton()
})
</script>

<template>
    <ToastProvider>
        <TooltipProvider>
            <div class="min-h-screen font-sans antialiased text-foreground">
                <router-view />
            </div>
        </TooltipProvider>
    </ToastProvider>
</template>

<style></style>
