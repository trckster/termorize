<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { UserSettings, UserTelegramScheduleItem } from '@/api/auth.ts'
import { settingsApi } from '@/api/settings.ts'
import { useToast } from '@/composables/useToast.ts'
import { useAuthStore } from '@/stores/auth.ts'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { InputNumber } from '@/components/ui/input-number'
import { ToggleSwitch } from '@/components/ui/toggle-switch'

const props = defineProps<{
    settings?: UserSettings
}>()

const authStore = useAuthStore()
const { addToast } = useToast()

const botEnabled = ref(false)
const dailyQuestionsEnabled = ref(false)
const dailyQuestionsCount = ref(10)
const dailyQuestionsSchedule = ref<UserTelegramScheduleItem[]>([])
const isSaving = ref(false)

const timezoneLabel = computed(() => props.settings?.time_zone || 'UTC')

const parseTime = (time: string) => {
    if (!/^\d{2}:\d{2}$/.test(time)) return null

    const [hoursValue, minutesValue] = time.split(':')
    const hours = Number(hoursValue)
    const minutes = Number(minutesValue)

    if (!Number.isInteger(hours) || !Number.isInteger(minutes)) return null
    if (hours < 0 || hours > 23 || minutes < 0 || minutes > 59) return null

    return hours * 60 + minutes
}

const countValidationError = computed(() => {
    if (!Number.isInteger(dailyQuestionsCount.value)) return 'Daily questions count must be an integer.'
    if (dailyQuestionsCount.value <= 0 || dailyQuestionsCount.value > 100) {
        return 'Daily questions count must be between 1 and 100.'
    }

    return ''
})

const scheduleValidationError = computed(() => {
    const intervals = dailyQuestionsSchedule.value
        .map((item, index) => {
            const from = parseTime(item.from)
            const to = parseTime(item.to)

            if (from === null || to === null) {
                return {
                    index,
                    from: -1,
                    to: -1,
                    valid: false,
                    reason: 'All times must use HH:mm format and stay in range 00:00 to 23:59.',
                }
            }

            if (from >= to) {
                return {
                    index,
                    from,
                    to,
                    valid: false,
                    reason: 'Each schedule interval must have "from" earlier than "to".',
                }
            }

            return {
                index,
                from,
                to,
                valid: true,
                reason: '',
            }
        })
        .sort((a, b) => a.from - b.from)

    const invalidInterval = intervals.find((item) => !item.valid)
    if (invalidInterval) return invalidInterval.reason

    for (let index = 1; index < intervals.length; index++) {
        const previous = intervals[index - 1]
        const current = intervals[index]
        if (current.from < previous.to) {
            return 'Schedule intervals cannot overlap.'
        }
    }

    return ''
})

const isValid = computed(() => !countValidationError.value && !scheduleValidationError.value)

const hasChanged = computed(() => {
    if (!props.settings) return false

    const currentTelegram = props.settings.telegram
    if (dailyQuestionsEnabled.value !== currentTelegram.daily_questions_enabled) return true
    if (dailyQuestionsCount.value !== currentTelegram.daily_questions_count) return true

    const currentSchedule = JSON.stringify(currentTelegram.daily_questions_schedule)
    const editedSchedule = JSON.stringify(dailyQuestionsSchedule.value)
    return currentSchedule !== editedSchedule
})

const setScheduleTime = (index: number, key: 'from' | 'to', value: string) => {
    dailyQuestionsSchedule.value[index] = {
        ...dailyQuestionsSchedule.value[index],
        [key]: value,
    }
}

const addScheduleItem = () => {
    dailyQuestionsSchedule.value.push({ from: '09:00', to: '10:00' })
}

const removeScheduleItem = (index: number) => {
    dailyQuestionsSchedule.value.splice(index, 1)
}

