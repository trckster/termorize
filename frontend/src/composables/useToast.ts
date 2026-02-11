import { ref, computed } from 'vue'

export interface Toast {
    id: string
    title: string
    description?: string
    variant?: 'default' | 'success' | 'destructive'
    duration?: number
}

const toasts = ref<Toast[]>([])

export function useToast() {
    const addToast = (toast: Omit<Toast, 'id'>) => {
        const id = Math.random().toString(36).substring(2, 9)
        const newToast: Toast = {
            id,
            duration: 5000,
            ...toast,
        }
        toasts.value.push(newToast)

        setTimeout(() => {
            removeToast(id)
        }, newToast.duration)
    }

    const removeToast = (id: string) => {
        const index = toasts.value.findIndex((t) => t.id === id)
        if (index > -1) {
            toasts.value.splice(index, 1)
        }
    }

    return {
        toasts: computed(() => toasts.value),
        addToast,
        removeToast,
    }
}
