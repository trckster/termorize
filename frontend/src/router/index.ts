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

router.beforeEach(async (to, _from, next) => {
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
