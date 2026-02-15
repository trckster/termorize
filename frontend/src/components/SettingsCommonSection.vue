<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { User } from '@/api/auth.ts'
import { settingsApi } from '@/api/settings.ts'
import { useAuthStore } from '@/stores/auth.ts'
import { useToast } from '@/composables/useToast.ts'
import { formatDate } from '@/lib/utils.ts'
import { Button } from '@/components/ui/button'
import { Combobox } from '@/components/ui/combobox'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

const props = defineProps<{
    user: User | null
}>()

const authStore = useAuthStore()
const { addToast } = useToast()

const browserTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'

const getTimezones = () => {
    const fallback = ['UTC']
    const intlWithSupportedValues = Intl as typeof Intl & {
        supportedValuesOf?: (key: string) => string[]
    }

    if (!intlWithSupportedValues.supportedValuesOf) return fallback

    try {
        return intlWithSupportedValues.supportedValuesOf('timeZone')
    } catch {
        return fallback
    }
}

const allTimezones = Array.from(new Set([browserTimezone, ...getTimezones()]))
const timezone = ref(props.user?.settings.time_zone || browserTimezone)

watch(
    () => props.user?.settings.time_zone,
    (nextTimezone) => {
        timezone.value = nextTimezone || browserTimezone
    }
)

const timezoneOptions = computed(() => allTimezones.map((item) => ({ value: item, label: item })))
const isSaving = ref(false)

const hasTimezoneChanged = computed(() => {
    const currentTimezone = props.user?.settings.time_zone || browserTimezone
    return timezone.value !== currentTimezone
})

const saveTimezone = async () => {
    if (!props.user || !hasTimezoneChanged.value || isSaving.value) return

    isSaving.value = true

    try {
        authStore.user = await settingsApi.updateSettings({
            ...props.user.settings,
            time_zone: timezone.value,
        })

        addToast({
            title: 'Saved',
            description: 'Settings were saved successfully.',
            variant: 'success',
            duration: 3000,
        })
    } catch (error) {
        console.error('Failed to save settings:', error)
        addToast({
            title: 'Error',
            description: 'Failed to save settings. Please try again.',
            variant: 'destructive',
            duration: 5000,
        })
    } finally {
        isSaving.value = false
    }
}

const fields = computed(() => [
    {
        key: 'id',
        label: 'ID',
        value: props.user?.id,
        explanation: 'Unique account identifier in Termorize.',
    },
    {
        key: 'created_at',
        label: 'Creation Date',
        value: props.user?.created_at ? formatDate(props.user.created_at) : 'Not available',
        explanation: 'Date and time when your Termorize account was created.',
    },
    {
        key: 'name',
        label: 'Name',
        value: props.user?.name || 'Not available',
        explanation: 'Your Telegram name.',
    },
    {
        key: 'username',
        label: 'Username',
        value: props.user?.username ? `@${props.user.username}` : 'Not available',
        explanation: 'Your Telegram username.',
    },
])
</script>

<template>
    <Card>
        <CardHeader>
            <CardTitle>Common</CardTitle>
            <CardDescription>Basic account information from your profile.</CardDescription>
        </CardHeader>
        <CardContent class="space-y-4">
            <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
                <div v-for="field in fields" :key="field.key" class="space-y-2 rounded-lg p-4">
                    <p class="text-sm font-semibold text-foreground">{{ field.label }}</p>
                    <input
                        :value="field.value"
                        disabled
                        class="w-full rounded-md border border-border bg-muted px-3 py-2 text-sm text-muted-foreground"
                    />
                    <p class="text-xs text-muted-foreground">{{ field.explanation }}</p>
                </div>

                <div class="space-y-2 rounded-lg p-4">
                    <p class="text-sm font-semibold text-foreground">Timezone</p>
                    <Combobox
                        v-model="timezone"
                        :options="timezoneOptions"
                        placeholder="Select timezone"
                        search-placeholder="Search timezone..."
                        empty-text="No timezone found."
                    />
                    <p class="text-xs text-muted-foreground">
                        Your preferred timezone used for daily schedule and time-based features.
                    </p>
                </div>
            </div>
            <div class="px-4" v-if="hasTimezoneChanged">
                <Button v-if="hasTimezoneChanged" :disabled="isSaving" @click="saveTimezone">
                    {{ isSaving ? 'Saving...' : 'Save' }}
                </Button>
            </div>
        </CardContent>
    </Card>
</template>
