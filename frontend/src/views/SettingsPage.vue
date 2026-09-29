<script setup lang="ts">
import { computed, ref } from 'vue'
import { Loader2, TriangleAlert } from 'lucide-vue-next'
import { settingsApi } from '@/api/settings'
import { useToast } from '@/composables/useToast'
import { Button } from '@/components/ui/button'
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
const { addToast } = useToast()
const commonSection = ref<InstanceType<typeof SettingsCommonSection>>()
const languagesSection = ref<InstanceType<typeof SettingsLanguagesSection>>()
const telegramSection = ref<InstanceType<typeof SettingsTelegramSection>>()
const isSaving = ref(false)
const changes = computed(() =>
    [commonSection.value?.changes, languagesSection.value?.changes, telegramSection.value?.changes].filter(Boolean)
)
const hasChanges = computed(() => changes.value.length > 0)
const validationError = computed(() => telegramSection.value?.validationError || '')

function reviewValidation() {
    const section = document.getElementById('settings-telegram')
    const input = section?.querySelector<HTMLElement>('[aria-invalid="true"]:not(:disabled)')
    if (input) {
        input.scrollIntoView({ block: 'center' })
        input.focus({ preventScroll: true })
    } else section?.scrollIntoView({ block: 'start' })
}

async function save() {
    if (!authStore.user || !hasChanges.value || isSaving.value) return
    if (validationError.value) return reviewValidation()
    isSaving.value = true
    try {
        // One request merges every edited section, preserving unrelated settings.
        authStore.user = await settingsApi.updateSettings(Object.assign({}, authStore.user.settings, ...changes.value))
        addToast({
            title: t.value.toastSavedTitle,
            description: t.value.toastSavedDescription,
            variant: 'success',
            duration: 3000,
        })
    } catch {
        addToast({
            title: t.value.toastErrorTitle,
            description: t.value.toastSaveErrorDescription,
            variant: 'destructive',
            duration: 5000,
        })
    } finally {
        isSaving.value = false
    }
}

const user = computed(() => authStore.user)
const userSettings = computed(() => user.value?.settings)
const guestExpiresAt = computed(() => user.value?.guest_expires_at ?? null)
</script>

<template>
    <main class="px-4 pb-28 pt-4 sm:px-6 sm:pt-8">
        <div class="mx-auto max-w-5xl space-y-4 sm:space-y-6">
            <div>
                <h1 class="text-2xl font-semibold tracking-tight text-foreground sm:text-3xl sm:font-bold">
                    {{ t.settingsTitle }}
                </h1>
                <p class="mt-2 max-w-2xl text-sm leading-6 text-muted-foreground">
                    {{ t.settingsDescription }}
                </p>
            </div>

            <fieldset :disabled="isSaving" class="min-w-0 space-y-4 sm:space-y-6" :aria-busy="isSaving">
                <legend class="sr-only">{{ t.settingsTitle }}</legend>
                <SettingsCommonSection ref="commonSection" :user="user" />
                <SettingsAppearanceSection />
                <SettingsLanguagesSection ref="languagesSection" :settings="userSettings" />
                <SettingsTelegramSection
                    id="settings-telegram"
                    ref="telegramSection"
                    :settings="userSettings"
                    :is-guest="Boolean(guestExpiresAt)"
                    :is-saving="isSaving"
                    :timezone="commonSection?.changes?.time_zone ?? userSettings?.time_zone"
                />
            </fieldset>

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
        <div
            v-if="hasChanges"
            class="pointer-events-none fixed inset-x-4 bottom-[calc(5rem+env(safe-area-inset-bottom))] z-40 flex justify-center md:bottom-6"
        >
            <section
                :aria-label="t.settingsUnsavedChanges"
                class="pointer-events-auto flex max-w-full items-center gap-3 rounded-full border border-border bg-card py-2 pl-5 pr-2 text-card-foreground shadow-lg sm:gap-5"
            >
                <p class="text-sm font-medium" role="status">
                    {{ validationError ? t.settingsNeedsReview : t.settingsUnsavedChanges }}
                </p>
                <Button :disabled="isSaving" class="shrink-0 rounded-full" @click="save">
                    <Loader2 v-if="isSaving" class="motion-safe:animate-spin" aria-hidden="true" />
                    {{ isSaving ? t.saving : validationError ? t.settingsReview : t.save }}
                </Button>
            </section>
        </div>
    </main>
</template>
