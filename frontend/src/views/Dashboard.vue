<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import Card from '@/components/ui/Card.vue'
import { Server, Wifi, WifiOff, Monitor } from 'lucide-vue-next'

const store = useAppStore()
const router = useRouter()

const totalHosts = computed(() => store.hosts.length)
const connectedCount = computed(() => {
  return store.hosts.filter(h => store.isConnected(h.id)).length
})
</script>

<template>
  <div class="p-6">
    <h1 class="text-2xl font-bold mb-6">仪表盘</h1>

    <!-- 统计卡片 -->
    <div class="grid grid-cols-3 gap-4 mb-8">
      <Card class="p-4">
        <div class="flex items-center gap-3">
          <div class="h-10 w-10 rounded-lg bg-primary/10 flex items-center justify-center">
            <Server class="h-5 w-5 text-primary" />
          </div>
          <div>
            <p class="text-sm text-muted-foreground">宿主机</p>
            <p class="text-2xl font-bold">{{ totalHosts }}</p>
          </div>
        </div>
      </Card>

      <Card class="p-4">
        <div class="flex items-center gap-3">
          <div class="h-10 w-10 rounded-lg bg-green-500/10 flex items-center justify-center">
            <Wifi class="h-5 w-5 text-green-500" />
          </div>
          <div>
            <p class="text-sm text-muted-foreground">已连接</p>
            <p class="text-2xl font-bold">{{ connectedCount }}</p>
          </div>
        </div>
      </Card>

      <Card class="p-4">
        <div class="flex items-center gap-3">
          <div class="h-10 w-10 rounded-lg bg-muted flex items-center justify-center">
            <WifiOff class="h-5 w-5 text-muted-foreground" />
          </div>
          <div>
            <p class="text-sm text-muted-foreground">未连接</p>
            <p class="text-2xl font-bold">{{ totalHosts - connectedCount }}</p>
          </div>
        </div>
      </Card>
    </div>

    <!-- 宿主机列表 -->
    <h2 class="text-lg font-semibold mb-4">宿主机列表</h2>

    <div v-if="store.hosts.length === 0" class="text-center py-16 text-muted-foreground">
      <Monitor class="h-12 w-12 mx-auto mb-3 opacity-50" />
      <p>还没有添加宿主机</p>
      <p class="text-sm mt-1">点击左侧「添加」按钮开始</p>
    </div>

    <div v-else class="grid gap-3">
      <Card
        v-for="host in store.hosts"
        :key="host.id"
        class="p-4 cursor-pointer hover:border-primary/50 transition-colors"
        @click="router.push(`/host/${host.id}`)"
      >
        <div class="flex items-center gap-4">
          <div class="h-10 w-10 rounded-lg bg-muted flex items-center justify-center">
            <Server class="h-5 w-5" />
          </div>
          <div class="flex-1">
            <p class="font-medium">{{ host.name }}</p>
            <p class="text-sm text-muted-foreground">{{ host.user }}@{{ host.host }}:{{ host.port }}</p>
          </div>
          <span
            class="px-2 py-1 rounded text-xs"
            :class="store.isConnected(host.id)
              ? 'bg-green-500/10 text-green-600 dark:text-green-400'
              : 'bg-muted text-muted-foreground'"
          >
            {{ store.isConnected(host.id) ? '已连接' : '未连接' }}
          </span>
        </div>
      </Card>
    </div>
  </div>
</template>
