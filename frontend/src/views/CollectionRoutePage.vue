<script setup lang="ts">
import Header from '@/components/Header.vue'
import BottomNav from '@/components/BottomNav.vue'
import CollectionDetailPage from '@/views/CollectionDetailPage.vue'
import PublicCollectionPreviewPage from '@/views/PublicCollectionPreviewPage.vue'
import { useAuthStore } from '@/stores/auth'
import { useRoute } from 'vue-router'

const authStore = useAuthStore()
const route = useRoute()
</script>

<template>
    <template v-if="authStore.isAuthenticated">
        <Header />
        <div class="pb-safe-nav">
            <CollectionDetailPage />
        </div>
        <BottomNav />
    </template>
    <PublicCollectionPreviewPage v-else mode="direct" :identifier="String(route.params.id || '')" />
</template>
