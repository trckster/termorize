<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ArrowRight, BarChart3, BookmarkCheck, Check, Clock3, Languages, Play, Search, Send, X } from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import { getTelegramWebAppInitData, isTelegramWebApp } from '@/lib/telegram'
import translationPreview from '@/assets/landing-translation.png'

const router = useRouter()
const authStore = useAuthStore()

const isTelegramLoading = ref(false)
const isGuestLoading = ref(false)
const isLoading = computed(() => isTelegramLoading.value || isGuestLoading.value)
const error = ref<string | null>(null)
const isInsideTelegram = isTelegramWebApp()

const originalTitle = document.title

onMounted(() => {
    document.title = 'Termorize: Remember the words you look up'

    const initData = getTelegramWebAppInitData()
    if (initData) {
        void startTelegramLogin()
    }
})

onBeforeUnmount(() => {
    document.title = originalTitle
})

const startTelegramLogin = async () => {
    if (isLoading.value) return

    try {
        error.value = null
        isTelegramLoading.value = true

        const initData = getTelegramWebAppInitData()
        if (initData) {
            await authStore.completeTelegramLogin({ init_data: initData })
            await router.replace({ name: 'translation' })
            return
        }

        const authUrl = await authStore.startTelegramLogin()
        window.location.assign(authUrl)
    } catch (err) {
        error.value = getErrorMessage(err)
        isTelegramLoading.value = false
    }
}

const tryWithoutRegistration = async () => {
    if (isLoading.value) return

    try {
        error.value = null
        isGuestLoading.value = true
        await authStore.continueAsGuest()
        await router.replace({ name: 'translation' })
    } catch (err) {
        error.value = getErrorMessage(err, 'Could not create a temporary account. Please try again.')
        isGuestLoading.value = false
    }
}

