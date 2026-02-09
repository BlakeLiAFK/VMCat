<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAppStore, type VM } from '@/stores/app'
import { useToast } from '@/composables/useToast'
import {
  HostConnect, HostDisconnect, HostIsConnected, HostDelete,
  VMList, VMStart, VMShutdown, VMDestroy, VMReboot,
  HostResourceStats,
} from '../../wailsjs/go/main/App'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import HostFormDialog from '@/components/HostFormDialog.vue'
import VMCreateDialog from '@/components/VMCreateDialog.vue'
import {
  Plug, PlugZap, Play, Square, RotateCw, Skull,
  Monitor, Pencil, Trash2, Loader2, Terminal, Plus,
  Cpu, MemoryStick, HardDrive, CheckSquare, XSquare,
} from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const store = useAppStore()
const toast = useToast()

const hostId = computed(() => route.params.id as string)
const host = computed(() => store.hosts.find(h => h.id === hostId.value))
const connected = ref(false)
const connecting = ref(false)
const vms = ref<VM[]>([])
const loadingVMs = ref(false)
const showEdit = ref(false)
const actionLoading = ref<Record<string, boolean>>({})
const selectedVMs = ref<Set<string>>(new Set())
const stats = ref<any>(null)
const showCreateVM = ref(false)

let refreshTimer: ReturnType<typeof setInterval> | null = null

async function checkConnection() {
  try {
    connected.value = await HostIsConnected(hostId.value)
    if (connected.value) {
      store.markConnected(hostId.value)
    }
  } catch {
    connected.value = false
  }
}

async function connect() {
  connecting.value = true
  try {
    await HostConnect(hostId.value)
    connected.value = true
    store.markConnected(hostId.value)
    toast.success('连接成功')
    await loadVMs()
    loadStats()
  } catch (e: any) {
    toast.error('连接失败: ' + e.toString())
  } finally {
    connecting.value = false
  }
}

async function disconnect() {
  await HostDisconnect(hostId.value)
  connected.value = false
  store.markDisconnected(hostId.value)
  vms.value = []
  stats.value = null
  toast.info('已断开连接')
}

async function loadVMs() {
  if (!connected.value) return
  loadingVMs.value = true
  try {
    const list = await VMList(hostId.value)
    vms.value = list || []
  } catch (e: any) {
    console.error('load vms:', e)
  } finally {
    loadingVMs.value = false
  }
}

async function loadStats() {
  if (!connected.value) return
  try {
    stats.value = await HostResourceStats(hostId.value)
  } catch {
    // 静默忽略
  }
}

async function vmAction(vmName: string, action: string) {
  const key = `${vmName}-${action}`
  actionLoading.value[key] = true
  try {
    const labels: Record<string, string> = {
      start: '启动', shutdown: '关机', destroy: '强制关闭', reboot: '重启',
    }
    switch (action) {
      case 'start': await VMStart(hostId.value, vmName); break
      case 'shutdown': await VMShutdown(hostId.value, vmName); break
      case 'destroy': await VMDestroy(hostId.value, vmName); break
      case 'reboot': await VMReboot(hostId.value, vmName); break
    }
    toast.success(`${vmName} ${labels[action] || action} 成功`)
    setTimeout(loadVMs, 1500)
  } catch (e: any) {
    toast.error(`操作失败: ${e.toString()}`)
  } finally {
    actionLoading.value[key] = false
  }
}

async function batchAction(action: string) {
  const names = Array.from(selectedVMs.value)
  if (names.length === 0) return
  for (const name of names) {
    await vmAction(name, action)
  }
  selectedVMs.value.clear()
}

async function deleteHost() {
  if (!confirm('确认删除此宿主机?')) return
  try {
    await HostDelete(hostId.value)
    toast.success('已删除')
    router.push('/')
  } catch (e: any) {
    toast.error('删除失败: ' + e.toString())
  }
}

function toggleSelect(name: string) {
  if (selectedVMs.value.has(name)) {
    selectedVMs.value.delete(name)
  } else {
    selectedVMs.value.add(name)
  }
}

function toggleSelectAll() {
  if (selectedVMs.value.size === vms.value.length) {
    selectedVMs.value.clear()
  } else {
    selectedVMs.value = new Set(vms.value.map(v => v.name))
  }
}

function stateVariant(state: string) {
  if (state === 'running') return 'success'
  if (state === 'shut off') return 'secondary'
  if (state === 'paused') return 'warning'
  return 'outline'
}

function stateLabel(state: string) {
  const map: Record<string, string> = {
    'running': '运行中', 'shut off': '已关机', 'paused': '已暂停',
    'idle': '空闲', 'crashed': '已崩溃',
  }
  return map[state] || state
}

function isActionLoading(vmName: string, action: string) {
  return actionLoading.value[`${vmName}-${action}`] || false
}

