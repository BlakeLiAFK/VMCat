<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAppStore, type VM } from '@/stores/app'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'
import { useSettings } from '@/composables/useSettings'
import {
  HostConnect, HostDisconnect, HostIsConnected, HostDelete, HostList,
  VMList, VMStart, VMShutdown, VMDestroy, VMReboot,
  HostResourceStats, HostGetFingerprint, HostResetHostKey, HostCheckTools,
} from '../../wailsjs/go/main/App'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import HostFormDialog from '@/components/HostFormDialog.vue'
import VMCreateDialog from '@/components/VMCreateDialog.vue'
import StoragePanel from '@/components/StoragePanel.vue'
import NetworkPanel from '@/components/NetworkPanel.vue'
import ImageManager from '@/components/ImageManager.vue'
import QuickCreateDialog from '@/components/QuickCreateDialog.vue'
import {
  Plug, PlugZap, Play, Square, RotateCw, Skull,
  Monitor, Pencil, Trash2, Loader2, Terminal, Plus, Zap,
  Cpu, MemoryStick, HardDrive, Check, Minus, Search, Filter, Database, ShieldCheck, Globe, Image,
  AlertTriangle,
} from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const store = useAppStore()
const toast = useToast()
const { request: confirmRequest } = useConfirm()

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
const showQuickCreate = ref(false)
const activeTab = ref<'vms' | 'storage' | 'network' | 'images'>('vms')
const searchQuery = ref('')
const stateFilter = ref('all')
const fingerprint = ref('')
const connectError = ref('')
const toolStatus = ref<Record<string, string>>({})
const { refreshIntervalMs } = useSettings()

const virshMissing = computed(() => connected.value && toolStatus.value.virsh === '')

const filteredVMs = computed(() => {
  let list = vms.value
  if (searchQuery.value.trim()) {
    const q = searchQuery.value.trim().toLowerCase()
    list = list.filter(v => v.name.toLowerCase().includes(q))
  }
  if (stateFilter.value !== 'all') {
    list = list.filter(v => v.state === stateFilter.value)
  }
  return list
})

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
  connectError.value = ''
  try {
    await HostConnect(hostId.value)
    connected.value = true
    store.markConnected(hostId.value)
    toast.success('连接成功')
    await loadVMs()
    loadStats()
    loadFingerprint()
    checkTools()
  } catch (e: any) {
    const msg = e.toString()
    connectError.value = msg
    if (msg.includes('密钥不匹配')) {
      toast.error(msg + '\n可在下方忘记旧指纹后重试')
    } else {
      toast.error('连接失败: ' + msg)
    }
    loadFingerprint()
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

async function loadFingerprint() {
  try {
    fingerprint.value = await HostGetFingerprint(hostId.value)
  } catch {
    fingerprint.value = ''
  }
}

async function checkTools() {
  try {
    toolStatus.value = await HostCheckTools(hostId.value) || {}
  } catch {
    toolStatus.value = {}
  }
}

async function resetHostKey() {
  const ok = await confirmRequest(
    '忘记主机指纹',
    '忘记后下次连接将重新接受服务端公钥指纹。',
  )
  if (!ok) return
  try {
    await HostResetHostKey(hostId.value)
    fingerprint.value = ''
    toast.success('已忘记主机指纹')
  } catch (e: any) {
    toast.error('操作失败: ' + e.toString())
  }
}

const actionLabels: Record<string, string> = {
  start: '启动', shutdown: '关机', destroy: '强制关闭', reboot: '重启',
}

async function vmAction(vmName: string, action: string) {
  // 启动无需确认，其他操作需要二次确认
  if (action !== 'start') {
    const ok = await confirmRequest(
      `${actionLabels[action] || action}确认`,
      `确认对 "${vmName}" 执行${actionLabels[action] || action}操作?`,
      { variant: action === 'destroy' ? 'destructive' : 'default' },
    )
    if (!ok) return
  }
  const key = `${vmName}-${action}`
  actionLoading.value[key] = true
  try {
    switch (action) {
      case 'start': await VMStart(hostId.value, vmName); break
      case 'shutdown': await VMShutdown(hostId.value, vmName); break
      case 'destroy': await VMDestroy(hostId.value, vmName); break
      case 'reboot': await VMReboot(hostId.value, vmName); break
    }
    toast.success(`${vmName} ${actionLabels[action] || action} 成功`)
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
  const label = actionLabels[action] || action
  const ok = await confirmRequest(
    `批量${label}确认`,
    `确认对 ${names.length} 台虚拟机执行${label}操作?`,
    { variant: action === 'destroy' ? 'destructive' : 'default' },
  )
  if (!ok) return
  for (const name of names) {
    // 批量操作已整体确认，单个不再弹窗，直接执行
    const key = `${name}-${action}`
    actionLoading.value[key] = true
    try {
      switch (action) {
        case 'start': await VMStart(hostId.value, name); break
        case 'shutdown': await VMShutdown(hostId.value, name); break
        case 'destroy': await VMDestroy(hostId.value, name); break
        case 'reboot': await VMReboot(hostId.value, name); break
      }
      toast.success(`${name} ${label} 成功`)
    } catch (e: any) {
      toast.error(`${name} 操作失败: ${e.toString()}`)
    } finally {
      actionLoading.value[key] = false
    }
  }
  selectedVMs.value.clear()
  setTimeout(loadVMs, 1500)
}

async function deleteHost() {
  const ok = await confirmRequest('删除宿主机', '确认删除此宿主机? 此操作不可撤销!', { variant: 'destructive', confirmText: '删除' })
  if (!ok) return
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

// 编辑保存后刷新侧栏宿主机列表
async function onHostSaved() {
  showEdit.value = false
  try {
    const list = await HostList()
    store.setHosts(list || [])
  } catch { /* 静默 */ }
}

onMounted(async () => {
  loadFingerprint()
  await checkConnection()
  if (!connected.value) {
    // 自动连接
    await connect()
  } else {
    await loadVMs()
    loadStats()
    checkTools()
  }
  // 按配置间隔自动刷新 VM 列表
  refreshTimer = setInterval(() => {
    if (connected.value) {
      loadVMs()
      loadStats()
    }
  }, refreshIntervalMs())
})

onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer)
})

