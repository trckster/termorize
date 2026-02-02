import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
    history: createWebHistory(),
    routes: [
        {
            path: '/login',
            name: 'login',
            component: () => import('@/views/LoginPage.vue'),
            meta: { guest: true },
        },
        {
            path: '/',
            name: 'translation',
            component: () => import('@/views/TranslationPage.vue'),
            meta: { requiresAuth: true },
        },
        {
            path: '/vocabulary',
            name: 'vocabulary',
            component: () => import('@/views/VocabularyPage.vue'),
            meta: { requiresAuth: true },
        },
    ],
})

router.beforeEach(async (to, _from, next) => {
    const authStore = useAuthStore()

    if (!authStore.hasCheckedAuth) {
        await authStore.getCurrentUser().catch(console.error)
    }

    if (to.meta.requiresAuth && !authStore.isAuthenticated) {
        next('/login')
    } else if (to.meta.guest && authStore.isAuthenticated) {
        next('/')
    } else {
        next()
    }
})

export default router
