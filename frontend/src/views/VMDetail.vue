<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useToast } from '@/composables/useToast'
import {
  VMGet, VMStart, VMShutdown, VMDestroy, VMReboot, VMSuspend, VMResume, VMDelete,
  SnapshotList, SnapshotCreate, SnapshotDelete, SnapshotRevert,
} from '../../wailsjs/go/main/App'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import Input from '@/components/ui/Input.vue'
import VMEditDialog from '@/components/VMEditDialog.vue'
import VMHardwareDialog from '@/components/VMHardwareDialog.vue'
import VMXMLDialog from '@/components/VMXMLDialog.vue'
import VMCloneDialog from '@/components/VMCloneDialog.vue'
import {
  ArrowLeft, Play, Square, RotateCw, Skull, Pause, PlayCircle,
  Cpu, MemoryStick, HardDrive, Network, Loader2,
  Camera, RotateCcw, Trash2, Plus, Terminal, Monitor as MonitorIcon, ScreenShare,
  Pencil, Copy, Code, Settings, Wrench,
} from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const toast = useToast()

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

const isWindows = computed(() => {
  const name = vmName.value.toLowerCase()
  return name.includes('win') || name.includes('windows')
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
  } catch (e: any) {
    console.error('load vm:', e)
    toast.error('加载 VM 信息失败')
  } finally {
    loading.value = false
  }
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
  if (!name) { toast.warning('请输入快照名称'); return }
  creatingSnap.value = true
  try {
    await SnapshotCreate(hostId.value, vmName.value, name)
    toast.success(`快照 ${name} 创建成功`)
    newSnapName.value = ''
    await loadSnapshots()
  } catch (e: any) { toast.error('创建快照失败: ' + e.toString()) }
  finally { creatingSnap.value = false }
}

async function revertSnapshot(snapName: string) {
  if (!confirm(`确认恢复到快照 "${snapName}"?`)) return
  snapActionLoading.value[snapName] = true
  try {
    await SnapshotRevert(hostId.value, vmName.value, snapName)
    toast.success(`已恢复到快照 ${snapName}`)
    await load()
  } catch (e: any) { toast.error('恢复快照失败: ' + e.toString()) }
  finally { snapActionLoading.value[snapName] = false }
}

async function deleteSnapshot(snapName: string) {
  if (!confirm(`确认删除快照 "${snapName}"?`)) return
  snapActionLoading.value[`del-${snapName}`] = true
  try {
    await SnapshotDelete(hostId.value, vmName.value, snapName)
    toast.success(`快照 ${snapName} 已删除`)
    await loadSnapshots()
  } catch (e: any) { toast.error('删除快照失败: ' + e.toString()) }
  finally { snapActionLoading.value[`del-${snapName}`] = false }
}

async function doAction(action: string) {
  actionLoading.value = action
  const labels: Record<string, string> = {
    start: '启动', shutdown: '关机', destroy: '强制关闭',
    reboot: '重启', suspend: '暂停', resume: '恢复',
  }
  try {
    switch (action) {
      case 'start': await VMStart(hostId.value, vmName.value); break
      case 'shutdown': await VMShutdown(hostId.value, vmName.value); break
      case 'destroy': await VMDestroy(hostId.value, vmName.value); break
      case 'reboot': await VMReboot(hostId.value, vmName.value); break
      case 'suspend': await VMSuspend(hostId.value, vmName.value); break
      case 'resume': await VMResume(hostId.value, vmName.value); break
    }
    toast.success(`${vmName.value} ${labels[action] || action} 成功`)
    setTimeout(load, 1500)
  } catch (e: any) { toast.error('操作失败: ' + e.toString()) }
  finally { actionLoading.value = '' }
}

async function deleteVM() {
  const removeStorage = confirm('是否同时删除磁盘文件?\n\n确定 = 删除磁盘\n取消 = 仅取消定义')
  if (!confirm(`确认删除虚拟机 "${vmName.value}"? 此操作不可撤销!`)) return
  try {
    await VMDelete(hostId.value, vmName.value, removeStorage)
    toast.success('虚拟机已删除')
    router.push(`/host/${hostId.value}`)
  } catch (e: any) { toast.error('删除失败: ' + e.toString()) }
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
    'running': '运行中', 'shut off': '已关机', 'paused': '已暂停',
    'idle': '空闲', 'crashed': '已崩溃',
  }
  return map[state] || state
}

function formatMem(mb: number) {
  return mb >= 1024 ? (mb / 1024).toFixed(1) + ' GB' : mb + ' MB'
}

onMounted(() => { load(); loadSnapshots() })
</script>

