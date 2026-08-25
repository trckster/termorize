<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { adminApi, type AdminUser } from '@/api/admin.ts'
import { useI18n } from '@/composables/useI18n'
import { formatDate, formatNumber, formatRelativeTime } from '@/lib/utils.ts'

const { t } = useI18n()
const users = ref<AdminUser[]>([])
const total = ref(0)
const isLoading = ref(true)
const hasError = ref(false)

const fetchUsers = async () => {
    isLoading.value = true
    hasError.value = false

    try {
        const response = await adminApi.getUsers()
        users.value = response.data
        total.value = response.total
    } catch {
        hasError.value = true
    } finally {
        isLoading.value = false
    }
}

onMounted(() => void fetchUsers())
</script>

<template>
    <main class="px-4 py-6 sm:px-6 sm:py-8">
        <div class="mx-auto max-w-6xl space-y-6">
            <header class="flex items-end justify-between gap-6">
                <div>
                    <h1 class="text-2xl font-semibold tracking-tight text-foreground">{{ t.usersHeading }}</h1>
                    <p class="mt-1.5 max-w-2xl text-sm leading-6 text-muted-foreground">
                        {{ t.usersDescription }}
                    </p>
                </div>

                <dl class="shrink-0 text-right">
                    <dt class="text-xs font-medium text-muted-foreground">{{ t.usersTotal }}</dt>
                    <dd class="mt-1 text-2xl font-semibold tabular-nums text-foreground">
                        <span
                            v-if="isLoading"
                            class="inline-block h-7 w-12 animate-pulse rounded bg-muted motion-reduce:animate-none"
                        />
                        <template v-else>{{ formatNumber(total) }}</template>
                    </dd>
                </dl>
            </header>

            <div
                v-if="hasError"
                role="alert"
                class="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-destructive/25 bg-destructive/5 px-4 py-3 text-sm text-destructive"
            >
                <span>{{ t.usersLoadError }}</span>
                <button
                    type="button"
                    class="font-medium underline underline-offset-4 hover:no-underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                    @click="fetchUsers"
                >
                    {{ t.commonRetry }}
                </button>
            </div>

            <div v-else class="overflow-hidden rounded-xl border border-border bg-card">
                <div v-if="isLoading" class="divide-y divide-border" aria-label="Loading users">
                    <div v-for="index in 8" :key="index" class="flex gap-6 px-4 py-4 sm:px-5">
                        <span class="h-5 w-28 animate-pulse rounded bg-muted motion-reduce:animate-none" />
                        <span
                            class="hidden h-5 flex-1 animate-pulse rounded bg-muted motion-reduce:animate-none sm:block"
                        />
                        <span class="ml-auto h-5 w-32 animate-pulse rounded bg-muted motion-reduce:animate-none" />
                    </div>
                </div>

                <div v-else-if="users.length === 0" class="px-6 py-16 text-center">
                    <p class="text-sm font-medium text-foreground">{{ t.usersEmpty }}</p>
                    <p class="mt-1 text-sm text-muted-foreground">{{ t.usersEmptyDescription }}</p>
                </div>

                <div v-else class="overflow-x-auto">
                    <table class="w-full min-w-[44rem] text-left text-sm">
                        <thead class="border-b border-border bg-muted/35 text-xs font-medium text-muted-foreground">
                            <tr>
                                <th scope="col" class="px-5 py-3">{{ t.usersId }}</th>
                                <th scope="col" class="px-5 py-3">{{ t.usersName }}</th>
                                <th scope="col" class="px-5 py-3">{{ t.usersUsername }}</th>
                                <th scope="col" class="px-5 py-3 text-right">{{ t.usersVocabularySize }}</th>
                                <th scope="col" class="px-5 py-3 text-right">{{ t.usersLatestUsage }}</th>
                            </tr>
                        </thead>
                        <tbody class="divide-y divide-border">
                            <tr
                                v-for="user in users"
                                :key="user.id"
                                :class="user.deleted_at ? 'bg-muted/15 hover:bg-muted/30' : 'hover:bg-muted/25'"
                            >
                                <td class="px-5 py-3.5 font-mono text-xs tabular-nums text-muted-foreground">
                                    {{ user.id }}
                                </td>
                                <td class="px-5 py-3.5 font-medium text-foreground">
                                    <div class="flex items-center gap-2">
                                        <span>{{ user.name || t.usersNotAvailable }}</span>
                                        <span
                                            v-if="user.deleted_at"
                                            :title="formatDate(user.deleted_at)"
                                            class="inline-flex shrink-0 items-center rounded-md border border-destructive/30 bg-destructive/10 px-1.5 py-0.5 text-[0.6875rem] font-semibold leading-none text-destructive"
                                        >
                                            {{ t.usersDeleted }}
                                        </span>
                                    </div>
                                </td>
                                <td class="px-5 py-3.5 text-muted-foreground">
                                    {{ user.username ? `@${user.username}` : t.usersNotAvailable }}
                                </td>
                                <td class="px-5 py-3.5 text-right tabular-nums text-foreground">
                                    {{ formatNumber(user.vocabulary_size) }}
                                </td>
                                <td class="px-5 py-3.5 text-right">
                                    <time
                                        v-if="user.latest_usage"
                                        :datetime="user.latest_usage"
                                        :title="formatDate(user.latest_usage)"
                                        class="text-foreground"
                                    >
                                        {{ formatRelativeTime(user.latest_usage) }}
                                    </time>
                                    <span v-else class="text-muted-foreground">{{ t.usersNotAvailable }}</span>
                                </td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
    </main>
</template>
