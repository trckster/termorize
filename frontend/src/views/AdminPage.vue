<script setup lang="ts">
import { useI18n } from '@/composables/useI18n'
const { t } = useI18n()
</script>

<template>
    <div>
        <header class="mx-auto max-w-7xl px-4 pt-6 sm:px-6">
            <h1 class="mb-4 text-2xl font-semibold">{{ t.navAdmin }}</h1>
            <nav :aria-label="t.navAdmin" class="flex gap-2 overflow-x-auto border-b border-border pb-3">
                <RouterLink
                    v-for="tab in [
                        { path: '/admin/users', label: t.navUsers },
                        { path: '/admin/word-audios', label: t.navWordAudios },
                        { path: '/admin/descriptions', label: t.navDescriptions },
                    ]"
                    :key="tab.path"
                    :to="tab.path"
                    class="whitespace-nowrap rounded-md px-3 py-2 text-sm font-medium text-muted-foreground hover:bg-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                    active-class="bg-accent !text-foreground"
                >
                    {{ tab.label }}
                </RouterLink>
            </nav>
        </header>
        <RouterView />
    </div>
</template>
