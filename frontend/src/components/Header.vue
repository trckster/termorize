<template>
    <header class="border-b border-border bg-background">
        <div class="flex items-center justify-between px-6 py-4">
            <nav class="flex gap-8">
                <router-link
                    to="/"
                    :class="[
                        'text-sm font-medium transition-colors hover:text-foreground/80',
                        route.path === '/' ? 'text-foreground' : 'text-muted-foreground',
                    ]"
                >
                    Home
                </router-link>
                <router-link
                    to="/vocabulary"
                    :class="[
                        'text-sm font-medium transition-colors hover:text-foreground/80',
                        route.path === '/vocabulary' ? 'text-foreground' : 'text-muted-foreground',
                    ]"
                >
                    Vocabulary
                </router-link>
            </nav>

            <div class="flex items-center gap-4">
                <div class="text-right">
                    <p class="text-sm font-medium text-foreground">{{ user?.name }}</p>
                    <p class="text-xs text-muted-foreground">@{{ user?.username }}</p>
                </div>
                <button
                    @click="handleLogout"
                    class="inline-flex items-center justify-center px-3 py-1.5 text-sm font-medium rounded-md bg-red-500 text-white hover:bg-red-600 transition-colors"
                >
                    Logout
                </button>
            </div>
        </div>
    </header>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const user = computed(() => authStore.user)

const handleLogout = async () => {
    await authStore.logout()
    router.push('/login')
}
</script>
