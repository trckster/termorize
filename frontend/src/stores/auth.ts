import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { authApi, type User } from '@/api/auth.ts'

export const useAuthStore = defineStore('auth', () => {
    const user = ref<User | null>(null)
    const isAuthenticated = computed(() => !!user.value)
    const hasCheckedAuth = ref(false)

    const login = async (authData: any) => {
        user.value = await authApi.login(authData)
    }

    const logout = async () => {
        await authApi.logout()
        user.value = null
    }

    const getCurrentUser = async () => {
        if (isAuthenticated.value) {
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
        login,
        logout,
        getCurrentUser,
    }
})
