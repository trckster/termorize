<script setup lang="ts">
import { onMounted } from 'vue'
import { TooltipProvider } from '@/components/ui/tooltip'
import ToastProvider from '@/components/ToastProvider.vue'

onMounted(() => {
    const theme = localStorage.getItem('theme')
    if (theme === 'dark' || (!theme && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
        document.documentElement.classList.add('dark')
    } else {
        document.documentElement.classList.remove('dark')
    }
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

<style>
/* Remove default vite styles if any or just keep it clean */
</style>
