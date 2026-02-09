<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'
import {
  VMGet, VMStart, VMShutdown, VMDestroy, VMReboot, VMSuspend, VMResume, VMDelete,
  VMStats, SnapshotList, SnapshotCreate, SnapshotDelete, SnapshotRevert,
  VMNoteGet, VMNoteSet, NATRuleList,
} from '../../wailsjs/go/main/App'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import Input from '@/components/ui/Input.vue'
import VMEditDialog from '@/components/VMEditDialog.vue'
import VMHardwareDialog from '@/components/VMHardwareDialog.vue'
import VMXMLDialog from '@/components/VMXMLDialog.vue'
import VMCloneDialog from '@/components/VMCloneDialog.vue'
import VMMigrateDialog from '@/components/VMMigrateDialog.vue'
import VMStatsChart from '@/components/VMStatsChart.vue'
import {
  ArrowLeft, Play, Square, RotateCw, Skull, Pause, PlayCircle,
  Cpu, MemoryStick, HardDrive, Network, Loader2,
  Camera, RotateCcw, Trash2, Plus, Terminal, Monitor as MonitorIcon, ScreenShare,
  Pencil, Copy, Code, Settings, Wrench,
  Activity, ArrowDown, ArrowUp, TrendingUp, MessageSquare, Save, ArrowRightLeft,
} from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const toast = useToast()
const { request: confirmRequest } = useConfirm()

const hostId = computed(() => route.params.id as string)
const vmName = computed(() => route.params.name as string)
const detail = ref<any>(null)
const loading = ref(true)
const actionLoading = ref('')

// 对话框状态
const showEdit = ref(false)
const showHardware = ref(false)
const showXML = ref(false)
const showClone = ref(false)
const showMigrate = ref(false)

const isWindows = computed(() => {
  const name = vmName.value.toLowerCase()
  return name.includes('win') || name.includes('windows')
})

// 备注
const note = ref('')
const noteSaving = ref(false)
const showStatsChart = ref(false)
const vmNATRules = ref<any[]>([])

// 实时资源统计
const stats = ref<any>(null)
let statsTimer: ReturnType<typeof setInterval> | null = null

