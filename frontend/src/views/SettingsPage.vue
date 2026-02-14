<script setup lang="ts">
import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth.ts'
import SettingsCommonSection from '@/components/SettingsCommonSection.vue'
import SettingsLanguagesSection from '@/components/SettingsLanguagesSection.vue'
import SettingsTelegramSection from '@/components/SettingsTelegramSection.vue'

const authStore = useAuthStore()

const user = computed(() => authStore.user)
const userSettings = computed(() => user.value?.settings)
</script>

<template>
    <main class="px-6 py-8">
        <div class="max-w-5xl mx-auto space-y-6">
            <div>
                <h1 class="text-3xl font-bold text-foreground">Settings</h1>
                <p class="text-sm text-muted-foreground mt-2">
                    Review your account, language, and Telegram preferences.
                </p>
            </div>

            <SettingsCommonSection :user="user" />
            <SettingsLanguagesSection :settings="userSettings" />
            <SettingsTelegramSection :settings="userSettings" />
        </div>
    </main>
</template>
