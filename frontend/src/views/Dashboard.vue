<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { useSettings } from '@/composables/useSettings'
import { VMList, HostResourceStats, AuditListAll, SettingGet } from '../../wailsjs/go/main/App'
import Card from '@/components/ui/Card.vue'
import { Server, Wifi, WifiOff, Monitor, Play, Box, Cpu, MemoryStick, HardDrive, FileText, AlertTriangle } from 'lucide-vue-next'

const { t } = useI18n()
const store = useAppStore()
const router = useRouter()
const { refreshIntervalMs } = useSettings()

// VM 统计数据: hostId -> VM[]
const hostVMs = ref<Record<string, any[]>>({})
const hostStats = ref<Record<string, any>>({})
const loadingVMs = ref(false)
const recentAudit = ref<any[]>([])
const alertThresholds = ref({ cpu: 90, mem: 90, disk: 85 })
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

// 告警宿主机
const alertedHosts = computed(() => {
  return store.hosts.filter(h => {
    const s = hostStats.value[h.id]
    if (!s) return false
    return s.cpuPercent >= alertThresholds.value.cpu ||
           s.memPercent >= alertThresholds.value.mem ||
           s.diskPercent >= alertThresholds.value.disk
  })
})

// VM 状态分布
const vmStateDistribution = computed(() => {
  const dist: Record<string, number> = {}
  for (const vms of Object.values(hostVMs.value)) {
    for (const v of vms as any[]) {
      dist[v.state] = (dist[v.state] || 0) + 1
    }
  }
  return dist
})

const stateLabels = computed<Record<string, string>>(() => ({
  'running': t('vm.running'), 'shut off': t('vm.shutOff'), 'paused': t('vm.paused'),
  'idle': t('vm.idle'), 'crashed': t('vm.crashed'),
}))

function getHostName(hostId: string): string {
  return store.hosts.find(h => h.id === hostId)?.name || hostId.slice(0, 8)
}

function getHostVMCount(hostId: string) {
  return hostVMs.value[hostId]?.length || 0
}
function getHostRunningCount(hostId: string) {
  return hostVMs.value[hostId]?.filter((v: any) => v.state === 'running').length || 0
}

async function loadAlertThresholds() {
  try {
    const cpu = await SettingGet('alert_cpu_threshold').catch(() => '')
    const mem = await SettingGet('alert_mem_threshold').catch(() => '')
    const disk = await SettingGet('alert_disk_threshold').catch(() => '')
    alertThresholds.value = {
      cpu: cpu ? Number(cpu) : 90,
      mem: mem ? Number(mem) : 90,
      disk: disk ? Number(disk) : 85,
    }
  } catch { /* */ }
}

async function loadRecentAudit() {
  try {
    recentAudit.value = (await AuditListAll(10)) || []
  } catch { recentAudit.value = [] }
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
  loadRecentAudit()
  loadAlertThresholds()
  refreshTimer = setInterval(() => { loadAllVMs(); loadRecentAudit() }, refreshIntervalMs())
})

onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer)
})
</script>

