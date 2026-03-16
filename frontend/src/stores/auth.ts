import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { authApi, type TelegramLoginCallbackPayload, type User } from '@/api/auth.ts'

export const useAuthStore = defineStore('auth', () => {
    const user = ref<User | null>(null)
    const isAuthenticated = computed(() => !!user.value)
    const hasCheckedAuth = ref(false)

    const startTelegramLogin = async () => {
        return authApi.startTelegramLogin()
    }

    const completeTelegramLogin = async (payload: TelegramLoginCallbackPayload) => {
        user.value = await authApi.completeTelegramLogin(payload)
        hasCheckedAuth.value = true
    }

    const logout = async () => {
        await authApi.logout()
        user.value = null
    }

    const getCurrentUser = async () => {
        if (isAuthenticated.value) {
            hasCheckedAuth.value = true
            return user.value
        }

        user.value = await authApi.getCurrentUser().catch(() => null)
        hasCheckedAuth.value = true
        return user.value
    }

    return {
        user,
        isAuthenticated,
        hasCheckedAuth,
        startTelegramLogin,
        completeTelegramLogin,
        logout,
        getCurrentUser,
    }
})
