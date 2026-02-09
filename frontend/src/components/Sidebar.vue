<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { useTheme } from '@/composables/useTheme'
import { useSettings } from '@/composables/useSettings'
import { HostList, HostIsConnected, VMList, AppVersion } from '../../wailsjs/go/main/App'
import { Server, Plus, Settings, LayoutDashboard, Sun, Moon } from 'lucide-vue-next'
import HostFormDialog from '@/components/HostFormDialog.vue'

const router = useRouter()
const route = useRoute()
const store = useAppStore()
const { isDark, toggle: toggleTheme } = useTheme()
const showAddHost = ref(false)
const appVersion = ref('')
const { refreshIntervalMs } = useSettings()

let refreshTimer: ReturnType<typeof setInterval> | null = null

async function loadHosts() {
  try {
    const list = await HostList()
    store.setHosts(list || [])
    for (const h of (list || [])) {
      const connected = await HostIsConnected(h.id)
      if (connected) {
        store.markConnected(h.id)
        // 拉取 VM 计数
        try {
          const vms = await VMList(h.id)
          const arr = vms || []
          store.setHostVMCount(h.id, arr.length, arr.filter(v => v.state === 'running').length)
        } catch { /* 静默 */ }
      } else {
        store.markDisconnected(h.id)
      }
    }
  } catch (e) {
    console.error('load hosts:', e)
  }
}

function navigateTo(path: string) {
  router.push(path)
}

function isActive(path: string) {
  return route.path === path
}

function isHostActive(id: string) {
  return route.path.startsWith(`/host/${id}`)
}

onMounted(() => {
  loadHosts()
  AppVersion().then(v => { appVersion.value = v }).catch(() => {})
  // 按配置间隔自动刷新连接状态
  refreshTimer = setInterval(loadHosts, refreshIntervalMs())
})

onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer)
})

defineExpose({ loadHosts })
</script>

<template>
  <aside class="w-56 h-full border-r bg-muted/30 flex flex-col">
    <!-- Logo -->
    <div class="h-12 flex items-center px-4 border-b">
      <span class="font-bold text-lg tracking-tight">VMCat</span>
      <span class="ml-auto text-xs text-muted-foreground">v{{ appVersion }}</span>
    </div>

    <!-- 导航 -->
    <nav class="flex-1 overflow-auto py-2">
      <button
        class="w-full flex items-center gap-2 px-4 py-2 text-sm hover:bg-accent transition-colors"
        :class="{ 'bg-accent text-accent-foreground': isActive('/') }"
        @click="navigateTo('/')"
      >
        <LayoutDashboard class="h-4 w-4" />
        <span>仪表盘</span>
      </button>

      <div class="mt-3 px-4">
        <span class="text-xs font-medium text-muted-foreground uppercase tracking-wider">宿主机</span>
      </div>

      <div class="mt-1">
        <button
          v-for="host in store.hosts"
          :key="host.id"
          class="w-full flex items-center gap-2 px-4 py-2 text-sm hover:bg-accent transition-colors"
          :class="{ 'bg-accent text-accent-foreground': isHostActive(host.id) }"
          @click="navigateTo(`/host/${host.id}`)"
        >
          <Server class="h-4 w-4" />
          <span class="truncate flex-1 text-left">{{ host.name }}</span>
          <span
            v-if="store.isConnected(host.id) && store.getHostVMCount(host.id).total > 0"
            class="text-xs text-muted-foreground flex-shrink-0"
          >
            {{ store.getHostVMCount(host.id).running }}/{{ store.getHostVMCount(host.id).total }}
          </span>
          <span
            class="h-2 w-2 rounded-full flex-shrink-0"
            :class="store.isConnected(host.id) ? 'bg-green-500' : 'bg-muted-foreground/30'"
          />
        </button>
      </div>
    </nav>

    <!-- 底部操作 -->
    <div class="border-t p-2 flex gap-1">
      <button
        class="flex-1 flex items-center justify-center gap-1 px-2 py-2 text-sm rounded hover:bg-accent transition-colors"
        @click="showAddHost = true"
      >
        <Plus class="h-4 w-4" />
        <span>添加</span>
      </button>
      <button
        class="flex items-center justify-center px-2 py-2 rounded hover:bg-accent transition-colors"
        @click="toggleTheme"
        :title="isDark ? '切换亮色' : '切换暗色'"
      >
        <Sun v-if="isDark" class="h-4 w-4" />
        <Moon v-else class="h-4 w-4" />
      </button>
      <button
        class="flex items-center justify-center px-2 py-2 rounded hover:bg-accent transition-colors"
        @click="navigateTo('/settings')"
      >
        <Settings class="h-4 w-4" />
      </button>
    </div>

    <HostFormDialog
      :open="showAddHost"
      @update:open="showAddHost = $event"
      @saved="loadHosts"
    />
  </aside>
</template>
