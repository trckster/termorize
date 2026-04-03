<script setup lang="ts">
import { onMounted, onBeforeUnmount, watch } from 'vue'
import { TooltipProvider } from '@/components/ui/tooltip'
import ToastProvider from '@/components/ToastProvider.vue'
import { getLocaleDirection, useI18n } from '@/composables/useI18n'

const { locale } = useI18n()

const applyTheme = () => {
    const theme = localStorage.getItem('theme')
    if (theme === 'dark' || (!theme && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
        document.documentElement.classList.add('dark')
    } else {
        document.documentElement.classList.remove('dark')
    }
}

const systemThemeQuery = window.matchMedia('(prefers-color-scheme: dark)')

const handleSystemThemeChange = () => {
    if (!localStorage.getItem('theme')) {
        applyTheme()
    }
}

onMounted(() => {
    applyTheme()
    systemThemeQuery.addEventListener('change', handleSystemThemeChange)
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
