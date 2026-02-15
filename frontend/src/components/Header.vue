<template>
    <header class="border-b border-border bg-background">
        <div class="flex items-center justify-between px-6 py-4">
            <nav class="flex gap-8">
                <router-link
                    to="/"
                    :class="[
                        'text-sm font-medium transition-colors hover:text-foreground',
                        route.path === '/' ? 'text-foreground' : 'text-muted-foreground',
                    ]"
                >
                    Home
                </router-link>
                <router-link
                    to="/vocabulary"
                    :class="[
                        'text-sm font-medium transition-colors hover:text-foreground',
                        route.path === '/vocabulary' ? 'text-foreground' : 'text-muted-foreground',
                    ]"
                >
                    Vocabulary
                </router-link>
            </nav>

            <div class="flex items-center gap-4">
                <div ref="profileMenuRef" class="relative">
                    <button
                        @click.stop="toggleProfileMenu"
                        class="inline-flex items-center gap-3 rounded-md px-2 py-1 text-left transition-colors hover:bg-accent"
                        aria-haspopup="menu"
                        :aria-expanded="isProfileMenuOpen"
                    >
                        <div class="text-right">
                            <p class="text-sm font-medium text-foreground">{{ user?.name }}</p>
                            <p class="text-xs text-muted-foreground">@{{ user?.username }}</p>
                        </div>
                        <ChevronDown
                            class="h-4 w-4 text-muted-foreground transition-transform"
                            :class="isProfileMenuOpen ? 'rotate-180' : ''"
                        />
                    </button>

                    <div
                        v-if="isProfileMenuOpen"
                        class="absolute right-0 top-full z-50 mt-2 w-60 rounded-md border border-border bg-popover p-2 text-popover-foreground shadow-md"
                        role="menu"
                    >
                        <div class="flex items-center justify-between rounded-sm px-2 py-2">
                            <div class="flex items-center gap-2 text-sm font-medium">
                                <Sun v-if="isDark" class="h-4 w-4" />
                                <Moon v-else class="h-4 w-4" />
                                <span>Change theme</span>
                            </div>
                            <button
                                @click.stop="toggleTheme"
                                type="button"
                                class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors"
                                :class="isDark ? 'bg-primary' : 'bg-muted'"
                                role="switch"
                                :aria-checked="isDark"
                            >
                                <span
                                    class="inline-block h-5 w-5 transform rounded-full bg-background transition-transform"
                                    :class="isDark ? 'translate-x-5' : 'translate-x-1'"
                                />
                            </button>
                        </div>

                        <button
                            @click="goToSettings"
                            class="flex w-full items-center gap-2 rounded-sm px-2 py-2 text-sm font-medium transition-colors hover:bg-accent"
                            role="menuitem"
                        >
                            <Settings class="h-4 w-4" />
                            <span>Settings</span>
                        </button>

                        <div class="my-1 border-t border-border"></div>

                        <button
                            @click="handleLogout"
                            class="mt-1 flex w-full items-center gap-2 rounded-sm px-2 py-2 text-sm font-medium text-destructive transition-colors hover:bg-destructive hover:text-primary-foreground"
                            role="menuitem"
                        >
                            <LogOut class="h-4 w-4" />
                            <span>Logout</span>
                        </button>
                    </div>
                </div>
            </div>
        </div>
    </header>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onBeforeUnmount } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { Sun, Moon, ChevronDown, Settings, LogOut } from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const user = computed(() => authStore.user)
const isDark = ref(false)
const isProfileMenuOpen = ref(false)
const profileMenuRef = ref<HTMLElement | null>(null)

const closeProfileMenu = () => {
    isProfileMenuOpen.value = false
}

const toggleProfileMenu = () => {
    isProfileMenuOpen.value = !isProfileMenuOpen.value
}

const handleClickOutside = (event: MouseEvent) => {
    if (!profileMenuRef.value) return
    if (!profileMenuRef.value.contains(event.target as Node)) {
        closeProfileMenu()
    }
}

onMounted(() => {
    isDark.value = document.documentElement.classList.contains('dark')
    document.addEventListener('click', handleClickOutside)
})

onBeforeUnmount(() => {
    document.removeEventListener('click', handleClickOutside)
})

const toggleTheme = () => {
    isDark.value = !isDark.value
    if (isDark.value) {
        document.documentElement.classList.add('dark')
        localStorage.setItem('theme', 'dark')
    } else {
        document.documentElement.classList.remove('dark')
        localStorage.setItem('theme', 'light')
    }
}

const goToSettings = () => {
    closeProfileMenu()
    router.push('/settings')
}

const handleLogout = async () => {
    closeProfileMenu()
    await authStore.logout()
    router.push('/login')
}
</script>
