<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Loader2 } from 'lucide-vue-next'
import { collectionsApi } from '@/api/collections'
import { Button } from '@/components/ui/button'
import Header from '@/components/Header.vue'
import BottomNav from '@/components/BottomNav.vue'
import PublicCollectionPreviewPage from '@/views/PublicCollectionPreviewPage.vue'
import { useAuthStore } from '@/stores/auth'
import { useI18n } from '@/composables/useI18n'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const isLoading = ref(true)
const hasError = ref(false)
let activeIdentifier = ''

watch(
    [() => authStore.isAuthenticated, () => route.params.shareId],
    async ([isAuthenticated, routeIdentifier]) => {
        if (!isAuthenticated) {
            isLoading.value = false
            hasError.value = false
            activeIdentifier = ''
            return
        }

        const identifier = typeof routeIdentifier === 'string' ? routeIdentifier : ''
        if (!identifier || identifier === activeIdentifier) return

        activeIdentifier = identifier
        isLoading.value = true
        hasError.value = false

        try {
            const collection = await collectionsApi.joinByShareIdentifier(identifier)
            await router.replace(`/collections/${collection.id}`)
        } catch {
            hasError.value = true
            isLoading.value = false
        }
    },
    { immediate: true }
)
</script>

<template>
    <template v-if="!authStore.isAuthenticated">
        <PublicCollectionPreviewPage mode="join" :identifier="String(route.params.shareId || '')" />
    </template>

    <template v-else>
        <Header />
        <div class="pb-safe-nav">
            <main class="px-4 py-4 sm:px-6 sm:py-8">
                <div class="mx-auto flex min-h-72 max-w-md flex-col items-center justify-center text-center">
                    <div v-if="isLoading" class="flex flex-col items-center gap-3" role="status">
                        <Loader2 class="h-6 w-6 animate-spin text-muted-foreground motion-reduce:animate-none" />
                        <p class="text-sm text-muted-foreground">{{ t.collectionJoinLoading }}</p>
                    </div>

                    <div v-else-if="hasError" class="flex flex-col items-center gap-3">
                        <h1 class="text-xl font-semibold text-foreground">{{ t.collectionJoinErrorTitle }}</h1>
                        <p class="max-w-md text-sm text-muted-foreground">{{ t.collectionJoinErrorDescription }}</p>
                        <Button variant="outline" class="mt-2" @click="router.push('/collections')">
                            {{ t.collectionJoinBack }}
                        </Button>
                    </div>
                </div>
            </main>
        </div>
        <BottomNav />
    </template>
</template>