function getErrorMessage(err: unknown, fallback = 'Login failed. Please try again.'): string {
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
    <div class="landing-view">
        <nav class="landing-nav pt-safe" aria-label="Primary navigation">
            <div class="landing-wrap nav-inner">
                <a class="brand" href="#top" aria-label="Termorize home">
                    <span class="brand-mark" aria-hidden="true">
                        <svg viewBox="0 0 40 40" fill="none">
                            <rect x="1" y="1" width="38" height="38" rx="10" fill="currentColor" />
                            <path
                                d="M12 14h17M20.5 14v16"
                                stroke="hsl(var(--primary-foreground))"
                                stroke-width="4"
                                stroke-linecap="round"
                            />
                            <path
                                d="M12 20h12"
                                stroke="hsl(var(--primary-foreground))"
                                stroke-width="2.4"
                                stroke-linecap="round"
                                opacity=".48"
                            />
                        </svg>
                    </span>
                    <span>Termorize</span>
                </a>

                <div class="nav-links">
                    <a href="#how">How it works</a>
                    <a href="#telegram">Telegram</a>
                </div>

                <button
                    class="button button-primary nav-button"
                    type="button"
                    :disabled="isLoading"
                    @click="tryWithoutRegistration"
                >
                    <span class="nav-label-full">Start practicing</span>
                    <span class="nav-label-short">Try it</span>
                    <ArrowRight aria-hidden="true" />
                </button>
            </div>
        </nav>

        <div v-if="error" class="login-error" role="alert" aria-live="assertive">
            <span>{{ error }}</span>
            <button type="button" aria-label="Dismiss error" @click="error = null">
                <X aria-hidden="true" />
            </button>
        </div>

        <main>
            <header id="top" class="hero">
                <div class="landing-wrap">
                    <div class="hero-intro">
                        <h1>Remember more of the words you look up.</h1>

                        <div class="hero-pitch">
                            <p>
                                Translate a word, save it, and practice at the right pace—on the web or through
                                Telegram.
                            </p>

                            <div class="hero-actions">
                                <button
                                    class="button button-primary button-large"
                                    type="button"
                                    :disabled="isLoading"
                                    @click="tryWithoutRegistration"
                                >
                                    <Play aria-hidden="true" />
                                    <span v-if="isGuestLoading">Preparing your practice...</span>
                                    <span v-else>Start practicing</span>
                                </button>
                                <button
                                    class="button button-secondary button-large"
                                    type="button"
                                    :disabled="isLoading"
                                    @click="startTelegramLogin"
                                >
                                    <Send aria-hidden="true" />
                                    <span v-if="isTelegramLoading">{{
                                        isInsideTelegram ? 'Signing in...' : 'Redirecting...'
                                    }}</span>
                                    <span v-else>Use Telegram</span>
                                </button>
                            </div>

                            <ul class="access-facts" aria-label="Guest access details">
                                <li><Check aria-hidden="true" /> No signup</li>
                                <li>50 example words</li>
                                <li>7 days of guest access</li>
                            </ul>
                        </div>
                    </div>

                    <div class="product-showcase">
                        <div class="app-frame">
                            <div class="app-frame-bar">
                                <span class="live-dot" aria-hidden="true"></span>
                                <span>Termorize web app</span>
                                <span class="app-frame-view">Translation</span>
                            </div>

                            <div class="app-preview">
                                <img
                                    :src="translationPreview"
                                    alt="Termorize translation workspace showing an English to Italian translation"
                                />
                            </div>

                            <div class="preview-flow" aria-label="Learning flow">
                                <div class="preview-step">
                                    <Languages aria-hidden="true" />
                                    <span><strong>Translate</strong> in your language pair</span>
                                </div>
                                <div class="preview-step">
                                    <BookmarkCheck aria-hidden="true" />
                                    <span><strong>Save</strong> it to your vocabulary</span>
                                </div>
                                <div class="preview-step">
                                    <Play aria-hidden="true" />
                                    <span><strong>Practice</strong> when it is due</span>
                                </div>
                            </div>
                        </div>

                        <aside class="telegram-note" aria-label="Telegram practice reminder example">
                            <span class="telegram-note-icon" aria-hidden="true"><Send /></span>
                            <span class="telegram-note-copy">
                                <strong>Practice is ready</strong>
                                <span>10 words · about 3 min</span>
                            </span>
                            <ArrowRight aria-hidden="true" />
                        </aside>
                    </div>
                </div>
            </header>

            <section id="how" class="learning-loop">
                <div class="landing-wrap">
                    <div class="section-intro">
                        <h2>A short path from lookup to recall.</h2>
                        <p>
                            Termorize keeps the useful parts of vocabulary learning together, so every saved word has a
                            clear next step.
                        </p>
                    </div>

                    <div class="loop-layout">
                        <ol class="loop-steps">
                            <li>
                                <span class="step-icon"><Search aria-hidden="true" /></span>
                                <div>
                                    <h3>Capture it while it is fresh.</h3>
                                    <p>Translate with your chosen language pair and save the result in one shortcut.</p>
                                </div>
                            </li>
                            <li>
                                <span class="step-icon"><Play aria-hidden="true" /></span>
                                <div>
                                    <h3>Bring it back into practice.</h3>
                                    <p>Run a quick exercise on the web or let Telegram deliver it on schedule.</p>
                                </div>
                            </li>
                            <li>
                                <span class="step-icon"><BarChart3 aria-hidden="true" /></span>
                                <div>
                                    <h3>See what needs another pass.</h3>
                                    <p>Progress stays attached to each word, keeping your next review focused.</p>
                                </div>
                            </li>
                        </ol>

                        <div class="memory-panel" aria-label="Vocabulary progress example">
                            <div class="memory-panel-head">
                                <div>
                                    <span>Vocabulary</span>
                                    <strong>Saved words</strong>
                                </div>
                                <span class="search-control"><Search aria-hidden="true" /> Search</span>
                            </div>

                            <div class="word-list">
                                <div class="word-row">
                                    <div class="word-pair">
                                        <strong>resilient</strong>
                                        <span>resiliente</span>
                                    </div>
                                    <div class="word-state">
                                        <div class="state-label"><span>Learning</span><strong>67%</strong></div>
                                        <span
                                            class="progress-track"
                                            role="progressbar"
                                            aria-label="Resilient learning progress"
                                            aria-valuemin="0"
                                            aria-valuemax="100"
                                            aria-valuenow="67"
                                            ><span style="width: 67%"></span
                                        ></span>
                                    </div>
                                </div>
                                <div class="word-row">
                                    <div class="word-pair">
                                        <strong>serendipity</strong>
                                        <span>serendipità</span>
                                    </div>
                                    <div class="word-state">
                                        <div class="state-label"><span>Learning</span><strong>31%</strong></div>
                                        <span
                                            class="progress-track"
                                            role="progressbar"
                                            aria-label="Serendipity learning progress"
                                            aria-valuemin="0"
                                            aria-valuemax="100"
                                            aria-valuenow="31"
                                            ><span style="width: 31%"></span
                                        ></span>
                                    </div>
                                </div>
                                <div class="word-row">
                                    <div class="word-pair">
                                        <strong>eloquent</strong>
                                        <span>eloquente</span>
                                    </div>
                                    <div class="word-state">
                                        <div class="state-label"><span>Learning</span><strong>94%</strong></div>
                                        <span
                                            class="progress-track"
                                            role="progressbar"
                                            aria-label="Eloquent learning progress"
                                            aria-valuemin="0"
                                            aria-valuemax="100"
                                            aria-valuenow="94"
                                            ><span style="width: 94%"></span
                                        ></span>
                                    </div>
                                </div>
                            </div>

                            <div class="memory-panel-foot">
                                <span><BookmarkCheck aria-hidden="true" /> Personal vocabulary</span>
                                <span><BarChart3 aria-hidden="true" /> Progress per word</span>
                            </div>
                        </div>
                    </div>
                </div>
            </section>

            <section id="telegram" class="telegram-section">
                <div class="landing-wrap">
                    <div class="telegram-panel">
                        <div class="telegram-copy">
                            <h2>Your next review can arrive in Telegram.</h2>
                            <p>
                                Pick the time and daily count. The bot practices from the same vocabulary you use on the
                                web.
                            </p>

                            <ul class="telegram-facts">
                                <li><Clock3 aria-hidden="true" /> Your timezone</li>
                                <li><Play aria-hidden="true" /> 1–100 exercises</li>
                                <li><Check aria-hidden="true" /> One shared account</li>
                            </ul>

                            <button
                                class="button button-telegram button-large"
                                type="button"
                                :disabled="isLoading"
                                @click="startTelegramLogin"
                            >
                                <Send aria-hidden="true" />
                                <span v-if="isTelegramLoading">{{
                                    isInsideTelegram ? 'Signing in...' : 'Redirecting...'
                                }}</span>
                                <span v-else>Continue with Telegram</span>
                            </button>
                        </div>

                        <div class="telegram-demo" aria-hidden="true">
                            <div class="chat-head">
                                <span class="chat-avatar"><span>T</span></span>
                                <span><strong>Termorize</strong><small>bot</small></span>
                                <span class="chat-time">08:30</span>
                            </div>
                            <div class="chat-body">
                                <div class="chat-bubble">
                                    <small>Daily practice</small>
                                    <p>Choose the translation for <strong>resilient</strong></p>
                                    <div class="answer-list">
                                        <span class="answer selected"><Check /> resiliente</span>
                                        <span class="answer">serendipità</span>
                                        <span class="answer">eloquente</span>
                                    </div>
                                    <span class="chat-progress">1 of 10</span>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </section>

            <section class="closing-section">
                <div class="landing-wrap closing-inner">
                    <h2>Make the next word stick.</h2>
                    <div class="closing-actions">
                        <button
                            class="button button-primary button-large"
                            type="button"
                            :disabled="isLoading"
                            @click="tryWithoutRegistration"
                        >
                            <Play aria-hidden="true" />
                            <span v-if="isGuestLoading">Preparing your practice...</span>
                            <span v-else>Start without signup</span>
                        </button>
                        <button
                            class="button button-secondary button-large"
                            type="button"
                            :disabled="isLoading"
                            @click="startTelegramLogin"
                        >
                            <Send aria-hidden="true" />
                            <span>Use Telegram</span>
                        </button>
                    </div>
                </div>
            </section>
        </main>

        <footer class="landing-footer">
            <div class="landing-wrap footer-inner">
                <a class="brand" href="#top" aria-label="Back to top">
                    <span class="brand-mark" aria-hidden="true">
                        <svg viewBox="0 0 40 40" fill="none">
                            <rect x="1" y="1" width="38" height="38" rx="10" fill="currentColor" />
                            <path
                                d="M12 14h17M20.5 14v16"
                                stroke="hsl(var(--primary-foreground))"
                                stroke-width="4"
                                stroke-linecap="round"
                            />
                            <path
                                d="M12 20h12"
                                stroke="hsl(var(--primary-foreground))"
                                stroke-width="2.4"
                                stroke-linecap="round"
                                opacity=".48"
                            />
                        </svg>
                    </span>
                    <span>Termorize</span>
                </a>
                <span class="footer-note">© 2026 · Vocabulary practice on web and Telegram</span>
            </div>
        </footer>
    </div>
</template>

<style src="../assets/landing.css"></style>
