<script setup lang="ts">
import { Send } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'

defineProps<{
    loading?: boolean
    insideTelegram?: boolean
}>()

const emit = defineEmits<{
    (e: 'start'): void
}>()
</script>

<template>
    <Button
        type="button"
        size="lg"
        class="w-full gap-3 bg-[#229ED9] text-white hover:bg-[#1d8fc5] focus-visible:ring-[#229ED9]"
        :disabled="loading"
        @click="emit('start')"
    >
        <Send class="size-4" />
        {{ loading ? (insideTelegram ? 'Authorizing in Telegram...' : 'Redirecting to Telegram...') : (insideTelegram ? 'Continue in Telegram' : 'Continue via Telegram') }}
    </Button>
</template>
