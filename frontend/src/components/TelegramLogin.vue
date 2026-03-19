<script setup lang="ts">
import { Send } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { useI18n } from '@/composables/useI18n'

defineProps<{
    loading?: boolean
    insideTelegram?: boolean
}>()

const emit = defineEmits<{
    (e: 'start'): void
}>()

const { t } = useI18n()
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
        {{ loading ? (insideTelegram ? t.telegramLoginButtonLoading : t.telegramLoginButtonRedirecting) : (insideTelegram ? t.telegramLoginButtonInsideTelegram : t.telegramLoginButtonVia) }}
    </Button>
</template>
