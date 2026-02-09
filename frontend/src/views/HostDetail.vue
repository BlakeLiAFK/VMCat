<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAppStore, type VM } from '@/stores/app'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'
import { useSettings } from '@/composables/useSettings'
import { SettingGet } from '../../wailsjs/go/main/App'
import {
  HostConnect, HostDisconnect, HostIsConnected, HostDelete, HostList,
  VMList, VMStart, VMShutdown, VMDestroy, VMReboot, VMDelete, VMClone, VMMigrate,
  HostResourceStats, HostGetFingerprint, HostResetHostKey, HostCheckTools,
} from '../../wailsjs/go/main/App'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import HostStatsChart from '@/components/HostStatsChart.vue'
import HostFormDialog from '@/components/HostFormDialog.vue'
import VMCreateDialog from '@/components/VMCreateDialog.vue'
import StoragePanel from '@/components/StoragePanel.vue'
import NetworkPanel from '@/components/NetworkPanel.vue'
import ImageManager from '@/components/ImageManager.vue'
import LibvirtSetupDialog from '@/components/LibvirtSetupDialog.vue'
import QuickCreateDialog from '@/components/QuickCreateDialog.vue'
import ContextMenu from '@/components/ui/ContextMenu.vue'
import type { MenuItem } from '@/components/ui/ContextMenu.vue'
import BatchProgressPanel from '@/components/BatchProgressPanel.vue'
import type { BatchTask } from '@/components/BatchProgressPanel.vue'
import VMMigrateDialog from '@/components/VMMigrateDialog.vue'
import BatchDeployDialog from '@/components/BatchDeployDialog.vue'
import {
  Plug, PlugZap, Play, Square, RotateCw, Skull,
  Monitor, Pencil, Trash2, Loader2, Terminal, Plus, Zap,
  Cpu, MemoryStick, HardDrive, Check, Minus, Search, Filter, Database, ShieldCheck, Globe, Image,
  AlertTriangle, Copy, ArrowRightLeft, Rocket, Download,
} from 'lucide-vue-next'

const { t } = useI18n()
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
const showStatsChart = ref(false)
const searchQuery = ref('')
const stateFilter = ref('all')
const fingerprint = ref('')
const connectError = ref('')
const toolStatus = ref<Record<string, string>>({})
const { refreshIntervalMs } = useSettings()
const alertThresholds = ref({ cpu: 90, mem: 90, disk: 85 })
const batchTasks = ref<BatchTask[]>([])
const showBatchProgress = ref(false)
const showBatchMigrate = ref(false)
const batchMigrateVM = ref('')
const showBatchDeploy = ref(false)
const showLibvirtSetup = ref(false)

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
  } catch { /* 静默 */ }
}

function isAlerted(value: number, threshold: number): boolean {
  return value >= threshold
}

const vmContextMenuRef = ref<InstanceType<typeof ContextMenu>>()
const vmContextMenuItems = ref<MenuItem[]>([])

const virshMissing = computed(() => connected.value && toolStatus.value.virsh === '')

// 响应式操作标签
const actionLabels = computed<Record<string, string>>(() => ({
  start: t('vm.start'),
  shutdown: t('vm.shutdown'),
  destroy: t('vm.destroy'),
  reboot: t('vm.reboot'),
}))

function showVMContextMenu(e: MouseEvent, v: VM) {
  const items: MenuItem[] = []
  if (v.state === 'shut off') {
    items.push({ label: t('vm.start'), action: () => vmAction(v.name, 'start') })
  }
  if (v.state === 'running') {
    items.push({ label: t('vm.shutdown'), action: () => vmAction(v.name, 'shutdown') })
    items.push({ label: t('vm.reboot'), action: () => vmAction(v.name, 'reboot') })
    items.push({ label: t('vm.destroy'), variant: 'destructive', action: () => vmAction(v.name, 'destroy') })
  }
  items.push({ label: '', action: () => {}, divider: true })
  items.push({ label: t('host.openTerminal'), action: () => router.push(`/host/${hostId.value}/terminal`) })
  if (v.state === 'running') {
    items.push({ label: t('vm.vncConsole'), action: () => router.push(`/host/${hostId.value}/vm/${v.name}/vnc`) })
  }
  items.push({ label: '', action: () => {}, divider: true })
  items.push({ label: t('vm.clone'), action: () => cloneVM(v.name) })
  if (v.state === 'running') {
    items.push({ label: t('vm.migrate'), action: () => { batchMigrateVM.value = v.name; showBatchMigrate.value = true } })
  }
  items.push({ label: t('vm.delete'), variant: 'destructive', action: () => deleteVM(v.name) })
  vmContextMenuItems.value = items
  vmContextMenuRef.value?.show(e)
}

