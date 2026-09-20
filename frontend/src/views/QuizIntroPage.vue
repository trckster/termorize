<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Play, X } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { useI18n } from '@/composables/useI18n'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const root = ref<HTMLElement | null>(null)
const starting = ref(false)
const returnTo = computed(() => {
    const path = route.query.from
    if (typeof path !== 'string' || !path.startsWith('/') || path.startsWith('//')) {
        return '/translation'
    }
    const resolved = router.resolve(path)
    const isAppPage =
        (resolved.meta.requiresAuth &&
            !['quiz', 'quiz-intro', 'collection-practice'].includes(String(resolved.name))) ||
        resolved.name === 'collection-detail'
    return isAppPage ? resolved.fullPath : '/translation'
})

onMounted(() => root.value?.focus())

async function startQuiz() {
    if (starting.value) return
    starting.value = true
    try {
        await router.push({ name: 'quiz' })
    } finally {
        starting.value = false
    }
}

async function closeIntro() {
    await router.push(returnTo.value)
    await nextTick()
    document.querySelector<HTMLElement>('[data-practice-trigger]')?.focus()
}
</script>

<template>
    <main
        ref="root"
        class="min-h-screen bg-background focus:outline-none"
        tabindex="-1"
        @keydown.enter.self.prevent="startQuiz"
        @keydown.esc.prevent="closeIntro"
    >
        <div class="border-b border-border px-4 py-3 sm:px-6">
            <div class="flex items-center justify-between gap-3">
                <span class="text-sm font-medium text-muted-foreground">{{ t.quizTitle }}</span>
                <button
                    :aria-label="t.quizBackToApp"
                    class="inline-flex size-11 items-center justify-center rounded-sm text-muted-foreground transition-colors hover:text-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                    @click="closeIntro"
                >
                    <X class="size-5" aria-hidden="true" />
                </button>
            </div>
        </div>

        <div class="flex min-h-[calc(100dvh-69px)] items-center justify-center px-4 py-10 sm:px-6 sm:py-12">
            <div class="w-full max-w-xl text-center">
                <h1 class="text-3xl font-semibold tracking-tight sm:text-4xl">{{ t.quizIntroTitle }}</h1>
                <p class="mt-4 text-base leading-7 text-muted-foreground">{{ t.quizIntroDescription }}</p>
                <Button size="lg" class="mt-8 min-h-11 min-w-44" :disabled="starting" @click="startQuiz">
                    <Play class="size-4" aria-hidden="true" />{{ starting ? t.quizStarting : t.quizStart }}
                </Button>
            </div>
        </div>
    </main>
</template>
