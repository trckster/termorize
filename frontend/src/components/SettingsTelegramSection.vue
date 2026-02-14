<script setup lang="ts">
import { computed } from 'vue'
import type { UserSettings } from '@/api/auth.ts'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

const props = defineProps<{
    settings?: UserSettings
}>()

const boolLabel = (value?: boolean) => {
    if (typeof value !== 'boolean') return 'Not available'
    return value ? 'Yes' : 'No'
}

const scheduleLabel = computed(() => {
    const schedule = props.settings?.telegram.daily_questions_schedule || []
    if (schedule.length === 0) return 'Not configured'

    return schedule.map((item) => `${item.from}-${item.to}`).join(', ')
})

const fields = computed(() => {
    return [
        {
            key: 'bot_enabled',
            label: 'Bot Enabled',
            value: boolLabel(props.settings?.telegram.bot_enabled),
            explanation: 'Shows whether Telegram bot integration is active for your account.',
        },
        {
            key: 'daily_questions_enabled',
            label: 'Daily Questions Enabled',
            value: boolLabel(props.settings?.telegram.daily_questions_enabled),
            explanation: 'Controls if the bot sends your daily vocabulary questions.',
        },
        {
            key: 'daily_questions_count',
            label: 'Daily Questions Count',
            value: props.settings?.telegram.daily_questions_count?.toString() || '0',
            explanation: 'How many questions the bot sends to you each day.',
        },
        {
            key: 'daily_questions_schedule',
            label: 'Questions Schedule',
            value: scheduleLabel.value,
            explanation: 'Time windows in your timezone when daily questions may be delivered.',
        },
    ]
})
</script>

<template>
    <Card>
        <CardHeader>
            <CardTitle>Telegram</CardTitle>
            <CardDescription>Bot and notification controls for your Telegram account.</CardDescription>
        </CardHeader>
        <CardContent class="space-y-4">
            <div v-for="field in fields" :key="field.key" class="rounded-lg border border-border p-4 space-y-2">
                <p class="text-sm font-semibold text-foreground">{{ field.label }}</p>
                <p class="text-sm text-foreground">{{ field.value }}</p>
                <p class="text-xs text-muted-foreground">{{ field.explanation }}</p>
            </div>
        </CardContent>
    </Card>
</template>