function formatBytes(bytes: number) {
  if (!bytes || bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return (bytes / Math.pow(1024, i)).toFixed(1) + ' ' + units[i]
}

async function loadStats() {
  if (detail.value?.state !== 'running') {
    stats.value = null
    return
  }
  try {
    stats.value = await VMStats(hostId.value, vmName.value)
  } catch { /* 静默 */ }
}

function startStatsPolling() {
  stopStatsPolling()
  loadStats()
  statsTimer = setInterval(loadStats, 5000)
}

function stopStatsPolling() {
  if (statsTimer) { clearInterval(statsTimer); statsTimer = null }
}

watch(() => detail.value?.state, (state) => {
  if (state === 'running') startStatsPolling()
  else stopStatsPolling()
})

// 快照相关
const snapshots = ref<any[]>([])
const loadingSnaps = ref(false)
const newSnapName = ref('')
const creatingSnap = ref(false)
const snapActionLoading = ref<Record<string, boolean>>({})

async function load() {
  loading.value = true
  try {
    detail.value = await VMGet(hostId.value, vmName.value)
    loadVMNATRules()
  } catch (e: any) {
    console.error('load vm:', e)
    toast.error('加载 VM 信息失败')
  } finally {
    loading.value = false
  }
}

async function loadVMNATRules() {
  try {
    const all = (await NATRuleList(hostId.value)) || []
    const ips = (detail.value?.nics || []).map((n: any) => n.ip).filter(Boolean)
    vmNATRules.value = ips.length > 0 ? all.filter((r: any) => ips.includes(r.vmIP)) : []
  } catch { vmNATRules.value = [] }
}

async function loadSnapshots() {
  loadingSnaps.value = true
  try {
    const list = await SnapshotList(hostId.value, vmName.value)
    snapshots.value = list || []
  } catch { /* 静默 */ }
  finally { loadingSnaps.value = false }
}

async function createSnapshot() {
  const name = newSnapName.value.trim()
  if (!name) { toast.warning(t('vm.enterSnapshotName')); return }
  creatingSnap.value = true
  try {
    await SnapshotCreate(hostId.value, vmName.value, name)
    toast.success(t('vm.snapshotCreated', { name }))
    newSnapName.value = ''
    await loadSnapshots()
  } catch (e: any) { toast.error(t('vm.snapshotCreateFailed') + ': ' + e.toString()) }
  finally { creatingSnap.value = false }
}

async function revertSnapshot(snapName: string) {
  const ok = await confirmRequest(t('vm.snapshotRevertTitle'), t('vm.revertConfirm', { name: snapName }))
  if (!ok) return
  snapActionLoading.value[snapName] = true
  try {
    await SnapshotRevert(hostId.value, vmName.value, snapName)
    toast.success(t('vm.revertSuccess', { name: snapName }))
    await load()
  } catch (e: any) { toast.error(t('vm.revertFailed') + ': ' + e.toString()) }
  finally { snapActionLoading.value[snapName] = false }
}

async function deleteSnapshot(snapName: string) {
  const ok = await confirmRequest(t('vm.snapshotConfirmTitle'), t('vm.deleteSnapshotConfirm', { name: snapName }), { variant: 'destructive', confirmText: t('common.delete') })
  if (!ok) return
  snapActionLoading.value[`del-${snapName}`] = true
  try {
    await SnapshotDelete(hostId.value, vmName.value, snapName)
    toast.success(t('vm.snapshotDeleted', { name: snapName }))
    await loadSnapshots()
  } catch (e: any) { toast.error(t('vm.snapshotDeleteFailed') + ': ' + e.toString()) }
  finally { snapActionLoading.value[`del-${snapName}`] = false }
}

const actionLabels: Record<string, string> = {
  start: 'vm.start', shutdown: 'vm.shutdown', destroy: 'vm.destroy',
  reboot: 'vm.reboot', suspend: 'vm.suspend', resume: 'vm.resume',
}

async function doAction(action: string) {
  // 启动和恢复无需确认
  if (action !== 'start' && action !== 'resume') {
    const ok = await confirmRequest(
      t('common.actionConfirm', { action: t(actionLabels[action]) }),
      t('vm.deleteConfirmMsg', { name: vmName.value, action: t(actionLabels[action]) }),
      { variant: action === 'destroy' ? 'destructive' : 'default' },
    )
    if (!ok) return
  }
  actionLoading.value = action
  try {
    switch (action) {
      case 'start': await VMStart(hostId.value, vmName.value); break
      case 'shutdown': await VMShutdown(hostId.value, vmName.value); break
      case 'destroy': await VMDestroy(hostId.value, vmName.value); break
      case 'reboot': await VMReboot(hostId.value, vmName.value); break
      case 'suspend': await VMSuspend(hostId.value, vmName.value); break
      case 'resume': await VMResume(hostId.value, vmName.value); break
    }
    toast.success(t('vm.operationActionSuccess', { name: vmName.value, action: t(actionLabels[action]) || action }))
    setTimeout(load, 1500)
  } catch (e: any) { toast.error(t('common.operationFailed') + ': ' + e.toString()) }
  finally { actionLoading.value = '' }
}

async function deleteVM() {
  const removeStorage = await confirmRequest(
    t('vm.deleteDiskConfirmTitle'),
    t('vm.deleteDiskConfirmMsg'),
    { variant: 'destructive', confirmText: t('vm.deleteDisk'), cancelText: t('vm.onlyUndefine') },
  )
  const ok = await confirmRequest(
    t('vm.deleteConfirmTitle'),
    t('vm.deleteVMConfirm', { name: vmName.value }),
    { variant: 'destructive', confirmText: t('common.delete') },
  )
  if (!ok) return
  try {
    await VMDelete(hostId.value, vmName.value, removeStorage)
    toast.success(t('vm.vmDeleted'))
    router.push(`/host/${hostId.value}`)
  } catch (e: any) { toast.error(t('common.deleteFailed') + ': ' + e.toString()) }
}

function onEditSaved(newName?: string) {
  if (newName && newName !== vmName.value) {
    router.replace(`/host/${hostId.value}/vm/${newName}`)
  }
  load()
}

function stateVariant(state: string) {
  if (state === 'running') return 'success'
  if (state === 'shut off') return 'secondary'
  if (state === 'paused') return 'warning'
  return 'outline'
}

function stateLabel(state: string) {
  const map: Record<string, string> = {
    'running': 'vm.running', 'shut off': 'vm.shutOff', 'paused': 'vm.paused',
    'idle': 'vm.idle', 'crashed': 'vm.crashed',
  }
  return t(map[state] || state)
}

function formatMem(mb: number) {
  return mb >= 1024 ? (mb / 1024).toFixed(1) + ' GB' : mb + ' MB'
}

async function loadNote() {
  try {
    note.value = await VMNoteGet(hostId.value, vmName.value)
  } catch { note.value = '' }
}

async function saveNote() {
  noteSaving.value = true
  try {
    await VMNoteSet(hostId.value, vmName.value, note.value)
    toast.success(t('vm.noteSaved'))
  } catch (e: any) { toast.error(t('common.saveFailed') + ': ' + e.toString()) }
  finally { noteSaving.value = false }
}

onMounted(() => { load(); loadSnapshots(); loadNote() })
onUnmounted(() => { stopStatsPolling() })
</script>

<template>
  <div class="p-6">
    <!-- 返回 -->
    <button
      class="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground mb-4 transition-colors"
      @click="router.push(`/host/${hostId}`)"
    >
      <ArrowLeft class="h-4 w-4" />
      {{ t('vm.backToList') }}
    </button>

    <!-- 加载中 -->
    <div v-if="loading" class="text-center py-20">
      <Loader2 class="h-8 w-8 mx-auto animate-spin text-muted-foreground" />
    </div>

    <template v-else-if="detail">
      <!-- 标题栏 -->
      <div class="flex items-center justify-between mb-6">
        <div class="flex items-center gap-3">
          <h1 class="text-2xl font-bold">{{ detail.name }}</h1>
          <Badge :variant="stateVariant(detail.state)">{{ stateLabel(detail.state) }}</Badge>
        </div>
        <div class="flex items-center gap-2 flex-wrap">
          <Button
            v-if="detail.state === 'running' && detail.vncPort > 0"
            variant="outline" size="sm"
            @click="router.push(`/host/${hostId}/vm/${vmName}/vnc`)"
          >
            <ScreenShare class="h-4 w-4" /> VNC
          </Button>
          <Button
            v-if="detail.state === 'running' && !isWindows"
            variant="outline" size="sm"
            @click="router.push({ path: `/host/${hostId}/terminal`, query: { cmd: `virsh console ${vmName}` } })"
          >
            <Terminal class="h-4 w-4" /> {{ t('vm.console') }}
          </Button>
          <Button v-if="detail.state === 'shut off'" @click="doAction('start')" :loading="actionLoading === 'start'">
            <Play class="h-4 w-4" /> {{ t('vm.start') }}
          </Button>
          <Button v-if="detail.state === 'running'" variant="outline" @click="doAction('shutdown')" :loading="actionLoading === 'shutdown'">
            <Square class="h-4 w-4" /> {{ t('vm.shutdown') }}
          </Button>
          <Button v-if="detail.state === 'running'" variant="outline" @click="doAction('reboot')" :loading="actionLoading === 'reboot'">
            <RotateCw class="h-4 w-4" /> {{ t('vm.reboot') }}
          </Button>
          <Button v-if="detail.state === 'running'" variant="outline" @click="doAction('suspend')" :loading="actionLoading === 'suspend'">
            <Pause class="h-4 w-4" /> {{ t('vm.suspend') }}
          </Button>
          <Button v-if="detail.state === 'paused'" @click="doAction('resume')" :loading="actionLoading === 'resume'">
            <PlayCircle class="h-4 w-4" /> {{ t('vm.resume') }}
          </Button>
          <Button v-if="detail.state === 'running'" variant="destructive" @click="doAction('destroy')" :loading="actionLoading === 'destroy'">
            <Skull class="h-4 w-4" /> {{ t('vm.destroy') }}
          </Button>
          <div class="w-px h-6 bg-border mx-1" />
          <Button variant="outline" size="sm" @click="showEdit = true" :title="t('common.edit')">
            <Pencil class="h-4 w-4" /> {{ t('common.edit') }}
          </Button>
          <Button variant="outline" size="sm" @click="showHardware = true" :title="t('vm.hardware')">
            <Wrench class="h-4 w-4" /> {{ t('vm.hardware') }}
          </Button>
          <Button variant="outline" size="sm" @click="showClone = true" :title="t('vm.clone')">
            <Copy class="h-4 w-4" /> {{ t('vm.clone') }}
          </Button>
          <Button v-if="detail.state === 'running' || detail.state === 'shut off'" variant="outline" size="sm" @click="showMigrate = true" :title="t('vm.migrate')">
            <ArrowRightLeft class="h-4 w-4" /> {{ t('vm.migrate') }}
          </Button>
          <Button variant="outline" size="sm" @click="showXML = true" :title="t('vm.xml')">
            <Code class="h-4 w-4" /> {{ t('vm.xml') }}
          </Button>
          <Button variant="destructive" size="sm" @click="deleteVM" :title="t('common.delete')">
            <Trash2 class="h-4 w-4" /> {{ t('common.delete') }}
          </Button>
        </div>
      </div>

      <!-- 备注 -->
      <div class="mb-4 flex items-center gap-2">
        <MessageSquare class="h-3.5 w-3.5 text-muted-foreground flex-shrink-0" />
        <input
          v-model="note"
          :placeholder="t('vm.note')"
          class="flex-1 text-sm bg-transparent border-b border-transparent hover:border-input focus:border-primary focus:outline-none py-0.5 transition-colors"
          @blur="saveNote"
          @keyup.enter="($event.target as HTMLInputElement)?.blur()"
        />
      </div>

      <!-- 资源信息 -->
      <div class="grid grid-cols-4 gap-4 mb-6">
        <Card class="p-4">
          <div class="flex items-center gap-3">
            <Cpu class="h-5 w-5 text-muted-foreground" />
            <div>
              <p class="text-sm text-muted-foreground">CPU</p>
              <p class="text-lg font-semibold">{{ detail.cpus }} vCPU</p>
            </div>
          </div>
        </Card>
        <Card class="p-4">
          <div class="flex items-center gap-3">
            <MemoryStick class="h-5 w-5 text-muted-foreground" />
            <div>
              <p class="text-sm text-muted-foreground">{{ t('vm.memory') }}</p>
              <p class="text-lg font-semibold">{{ formatMem(detail.memoryMB) }}</p>
            </div>
          </div>
        </Card>
        <Card class="p-4">
          <div class="flex items-center gap-3">
            <HardDrive class="h-5 w-5 text-muted-foreground" />
            <div>
              <p class="text-sm text-muted-foreground">{{ t('vm.disks') }}</p>
              <p class="text-lg font-semibold">{{ t('vm.diskCount', { count: detail.disks?.length || 0 }) }}</p>
            </div>
          </div>
        </Card>
        <Card class="p-4">
          <div class="flex items-center gap-3">
            <MonitorIcon class="h-5 w-5 text-muted-foreground" />
            <div>
              <p class="text-sm text-muted-foreground">VNC</p>
              <p class="text-lg font-semibold">{{ detail.vncPort > 0 ? ':' + detail.vncPort : '-' }}</p>
            </div>
          </div>
        </Card>
      </div>

      <!-- 实时资源监控 -->
      <Card class="mb-6" v-if="detail.state === 'running' && stats">
        <div class="p-4 border-b">
          <h3 class="font-semibold flex items-center gap-2">
            <Activity class="h-4 w-4" /> {{ t('vm.realTimeMonitor') }}
          </h3>
        </div>
        <div class="grid grid-cols-4 gap-4 p-4">
          <div>
            <p class="text-xs text-muted-foreground mb-1">{{ t('vm.cpuUsage') }}</p>
            <p class="text-xl font-bold" :class="stats.cpuPercent > 80 ? 'text-red-500' : stats.cpuPercent > 50 ? 'text-amber-500' : 'text-green-500'">
              {{ stats.cpuPercent.toFixed(1) }}%
            </p>
            <div class="h-1.5 bg-muted rounded-full mt-2 overflow-hidden">
              <div class="h-full rounded-full transition-all" :class="stats.cpuPercent > 80 ? 'bg-red-500' : stats.cpuPercent > 50 ? 'bg-amber-500' : 'bg-green-500'" :style="{ width: Math.min(stats.cpuPercent, 100) + '%' }" />
            </div>
          </div>
          <div>
            <p class="text-xs text-muted-foreground mb-1">{{ t('vm.memory') }}</p>
            <p class="text-xl font-bold">{{ formatBytes(stats.memRSS * 1024) }}</p>
            <p class="text-xs text-muted-foreground mt-1">{{ t('vm.allocated') }}: {{ formatBytes(stats.memActual * 1024) }}</p>
          </div>
          <div>
            <p class="text-xs text-muted-foreground mb-1">{{ t('vm.networkIO') }}</p>
            <div class="flex items-center gap-1 mt-1">
              <ArrowDown class="h-3 w-3 text-blue-500" />
              <span class="text-sm font-medium">{{ formatBytes(stats.netRxBytes) }}</span>
            </div>
            <div class="flex items-center gap-1 mt-1">
              <ArrowUp class="h-3 w-3 text-green-500" />
              <span class="text-sm font-medium">{{ formatBytes(stats.netTxBytes) }}</span>
            </div>
          </div>
          <div>
            <p class="text-xs text-muted-foreground mb-1">{{ t('vm.diskIO') }}</p>
            <div class="flex items-center gap-1 mt-1">
              <ArrowDown class="h-3 w-3 text-blue-500" />
              <span class="text-sm font-medium">{{ formatBytes(stats.blockRdBytes) }}</span>
            </div>
            <div class="flex items-center gap-1 mt-1">
              <ArrowUp class="h-3 w-3 text-green-500" />
              <span class="text-sm font-medium">{{ formatBytes(stats.blockWrBytes) }}</span>
            </div>
          </div>
        </div>
      </Card>

      <!-- 趋势图 -->
      <div v-if="detail.state === 'running'" class="mb-4">
        <button
          class="text-xs text-muted-foreground hover:text-foreground transition-colors flex items-center gap-1"
          @click="showStatsChart = !showStatsChart"
        >
          <TrendingUp class="h-3 w-3" />
          {{ showStatsChart ? t('vm.collapseChart') : t('vm.expandChart') }}
        </button>
        <Card v-if="showStatsChart" class="mt-2 p-4">
          <VMStatsChart :hostId="hostId" :vmName="vmName" />
        </Card>
      </div>

      <!-- 网络信息 -->
      <Card class="mb-6" v-if="detail.nics?.length">
        <div class="p-4 border-b">
          <h3 class="font-semibold flex items-center gap-2">
            <Network class="h-4 w-4" /> {{ t('vm.networkInterface') }}
          </h3>
        </div>
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b text-muted-foreground">
              <th class="text-left p-3 font-medium">{{ t('vm.thMAC') }}</th>
              <th class="text-left p-3 font-medium">{{ t('vm.thBridgeNetwork') }}</th>
              <th class="text-left p-3 font-medium">{{ t('vm.thIP') }}</th>
              <th class="text-left p-3 font-medium">{{ t('vm.thModel') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="nic in detail.nics" :key="nic.mac" class="border-b last:border-0">
              <td class="p-3 font-mono text-xs selectable">{{ nic.mac }}</td>
              <td class="p-3">{{ nic.bridge || nic.network || '-' }}</td>
              <td class="p-3 font-mono selectable">{{ nic.ip || '-' }}</td>
              <td class="p-3">{{ nic.model || '-' }}</td>
            </tr>
          </tbody>
        </table>
      </Card>

      <!-- NAT 端口转发 -->
      <Card class="mb-6" v-if="vmNATRules.length">
        <div class="p-4 border-b">
          <h3 class="font-semibold flex items-center gap-2">
            <ArrowRightLeft class="h-4 w-4" /> {{ t('vm.natPortForward') }}
          </h3>
        </div>
        <div class="p-3 space-y-2">
          <div v-for="(rule, i) in vmNATRules" :key="i"
            class="flex items-center gap-3 p-2.5 rounded-lg bg-muted/30 text-sm">
            <span class="px-2 py-0.5 rounded text-xs bg-blue-500/10 text-blue-600 font-mono uppercase">{{ rule.proto }}</span>
            <span class="font-mono">{{ t('vm.hostPort') }}:{{ rule.hostPort }}</span>
            <ArrowRightLeft class="h-3.5 w-3.5 text-muted-foreground" />
            <span class="font-mono">{{ rule.vmIP }}:{{ rule.vmPort }}</span>
            <span v-if="rule.comment" class="text-xs text-muted-foreground">({{ rule.comment }})</span>
          </div>
        </div>
      </Card>

      <!-- 磁盘信息 -->
      <Card class="mb-6" v-if="detail.disks?.length">
        <div class="p-4 border-b">
          <h3 class="font-semibold flex items-center gap-2">
            <HardDrive class="h-4 w-4" /> {{ t('vm.disks') }}
          </h3>
        </div>
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b text-muted-foreground">
              <th class="text-left p-3 font-medium">{{ t('vm.thDevice') }}</th>
              <th class="text-left p-3 font-medium">{{ t('vm.thPath') }}</th>
              <th class="text-left p-3 font-medium">{{ t('vm.thFormat') }}</th>
              <th class="text-left p-3 font-medium">{{ t('vm.thSize') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="disk in detail.disks" :key="disk.device" class="border-b last:border-0">
              <td class="p-3">{{ disk.device }}</td>
              <td class="p-3 font-mono text-xs selectable">{{ disk.path }}</td>
              <td class="p-3">{{ disk.format || '-' }}</td>
              <td class="p-3">{{ disk.sizeGB ? disk.sizeGB.toFixed(1) + ' GB' : '-' }}</td>
            </tr>
          </tbody>
        </table>
      </Card>

      <!-- 快照管理 -->
      <Card>
        <div class="p-4 border-b flex items-center justify-between">
          <h3 class="font-semibold flex items-center gap-2">
            <Camera class="h-4 w-4" /> {{ t('vm.snapshots') }}
          </h3>
          <Button variant="outline" size="sm" @click="loadSnapshots" :loading="loadingSnaps">
            <RotateCw class="h-3.5 w-3.5" />
          </Button>
        </div>

        <!-- 创建快照 -->
        <div class="p-4 border-b flex gap-2">
          <Input
            v-model="newSnapName"
            :placeholder="t('vm.snapshotName')"
            class="flex-1"
            @keyup.enter="createSnapshot"
          />
          <Button :loading="creatingSnap" @click="createSnapshot">
            <Plus class="h-4 w-4" /> {{ t('vm.createSnapshot') }}
          </Button>
        </div>

        <!-- 快照列表 -->
        <div v-if="snapshots.length > 0">
          <div
            v-for="snap in snapshots"
            :key="snap.name"
            class="flex items-center justify-between p-4 border-b last:border-0 hover:bg-muted/50"
          >
            <div>
              <p class="font-medium text-sm">{{ snap.name }}</p>
              <p class="text-xs text-muted-foreground">{{ snap.createdAt }} - {{ snap.state }}</p>
            </div>
            <div class="flex gap-1">
              <Button
                variant="ghost" size="icon" :title="t('vm.revertSnapshot')"
                :loading="snapActionLoading[snap.name]"
                @click="revertSnapshot(snap.name)"
              >
                <RotateCcw class="h-4 w-4" />
              </Button>
              <Button
                variant="ghost" size="icon" :title="t('vm.deleteSnapshot')"
                :loading="snapActionLoading[`del-${snap.name}`]"
                @click="deleteSnapshot(snap.name)"
              >
                <Trash2 class="h-4 w-4 text-destructive" />
              </Button>
            </div>
          </div>
        </div>
        <div v-else class="p-8 text-center text-sm text-muted-foreground">
          {{ t('vm.noSnapshots') }}
        </div>
      </Card>
    </template>

    <!-- 对话框 -->
    <VMEditDialog
      :open="showEdit" :hostId="hostId" :vmName="vmName" :detail="detail"
      @update:open="showEdit = $event" @saved="onEditSaved"
    />
    <VMHardwareDialog
      :open="showHardware" :hostId="hostId" :vmName="vmName" :detail="detail"
      @update:open="showHardware = $event" @saved="load"
    />
    <VMXMLDialog
      :open="showXML" :hostId="hostId" :vmName="vmName"
      @update:open="showXML = $event" @saved="load"
    />
    <VMCloneDialog
      :open="showClone" :hostId="hostId" :vmName="vmName"
      @update:open="showClone = $event" @saved="load"
    />
    <VMMigrateDialog
      :open="showMigrate" :hostId="hostId" :vmName="vmName" :vmState="detail?.state"
      @update:open="showMigrate = $event" @migrated="router.push(`/host/${hostId}`)"
    />
  </div>
</template>