const saveTelegramSettings = async () => {
    if (!props.settings || !hasChanged.value || !isValid.value || isSaving.value) return

    isSaving.value = true

    try {
        authStore.user = await settingsApi.updateSettings({
            ...props.settings,
            telegram: {
                bot_enabled: botEnabled.value,
                daily_questions_enabled: dailyQuestionsEnabled.value,
                daily_questions_count: dailyQuestionsCount.value,
                daily_questions_schedule: dailyQuestionsSchedule.value,
            },
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

watch(
    () => props.settings,
    (nextSettings) => {
        botEnabled.value = nextSettings?.telegram.bot_enabled || false
        dailyQuestionsEnabled.value = nextSettings?.telegram.daily_questions_enabled || false
        dailyQuestionsCount.value = nextSettings?.telegram.daily_questions_count || 1
        dailyQuestionsSchedule.value = (nextSettings?.telegram.daily_questions_schedule || []).map((item) => ({
            from: item.from,
            to: item.to,
        }))
    },
    { immediate: true }
)
</script>

<template>
    <Card>
        <CardHeader>
            <CardTitle>Telegram</CardTitle>
            <CardDescription>Bot and notification controls for your Telegram account.</CardDescription>
        </CardHeader>
        <CardContent class="relative" :disabled="!props.settings?.telegram.bot_enabled">
            <template v-slot:disable-reason>
                <Card class="p-5 flex flex-col items-center">
                    <span> To enable Telegram bot, send him any message: </span>
                    <a
                        href="https://t.me/termorize_bot"
                        target="_blank"
                        rel="noopener noreferrer"
                        class="text-blue-500 underline"
                    >
                        @termorize_bot
                    </a>
                </Card>
            </template>

            <div class="grid grid-cols-1 md:grid-cols-2 p-4">
                <div class="space-y-2">
                    <p class="text-sm font-semibold text-foreground">Daily Questions Enabled</p>
                    <div class="h-10 flex items-center">
                        <ToggleSwitch v-model="dailyQuestionsEnabled" :disabled="isSaving" />
                    </div>
                    <p class="text-xs text-muted-foreground">
                        Controls if the bot sends you daily vocabulary exercises.
                    </p>
                </div>
                <div class="space-y-2" :class="dailyQuestionsEnabled ? '' : 'opacity-60'">
                    <p class="text-sm font-semibold text-foreground">Daily Questions Count</p>
                    <div class="h-10 flex items-center">
                        <InputNumber v-model="dailyQuestionsCount" min="1" max="100" step="1" :disabled="isSaving" />
                    </div>
                    <p class="text-xs text-muted-foreground">
                        How many exercises per day are you ready to complete? <br />
                        Must be from 1 to 100.
                    </p>
                    <p v-if="countValidationError" class="text-xs text-destructive">{{ countValidationError }}</p>
                </div>
            </div>

            <div class="p-4" :class="dailyQuestionsEnabled ? '' : 'opacity-60'">
                <div class="space-y-2">
                    <div class="flex items-center justify-between gap-4">
                        <p class="text-sm font-semibold text-foreground">Questions Schedule</p>
                        <p class="text-xs text-muted-foreground">Timezone: {{ timezoneLabel }}</p>
                    </div>

                    <div class="space-y-2">
                        <div
                            v-for="(item, index) in dailyQuestionsSchedule"
                            :key="index"
                            class="flex items-center gap-2"
                        >
                            <span class="text-muted-foreground">From</span>
                            <input
                                :value="item.from"
                                type="text"
                                inputmode="numeric"
                                placeholder="HH:mm"
                                :disabled="isSaving"
                                class="w-full rounded-md border border-border bg-background px-3 py-2 text-sm"
                                @input="setScheduleTime(index, 'from', ($event.target as HTMLInputElement).value)"
                            />
                            <span class="text-muted-foreground">to</span>
                            <input
                                :value="item.to"
                                type="text"
                                inputmode="numeric"
                                placeholder="HH:mm"
                                :disabled="isSaving"
                                class="w-full rounded-md border border-border bg-background px-3 py-2 text-sm"
                                @input="setScheduleTime(index, 'to', ($event.target as HTMLInputElement).value)"
                            />
                            <Button variant="outline" size="sm" :disabled="isSaving" @click="removeScheduleItem(index)">
                                Delete
                            </Button>
                        </div>
                    </div>

                    <Button variant="outline" size="sm" :disabled="isSaving" @click="addScheduleItem">
                        + Interval
                    </Button>

                    <p class="text-xs text-muted-foreground">
                        Set one or more time intervals in HH:mm format.<br />
                        This time is used to determine, when bot can send exercises to you in Telegram.
                    </p>
                    <p v-if="scheduleValidationError" class="text-xs text-destructive">{{ scheduleValidationError }}</p>
                </div>
            </div>

            <div v-if="hasChanged" class="px-4">
                <Button :disabled="isSaving || !isValid" @click="saveTelegramSettings">
                    {{ isSaving ? 'Saving...' : 'Save' }}
                </Button>
            </div>
        </CardContent>
    </Card>
</template>
