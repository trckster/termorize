import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
    history: createWebHistory(),
    routes: [
        {
            path: '/',
            name: 'root',
            component: () => import('@/views/RootView.vue'),
        },
        {
            path: '/login',
            name: 'login',
            component: () => import('@/views/LoginPage.vue'),
            meta: { guest: true },
        },
        {
            path: '/login/telegram/callback',
            name: 'telegram-login-callback',
            component: () => import('@/views/TelegramLoginCallbackPage.vue'),
            meta: { guest: true },
        },
        {
            path: '/quiz',
            name: 'quiz',
            component: () => import('@/views/QuizPage.vue'),
            meta: { requiresAuth: true },
        },
        {
            path: '/',
            component: () => import('@/layouts/MainLayout.vue'),
            meta: { requiresAuth: true },
            children: [
                {
                    path: 'translation',
                    name: 'translation',
                    component: () => import('@/views/TranslationPage.vue'),
                },
                {
                    path: 'vocabulary',
                    name: 'vocabulary',
                    component: () => import('@/views/VocabularyPage.vue'),
                },
                {
                    path: 'collections',
                    name: 'collections',
                    component: () => import('@/views/CollectionsPage.vue'),
                },
                {
                    path: 'collections/join/:token',
                    name: 'collection-join',
                    component: () => import('@/views/CollectionJoinPage.vue'),
                },
                {
                    path: 'collections/:id',
                    name: 'collection-detail',
                    component: () => import('@/views/CollectionDetailPage.vue'),
                },
                {
                    path: 'exercises',
                    name: 'exercises',
                    component: () => import('@/views/ExercisesPage.vue'),
                },
                {
                    path: 'statistics',
                    name: 'statistics',
                    component: () => import('@/views/StatisticsPage.vue'),
                },
                {
                    path: 'settings',
                    name: 'settings',
                    component: () => import('@/views/SettingsPage.vue'),
                },
            ],
        },
    ],
})

let lastVersionCheck = 0
const VERSION_CHECK_INTERVAL = 60_000

const hasNewAppVersion = async () => {
    if (!import.meta.env.PROD || Date.now() - lastVersionCheck < VERSION_CHECK_INTERVAL) {
        return false
    }

    lastVersionCheck = Date.now()

    try {
        const response = await fetch('/', {
            cache: 'no-store',
            headers: { Accept: 'text/html' },
        })

        if (!response.ok) {
            return false
        }

        const html = await response.text()
        const version = html.match(/<meta name="app-version" content="([^"]+)">/)?.[1]

        return Boolean(version && version !== __APP_VERSION__)
    } catch {
        // A version check must never prevent navigation while offline or during a deployment.
        return false
    }
}

router.beforeEach(async (to, _from, next) => {
    if (await hasNewAppVersion()) {
        window.location.assign(to.fullPath)
        next(false)
        return
    }

    const authStore = useAuthStore()

    if (!authStore.hasCheckedAuth) {
        await authStore.getCurrentUser().catch(console.error)
    }

    if (to.meta.requiresAuth && !authStore.isAuthenticated) {
        next('/')
    } else if (to.meta.guest && authStore.isAuthenticated) {
        next({ name: 'translation' })
    } else {
        next()
    }
})

export default router
