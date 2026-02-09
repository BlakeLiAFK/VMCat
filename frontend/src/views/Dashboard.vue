<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { useSettings } from '@/composables/useSettings'
import { VMList, HostResourceStats } from '../../wailsjs/go/main/App'
import Card from '@/components/ui/Card.vue'
import { Server, Wifi, WifiOff, Monitor, Play, Box, Cpu, MemoryStick, HardDrive } from 'lucide-vue-next'

const store = useAppStore()
const router = useRouter()
const { refreshIntervalMs } = useSettings()

// VM 统计数据: hostId -> VM[]
const hostVMs = ref<Record<string, any[]>>({})
const hostStats = ref<Record<string, any>>({})
const loadingVMs = ref(false)
let refreshTimer: ReturnType<typeof setInterval> | null = null

const totalHosts = computed(() => store.hosts.length)
const connectedCount = computed(() => {
  return store.hosts.filter(h => store.isConnected(h.id)).length
})
const totalVMs = computed(() => {
  return Object.values(hostVMs.value).reduce((sum, vms) => sum + vms.length, 0)
})
const runningVMs = computed(() => {
  return Object.values(hostVMs.value).reduce((sum, vms) => sum + vms.filter(v => v.state === 'running').length, 0)
})

// 资源聚合
const avgCPU = computed(() => {
  const vals = Object.values(hostStats.value).filter(s => s?.cpuPercent != null)
  if (vals.length === 0) return 0
  return vals.reduce((sum, s) => sum + s.cpuPercent, 0) / vals.length
})
const avgMem = computed(() => {
  const vals = Object.values(hostStats.value).filter(s => s?.memPercent != null)
  if (vals.length === 0) return 0
  return vals.reduce((sum, s) => sum + s.memPercent, 0) / vals.length
})
const avgDisk = computed(() => {
  const vals = Object.values(hostStats.value).filter(s => s?.diskPercent != null)
  if (vals.length === 0) return 0
  return vals.reduce((sum, s) => sum + s.diskPercent, 0) / vals.length
})

function getHostVMCount(hostId: string) {
  return hostVMs.value[hostId]?.length || 0
}
function getHostRunningCount(hostId: string) {
  return hostVMs.value[hostId]?.filter((v: any) => v.state === 'running').length || 0
}

async function loadAllVMs() {
  loadingVMs.value = true
  const connected = store.hosts.filter(h => store.isConnected(h.id))
  const results: Record<string, any[]> = {}
  const statsResults: Record<string, any> = {}
  await Promise.all(connected.map(async (h) => {
    try {
      const vms = await VMList(h.id)
      results[h.id] = vms || []
    } catch { results[h.id] = [] }
    try {
      statsResults[h.id] = await HostResourceStats(h.id)
    } catch { /* 静默 */ }
  }))
  hostVMs.value = results
  hostStats.value = statsResults
  loadingVMs.value = false
}

onMounted(() => {
  loadAllVMs()
  refreshTimer = setInterval(loadAllVMs, refreshIntervalMs())
})

onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer)
})
</script>

<template>
  <div class="p-6">
    <h1 class="text-2xl font-bold mb-6">仪表盘</h1>

    <!-- 统计卡片 -->
    <div class="grid grid-cols-5 gap-4 mb-8">
      <Card class="p-4">
        <div class="flex items-center gap-3">
          <div class="h-10 w-10 rounded-lg bg-primary/10 flex items-center justify-center">
            <Server class="h-5 w-5 text-primary" />
          </div>
          <div>
            <p class="text-sm text-muted-foreground">宿主机</p>
            <p class="text-2xl font-bold">{{ connectedCount }}<span class="text-base font-normal text-muted-foreground">/{{ totalHosts }}</span></p>
          </div>
        </div>
      </Card>

      <Card class="p-4">
        <div class="flex items-center gap-3">
          <div class="h-10 w-10 rounded-lg bg-emerald-500/10 flex items-center justify-center">
            <Play class="h-5 w-5 text-emerald-500" />
          </div>
          <div>
            <p class="text-sm text-muted-foreground">运行中 VM</p>
            <p class="text-2xl font-bold">{{ runningVMs }}<span class="text-base font-normal text-muted-foreground">/{{ totalVMs }}</span></p>
          </div>
        </div>
      </Card>

      <Card class="p-4">
        <div class="flex items-center gap-3">
          <div class="h-10 w-10 rounded-lg bg-blue-500/10 flex items-center justify-center">
            <Cpu class="h-5 w-5 text-blue-500" />
          </div>
          <div>
            <p class="text-sm text-muted-foreground">CPU 均值</p>
            <p class="text-2xl font-bold">{{ connectedCount > 0 ? avgCPU.toFixed(1) + '%' : '-' }}</p>
          </div>
        </div>
      </Card>

      <Card class="p-4">
        <div class="flex items-center gap-3">
          <div class="h-10 w-10 rounded-lg bg-purple-500/10 flex items-center justify-center">
            <MemoryStick class="h-5 w-5 text-purple-500" />
          </div>
          <div>
            <p class="text-sm text-muted-foreground">内存均值</p>
            <p class="text-2xl font-bold">{{ connectedCount > 0 ? avgMem.toFixed(1) + '%' : '-' }}</p>
          </div>
        </div>
      </Card>

      <Card class="p-4">
        <div class="flex items-center gap-3">
          <div class="h-10 w-10 rounded-lg bg-orange-500/10 flex items-center justify-center">
            <HardDrive class="h-5 w-5 text-orange-500" />
          </div>
          <div>
            <p class="text-sm text-muted-foreground">磁盘均值</p>
            <p class="text-2xl font-bold">{{ connectedCount > 0 ? avgDisk.toFixed(1) + '%' : '-' }}</p>
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
          <div class="flex-1 min-w-0">
            <p class="font-medium">{{ host.name }}</p>
            <p class="text-sm text-muted-foreground truncate">{{ host.user }}@{{ host.host }}:{{ host.port }}</p>
          </div>
          <!-- 已连接：显示资源小指示器 -->
          <template v-if="store.isConnected(host.id)">
            <div v-if="hostStats[host.id]" class="flex items-center gap-3 text-xs text-muted-foreground">
              <div class="w-16">
                <span>CPU {{ hostStats[host.id].cpuPercent.toFixed(0) }}%</span>
                <div class="h-1 bg-muted rounded-full mt-0.5 overflow-hidden">
                  <div class="h-full bg-blue-500 rounded-full" :style="{ width: hostStats[host.id].cpuPercent + '%' }" />
                </div>
              </div>
              <div class="w-16">
                <span>内存 {{ hostStats[host.id].memPercent.toFixed(0) }}%</span>
                <div class="h-1 bg-muted rounded-full mt-0.5 overflow-hidden">
                  <div class="h-full bg-purple-500 rounded-full" :style="{ width: hostStats[host.id].memPercent + '%' }" />
                </div>
              </div>
            </div>
            <div v-if="getHostVMCount(host.id) > 0" class="text-right">
              <p class="text-sm font-medium">{{ getHostRunningCount(host.id) }}/{{ getHostVMCount(host.id) }}</p>
              <p class="text-xs text-muted-foreground">VM</p>
            </div>
          </template>
          <span
            class="px-2 py-1 rounded text-xs flex-shrink-0"
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
