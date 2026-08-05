<template>
    <header class="pt-safe border-b border-border bg-background">
        <div class="flex min-h-[3.75rem] items-center justify-between px-4 py-2 sm:min-h-0 sm:px-6 sm:py-4">
            <router-link
                to="/translation"
                class="-ml-2 inline-flex min-h-11 items-center rounded-md px-2 text-sm font-semibold tracking-tight transition-colors hover:bg-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 md:hidden"
            >
                Termorize
            </router-link>
            <nav class="hidden md:flex gap-8">
                <router-link
                    to="/translation"
                    :aria-current="route.path === '/translation' ? 'page' : undefined"
                    :class="[
                        'text-sm font-medium transition-colors hover:text-foreground',
                        route.path === '/translation' ? 'text-foreground' : 'text-muted-foreground',
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
                    to="/collections"
                    :aria-current="route.path.startsWith('/collections') ? 'page' : undefined"
                    :class="[
                        'text-sm font-medium transition-colors hover:text-foreground',
                        route.path.startsWith('/collections') ? 'text-foreground' : 'text-muted-foreground',
                    ]"
                >
                    {{ t.navCollections }}
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
                <router-link
                    v-if="user?.is_admin"
                    to="/users"
                    :aria-current="route.path === '/users' ? 'page' : undefined"
                    :class="[
                        'text-sm font-medium transition-colors hover:text-foreground',
                        route.path === '/users' ? 'text-foreground' : 'text-muted-foreground',
                    ]"
                >
                    {{ t.navUsers }}
                </router-link>
            </nav>

            <div class="flex min-w-0 items-center gap-4">
                <div ref="profileMenuRef" class="relative">
                    <button
                        ref="profileMenuButtonRef"
                        @click.stop="toggleProfileMenu"
                        class="inline-flex h-11 w-11 min-w-0 items-center justify-center rounded-md text-left transition-colors hover:bg-accent focus:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 min-[480px]:h-auto min-[480px]:w-auto min-[480px]:max-w-[11rem] min-[480px]:gap-2 min-[480px]:px-2 min-[480px]:py-2 sm:max-w-none sm:gap-3"
                        aria-haspopup="menu"
                        :aria-label="t.headerOpenProfileMenu"
                        :aria-expanded="isProfileMenuOpen"
                        :aria-controls="profileMenuId"
                    >
                        <UserRound class="h-5 w-5 text-muted-foreground min-[480px]:hidden" />
                        <div class="hidden min-w-0 text-right min-[480px]:block">
                            <p class="truncate text-sm font-medium text-foreground">{{ user?.name }}</p>
                            <p class="truncate text-xs text-muted-foreground">@{{ user?.username }}</p>
                        </div>
                        <ChevronDown
                            class="hidden h-4 w-4 text-muted-foreground transition-transform min-[480px]:block"
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
                        <div class="border-b border-border px-2 py-2 min-[480px]:hidden">
                            <p class="truncate text-sm font-medium text-foreground">{{ user?.name }}</p>
                            <p class="truncate text-xs text-muted-foreground">@{{ user?.username }}</p>
                        </div>
                        <div class="flex items-center justify-between rounded-sm px-2 py-2">
                            <div :id="themeSwitchLabelId" class="flex items-center gap-2 text-sm font-medium">
                                <Sun v-if="isDark" class="h-4 w-4" />
                                <Moon v-else class="h-4 w-4" />
                                <span>{{ t.headerChangeTheme }}</span>
                            </div>
                            <ToggleSwitch
                                :model-value="isDark"
                                :labelledby="themeSwitchLabelId"
                                @update:model-value="setDark"
                                @click.stop
                            />
                        </div>

                        <button
                            ref="firstMenuActionRef"
                            @click="goToSettings"
                            role="menuitem"
                            class="flex min-h-11 w-full items-center gap-2 rounded-sm px-2 py-2 text-sm font-medium transition-colors hover:bg-accent focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                        >
                            <Settings class="h-4 w-4" />
                            <span>{{ t.headerSettings }}</span>
                        </button>

                        <div class="my-1 border-t border-border"></div>

                        <button
                            v-if="!isMiniApp"
                            @click="handleLogout"
                            role="menuitem"
                            class="mt-1 flex min-h-11 w-full items-center gap-2 rounded-sm px-2 py-2 text-sm font-medium text-destructive transition-colors hover:bg-destructive hover:text-destructive-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                        >
                            <LogOut class="h-4 w-4" />
                            <span>{{ t.headerLogout }}</span>
                        </button>

                        <button
                            v-if="isMiniApp"
                            @click="handleLogout"
                            role="menuitem"
                            class="mt-1 flex min-h-11 w-full items-center gap-2 rounded-sm px-2 py-2 text-sm font-medium text-destructive transition-colors hover:bg-destructive hover:text-destructive-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
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
import { Sun, Moon, ChevronDown, Settings, LogOut, UserRound } from 'lucide-vue-next'
import { ToggleSwitch } from '@/components/ui/toggle-switch'
import { isTelegramWebApp } from '@/lib/telegram.ts'
import { useI18n } from '@/composables/useI18n'
import { useTheme } from '@/composables/useTheme'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const { t } = useI18n()
const { isDark, setDark } = useTheme()

const user = computed(() => authStore.user)
const isMiniApp = isTelegramWebApp()
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
    document.addEventListener('click', handleClickOutside)
})

onBeforeUnmount(() => {
    document.removeEventListener('click', handleClickOutside)
})

const goToSettings = () => {
    closeProfileMenu()
    router.push('/settings')
}

const handleLogout = async () => {
    closeProfileMenu()
    await authStore.logout()
    router.push('/')
}
</script>
