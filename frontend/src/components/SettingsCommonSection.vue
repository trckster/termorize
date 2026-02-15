<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Check, Copy } from 'lucide-vue-next'
import type { User } from '@/api/auth.ts'
import { formatDate } from '@/lib/utils.ts'
import { Button } from '@/components/ui/button'
import { Combobox } from '@/components/ui/combobox'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

const props = defineProps<{
    user: User | null
}>()

const idValue = computed(() => props.user?.id?.toString() || 'Not available')
const copied = ref(false)
const browserTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'

const copyId = async () => {
    if (!props.user?.id || typeof navigator === 'undefined' || !navigator.clipboard) return

    await navigator.clipboard.writeText(props.user.id.toString())
    copied.value = true

    setTimeout(() => {
        copied.value = false
    }, 1400)
}

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

const fields = computed(() => [
    {
        key: 'name',
        label: 'Name',
        value: props.user?.name || 'Not available',
        explanation: 'Your display name shown in the app header and profile.',
    },
    {
        key: 'username',
        label: 'Username',
        value: props.user?.username ? `@${props.user.username}` : 'Not available',
        explanation: 'Your Telegram username connected to this account.',
    },
    {
        key: 'created_at',
        label: 'Creation Date',
        value: props.user?.created_at ? formatDate(props.user.created_at) : 'Not available',
        explanation: 'Date and time when your Termorize account was created.',
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
            <div class="space-y-2">
                <p class="text-sm font-semibold text-foreground">ID</p>
                <div class="relative">
                    <input
                        :value="idValue"
                        disabled
                        class="w-full px-3 py-2 pr-12 text-sm rounded-md border border-border bg-muted text-muted-foreground disabled:cursor-not-allowed"
                    />
                    <Button
                        type="button"
                        variant="ghost"
                        size="icon-sm"
                        class="absolute right-1 top-1"
                        :disabled="!props.user?.id"
                        @click="copyId"
                    >
                        <Check v-if="copied" class="h-4 w-4" />
                        <Copy v-else class="h-4 w-4" />
                    </Button>
                </div>
                <p class="text-xs text-muted-foreground">Unique account identifier in Termorize.</p>
            </div>

            <div v-for="field in fields" :key="field.key" class="space-y-2">
                <p class="text-sm font-semibold text-foreground">{{ field.label }}</p>
                <input
                    :value="field.value"
                    disabled
                    class="w-full px-3 py-2 text-sm rounded-md border border-border bg-muted text-muted-foreground disabled:cursor-not-allowed"
                />
                <p class="text-xs text-muted-foreground">{{ field.explanation }}</p>
            </div>

            <div class="space-y-2">
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
        </CardContent>
    </Card>
</template>
