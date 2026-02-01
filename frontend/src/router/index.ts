import { createRouter, createWebHistory } from 'vue-router'

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
            name: 'account',
            component: () => import('@/views/AccountPage.vue'),
            meta: { requiresAuth: true },
        },
    ],
})

router.beforeEach((to, _from, next) => {
    // TODO Auth logic
    next()
})

export default router
