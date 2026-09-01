<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
    ArrowRight,
    BarChart3,
    BookmarkCheck,
    Clock3,
    EllipsisVertical,
    Languages,
    Play,
    Search,
    Send,
    X,
} from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import { getTelegramWebAppInitData, isTelegramWebApp } from '@/lib/telegram'
import { clearPostAuthPath } from '@/lib/postAuthRedirect'

type ActionSource =
    | 'telegram-auto'
    | 'guest-nav'
    | 'guest-hero'
    | 'telegram-hero'
    | 'telegram-panel'
    | 'guest-closing'
    | 'telegram-closing'

const router = useRouter()
const authStore = useAuthStore()

const isTelegramLoading = ref(false)
const isGuestLoading = ref(false)
const isLoading = computed(() => isTelegramLoading.value || isGuestLoading.value)
const error = ref<string | null>(null)
const isInsideTelegram = isTelegramWebApp()
const activeAction = ref<ActionSource | null>(null)
const telegramLoadingLabel = computed(() => (isInsideTelegram ? 'Signing in...' : 'Redirecting...'))
const actionStatus = computed(() => {
    if (isGuestLoading.value) return 'Preparing your practice...'
    if (isTelegramLoading.value) return telegramLoadingLabel.value
    return ''
})

const isActionBusy = (source: ActionSource) => activeAction.value === source && isLoading.value

const originalTitle = document.title

onMounted(() => {
    document.title = 'Termorize: Remember the words you look up'

    const initData = getTelegramWebAppInitData()
    if (initData) {
        void startTelegramLogin('telegram-auto')
    }
})

onBeforeUnmount(() => {
    document.title = originalTitle
})

const startTelegramLogin = async (source: ActionSource) => {
    if (isLoading.value) return

    try {
        clearPostAuthPath()
        error.value = null
        activeAction.value = source
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
        activeAction.value = null
    }
}

