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
const isDesignPreview = import.meta.env.DEV && import.meta.env.VITE_DESIGN_PREVIEW === '1'
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
                <div
                    v-if="isDesignPreview"
                    class="flex flex-wrap items-center justify-center gap-x-4 border-b border-border bg-secondary px-4 py-1 text-xs text-secondary-foreground"
                >
                    <span>Local preview · sample data</span>
                    <RouterLink
                        to="/settings"
                        class="inline-flex min-h-11 items-center rounded-sm underline underline-offset-4 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                        >Preview settings</RouterLink
                    >
                </div>
                <router-view />
            </div>
        </TooltipProvider>
    </ToastProvider>
</template>

<style></style>
