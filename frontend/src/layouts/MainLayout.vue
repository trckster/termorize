<template>
    <Header />
    <div class="pb-safe-nav">
        <router-view />
    </div>
    <BottomNav />
</template>

<script setup lang="ts">
import Header from '@/components/Header.vue'
import BottomNav from '@/components/BottomNav.vue'
</script>