<template>
  <div class="p-6">
    <h1 class="text-2xl font-bold mb-6">{{ t('dashboard.title') }}</h1>

    <!-- 统计卡片 -->
    <div class="grid grid-cols-5 gap-4 mb-8">
      <Card class="p-4">
        <div class="flex items-center gap-3">
          <div class="h-10 w-10 rounded-lg bg-primary/10 flex items-center justify-center">
            <Server class="h-5 w-5 text-primary" />
          </div>
          <div>
            <p class="text-sm text-muted-foreground">{{ t('dashboard.hosts') }}</p>
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
            <p class="text-sm text-muted-foreground">{{ t('dashboard.runningVMs') }}</p>
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
            <p class="text-sm text-muted-foreground">{{ t('dashboard.cpuAvg') }}</p>
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
            <p class="text-sm text-muted-foreground">{{ t('dashboard.memAvg') }}</p>
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
            <p class="text-sm text-muted-foreground">{{ t('dashboard.diskAvg') }}</p>
            <p class="text-2xl font-bold">{{ connectedCount > 0 ? avgDisk.toFixed(1) + '%' : '-' }}</p>
          </div>
        </div>
      </Card>
    </div>

    <!-- 告警宿主机 -->
    <div v-if="alertedHosts.length > 0" class="mb-6">
      <h2 class="text-lg font-semibold mb-3 flex items-center gap-2">
        <AlertTriangle class="h-5 w-5 text-red-500" /> {{ t('dashboard.resourceAlert') }}
      </h2>
      <div class="grid gap-2">
        <Card
          v-for="h in alertedHosts" :key="h.id"
          class="p-3 border-red-500/50 bg-red-500/5 cursor-pointer hover:border-red-500 transition-colors"
          @click="router.push(`/host/${h.id}`)"
        >
          <div class="flex items-center gap-3">
            <Server class="h-4 w-4 text-red-500" />
            <span class="font-medium">{{ h.name }}</span>
            <div v-if="hostStats[h.id]" class="flex items-center gap-3 ml-auto text-xs">
              <span :class="hostStats[h.id].cpuPercent >= alertThresholds.cpu ? 'text-red-500 font-bold' : 'text-muted-foreground'">
                {{ t('chart.cpu') }} {{ hostStats[h.id].cpuPercent.toFixed(0) }}%
              </span>
              <span :class="hostStats[h.id].memPercent >= alertThresholds.mem ? 'text-red-500 font-bold' : 'text-muted-foreground'">
                {{ t('dashboard.memory') }} {{ hostStats[h.id].memPercent.toFixed(0) }}%
              </span>
              <span :class="hostStats[h.id].diskPercent >= alertThresholds.disk ? 'text-red-500 font-bold' : 'text-muted-foreground'">
                {{ t('dashboard.disk') }} {{ hostStats[h.id].diskPercent.toFixed(0) }}%
              </span>
            </div>
          </div>
        </Card>
      </div>
    </div>

    <!-- VM 状态分布 + 最近操作 -->
    <div class="grid grid-cols-2 gap-6 mb-6" v-if="totalVMs > 0 || recentAudit.length > 0">
      <!-- VM 状态分布 -->
      <Card v-if="totalVMs > 0" class="p-4">
        <h3 class="text-sm font-semibold mb-3">{{ t('dashboard.vmStateDistribution') }}</h3>
        <div class="space-y-2">
          <div v-for="(count, state) in vmStateDistribution" :key="state" class="flex items-center gap-2">
            <span class="text-xs w-14 text-muted-foreground">{{ stateLabels[state as string] || state }}</span>
            <div class="flex-1 h-2 bg-muted rounded-full overflow-hidden">
              <div
                class="h-full rounded-full transition-all"
                :class="state === 'running' ? 'bg-green-500' : state === 'shut off' ? 'bg-gray-400' : state === 'paused' ? 'bg-amber-500' : 'bg-red-500'"
                :style="{ width: (count / totalVMs * 100) + '%' }"
              />
            </div>
            <span class="text-xs font-medium w-8 text-right">{{ count }}</span>
          </div>
        </div>
      </Card>

      <!-- 最近操作 -->
      <Card v-if="recentAudit.length > 0" class="p-4">
        <div class="flex items-center justify-between mb-3">
          <h3 class="text-sm font-semibold flex items-center gap-1.5">
            <FileText class="h-3.5 w-3.5" /> {{ t('dashboard.recentActions') }}
          </h3>
          <button class="text-xs text-muted-foreground hover:text-foreground" @click="router.push('/audit')">{{ t('dashboard.viewAll') }}</button>
        </div>
        <div class="space-y-1.5">
          <div v-for="r in recentAudit" :key="r.id" class="flex items-center gap-2 text-xs">
            <span class="text-muted-foreground w-28 flex-shrink-0 truncate">{{ r.timestamp }}</span>
            <span class="px-1 py-0.5 rounded bg-muted text-[10px]">{{ r.action }}</span>
            <span class="truncate flex-1">{{ r.vmName || getHostName(r.hostId) }}</span>
          </div>
        </div>
      </Card>
    </div>

    <!-- 宿主机列表 -->
    <h2 class="text-lg font-semibold mb-4">{{ t('dashboard.hostList') }}</h2>

    <div v-if="store.hosts.length === 0" class="text-center py-16 text-muted-foreground">
      <Monitor class="h-12 w-12 mx-auto mb-3 opacity-50" />
      <p>{{ t('dashboard.noHosts') }}</p>
      <p class="text-sm mt-1">{{ t('dashboard.noHostsTip') }}</p>
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
                <span>{{ t('chart.cpu') }} {{ hostStats[host.id].cpuPercent.toFixed(0) }}%</span>
                <div class="h-1 bg-muted rounded-full mt-0.5 overflow-hidden">
                  <div class="h-full bg-blue-500 rounded-full" :style="{ width: hostStats[host.id].cpuPercent + '%' }" />
                </div>
              </div>
              <div class="w-16">
                <span>{{ t('dashboard.memory') }} {{ hostStats[host.id].memPercent.toFixed(0) }}%</span>
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
            {{ store.isConnected(host.id) ? t('dashboard.connected') : t('dashboard.disconnected') }}
          </span>
        </div>
      </Card>
    </div>
  </div>
</template>
