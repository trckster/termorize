<template>
    <header class="border-b border-border bg-background">
        <div class="flex items-center justify-between px-6 py-4">
            <span class="text-sm font-semibold tracking-tight md:hidden">Termorize</span>
            <nav class="hidden md:flex gap-8">
                <router-link
                    to="/"
                    :aria-current="route.path === '/' ? 'page' : undefined"
                    :class="[
                        'text-sm font-medium transition-colors hover:text-foreground',
                        route.path === '/' ? 'text-foreground' : 'text-muted-foreground',
                    ]"
                >
                    {{ t.navHome }}
                </router-link>
                <router-link
                    to="/vocabulary"
                    :aria-current="route.path === '/vocabulary' ? 'page' : undefined"
                    :class="[
                        'text-sm font-medium transition-colors hover:text-foreground',
                        route.path === '/vocabulary' ? 'text-foreground' : 'text-muted-foreground',
                    ]"
                >
                    {{ t.navVocabulary }}
                </router-link>
                <router-link
                    to="/exercises"
                    :aria-current="route.path === '/exercises' ? 'page' : undefined"
                    :class="[
                        'text-sm font-medium transition-colors hover:text-foreground',
                        route.path === '/exercises' ? 'text-foreground' : 'text-muted-foreground',
                    ]"
                >
                    {{ t.navExercises }}
                </router-link>
                <router-link
                    to="/statistics"
                    :aria-current="route.path === '/statistics' ? 'page' : undefined"
                    :class="[
                        'text-sm font-medium transition-colors hover:text-foreground',
                        route.path === '/statistics' ? 'text-foreground' : 'text-muted-foreground',
                    ]"
                >
                    {{ t.navStatistics }}
                </router-link>
            </nav>

            <div class="flex min-w-0 items-center gap-4">
                <div ref="profileMenuRef" class="relative">
                    <button
                        ref="profileMenuButtonRef"
                        @click.stop="toggleProfileMenu"
                        class="inline-flex min-w-0 items-center gap-3 rounded-md px-2 py-2 text-left transition-colors hover:bg-accent focus:outline-none focus:ring-2 focus:ring-primary"
                        aria-haspopup="menu"
                        :aria-label="t.headerOpenProfileMenu"
                        :aria-expanded="isProfileMenuOpen"
                        :aria-controls="profileMenuId"
                    >
                        <div class="min-w-0 text-right">
                            <p class="truncate text-sm font-medium text-foreground">{{ user?.name }}</p>
                            <p class="truncate text-xs text-muted-foreground">@{{ user?.username }}</p>
                        </div>
                        <ChevronDown
                            class="h-4 w-4 text-muted-foreground transition-transform"
                            :class="isProfileMenuOpen ? 'rotate-180' : ''"
                        />
                    </button>

                    <div
                        v-if="isProfileMenuOpen"
                        :id="profileMenuId"
                        class="absolute right-0 top-full z-50 mt-2 w-60 rounded-md border border-border bg-popover p-2 text-popover-foreground shadow-md"
                        role="menu"
                        :aria-label="t.headerOpenProfileMenu"
                        @keydown.esc.prevent="closeProfileMenu(true)"
                    >
                        <div class="flex items-center justify-between rounded-sm px-2 py-2">
                            <div :id="themeSwitchLabelId" class="flex items-center gap-2 text-sm font-medium">
                                <Sun v-if="isDark" class="h-4 w-4" />
                                <Moon v-else class="h-4 w-4" />
                                <span>{{ t.headerChangeTheme }}</span>
                            </div>
                            <ToggleSwitch
                                :model-value="isDark"
                                :labelledby="themeSwitchLabelId"
                                @update:model-value="setTheme"
                                @click.stop
                            />
                        </div>

                        <button
                            ref="firstMenuActionRef"
                            @click="goToSettings"
                            role="menuitem"
                            class="flex w-full items-center gap-2 rounded-sm px-2 py-2 text-sm font-medium transition-colors hover:bg-accent focus:outline-none focus:ring-2 focus:ring-primary"
                        >
                            <Settings class="h-4 w-4" />
                            <span>{{ t.headerSettings }}</span>
                        </button>

                        <div class="my-1 border-t border-border"></div>

                        <button
                            v-if="!isMiniApp"
                            @click="handleLogout"
                            role="menuitem"
                            class="mt-1 flex w-full items-center gap-2 rounded-sm px-2 py-2 text-sm font-medium text-destructive transition-colors hover:bg-destructive hover:text-primary-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                        >
                            <LogOut class="h-4 w-4" />
                            <span>{{ t.headerLogout }}</span>
                        </button>

                        <button
                            v-if="isMiniApp"
                            @click="handleLogout"
                            role="menuitem"
                            class="mt-1 flex w-full items-center gap-2 rounded-sm px-2 py-2 text-sm font-medium text-destructive transition-colors hover:bg-destructive hover:text-primary-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                        >
                            <LogOut class="h-4 w-4" />
                            <span>{{ t.headerRelogin }}</span>
                        </button>
                    </div>
                </div>
            </div>
        </div>
    </header>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, onMounted, onBeforeUnmount, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { Sun, Moon, ChevronDown, Settings, LogOut } from 'lucide-vue-next'
import { ToggleSwitch } from '@/components/ui/toggle-switch'
import { isTelegramWebApp } from '@/lib/telegram.ts'
import { useI18n } from '@/composables/useI18n'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const { t } = useI18n()

const user = computed(() => authStore.user)
const isMiniApp = isTelegramWebApp()
const isDark = ref(false)
const isProfileMenuOpen = ref(false)
const profileMenuRef = ref<HTMLElement | null>(null)
const profileMenuButtonRef = ref<HTMLButtonElement | null>(null)
const firstMenuActionRef = ref<HTMLButtonElement | null>(null)
const profileMenuId = 'profile-menu'
const themeSwitchLabelId = 'profile-theme-switch-label'

const closeProfileMenu = (restoreFocus: boolean = false) => {
    isProfileMenuOpen.value = false

    if (restoreFocus) {
        profileMenuButtonRef.value?.focus()
    }
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

watch(isProfileMenuOpen, async (open) => {
    if (!open) {
        return
    }

    await nextTick()
    firstMenuActionRef.value?.focus()
})

onMounted(() => {
    isDark.value = document.documentElement.classList.contains('dark')
    document.addEventListener('click', handleClickOutside)
})

onBeforeUnmount(() => {
    document.removeEventListener('click', handleClickOutside)
})

const setTheme = (nextIsDark: boolean) => {
    isDark.value = nextIsDark
    if (nextIsDark) {
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