function formatMem(mb: number) {
  return mb >= 1024 ? (mb / 1024).toFixed(1) + ' GB' : mb + ' MB'
}

onMounted(async () => {
  await checkConnection()
  if (!connected.value) {
    // 自动连接
    await connect()
  } else {
    await loadVMs()
    loadStats()
  }
  // 每 10 秒自动刷新 VM 列表
  refreshTimer = setInterval(() => {
    if (connected.value) {
      loadVMs()
      loadStats()
    }
  }, 10000)
})

onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer)
})

watch(hostId, async () => {
  vms.value = []
  stats.value = null
  selectedVMs.value.clear()
  await checkConnection()
  if (!connected.value) {
    await connect()
  } else {
    await loadVMs()
    loadStats()
  }
})
</script>

<template>
  <div class="p-6" v-if="host">
    <!-- 标题栏 -->
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-2xl font-bold">{{ host.name }}</h1>
        <p class="text-sm text-muted-foreground mt-1">{{ host.user }}@{{ host.host }}:{{ host.port }}</p>
      </div>
      <div class="flex items-center gap-2">
        <Button
          v-if="connected"
          variant="outline" size="sm"
          @click="router.push(`/host/${hostId}/terminal`)"
        >
          <Terminal class="h-4 w-4" />
          终端
        </Button>
        <Button v-if="!connected" variant="default" :loading="connecting" @click="connect">
          <Plug class="h-4 w-4" />
          连接
        </Button>
        <Button v-else variant="outline" @click="disconnect">
          <PlugZap class="h-4 w-4" />
          断开
        </Button>
        <Button variant="ghost" size="icon" @click="showEdit = true">
          <Pencil class="h-4 w-4" />
        </Button>
        <Button variant="ghost" size="icon" @click="deleteHost">
          <Trash2 class="h-4 w-4 text-destructive" />
        </Button>
      </div>
    </div>

    <!-- 未连接提示 -->
    <div v-if="!connected && !connecting" class="text-center py-20 text-muted-foreground">
      <Plug class="h-12 w-12 mx-auto mb-3 opacity-50" />
      <p>请先连接到此宿主机</p>
      <Button class="mt-4" :loading="connecting" @click="connect">
        <Plug class="h-4 w-4" />
        连接
      </Button>
    </div>

    <!-- 连接中 -->
    <div v-else-if="connecting" class="text-center py-20">
      <Loader2 class="h-8 w-8 mx-auto animate-spin text-muted-foreground" />
      <p class="text-sm text-muted-foreground mt-2">正在连接...</p>
    </div>

    <!-- 已连接 -->
    <div v-else>
      <!-- 资源概览 -->
      <div v-if="stats" class="grid grid-cols-4 gap-3 mb-6">
        <Card class="p-3">
          <div class="flex items-center gap-2">
            <Cpu class="h-4 w-4 text-blue-500" />
            <span class="text-sm text-muted-foreground">CPU</span>
          </div>
          <p class="text-lg font-semibold mt-1">{{ stats.cpuPercent.toFixed(1) }}%</p>
          <div class="mt-1 h-1.5 bg-muted rounded-full overflow-hidden">
            <div class="h-full bg-blue-500 rounded-full transition-all" :style="{ width: stats.cpuPercent + '%' }" />
          </div>
        </Card>
        <Card class="p-3">
          <div class="flex items-center gap-2">
            <MemoryStick class="h-4 w-4 text-purple-500" />
            <span class="text-sm text-muted-foreground">内存</span>
          </div>
          <p class="text-lg font-semibold mt-1">{{ stats.memPercent.toFixed(1) }}%</p>
          <p class="text-xs text-muted-foreground">{{ formatMem(stats.memUsed) }} / {{ formatMem(stats.memTotal) }}</p>
        </Card>
        <Card class="p-3">
          <div class="flex items-center gap-2">
            <HardDrive class="h-4 w-4 text-orange-500" />
            <span class="text-sm text-muted-foreground">磁盘</span>
          </div>
          <p class="text-lg font-semibold mt-1">{{ stats.diskPercent.toFixed(1) }}%</p>
          <p class="text-xs text-muted-foreground">{{ stats.diskUsed }}G / {{ stats.diskTotal }}G</p>
        </Card>
        <Card class="p-3">
          <div class="flex items-center gap-2">
            <Monitor class="h-4 w-4 text-green-500" />
            <span class="text-sm text-muted-foreground">VM</span>
          </div>
          <p class="text-lg font-semibold mt-1">{{ vms.filter(v => v.state === 'running').length }} / {{ vms.length }}</p>
          <p class="text-xs text-muted-foreground">运行中 / 总数</p>
        </Card>
      </div>

      <!-- VM 列表头 -->
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-lg font-semibold">虚拟机 ({{ vms.length }})</h2>
        <div class="flex items-center gap-2">
          <!-- 批量操作 -->
          <template v-if="selectedVMs.size > 0">
            <span class="text-sm text-muted-foreground">已选 {{ selectedVMs.size }}</span>
            <Button variant="outline" size="sm" @click="batchAction('start')">
              <Play class="h-3.5 w-3.5" /> 启动
            </Button>
            <Button variant="outline" size="sm" @click="batchAction('shutdown')">
              <Square class="h-3.5 w-3.5" /> 关机
            </Button>
          </template>
          <Button variant="outline" size="sm" @click="loadVMs" :loading="loadingVMs">
            <RotateCw class="h-3.5 w-3.5" />
            刷新
          </Button>
          <Button size="sm" @click="showCreateVM = true">
            <Plus class="h-3.5 w-3.5" />
            创建 VM
          </Button>
        </div>
      </div>

      <!-- 加载中 -->
      <div v-if="loadingVMs && vms.length === 0" class="text-center py-12">
        <Loader2 class="h-8 w-8 mx-auto animate-spin text-muted-foreground" />
        <p class="text-sm text-muted-foreground mt-2">加载虚拟机列表...</p>
      </div>

      <!-- VM 表格 -->
      <Card v-else-if="vms.length > 0">
        <table class="w-full">
          <thead>
            <tr class="border-b text-sm text-muted-foreground">
              <th class="text-left p-3 w-8">
                <button @click="toggleSelectAll" class="hover:text-foreground">
                  <CheckSquare v-if="selectedVMs.size === vms.length && vms.length > 0" class="h-4 w-4" />
                  <XSquare v-else class="h-4 w-4" />
                </button>
              </th>
              <th class="text-left p-3 font-medium">状态</th>
              <th class="text-left p-3 font-medium">名称</th>
              <th class="text-left p-3 font-medium">CPU</th>
              <th class="text-left p-3 font-medium">内存</th>
              <th class="text-right p-3 font-medium">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="v in vms"
              :key="v.name"
              class="border-b last:border-0 hover:bg-muted/50 cursor-pointer transition-colors"
              :class="{ 'bg-accent/30': selectedVMs.has(v.name) }"
              @click="router.push(`/host/${hostId}/vm/${v.name}`)"
            >
              <td class="p-3" @click.stop>
                <button @click="toggleSelect(v.name)" class="hover:text-foreground">
                  <CheckSquare v-if="selectedVMs.has(v.name)" class="h-4 w-4 text-primary" />
                  <XSquare v-else class="h-4 w-4 text-muted-foreground" />
                </button>
              </td>
              <td class="p-3">
                <Badge :variant="stateVariant(v.state)">{{ stateLabel(v.state) }}</Badge>
              </td>
              <td class="p-3 font-medium">{{ v.name }}</td>
              <td class="p-3 text-muted-foreground">{{ v.cpus }} vCPU</td>
              <td class="p-3 text-muted-foreground">{{ formatMem(v.memoryMB) }}</td>
              <td class="p-3 text-right" @click.stop>
                <div class="flex items-center justify-end gap-1">
                  <Button
                    v-if="v.state === 'shut off'"
                    variant="ghost" size="icon"
                    :loading="isActionLoading(v.name, 'start')"
                    @click="vmAction(v.name, 'start')"
                    title="启动"
                  >
                    <Play class="h-4 w-4 text-green-500" />
                  </Button>
                  <Button
                    v-if="v.state === 'running'"
                    variant="ghost" size="icon"
                    :loading="isActionLoading(v.name, 'shutdown')"
                    @click="vmAction(v.name, 'shutdown')"
                    title="关机"
                  >
                    <Square class="h-4 w-4" />
                  </Button>
                  <Button
                    v-if="v.state === 'running'"
                    variant="ghost" size="icon"
                    :loading="isActionLoading(v.name, 'reboot')"
                    @click="vmAction(v.name, 'reboot')"
                    title="重启"
                  >
                    <RotateCw class="h-4 w-4" />
                  </Button>
                  <Button
                    v-if="v.state === 'running'"
                    variant="ghost" size="icon"
                    :loading="isActionLoading(v.name, 'destroy')"
                    @click="vmAction(v.name, 'destroy')"
                    title="强制关闭"
                  >
                    <Skull class="h-4 w-4 text-destructive" />
                  </Button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </Card>

      <!-- 空状态 -->
      <div v-else class="text-center py-12 text-muted-foreground">
        <Monitor class="h-12 w-12 mx-auto mb-3 opacity-50" />
        <p>此宿主机上没有虚拟机</p>
      </div>
    </div>

    <!-- 编辑弹窗 -->
    <HostFormDialog
      :open="showEdit"
      :host="host"
      @update:open="showEdit = $event"
      @saved="showEdit = false"
    />

    <!-- 创建 VM 弹窗 -->
    <VMCreateDialog
      :open="showCreateVM"
      :hostId="hostId"
      @update:open="showCreateVM = $event"
      @saved="showCreateVM = false; loadVMs()"
    />
  </div>
</template>
