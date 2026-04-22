<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { BarChart3, BookOpen, Languages, Send } from 'lucide-vue-next'
import { useRouter } from 'vue-router'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import TelegramLogin from '@/components/TelegramLogin.vue'
import { getTelegramWebAppInitData, isTelegramWebApp } from '@/lib/telegram'
import { useAuthStore } from '@/stores/auth'
import { useI18n } from '@/composables/useI18n'

const router = useRouter()
const authStore = useAuthStore()
const { t } = useI18n()
const error = ref<string | null>(null)
const isLoading = ref(false)
const isInsideTelegram = isTelegramWebApp()
const featureItems = [
    {
        key: 'translate',
        titleKey: 'loginFeatureTranslateTitle' as const,
        descriptionKey: 'loginFeatureTranslateDescription' as const,
        icon: Languages,
    },
    {
        key: 'vocabulary',
        titleKey: 'loginFeatureVocabularyTitle' as const,
        descriptionKey: 'loginFeatureVocabularyDescription' as const,
        icon: BookOpen,
    },
    {
        key: 'practice',
        titleKey: 'loginFeaturePracticeTitle' as const,
        descriptionKey: 'loginFeaturePracticeDescription' as const,
        icon: BarChart3,
    },
    {
        key: 'telegram',
        titleKey: 'loginFeatureTelegramTitle' as const,
        descriptionKey: 'loginFeatureTelegramDescription' as const,
        icon: Send,
    },
]

onMounted(() => {
    if (getTelegramWebAppInitData()) {
        void startTelegramLogin()
    }
})

const startTelegramLogin = async () => {
    if (isLoading.value) {
        return
    }

    try {
        error.value = null
        isLoading.value = true

        const initData = getTelegramWebAppInitData()
        if (initData) {
            await authStore.completeTelegramLogin({ init_data: initData })
            await router.replace('/')
            return
        }

        const authUrl = await authStore.startTelegramLogin()
        window.location.assign(authUrl)
    } catch (err) {
        error.value = getErrorMessage(err, t.value.loginStartError)
        isLoading.value = false
    }
}

function getErrorMessage(err: unknown, fallback: string): string {
    if (err instanceof Error) {
        return err.message
    }

    if (typeof err === 'object' && err !== null && 'body' in err) {
        const body = (err as { body?: { error?: string; details?: string; message?: string } }).body
        return body?.details || body?.error || body?.message || fallback
    }

    return fallback
}
</script>

<template>
    <div class="min-h-screen bg-background px-4 py-10 sm:px-6 lg:px-8">
        <div
            class="mx-auto grid min-h-[calc(100vh-5rem)] max-w-6xl items-center gap-8 lg:grid-cols-[minmax(0,1fr)_26rem]"
        >
            <section class="space-y-6">
                <div class="space-y-4">
                    <p class="text-sm font-medium uppercase tracking-[0.25em] text-primary/80">{{ t.loginEyebrow }}</p>
                    <div class="space-y-3">
                        <h1 class="max-w-3xl text-4xl font-semibold tracking-tight text-foreground sm:text-5xl">
                            {{ t.loginHeroTitle }}
                        </h1>
                        <p class="max-w-2xl text-base text-muted-foreground sm:text-lg">
                            {{ t.loginHeroDescription }}
                        </p>
                    </div>
                </div>

                <div class="grid gap-4 sm:grid-cols-2">
                    <article
                        v-for="item in featureItems"
                        :key="item.key"
                        class="rounded-2xl border border-border/70 bg-card/70 p-5 shadow-sm"
                    >
                        <div
                            class="mb-4 flex h-11 w-11 items-center justify-center rounded-2xl bg-primary/10 text-primary"
                        >
                            <component :is="item.icon" class="h-5 w-5" />
                        </div>
                        <h2 class="text-base font-semibold text-foreground">{{ t[item.titleKey] }}</h2>
                        <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ t[item.descriptionKey] }}</p>
                    </article>
                </div>
            </section>

            <Card class="w-full border-border/70 bg-card/95 shadow-xl backdrop-blur-sm">
                <CardHeader class="space-y-3 text-center">
                    <CardTitle class="text-2xl font-bold text-foreground">{{ t.loginTitle }}</CardTitle>
                    <CardDescription class="text-muted-foreground">
                        {{ t.loginDescription }}
                    </CardDescription>
                </CardHeader>
                <CardContent class="flex flex-col items-center gap-4 pt-2">
                    <div
                        class="rounded-2xl border border-border/70 bg-muted/30 px-4 py-3 text-sm text-muted-foreground"
                    >
                        {{ t.loginCardNote }}
                    </div>
                    <TelegramLogin
                        class="mt-4"
                        :loading="isLoading"
                        :inside-telegram="isInsideTelegram"
                        @start="startTelegramLogin"
                    />
                    <div v-if="error" class="text-center text-sm text-destructive">{{ error }}</div>
                </CardContent>
            </Card>
        </div>
    </div>
</template>