watch(hostId, async () => {
  vms.value = []
  stats.value = null
  fingerprint.value = ''
  toolStatus.value = {}
  selectedVMs.value.clear()
  loadFingerprint()
  await checkConnection()
  if (!connected.value) {
    await connect()
  } else {
    await loadVMs()
    loadStats()
    checkTools()
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
        <div v-if="fingerprint" class="flex items-center gap-1.5 mt-1">
          <ShieldCheck class="h-3.5 w-3.5 text-green-500" />
          <span class="text-xs text-muted-foreground font-mono">{{ fingerprint }}</span>
          <button @click="resetHostKey" class="text-xs text-muted-foreground hover:text-destructive ml-1" title="忘记此主机指纹，下次连接重新验证">
            忘记指纹
          </button>
        </div>
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
      <!-- 密钥不匹配特殊提示 -->
      <Card v-if="connectError && connectError.includes('密钥不匹配')" class="max-w-md mx-auto mb-6 text-left border-destructive/50">
        <div class="p-4">
          <h3 class="text-sm font-semibold text-destructive flex items-center gap-2 mb-2">
            <ShieldCheck class="h-4 w-4" /> SSH 主机密钥不匹配
          </h3>
          <p class="text-xs text-muted-foreground mb-3">
            服务端指纹与上次连接时记录的不同。如果确认服务器已重装或更换了密钥，请忘记旧指纹后重试。
          </p>
          <div class="flex gap-2">
            <Button variant="destructive" size="sm" @click="resetHostKey">
              忘记指纹
            </Button>
            <Button variant="outline" size="sm" @click="connect">
              重试连接
            </Button>
          </div>
        </div>
      </Card>
      <!-- 普通连接失败提示 -->
      <Card v-else-if="connectError" class="max-w-md mx-auto mb-6 text-left">
        <div class="p-4">
          <p class="text-sm text-destructive mb-2">连接失败</p>
          <p class="text-xs text-muted-foreground font-mono break-all mb-3">{{ connectError }}</p>
          <Button variant="outline" size="sm" @click="connect">
            重试连接
          </Button>
        </div>
      </Card>
      <template v-else>
        <Plug class="h-12 w-12 mx-auto mb-3 opacity-50" />
        <p>请先连接到此宿主机</p>
        <Button class="mt-4" :loading="connecting" @click="connect">
          <Plug class="h-4 w-4" />
          连接
        </Button>
      </template>
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

      <!-- virsh 未安装提示 -->
      <Card v-if="virshMissing" class="mb-4 border-yellow-500/50 bg-yellow-500/5">
        <div class="p-3 flex items-center gap-3">
          <AlertTriangle class="h-4 w-4 text-yellow-500 flex-shrink-0" />
          <div class="flex-1 text-sm">
            <span class="font-medium text-yellow-600 dark:text-yellow-400">virsh/libvirt 未安装</span>
            <span class="text-muted-foreground ml-1">VM 管理功能不可用，可通过终端安装所需软件。</span>
          </div>
          <Button variant="outline" size="sm" @click="router.push(`/host/${hostId}/terminal`)">
            <Terminal class="h-3.5 w-3.5" /> 打开终端
          </Button>
        </div>
      </Card>

      <!-- Tab 切换 -->
      <div class="flex border-b mb-4">
        <button
          class="px-4 py-2.5 text-sm border-b-2 transition-colors -mb-px"
          :class="activeTab === 'vms' ? 'border-primary text-foreground font-medium' : 'border-transparent text-muted-foreground hover:text-foreground'"
          @click="activeTab = 'vms'"
        >
          <Monitor class="h-3.5 w-3.5 inline mr-1" /> 虚拟机
        </button>
        <button
          class="px-4 py-2.5 text-sm border-b-2 transition-colors -mb-px"
          :class="activeTab === 'storage' ? 'border-primary text-foreground font-medium' : 'border-transparent text-muted-foreground hover:text-foreground'"
          @click="activeTab = 'storage'"
        >
          <Database class="h-3.5 w-3.5 inline mr-1" /> 存储
        </button>
        <button
          class="px-4 py-2.5 text-sm border-b-2 transition-colors -mb-px"
          :class="activeTab === 'network' ? 'border-primary text-foreground font-medium' : 'border-transparent text-muted-foreground hover:text-foreground'"
          @click="activeTab = 'network'"
        >
          <Globe class="h-3.5 w-3.5 inline mr-1" /> 网络
        </button>
        <button
          class="px-4 py-2.5 text-sm border-b-2 transition-colors -mb-px"
          :class="activeTab === 'images' ? 'border-primary text-foreground font-medium' : 'border-transparent text-muted-foreground hover:text-foreground'"
          @click="activeTab = 'images'"
        >
          <Image class="h-3.5 w-3.5 inline mr-1" /> 镜像
        </button>
      </div>

      <!-- 存储管理 -->
      <StoragePanel v-if="activeTab === 'storage'" :hostId="hostId" :visible="activeTab === 'storage'" />

      <!-- 网络管理 -->
      <NetworkPanel v-if="activeTab === 'network'" :hostId="hostId" :visible="activeTab === 'network'" />

      <!-- 镜像管理 -->
      <ImageManager v-if="activeTab === 'images'" :hostId="hostId" :visible="activeTab === 'images'" />

      <!-- VM 列表头 -->
      <div v-if="activeTab === 'vms'" class="flex items-center justify-between mb-4">
        <h2 class="text-lg font-semibold">虚拟机 ({{ filteredVMs.length }}<span v-if="filteredVMs.length !== vms.length" class="text-muted-foreground font-normal">/{{ vms.length }}</span>)</h2>
        <div class="flex items-center gap-2">
          <!-- 搜索框 -->
          <div class="relative">
            <Search class="h-3.5 w-3.5 absolute left-2.5 top-1/2 -translate-y-1/2 text-muted-foreground" />
            <input
              v-model="searchQuery"
              placeholder="搜索 VM..."
              class="h-8 w-40 pl-8 pr-2 text-sm rounded-md border border-input bg-transparent focus:outline-none focus:ring-1 focus:ring-ring"
            />
          </div>
          <!-- 状态筛选 -->
          <select
            v-model="stateFilter"
            class="h-8 text-sm rounded-md border border-input bg-transparent px-2 focus:outline-none focus:ring-1 focus:ring-ring"
          >
            <option value="all">全部状态</option>
            <option value="running">运行中</option>
            <option value="shut off">已关机</option>
            <option value="paused">已暂停</option>
          </select>
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
          <Button size="sm" variant="outline" @click="showQuickCreate = true">
            <Zap class="h-3.5 w-3.5" />
            快速创建
          </Button>
          <Button size="sm" @click="showCreateVM = true">
            <Plus class="h-3.5 w-3.5" />
            创建 VM
          </Button>
        </div>
      </div>

      <!-- 加载中 -->
      <div v-if="activeTab === 'vms' && loadingVMs && vms.length === 0" class="text-center py-12">
        <Loader2 class="h-8 w-8 mx-auto animate-spin text-muted-foreground" />
        <p class="text-sm text-muted-foreground mt-2">加载虚拟机列表...</p>
      </div>

      <!-- VM 表格 -->
      <Card v-else-if="activeTab === 'vms' && vms.length > 0">
        <table class="w-full">
          <thead>
            <tr class="border-b text-sm text-muted-foreground">
              <th class="text-left p-3 w-8">
                <button @click="toggleSelectAll" class="hover:text-foreground flex items-center justify-center">
                  <span
                    class="h-4 w-4 rounded border flex items-center justify-center transition-colors"
                    :class="selectedVMs.size === vms.length && vms.length > 0
                      ? 'bg-primary border-primary text-primary-foreground'
                      : selectedVMs.size > 0
                        ? 'bg-primary/50 border-primary text-primary-foreground'
                        : 'border-muted-foreground/40'"
                  >
                    <Check v-if="selectedVMs.size === vms.length && vms.length > 0" class="h-3 w-3" />
                    <Minus v-else-if="selectedVMs.size > 0" class="h-3 w-3" />
                  </span>
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
              v-for="v in filteredVMs"
              :key="v.name"
              class="border-b last:border-0 hover:bg-muted/50 cursor-pointer transition-colors"
              :class="{ 'bg-accent/30': selectedVMs.has(v.name) }"
              @click="router.push(`/host/${hostId}/vm/${v.name}`)"
            >
              <td class="p-3" @click.stop>
                <button @click="toggleSelect(v.name)" class="hover:text-foreground flex items-center justify-center">
                  <span
                    class="h-4 w-4 rounded border flex items-center justify-center transition-colors"
                    :class="selectedVMs.has(v.name)
                      ? 'bg-primary border-primary text-primary-foreground'
                      : 'border-muted-foreground/40 hover:border-muted-foreground'"
                  >
                    <Check v-if="selectedVMs.has(v.name)" class="h-3 w-3" />
                  </span>
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
                    variant="outline" size="sm"
                    :loading="isActionLoading(v.name, 'start')"
                    @click="vmAction(v.name, 'start')"
                  >
                    <Play class="h-3.5 w-3.5 text-green-500" />
                    启动
                  </Button>
                  <Button
                    v-if="v.state === 'running'"
                    variant="outline" size="sm"
                    :loading="isActionLoading(v.name, 'shutdown')"
                    @click="vmAction(v.name, 'shutdown')"
                  >
                    <Square class="h-3.5 w-3.5" />
                    关机
                  </Button>
                  <Button
                    v-if="v.state === 'running'"
                    variant="outline" size="sm"
                    :loading="isActionLoading(v.name, 'reboot')"
                    @click="vmAction(v.name, 'reboot')"
                  >
                    <RotateCw class="h-3.5 w-3.5" />
                    重启
                  </Button>
                  <Button
                    v-if="v.state === 'running'"
                    variant="destructive" size="sm"
                    :loading="isActionLoading(v.name, 'destroy')"
                    @click="vmAction(v.name, 'destroy')"
                  >
                    <Skull class="h-3.5 w-3.5" />
                    强制关闭
                  </Button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </Card>

      <!-- 空状态 -->
      <div v-else-if="activeTab === 'vms'" class="text-center py-12 text-muted-foreground">
        <Monitor class="h-12 w-12 mx-auto mb-3 opacity-50" />
        <p>此宿主机上没有虚拟机</p>
      </div>
    </div>

    <!-- 编辑弹窗 -->
    <HostFormDialog
      :open="showEdit"
      :host="host"
      @update:open="showEdit = $event"
      @saved="onHostSaved"
    />

    <!-- 创建 VM 弹窗 -->
    <VMCreateDialog
      :open="showCreateVM"
      :hostId="hostId"
      @update:open="showCreateVM = $event"
      @saved="showCreateVM = false; loadVMs()"
    />

    <!-- 快速创建 VM 弹窗 -->
    <QuickCreateDialog
      :show="showQuickCreate"
      :hostId="hostId"
      @close="showQuickCreate = false"
      @created="loadVMs()"
    />
  </div>
</template>
