<script setup lang="ts">
import { computed } from 'vue'
import { TriangleAlert } from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth.ts'
import { useI18n } from '@/composables/useI18n'
import SettingsCommonSection from '@/components/SettingsCommonSection.vue'
import SettingsAppearanceSection from '@/components/SettingsAppearanceSection.vue'
import SettingsLanguagesSection from '@/components/SettingsLanguagesSection.vue'
import SettingsTelegramSection from '@/components/SettingsTelegramSection.vue'
import { formatDate } from '@/lib/utils.ts'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

const authStore = useAuthStore()
const { t } = useI18n()

const user = computed(() => authStore.user)
const userSettings = computed(() => user.value?.settings)
const guestExpiresAt = computed(() => user.value?.guest_expires_at ?? null)
</script>

<template>
    <main class="px-4 py-4 sm:px-6 sm:py-8">
        <div class="mx-auto max-w-5xl space-y-4 sm:space-y-6">
            <div>
                <h1 class="text-2xl font-semibold tracking-tight text-foreground sm:text-3xl sm:font-bold">
                    {{ t.settingsTitle }}
                </h1>
                <p class="mt-2 max-w-2xl text-sm leading-6 text-muted-foreground">
                    {{ t.settingsDescription }}
                </p>
            </div>

            <SettingsCommonSection :user="user" />
            <SettingsAppearanceSection />
            <SettingsLanguagesSection :settings="userSettings" />
            <SettingsTelegramSection :settings="userSettings" :is-guest="Boolean(guestExpiresAt)" />

            <Card
                v-if="guestExpiresAt"
                class="border-warning/40 bg-warning/5 shadow-none"
                aria-labelledby="guest-expiry-title"
            >
                <CardHeader>
                    <div class="flex items-start gap-3">
                        <TriangleAlert class="mt-0.5 h-5 w-5 shrink-0 text-warning" aria-hidden="true" />
                        <div class="space-y-1.5">
                            <CardTitle id="guest-expiry-title">{{ t.settingsGuestWarningTitle }}</CardTitle>
                            <CardDescription class="leading-6 text-foreground/80">
                                {{ t.settingsGuestWarningDescription }}
                                <time :datetime="guestExpiresAt" class="font-semibold tabular-nums text-foreground">
                                    {{ formatDate(guestExpiresAt) }} </time
                                >.
                            </CardDescription>
                        </div>
                    </div>
                </CardHeader>
                <CardContent>
                    <p class="text-sm leading-6 text-muted-foreground">{{ t.settingsGuestWarningNote }}</p>
                </CardContent>
            </Card>
        </div>
    </main>
</template>