<template>
  <div class="p-6">
    <!-- 返回 -->
    <button
      class="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground mb-4 transition-colors"
      @click="router.push(`/host/${hostId}`)"
    >
      <ArrowLeft class="h-4 w-4" />
      返回虚拟机列表
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
            <Terminal class="h-4 w-4" /> 控制台
          </Button>
          <Button v-if="detail.state === 'shut off'" @click="doAction('start')" :loading="actionLoading === 'start'">
            <Play class="h-4 w-4" /> 启动
          </Button>
          <Button v-if="detail.state === 'running'" variant="outline" @click="doAction('shutdown')" :loading="actionLoading === 'shutdown'">
            <Square class="h-4 w-4" /> 关机
          </Button>
          <Button v-if="detail.state === 'running'" variant="outline" @click="doAction('reboot')" :loading="actionLoading === 'reboot'">
            <RotateCw class="h-4 w-4" /> 重启
          </Button>
          <Button v-if="detail.state === 'running'" variant="outline" @click="doAction('suspend')" :loading="actionLoading === 'suspend'">
            <Pause class="h-4 w-4" /> 暂停
          </Button>
          <Button v-if="detail.state === 'paused'" @click="doAction('resume')" :loading="actionLoading === 'resume'">
            <PlayCircle class="h-4 w-4" /> 恢复
          </Button>
          <Button v-if="detail.state === 'running'" variant="destructive" @click="doAction('destroy')" :loading="actionLoading === 'destroy'">
            <Skull class="h-4 w-4" /> 强制关闭
          </Button>
          <div class="w-px h-6 bg-border mx-1" />
          <Button variant="outline" size="sm" @click="showEdit = true" title="编辑">
            <Pencil class="h-4 w-4" /> 编辑
          </Button>
          <Button variant="outline" size="sm" @click="showHardware = true" title="硬件">
            <Wrench class="h-4 w-4" /> 硬件
          </Button>
          <Button variant="outline" size="sm" @click="showClone = true" title="克隆">
            <Copy class="h-4 w-4" /> 克隆
          </Button>
          <Button variant="outline" size="sm" @click="showXML = true" title="XML">
            <Code class="h-4 w-4" /> XML
          </Button>
          <Button variant="destructive" size="sm" @click="deleteVM" title="删除">
            <Trash2 class="h-4 w-4" /> 删除
          </Button>
        </div>
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
              <p class="text-sm text-muted-foreground">内存</p>
              <p class="text-lg font-semibold">{{ formatMem(detail.memoryMB) }}</p>
            </div>
          </div>
        </Card>
        <Card class="p-4">
          <div class="flex items-center gap-3">
            <HardDrive class="h-5 w-5 text-muted-foreground" />
            <div>
              <p class="text-sm text-muted-foreground">磁盘</p>
              <p class="text-lg font-semibold">{{ detail.disks?.length || 0 }} 块</p>
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

      <!-- 网络信息 -->
      <Card class="mb-6" v-if="detail.nics?.length">
        <div class="p-4 border-b">
          <h3 class="font-semibold flex items-center gap-2">
            <Network class="h-4 w-4" /> 网络接口
          </h3>
        </div>
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b text-muted-foreground">
              <th class="text-left p-3 font-medium">MAC</th>
              <th class="text-left p-3 font-medium">桥接</th>
              <th class="text-left p-3 font-medium">IP</th>
              <th class="text-left p-3 font-medium">型号</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="nic in detail.nics" :key="nic.mac" class="border-b last:border-0">
              <td class="p-3 font-mono text-xs selectable">{{ nic.mac }}</td>
              <td class="p-3">{{ nic.bridge || '-' }}</td>
              <td class="p-3 font-mono selectable">{{ nic.ip || '-' }}</td>
              <td class="p-3">{{ nic.model || '-' }}</td>
            </tr>
          </tbody>
        </table>
      </Card>

      <!-- 磁盘信息 -->
      <Card class="mb-6" v-if="detail.disks?.length">
        <div class="p-4 border-b">
          <h3 class="font-semibold flex items-center gap-2">
            <HardDrive class="h-4 w-4" /> 磁盘
          </h3>
        </div>
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b text-muted-foreground">
              <th class="text-left p-3 font-medium">设备</th>
              <th class="text-left p-3 font-medium">路径</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="disk in detail.disks" :key="disk.device" class="border-b last:border-0">
              <td class="p-3">{{ disk.device }}</td>
              <td class="p-3 font-mono text-xs selectable">{{ disk.path }}</td>
            </tr>
          </tbody>
        </table>
      </Card>

      <!-- 快照管理 -->
      <Card>
        <div class="p-4 border-b flex items-center justify-between">
          <h3 class="font-semibold flex items-center gap-2">
            <Camera class="h-4 w-4" /> 快照
          </h3>
          <Button variant="outline" size="sm" @click="loadSnapshots" :loading="loadingSnaps">
            <RotateCw class="h-3.5 w-3.5" />
          </Button>
        </div>

        <!-- 创建快照 -->
        <div class="p-4 border-b flex gap-2">
          <Input
            v-model="newSnapName"
            placeholder="快照名称"
            class="flex-1"
            @keyup.enter="createSnapshot"
          />
          <Button :loading="creatingSnap" @click="createSnapshot">
            <Plus class="h-4 w-4" /> 创建
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
                variant="ghost" size="icon" title="恢复"
                :loading="snapActionLoading[snap.name]"
                @click="revertSnapshot(snap.name)"
              >
                <RotateCcw class="h-4 w-4" />
              </Button>
              <Button
                variant="ghost" size="icon" title="删除"
                :loading="snapActionLoading[`del-${snap.name}`]"
                @click="deleteSnapshot(snap.name)"
              >
                <Trash2 class="h-4 w-4 text-destructive" />
              </Button>
            </div>
          </div>
        </div>
        <div v-else class="p-8 text-center text-sm text-muted-foreground">
          暂无快照
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
  </div>
</template>
