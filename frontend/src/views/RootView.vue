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
    <div v-if="!authStore.hasCheckedAuth" class="min-h-screen bg-background" />
    <LandingPage v-else-if="!authStore.isAuthenticated" />
</template>
