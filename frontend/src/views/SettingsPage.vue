<script setup lang="ts">
import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth.ts'
import { useI18n } from '@/composables/useI18n'
import SettingsCommonSection from '@/components/SettingsCommonSection.vue'
import SettingsAppearanceSection from '@/components/SettingsAppearanceSection.vue'
import SettingsLanguagesSection from '@/components/SettingsLanguagesSection.vue'
import SettingsTelegramSection from '@/components/SettingsTelegramSection.vue'

const authStore = useAuthStore()
const { t } = useI18n()

const user = computed(() => authStore.user)
const userSettings = computed(() => user.value?.settings)
</script>

<template>
    <main class="px-4 py-4 sm:px-6 sm:py-8">
        <div class="max-w-5xl mx-auto space-y-6">
            <div>
                <h1 class="text-2xl font-semibold tracking-tight text-foreground sm:text-3xl sm:font-bold">
                    {{ t.settingsTitle }}
                </h1>
                <p class="text-sm text-muted-foreground mt-2">
                    {{ t.settingsDescription }}
                </p>
            </div>

            <SettingsCommonSection :user="user" />
            <SettingsAppearanceSection />
            <SettingsLanguagesSection :settings="userSettings" />
            <SettingsTelegramSection :settings="userSettings" />
        </div>
    </main>
</template>