async function cloneVM(vmName: string) {
  const newName = vmName + '-clone'
  const ok = await confirmRequest(t('host.cloneVM'), t('host.cloneVMMsg', { src: vmName, dst: newName }))
  if (!ok) return
  try {
    await VMClone(hostId.value, vmName, newName)
    toast.success(t('host.cloneSuccess', { name: newName }))
    setTimeout(loadVMs, 2000)
  } catch (e: any) {
    toast.error(t('host.cloneFailed') + ': ' + e.toString())
  }
}

async function deleteVM(vmName: string) {
  const ok = await confirmRequest(t('vm.deleteVM'), t('host.deleteVMConfirm', { name: vmName }), { variant: 'destructive', confirmText: t('common.delete') })
  if (!ok) return
  try {
    await VMDelete(hostId.value, vmName, false)
    toast.success(t('host.deleteVMSuccess', { name: vmName }))
    setTimeout(loadVMs, 1000)
  } catch (e: any) {
    toast.error(t('common.deleteFailed') + ': ' + e.toString())
  }
}

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
    toast.success(t('common.connectSuccess'))
    await loadVMs()
    loadStats()
    loadFingerprint()
    checkTools()
  } catch (e: any) {
    const msg = e.toString()
    connectError.value = msg
    if (msg.includes('key mismatch') || msg.includes('mismatch')) {
      toast.error(msg + '\n' + t('host.keyMismatchRetryTip'))
    } else {
      toast.error(t('host.connectFailedMsg', { msg }))
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
  toast.info(t('common.disconnected'))
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
    t('host.forgetFingerprintTitle'),
    t('host.forgetFingerprintMsg'),
  )
  if (!ok) return
  try {
    await HostResetHostKey(hostId.value)
    fingerprint.value = ''
    toast.success(t('host.forgotFingerprint'))
  } catch (e: any) {
    toast.error(t('common.operationFailed') + ': ' + e.toString())
  }
}

async function vmAction(vmName: string, action: string) {
  // 启动无需确认，其他操作需要二次确认
  const label = actionLabels.value[action] || action
  if (action !== 'start') {
    const ok = await confirmRequest(
      t('common.actionConfirm', { action: label }),
      t('common.actionConfirmMsg', { name: vmName, action: label }),
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
    toast.success(t('common.actionSuccess', { name: vmName, action: label }))
    setTimeout(loadVMs, 1500)
  } catch (e: any) {
    toast.error(t('common.operationFailed') + ': ' + e.toString())
  } finally {
    actionLoading.value[key] = false
  }
}

async function batchAction(action: string) {
  const names = Array.from(selectedVMs.value)
  if (names.length === 0) return
  const label = actionLabels.value[action] || action
  const ok = await confirmRequest(
    t('host.batchConfirm', { action: label }),
    t('host.batchConfirmMsg', { count: names.length, action: label }),
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
      toast.success(t('common.actionSuccess', { name, action: label }))
    } catch (e: any) {
      toast.error(t('common.operationFailed') + ': ' + e.toString())
    } finally {
      actionLoading.value[key] = false
    }
  }
  selectedVMs.value.clear()
  setTimeout(loadVMs, 1500)
}