const tryWithoutRegistration = async (source: ActionSource) => {
    if (isLoading.value) return

    try {
        error.value = null
        activeAction.value = source
        isGuestLoading.value = true
        await authStore.continueAsGuest()
        await router.replace({ name: 'translation' })
    } catch (err) {
        error.value = getErrorMessage(err, 'Could not create a temporary account. Please try again.')
        isGuestLoading.value = false
        activeAction.value = null
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
        <svg width="0" height="0" aria-hidden="true" style="position: absolute">
            <defs>
                <linearGradient id="brandGrad" x1="0" y1="0" x2="1" y2="1">
                    <stop offset="0" stop-color="oklch(0.68 0.16 152)" />
                    <stop offset="1" stop-color="oklch(0.46 0.13 152)" />
                </linearGradient>
            </defs>
        </svg>

        <nav class="landing-nav pt-safe" aria-label="Primary navigation">
            <div class="landing-wrap nav-inner">
                <a class="brand" href="#top" aria-label="Termorize home">
                    <span class="brand-mark" aria-hidden="true">
                        <svg viewBox="0 0 64 64" fill="none">
                            <rect x="2" y="2" width="60" height="60" rx="17" fill="url(#brandGrad)" />
                            <g stroke="#06140c" stroke-width="6.2" stroke-linecap="round">
                                <g opacity="0.22" transform="translate(7 7)">
                                    <path d="M20 23h24M32 23v23" />
                                </g>
                                <g opacity="0.45" transform="translate(3.5 3.5)">
                                    <path d="M20 23h24M32 23v23" />
                                </g>
                                <path d="M20 23h24M32 23v23" />
                            </g>
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
                    :aria-busy="isActionBusy('guest-nav')"
                    @click="tryWithoutRegistration('guest-nav')"
                >
                    <span class="nav-label-full">{{ isActionBusy('guest-nav') ? 'Preparing...' : 'Just Try' }}</span>
                    <span class="nav-label-short">{{ isActionBusy('guest-nav') ? 'Loading...' : 'Just Try' }}</span>
                    <ArrowRight aria-hidden="true" />
                </button>
            </div>
        </nav>

        <p class="sr-only" role="status" aria-live="polite">{{ actionStatus }}</p>

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
                                Translate and save new words. Telegram sends exercises from your saved vocabulary
                                automatically every day.
                            </p>

                            <div class="hero-actions">
                                <button
                                    class="button button-primary button-large"
                                    type="button"
                                    :disabled="isLoading"
                                    :aria-busy="isActionBusy('guest-hero')"
                                    @click="tryWithoutRegistration('guest-hero')"
                                >
                                    <Play aria-hidden="true" />
                                    <span v-if="isActionBusy('guest-hero')">Preparing your practice...</span>
                                    <span v-else>Just Try</span>
                                </button>
                                <button
                                    class="button button-telegram button-large"
                                    type="button"
                                    :disabled="isLoading"
                                    :aria-busy="isActionBusy('telegram-hero')"
                                    @click="startTelegramLogin('telegram-hero')"
                                >
                                    <Send aria-hidden="true" />
                                    <span v-if="isActionBusy('telegram-hero')">{{ telegramLoadingLabel }}</span>
                                    <span v-else>Sign in with Telegram</span>
                                </button>
                            </div>
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
                                <div
                                    class="translation-preview"
                                    role="img"
                                    aria-label="Termorize translation workspace showing whisper translated from English to Russian"
                                >
                                    <div class="translation-card">
                                        <div class="translation-card-head">
                                            <span>From</span>
                                            <strong><span class="language-code">EN</span> English</strong>
                                        </div>
                                        <div class="translation-card-word">whisper</div>
                                        <div class="translation-card-meta">
                                            <span>Source word</span>
                                            <span>7 characters</span>
                                        </div>
                                    </div>

                                    <span class="translation-direction" aria-hidden="true"><ArrowRight /></span>

                                    <div class="translation-card">
                                        <div class="translation-card-head">
                                            <span>To</span>
                                            <strong><span class="language-code">RU</span> Russian</strong>
                                        </div>
                                        <div class="translation-card-word" lang="ru">шёпот</div>
                                        <div class="translation-card-meta">
                                            <span>Translation</span>
                                            <span>5 characters</span>
                                        </div>
                                    </div>

                                    <div class="translation-save-hint">
                                        <BookmarkCheck aria-hidden="true" />
                                        <span>Ready to save to your vocabulary</span>
                                        <kbd>Ctrl + S</kbd>
                                    </div>
                                </div>
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

                        <aside class="telegram-note" aria-label="Automatic Telegram exercise delivery example">
                            <span class="telegram-note-icon" aria-hidden="true"><Send /></span>
                            <span class="telegram-note-copy">
                                <strong>Sent automatically</strong>
                                <span>Random daily exercises</span>
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
                                <span class="step-icon"><Send aria-hidden="true" /></span>
                                <div>
                                    <h3>Let practice come to you.</h3>
                                    <p>
                                        Turn on daily delivery, and Telegram automatically sends exercises from your
                                        saved vocabulary.
                                    </p>
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

                        <div class="memory-panel" aria-label="English to Russian vocabulary progress example">
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
                                        <strong>memory</strong>
                                        <span class="word-arrow" aria-hidden="true"><ArrowRight /></span>
                                        <span lang="ru">память</span>
                                    </div>
                                    <div class="word-state">
                                        <div class="state-label"><span>Learning</span><strong>67%</strong></div>
                                        <span
                                            class="progress-track"
                                            role="progressbar"
                                            aria-label="Memory learning progress"
                                            aria-valuemin="0"
                                            aria-valuemax="100"
                                            aria-valuenow="67"
                                            ><span style="width: 67%"></span
                                        ></span>
                                    </div>
                                </div>
                                <div class="word-row">
                                    <div class="word-pair">
                                        <strong>curiosity</strong>
                                        <span class="word-arrow" aria-hidden="true"><ArrowRight /></span>
                                        <span lang="ru">любопытство</span>
                                    </div>
                                    <div class="word-state">
                                        <div class="state-label"><span>Learning</span><strong>31%</strong></div>
                                        <span
                                            class="progress-track"
                                            role="progressbar"
                                            aria-label="Curiosity learning progress"
                                            aria-valuemin="0"
                                            aria-valuemax="100"
                                            aria-valuenow="31"
                                            ><span style="width: 31%"></span
                                        ></span>
                                    </div>
                                </div>
                                <div class="word-row">
                                    <div class="word-pair">
                                        <strong>courage</strong>
                                        <span class="word-arrow" aria-hidden="true"><ArrowRight /></span>
                                        <span lang="ru">смелость</span>
                                    </div>
                                    <div class="word-state">
                                        <div class="state-label"><span>Learning</span><strong>94%</strong></div>
                                        <span
                                            class="progress-track"
                                            role="progressbar"
                                            aria-label="Courage learning progress"
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
                            <h2>Exercises arrive automatically in Telegram.</h2>
                            <p>
                                Choose how many exercises you want each day and the time windows that suit you.
                                Termorize picks from your saved vocabulary and sends each exercise separately at a
                                random time within those windows.
                            </p>

                            <ul class="telegram-facts">
                                <li><Clock3 aria-hidden="true" /> Random times in your schedule</li>
                                <li><BookmarkCheck aria-hidden="true" /> 1–100 exercises each day</li>
                                <li><Send aria-hidden="true" /> From your saved vocabulary</li>
                            </ul>

                            <button
                                class="button button-telegram button-large"
                                type="button"
                                :disabled="isLoading"
                                :aria-busy="isActionBusy('telegram-panel')"
                                @click="startTelegramLogin('telegram-panel')"
                            >
                                <Send aria-hidden="true" />
                                <span v-if="isActionBusy('telegram-panel')">{{ telegramLoadingLabel }}</span>
                                <span v-else>Sign in with Telegram</span>
                            </button>
                        </div>

                        <div class="telegram-demo" role="group" aria-label="Telegram exercise preview">
                            <div class="chat-head">
                                <a
                                    class="chat-profile"
                                    href="https://t.me/termorize_bot"
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    aria-label="Open @termorize_bot on Telegram (opens in a new tab)"
                                >
                                    <span class="chat-avatar" aria-hidden="true">
                                        <img src="/favicon.svg" alt="" width="39" height="39" />
                                    </span>
                                    <span class="chat-identity">
                                        <strong>Termorize</strong>
                                        <small>@termorize_bot</small>
                                    </span>
                                </a>
                                <EllipsisVertical class="chat-menu" aria-hidden="true" />
                            </div>
                            <div class="chat-body">
                                <div class="chat-bubble">
                                    <p class="exercise-prompt">
                                        Translate word <strong>whisper</strong> to <span>🇷🇺 Russian</span>
                                    </p>
                                    <span class="exercise-instruction">Choose one of the options below.</span>
                                    <div class="exercise-options">
                                        <span class="exercise-option" lang="ru">голос</span>
                                        <span class="exercise-option" lang="ru">шёпот</span>
                                        <span class="exercise-option" lang="ru">ветер</span>
                                        <span class="exercise-option" lang="ru">тишина</span>
                                    </div>
                                    <span class="message-time">now</span>
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
                            :aria-busy="isActionBusy('guest-closing')"
                            @click="tryWithoutRegistration('guest-closing')"
                        >
                            <Play aria-hidden="true" />
                            <span v-if="isActionBusy('guest-closing')">Preparing your practice...</span>
                            <span v-else>Just Try</span>
                        </button>
                        <button
                            class="button button-telegram button-large"
                            type="button"
                            :disabled="isLoading"
                            :aria-busy="isActionBusy('telegram-closing')"
                            @click="startTelegramLogin('telegram-closing')"
                        >
                            <Send aria-hidden="true" />
                            <span v-if="isActionBusy('telegram-closing')">{{ telegramLoadingLabel }}</span>
                            <span v-else>Sign in with Telegram</span>
                        </button>
                    </div>
                </div>
            </section>
        </main>

        <footer class="landing-footer">
            <div class="landing-wrap footer-inner">
                <a class="brand" href="#top" aria-label="Termorize — back to top">
                    <span class="brand-mark" aria-hidden="true">
                        <svg viewBox="0 0 64 64" fill="none">
                            <rect x="2" y="2" width="60" height="60" rx="17" fill="url(#brandGrad)" />
                            <g stroke="#06140c" stroke-width="6.2" stroke-linecap="round">
                                <g opacity="0.22" transform="translate(7 7)">
                                    <path d="M20 23h24M32 23v23" />
                                </g>
                                <g opacity="0.45" transform="translate(3.5 3.5)">
                                    <path d="M20 23h24M32 23v23" />
                                </g>
                                <path d="M20 23h24M32 23v23" />
                            </g>
                        </svg>
                    </span>
                    <span>Termorize</span>
                </a>
                <div class="footer-meta">
                    <a class="footer-link" href="/extension-privacy.html">Extension privacy</a>
                    <span class="footer-note">© 2026 · Vocabulary exercises, delivered daily</span>
                </div>
            </div>
        </footer>
    </div>
</template>

<style src="../assets/landing.css"></style>
