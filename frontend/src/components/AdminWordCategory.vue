<script setup lang="ts">
import { ref, watch } from 'vue'
import type { Word } from '@/api/vocabulary'
import { partsOfSpeech, partOfSpeechLabel, type PartOfSpeech } from '@/lib/partOfSpeech'
import { useSettingsStore } from '@/stores/settings'
import { useI18n } from '@/composables/useI18n'
import { Button } from '@/components/ui/button'
import PartOfSpeechLabel from '@/components/PartOfSpeechLabel.vue'

const props = defineProps<{ word: Word; busy: boolean; inputId: string }>()
const emit = defineEmits<{ save: [word: Word, category: PartOfSpeech] }>()
const { t } = useI18n()
const settings = useSettingsStore()
const selected = ref<PartOfSpeech | ''>(props.word.part_of_speech ?? '')
watch(
    () => [props.word.id, props.word.part_of_speech],
    () => {
        selected.value = props.word.part_of_speech ?? ''
    }
)

function save() {
    if (selected.value) emit('save', props.word, selected.value)
}
</script>

<template>
    <div class="min-w-0 space-y-3">
        <div>
            <p :lang="word.language" class="break-words text-base font-medium">{{ word.word }}</p>
            <p class="mt-1 text-xs text-muted-foreground">
                {{ settings.languageOptions.find((option) => option.code === word.language)?.name ?? word.language }}
            </p>
            <PartOfSpeechLabel :value="word.part_of_speech" />
        </div>
        <form class="flex flex-wrap items-end gap-2" @submit.prevent="save">
            <div class="min-w-0 flex-1">
                <label :for="inputId" class="mb-1 block text-xs font-medium">{{ t.categorizationCategory }}</label>
                <select
                    :id="inputId"
                    v-model="selected"
                    :disabled="busy"
                    :aria-label="`${t.categorizationCategory}: ${word.word}`"
                    class="h-10 w-full rounded-md border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50"
                >
                    <option value="" disabled>{{ t.categorizationChoose }}</option>
                    <option v-for="category in partsOfSpeech" :key="category" :value="category">
                        {{ partOfSpeechLabel(category, 'en') }}
                    </option>
                </select>
            </div>
            <Button type="submit" variant="outline" :disabled="busy || !selected || selected === word.part_of_speech">{{
                t.categorizationSave
            }}</Button>
        </form>
    </div>
</template>
