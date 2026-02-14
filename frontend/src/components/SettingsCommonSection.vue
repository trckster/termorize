<script setup lang="ts">
import { computed } from 'vue'
import type { User } from '@/api/auth.ts'
import { formatDate } from '@/lib/utils.ts'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

const props = defineProps<{
    user: User | null
}>()

const fields = computed(() => {
    const currentUser = props.user

    const sectionFields = [
        {
            key: 'id',
            label: 'ID',
            value: currentUser?.id?.toString() || 'Not available',
            explanation: 'Unique account identifier in Termorize.',
        },
        {
            key: 'name',
            label: 'Name',
            value: currentUser?.name || 'Not available',
            explanation: 'Your display name shown in the app header and profile.',
        },
        {
            key: 'username',
            label: 'Username',
            value: currentUser?.username ? `@${currentUser.username}` : 'Not available',
            explanation: 'Your Telegram username connected to this account.',
        },
        {
            key: 'created_at',
            label: 'Creation Date',
            value: currentUser?.created_at ? formatDate(currentUser.created_at) : 'Not available',
            explanation: 'Date and time when your Termorize account was created.',
        },
        {
            key: 'time_zone',
            label: 'Timezone',
            value: currentUser?.settings.time_zone || 'Not available',
            explanation: 'Your preferred timezone used for daily schedule and time-based features.',
        },
    ]

    return sectionFields
})
</script>

<template>
    <Card>
        <CardHeader>
            <CardTitle>Common</CardTitle>
            <CardDescription>Basic account information from your profile.</CardDescription>
        </CardHeader>
        <CardContent class="space-y-4">
            <div v-for="field in fields" :key="field.key" class="rounded-lg border border-border p-4 space-y-2">
                <p class="text-sm font-semibold text-foreground">{{ field.label }}</p>
                <p class="text-sm text-foreground break-all">{{ field.value }}</p>
                <p class="text-xs text-muted-foreground">{{ field.explanation }}</p>
            </div>
        </CardContent>
    </Card>
</template>
