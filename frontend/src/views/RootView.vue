<script setup lang="ts">
import { watchEffect } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import LandingPage from './LandingPage.vue'

const router = useRouter()
const authStore = useAuthStore()

watchEffect(() => {
    if (authStore.hasCheckedAuth && authStore.isAuthenticated) {
        router.replace({ name: 'translation' })
    }
})
</script>

<template>
    <div
        v-if="!authStore.hasCheckedAuth"
        class="flex min-h-screen items-center justify-center bg-background px-6"
        role="status"
        aria-label="Loading Termorize"
    >
        <div class="w-full max-w-xs text-center">
            <p class="text-lg font-semibold tracking-tight text-foreground">Termorize</p>
            <div class="mx-auto mt-4 h-1 w-24 overflow-hidden rounded-full bg-muted">
                <div class="h-full w-2/3 animate-pulse rounded-full bg-primary motion-reduce:animate-none" />
            </div>
        </div>
    </div>
    <LandingPage v-else-if="!authStore.isAuthenticated" />
</template>
