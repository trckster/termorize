<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { User, UserSettings } from '@/api/auth.ts'
import { useI18n } from '@/composables/useI18n'
import { formatDate } from '@/lib/utils.ts'
import { Combobox } from '@/components/ui/combobox'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

const props = defineProps<{
    user: User | null
}>()

const { t } = useI18n()

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

const hasTimezoneChanged = computed(() => {
    const currentTimezone = props.user?.settings.time_zone || browserTimezone
    return timezone.value !== currentTimezone
})

const fields = computed(() => [
    {
        key: 'id',
        label: t.value.settingsCommonFieldId,
        value: props.user?.id,
        explanation: t.value.settingsCommonFieldIdExplanation,
    },
    {
        key: 'created_at',
        label: t.value.settingsCommonFieldCreatedAt,
        value: props.user?.created_at ? formatDate(props.user.created_at) : t.value.settingsCommonFieldNotAvailable,
        explanation: t.value.settingsCommonFieldCreatedAtExplanation,
    },
    {
        key: 'name',
        label: t.value.settingsCommonFieldName,
        value: props.user?.name || t.value.settingsCommonFieldNotAvailable,
        explanation: props.user?.guest_expires_at
            ? t.value.settingsCommonFieldGuestNameExplanation
            : t.value.settingsCommonFieldNameExplanation,
    },
    {
        key: 'username',
        label: t.value.settingsCommonFieldUsername,
        value: props.user?.username ? `@${props.user.username}` : t.value.settingsCommonFieldNotAvailable,
        explanation: props.user?.guest_expires_at
            ? t.value.settingsCommonFieldGuestUsernameExplanation
            : t.value.settingsCommonFieldUsernameExplanation,
    },
])
const changes = computed<Partial<UserSettings> | null>(() =>
    hasTimezoneChanged.value ? { time_zone: timezone.value } : null
)
defineExpose({ changes })
</script>

<template>
    <Card>
        <CardHeader>
            <CardTitle>{{ t.settingsCommonTitle }}</CardTitle>
            <CardDescription>
                {{ props.user?.guest_expires_at ? t.settingsCommonGuestDescription : t.settingsCommonDescription }}
            </CardDescription>
        </CardHeader>
        <CardContent class="space-y-4">
            <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
                <div v-for="field in fields" :key="field.key" class="space-y-2 sm:rounded-lg sm:p-4">
                    <label :for="`settings-${field.key}`" class="text-sm font-semibold text-foreground">
                        {{ field.label }}
                    </label>
                    <input
                        :id="`settings-${field.key}`"
                        :name="field.key"
                        :value="field.value"
                        autocomplete="off"
                        disabled
                        class="min-h-11 w-full rounded-md border border-border bg-muted px-3 py-2 text-base text-muted-foreground sm:text-sm"
                    />
                    <p class="text-xs text-muted-foreground">{{ field.explanation }}</p>
                </div>

                <div class="space-y-2 sm:rounded-lg sm:p-4">
                    <p class="text-sm font-semibold text-foreground">{{ t.settingsCommonFieldTimezone }}</p>
                    <Combobox
                        v-model="timezone"
                        :options="timezoneOptions"
                        :placeholder="t.settingsCommonTimezonePlaceholder"
                        :search-placeholder="t.settingsCommonTimezoneSearchPlaceholder"
                        :empty-text="t.settingsCommonTimezoneNotFound"
                        :aria-label="t.settingsCommonFieldTimezone"
                        name="time-zone"
                    />
                    <p class="text-xs text-muted-foreground">
                        {{ t.settingsCommonFieldTimezoneExplanation }}
                    </p>
                </div>
            </div>
        </CardContent>
    </Card>
</template>
