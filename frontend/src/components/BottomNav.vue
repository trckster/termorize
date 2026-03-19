<template>
    <nav
        class="fixed bottom-0 left-0 right-0 z-40 border-t border-border bg-background md:hidden"
        aria-label="Main navigation"
    >
        <div class="flex items-stretch" style="padding-bottom: env(safe-area-inset-bottom)">
            <router-link
                v-for="item in navItems"
                :key="item.to"
                :to="item.to"
                :aria-current="isActive(item.to) ? 'page' : undefined"
                class="flex flex-1 flex-col items-center gap-1 px-2 py-3 text-[12px] font-medium transition-colors rounded-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-inset"
                :class="isActive(item.to) ? 'text-foreground' : 'text-muted-foreground hover:text-foreground'"
            >
                <div class="relative flex items-center justify-center">
                    <div
                        v-if="isActive(item.to)"
                        class="absolute -top-3.5 left-1/2 h-0.5 w-6 -translate-x-1/2 rounded-full bg-primary"
                    />
                    <component :is="item.icon" class="h-5 w-5" />
                </div>
                <span>{{ item.label }}</span>
            </router-link>
        </div>
    </nav>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { Home, BookOpen, Brain } from 'lucide-vue-next'
import { useI18n } from '@/composables/useI18n'

const route = useRoute()
const { t } = useI18n()

const navItems = computed(() => [
    { to: '/', label: t.value.navHome, icon: Home },
    { to: '/vocabulary', label: t.value.navVocabulary, icon: BookOpen },
    { to: '/exercises', label: t.value.navExercises, icon: Brain },
])

const isActive = (path: string) => route.path === path
</script>
