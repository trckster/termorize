<template>
    <main class="px-4 py-4 sm:px-6 sm:py-8">
        <div class="mx-auto flex min-h-72 max-w-md flex-col items-center justify-center text-center">
            <div v-if="isLoading" class="flex flex-col items-center gap-3">
                <Loader2 class="h-6 w-6 animate-spin text-muted-foreground" />
                <p class="text-sm text-muted-foreground">{{ t.collectionJoinLoading }}</p>
            </div>

            <div v-else class="flex flex-col items-center gap-3">
                <h1 class="text-xl font-semibold text-foreground">{{ t.collectionJoinErrorTitle }}</h1>
                <p class="max-w-md text-sm text-muted-foreground">{{ t.collectionJoinErrorDescription }}</p>
                <Button variant="outline" class="mt-2" @click="router.push('/collections')">
                    {{ t.collectionJoinBack }}
                </Button>
            </div>
        </div>
    </main>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { collectionsApi } from '@/api/collections.ts'
import { useI18n } from '@/composables/useI18n'
import { Button } from '@/components/ui/button'
import { Loader2 } from 'lucide-vue-next'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const isLoading = ref(true)

onMounted(async () => {
    const token = route.params.token
    if (typeof token !== 'string' || !token) {
        isLoading.value = false
        return
    }

    try {
        const collection = await collectionsApi.joinByToken(token)
        await router.replace(`/collections/${collection.id}`)
    } catch {
        isLoading.value = false
    }
})
</script>
