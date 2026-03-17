import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { PHONE_MEDIA_QUERY } from '@/lib/screen.ts'

export function usePhoneViewport() {
    const isPhoneViewport = ref(false)
    let mediaQueryList: MediaQueryList | null = null

    const syncViewport = () => {
        isPhoneViewport.value = mediaQueryList?.matches ?? false
    }

    onMounted(() => {
        mediaQueryList = window.matchMedia(PHONE_MEDIA_QUERY)
        syncViewport()
        mediaQueryList.addEventListener('change', syncViewport)
    })

    onBeforeUnmount(() => {
        mediaQueryList?.removeEventListener('change', syncViewport)
    })

    return {
        isPhoneViewport: computed(() => isPhoneViewport.value),
    }
}