function exportVMsCSV() {
  const header = t('host.csvHeader') + '\n'
  const rows = filteredVMs.value.map(v =>
    `${v.name},${v.state},${v.cpus},${v.memoryMB}`
  ).join('\n')
  const blob = new Blob([header + rows], { type: 'text/csv' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `vmcat-vms-${host.value?.name || 'export'}-${new Date().toISOString().slice(0, 10)}.csv`
  a.click()
  URL.revokeObjectURL(url)
}

async function batchDelete() {
  const names = Array.from(selectedVMs.value)
  if (names.length === 0) return
  const ok = await confirmRequest(t('host.batchDelete'), t('host.batchDeleteConfirm', { count: names.length }), { variant: 'destructive', confirmText: t('common.delete') })
  if (!ok) return
  batchTasks.value = names.map(n => ({ name: n, status: 'pending' as const }))
  showBatchProgress.value = true
  for (let i = 0; i < names.length; i++) {
    batchTasks.value[i].status = 'running'
    try {
      await VMDelete(hostId.value, names[i], false)
      batchTasks.value[i].status = 'success'
    } catch (e: any) {
      batchTasks.value[i].status = 'error'
      batchTasks.value[i].error = e.toString()
    }
  }
  selectedVMs.value.clear()
  setTimeout(loadVMs, 1000)
}

async function deleteHost() {
  const ok = await confirmRequest(t('host.deleteHost'), t('host.deleteConfirm'), { variant: 'destructive', confirmText: t('common.delete') })
  if (!ok) return
  try {
    await HostDelete(hostId.value)
    toast.success(t('common.deleted'))
    router.push('/')
  } catch (e: any) {
    toast.error(t('common.deleteFailed') + ': ' + e.toString())
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

// 响应式状态标签
const stateLabels = computed<Record<string, string>>(() => ({
  'running': t('vm.running'),
  'shut off': t('vm.shutOff'),
  'paused': t('vm.paused'),
  'idle': t('vm.idle'),
  'crashed': t('vm.crashed'),
}))

function stateLabel(state: string) {
  return stateLabels.value[state] || state
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
  loadAlertThresholds()
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
          <button @click="resetHostKey" class="text-xs text-muted-foreground hover:text-destructive ml-1" :title="t('host.forgetFingerprintTip')">
            {{ t('host.forgetFingerprint') }}
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
          {{ t('host.terminal') }}
        </Button>
        <Button v-if="!connected" variant="default" :loading="connecting" @click="connect">
          <Plug class="h-4 w-4" />
          {{ t('host.connect') }}
        </Button>
        <Button v-else variant="outline" @click="disconnect">
          <PlugZap class="h-4 w-4" />
          {{ t('host.disconnect') }}
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
      <Card v-if="connectError && (connectError.includes('key mismatch') || connectError.includes('mismatch'))" class="max-w-md mx-auto mb-6 text-left border-destructive/50">
        <div class="p-4">
          <h3 class="text-sm font-semibold text-destructive flex items-center gap-2 mb-2">
            <ShieldCheck class="h-4 w-4" /> {{ t('host.fingerprintMismatch') }}
          </h3>
          <p class="text-xs text-muted-foreground mb-3">
            {{ t('host.fingerprintMismatchTip') }}
          </p>
          <div class="flex gap-2">
            <Button variant="destructive" size="sm" @click="resetHostKey">
              {{ t('host.forgetFingerprint') }}
            </Button>
            <Button variant="outline" size="sm" @click="connect">
              {{ t('host.retryConnect') }}
            </Button>
          </div>
        </div>
      </Card>
      <!-- 普通连接失败提示 -->
      <Card v-else-if="connectError" class="max-w-md mx-auto mb-6 text-left">
        <div class="p-4">
          <p class="text-sm text-destructive mb-2">{{ t('host.connectFailed') }}</p>
          <p class="text-xs text-muted-foreground font-mono break-all mb-3">{{ connectError }}</p>
          <Button variant="outline" size="sm" @click="connect">
            {{ t('host.retryConnect') }}
          </Button>
        </div>
      </Card>
      <template v-else>
        <Plug class="h-12 w-12 mx-auto mb-3 opacity-50" />
        <p>{{ t('host.pleaseConnect') }}</p>
        <Button class="mt-4" :loading="connecting" @click="connect">
          <Plug class="h-4 w-4" />
          {{ t('host.connect') }}
        </Button>
      </template>
    </div>

    <!-- 连接中 -->
    <div v-else-if="connecting" class="text-center py-20">
      <Loader2 class="h-8 w-8 mx-auto animate-spin text-muted-foreground" />
      <p class="text-sm text-muted-foreground mt-2">{{ t('host.connecting') }}</p>
    </div>

    <!-- 已连接 -->
    <div v-else>
      <!-- 资源概览 -->
      <div v-if="stats" class="grid grid-cols-4 gap-3 mb-6">
        <Card class="p-3" :class="isAlerted(stats.cpuPercent, alertThresholds.cpu) ? 'border-red-500/50 bg-red-500/5' : ''">
          <div class="flex items-center gap-2">
            <Cpu class="h-4 w-4" :class="isAlerted(stats.cpuPercent, alertThresholds.cpu) ? 'text-red-500' : 'text-blue-500'" />
            <span class="text-sm text-muted-foreground">CPU</span>
          </div>
          <p class="text-lg font-semibold mt-1" :class="isAlerted(stats.cpuPercent, alertThresholds.cpu) ? 'text-red-500' : ''">{{ stats.cpuPercent.toFixed(1) }}%</p>
          <div class="mt-1 h-1.5 bg-muted rounded-full overflow-hidden">
            <div class="h-full rounded-full transition-all" :class="isAlerted(stats.cpuPercent, alertThresholds.cpu) ? 'bg-red-500' : 'bg-blue-500'" :style="{ width: stats.cpuPercent + '%' }" />
          </div>
        </Card>
        <Card class="p-3" :class="isAlerted(stats.memPercent, alertThresholds.mem) ? 'border-red-500/50 bg-red-500/5' : ''">
          <div class="flex items-center gap-2">
            <MemoryStick class="h-4 w-4" :class="isAlerted(stats.memPercent, alertThresholds.mem) ? 'text-red-500' : 'text-purple-500'" />
            <span class="text-sm text-muted-foreground">{{ t('dashboard.memory') }}</span>
          </div>
          <p class="text-lg font-semibold mt-1" :class="isAlerted(stats.memPercent, alertThresholds.mem) ? 'text-red-500' : ''">{{ stats.memPercent.toFixed(1) }}%</p>
          <p class="text-xs text-muted-foreground">{{ formatMem(stats.memUsed) }} / {{ formatMem(stats.memTotal) }}</p>
        </Card>
        <Card class="p-3" :class="isAlerted(stats.diskPercent, alertThresholds.disk) ? 'border-red-500/50 bg-red-500/5' : ''">
          <div class="flex items-center gap-2">
            <HardDrive class="h-4 w-4" :class="isAlerted(stats.diskPercent, alertThresholds.disk) ? 'text-red-500' : 'text-orange-500'" />
            <span class="text-sm text-muted-foreground">{{ t('dashboard.disk') }}</span>
          </div>
          <p class="text-lg font-semibold mt-1" :class="isAlerted(stats.diskPercent, alertThresholds.disk) ? 'text-red-500' : ''">{{ stats.diskPercent.toFixed(1) }}%</p>
          <p class="text-xs text-muted-foreground">{{ stats.diskUsed }}G / {{ stats.diskTotal }}G</p>
        </Card>
        <Card class="p-3">
          <div class="flex items-center gap-2">
            <Monitor class="h-4 w-4 text-green-500" />
            <span class="text-sm text-muted-foreground">VM</span>
          </div>
          <p class="text-lg font-semibold mt-1">{{ vms.filter(v => v.state === 'running').length }} / {{ vms.length }}</p>
          <p class="text-xs text-muted-foreground">{{ t('dashboard.runningSlashTotal') }}</p>
        </Card>
      </div>

      <!-- 趋势图 -->
      <div v-if="stats" class="mb-4">
        <button
          class="text-xs text-muted-foreground hover:text-foreground transition-colors"
          @click="showStatsChart = !showStatsChart"
        >
          {{ showStatsChart ? t('vm.collapseChart') : t('vm.expandChart') }}
        </button>
        <Card v-if="showStatsChart" class="mt-2 p-4">
          <HostStatsChart :hostId="hostId" />
        </Card>
      </div>

      <!-- virsh 未安装提示 -->
      <Card v-if="virshMissing" class="mb-4 border-yellow-500/50 bg-yellow-500/5">
        <div class="p-3 flex items-center gap-3">
          <AlertTriangle class="h-4 w-4 text-yellow-500 flex-shrink-0" />
          <div class="flex-1 text-sm">
            <span class="font-medium text-yellow-600 dark:text-yellow-400">{{ t('host.virshMissing') }}</span>
            <span class="text-muted-foreground ml-1">{{ t('host.virshMissingTip') }}</span>
          </div>
          <div class="flex gap-2">
            <Button variant="default" size="sm" @click="showLibvirtSetup = true">
              <Download class="h-3.5 w-3.5" /> {{ t('host.oneClickInstall') }}
            </Button>
            <Button variant="outline" size="sm" @click="router.push(`/host/${hostId}/terminal`)">
              <Terminal class="h-3.5 w-3.5" /> {{ t('host.openTerminal') }}
            </Button>
          </div>
        </div>
      </Card>
      <LibvirtSetupDialog
        :open="showLibvirtSetup"
        :hostId="hostId"
        @update:open="showLibvirtSetup = $event"
        @installed="checkTools"
      />

      <!-- Tab 切换 -->
      <div class="flex border-b mb-4">
        <button
          class="px-4 py-2.5 text-sm border-b-2 transition-colors -mb-px"
          :class="activeTab === 'vms' ? 'border-primary text-foreground font-medium' : 'border-transparent text-muted-foreground hover:text-foreground'"
          @click="activeTab = 'vms'"
        >
          <Monitor class="h-3.5 w-3.5 inline mr-1" /> {{ t('tabs.vms') }}
        </button>
        <button
          class="px-4 py-2.5 text-sm border-b-2 transition-colors -mb-px"
          :class="activeTab === 'storage' ? 'border-primary text-foreground font-medium' : 'border-transparent text-muted-foreground hover:text-foreground'"
          @click="activeTab = 'storage'"
        >
          <Database class="h-3.5 w-3.5 inline mr-1" /> {{ t('tabs.storage') }}
        </button>
        <button
          class="px-4 py-2.5 text-sm border-b-2 transition-colors -mb-px"
          :class="activeTab === 'network' ? 'border-primary text-foreground font-medium' : 'border-transparent text-muted-foreground hover:text-foreground'"
          @click="activeTab = 'network'"
        >
          <Globe class="h-3.5 w-3.5 inline mr-1" /> {{ t('tabs.network') }}
        </button>
        <button
          class="px-4 py-2.5 text-sm border-b-2 transition-colors -mb-px"
          :class="activeTab === 'images' ? 'border-primary text-foreground font-medium' : 'border-transparent text-muted-foreground hover:text-foreground'"
          @click="activeTab = 'images'"
        >
          <Image class="h-3.5 w-3.5 inline mr-1" /> {{ t('tabs.images') }}
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
        <h2 class="text-lg font-semibold">{{ t('host.vmCount') }} ({{ filteredVMs.length }}<span v-if="filteredVMs.length !== vms.length" class="text-muted-foreground font-normal">/{{ vms.length }}</span>)</h2>
        <div class="flex items-center gap-2">
          <!-- 搜索框 -->
          <div class="relative">
            <Search class="h-3.5 w-3.5 absolute left-2.5 top-1/2 -translate-y-1/2 text-muted-foreground" />
            <input
              v-model="searchQuery"
              :placeholder="t('vm.searchVM')"
              class="h-8 w-40 pl-8 pr-2 text-sm rounded-md border border-input bg-transparent focus:outline-none focus:ring-1 focus:ring-ring"
            />
          </div>
          <!-- 状态筛选 -->
          <select
            v-model="stateFilter"
            class="h-8 text-sm rounded-md border border-input bg-transparent px-2 focus:outline-none focus:ring-1 focus:ring-ring"
          >
            <option value="all">{{ t('vm.allStates') }}</option>
            <option value="running">{{ t('vm.running') }}</option>
            <option value="shut off">{{ t('vm.shutOff') }}</option>
            <option value="paused">{{ t('vm.paused') }}</option>
          </select>
          <!-- 批量操作 -->
          <template v-if="selectedVMs.size > 0">
            <span class="text-sm text-muted-foreground">{{ t('vm.selected') }} {{ selectedVMs.size }}</span>
            <Button variant="outline" size="sm" @click="batchAction('start')">
              <Play class="h-3.5 w-3.5" /> {{ t('vm.start') }}
            </Button>
            <Button variant="outline" size="sm" @click="batchAction('shutdown')">
              <Square class="h-3.5 w-3.5" /> {{ t('vm.shutdown') }}
            </Button>
            <Button variant="destructive" size="sm" @click="batchDelete">
              <Trash2 class="h-3.5 w-3.5" /> {{ t('vm.delete') }}
            </Button>
          </template>
          <Button variant="outline" size="sm" @click="exportVMsCSV" :disabled="filteredVMs.length === 0" :title="t('common.export') + ' CSV'">
            <Download class="h-3.5 w-3.5" />
          </Button>
          <Button variant="outline" size="sm" @click="loadVMs" :loading="loadingVMs">
            <RotateCw class="h-3.5 w-3.5" />
            {{ t('common.refresh') }}
          </Button>
          <Button size="sm" variant="outline" @click="showQuickCreate = true">
            <Zap class="h-3.5 w-3.5" />
            {{ t('vm.quickCreate') }}
          </Button>
          <Button size="sm" variant="outline" @click="showBatchDeploy = true">
            <Rocket class="h-3.5 w-3.5" />
            {{ t('vm.batchDeploy') }}
          </Button>
          <Button size="sm" @click="showCreateVM = true">
            <Plus class="h-3.5 w-3.5" />
            {{ t('vm.create') }}
          </Button>
        </div>
      </div>

      <!-- 加载中 -->
      <div v-if="activeTab === 'vms' && loadingVMs && vms.length === 0" class="text-center py-12">
        <Loader2 class="h-8 w-8 mx-auto animate-spin text-muted-foreground" />
        <p class="text-sm text-muted-foreground mt-2">{{ t('host.loadingVMs') }}</p>
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
              <th class="text-left p-3 font-medium">{{ t('host.thState') }}</th>
              <th class="text-left p-3 font-medium">{{ t('host.thName') }}</th>
              <th class="text-left p-3 font-medium">{{ t('host.thCPU') }}</th>
              <th class="text-left p-3 font-medium">{{ t('host.thMemory') }}</th>
              <th class="text-right p-3 font-medium">{{ t('host.thActions') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="v in filteredVMs"
              :key="v.name"
              class="border-b last:border-0 hover:bg-muted/50 cursor-pointer transition-colors"
              :class="{ 'bg-accent/30': selectedVMs.has(v.name) }"
              @click="router.push(`/host/${hostId}/vm/${v.name}`)"
              @contextmenu.prevent="showVMContextMenu($event, v)"
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
                    {{ t('vm.start') }}
                  </Button>
                  <Button
                    v-if="v.state === 'running'"
                    variant="outline" size="sm"
                    :loading="isActionLoading(v.name, 'shutdown')"
                    @click="vmAction(v.name, 'shutdown')"
                  >
                    <Square class="h-3.5 w-3.5" />
                    {{ t('vm.shutdown') }}
                  </Button>
                  <Button
                    v-if="v.state === 'running'"
                    variant="outline" size="sm"
                    :loading="isActionLoading(v.name, 'reboot')"
                    @click="vmAction(v.name, 'reboot')"
                  >
                    <RotateCw class="h-3.5 w-3.5" />
                    {{ t('vm.reboot') }}
                  </Button>
                  <Button
                    v-if="v.state === 'running'"
                    variant="destructive" size="sm"
                    :loading="isActionLoading(v.name, 'destroy')"
                    @click="vmAction(v.name, 'destroy')"
                  >
                    <Skull class="h-3.5 w-3.5" />
                    {{ t('vm.destroy') }}
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
        <p>{{ t('host.noVMsOnHost') }}</p>
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

    <!-- VM 右键菜单 -->
    <ContextMenu ref="vmContextMenuRef" :items="vmContextMenuItems" />

    <!-- 批量操作进度 -->
    <BatchProgressPanel :tasks="batchTasks" :visible="showBatchProgress" @close="showBatchProgress = false" />

    <!-- 单个 VM 迁移弹窗 -->
    <VMMigrateDialog
      :open="showBatchMigrate"
      :hostId="hostId"
      :vmName="batchMigrateVM"
      @update:open="showBatchMigrate = $event"
      @migrated="loadVMs()"
    />

    <!-- 批量部署弹窗 -->
    <BatchDeployDialog
      :open="showBatchDeploy"
      @update:open="showBatchDeploy = $event"
      @deployed="loadVMs()"
    />
  </div>
</template>
