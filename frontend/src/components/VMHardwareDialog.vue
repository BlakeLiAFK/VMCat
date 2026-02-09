<script setup lang="ts">
import { ref, watch } from 'vue'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'
import {
  VMAttachDisk, VMDetachDisk, VMAttachInterface, VMDetachInterface,
  VMChangeMedia, VMEjectMedia, VMResizeDisk, VMSetGraphics,
  BridgeList, NetworkList, ISOList,
} from '../../wailsjs/go/main/App'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import { X, Plus, Trash2, Disc, HardDrive, Network, Monitor } from 'lucide-vue-next'

const props = defineProps<{
  open: boolean
  hostId: string
  vmName: string
  detail: any
}>()
const emit = defineEmits<{ 'update:open': [v: boolean]; saved: [] }>()
const toast = useToast()
const { request: confirmRequest } = useConfirm()

const tab = ref<'disk' | 'nic' | 'cdrom' | 'vnc'>('disk')
const saving = ref('')

// 磁盘
const diskSource = ref('')
const diskTarget = ref('vdb')
const diskDriver = ref('qcow2')

// 网卡
const nicSource = ref('')
const nicType = ref('bridge')
const nicModel = ref('virtio')
const bridges = ref<string[]>([])
const networks = ref<any[]>([])

// 光驱
const isos = ref<any[]>([])
const cdromTarget = ref('hda')
const cdromSource = ref('')

// VNC
const vncEnabled = ref(false)

// 磁盘扩容
const resizePath = ref('')
const resizeGB = ref(0)

watch(() => props.open, async (v) => {
  if (!v) return
  // 获取 VNC 状态
  vncEnabled.value = props.detail?.vncPort > 0
  // 设置光驱默认 target
  const disks = props.detail?.disks || []
  const cdDisk = disks.find((d: any) => d.device?.startsWith('hd') || d.device?.startsWith('sd'))
  if (cdDisk) cdromTarget.value = cdDisk.device
  // 加载选项
  try {
    const [b, n, i] = await Promise.all([
      BridgeList(props.hostId).catch(() => []),
      NetworkList(props.hostId).catch(() => []),
      ISOList(props.hostId).catch(() => []),
    ])
    bridges.value = b || []
    networks.value = n || []
    isos.value = i || []
    if (bridges.value.length > 0) nicSource.value = bridges.value[0]
  } catch { /* 静默 */ }
})

async function attachDisk() {
  if (!diskSource.value || !diskTarget.value) { toast.warning('请填写磁盘路径和目标设备'); return }
  saving.value = 'disk-add'
  try {
    await VMAttachDisk(props.hostId, props.vmName, {
      source: diskSource.value, target: diskTarget.value, driver: diskDriver.value, cache: '', devType: '',
    } as any)
    toast.success('磁盘已添加')
    diskSource.value = ''
    emit('saved')
  } catch (e: any) { toast.error('添加磁盘失败: ' + e.toString()) }
  finally { saving.value = '' }
}

async function detachDisk(target: string) {
  const ok = await confirmRequest('移除磁盘', `确认移除磁盘 ${target}?`, { variant: 'destructive', confirmText: '移除' })
  if (!ok) return
  saving.value = `disk-del-${target}`
  try {
    await VMDetachDisk(props.hostId, props.vmName, target)
    toast.success(`磁盘 ${target} 已移除`)
    emit('saved')
  } catch (e: any) { toast.error('移除失败: ' + e.toString()) }
  finally { saving.value = '' }
}

async function resizeDisk() {
  if (!resizePath.value || resizeGB.value <= 0) { toast.warning('请填写磁盘路径和目标大小'); return }
  saving.value = 'resize'
  try {
    await VMResizeDisk(props.hostId, resizePath.value, Number(resizeGB.value))
    toast.success('磁盘已扩容')
    resizePath.value = ''
    resizeGB.value = 0
  } catch (e: any) { toast.error('扩容失败: ' + e.toString()) }
  finally { saving.value = '' }
}

async function attachNIC() {
  if (!nicSource.value) { toast.warning('请选择网络'); return }
  saving.value = 'nic-add'
  try {
    await VMAttachInterface(props.hostId, props.vmName, {
      type: nicType.value, source: nicSource.value, model: nicModel.value,
    } as any)
    toast.success('网卡已添加')
    emit('saved')
  } catch (e: any) { toast.error('添加网卡失败: ' + e.toString()) }
  finally { saving.value = '' }
}

async function detachNIC(mac: string) {
  const ok = await confirmRequest('移除网卡', `确认移除网卡 ${mac}?`, { variant: 'destructive', confirmText: '移除' })
  if (!ok) return
  saving.value = `nic-del-${mac}`
  try {
    await VMDetachInterface(props.hostId, props.vmName, mac)
    toast.success('网卡已移除')
    emit('saved')
  } catch (e: any) { toast.error('移除失败: ' + e.toString()) }
  finally { saving.value = '' }
}

async function mountISO() {
  if (!cdromSource.value) { toast.warning('请选择 ISO'); return }
  saving.value = 'cdrom'
  try {
    await VMChangeMedia(props.hostId, props.vmName, cdromTarget.value, cdromSource.value)
    toast.success('ISO 已挂载')
  } catch (e: any) { toast.error('挂载失败: ' + e.toString()) }
  finally { saving.value = '' }
}

async function ejectISO() {
  saving.value = 'eject'
  try {
    await VMEjectMedia(props.hostId, props.vmName, cdromTarget.value)
    toast.success('光驱已弹出')
  } catch (e: any) { toast.error('弹出失败: ' + e.toString()) }
  finally { saving.value = '' }
}

async function toggleVNC() {
  saving.value = 'vnc'
  try {
    await VMSetGraphics(props.hostId, props.vmName, !vncEnabled.value)
    vncEnabled.value = !vncEnabled.value
    toast.success(vncEnabled.value ? 'VNC 已启用 (重启生效)' : 'VNC 已关闭 (重启生效)')
    emit('saved')
  } catch (e: any) { toast.error('设置失败: ' + e.toString()) }
  finally { saving.value = '' }
}

function close() { emit('update:open', false) }
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center">
      <div class="absolute inset-0 bg-black/50" @click="close" />
      <div class="relative bg-card border rounded-xl shadow-2xl w-[560px] max-h-[85vh] overflow-y-auto">
        <div class="flex items-center justify-between p-5 border-b">
          <h2 class="text-lg font-semibold">硬件管理</h2>
          <button @click="close" class="p-1 rounded hover:bg-accent"><X class="h-4 w-4" /></button>
        </div>

        <!-- Tab -->
        <div class="flex border-b px-5">
          <button v-for="t in (['disk', 'nic', 'cdrom', 'vnc'] as const)" :key="t"
            class="px-4 py-2.5 text-sm border-b-2 transition-colors -mb-px"
            :class="tab === t ? 'border-primary text-foreground font-medium' : 'border-transparent text-muted-foreground hover:text-foreground'"
            @click="tab = t"
          >
            <HardDrive v-if="t === 'disk'" class="h-3.5 w-3.5 inline mr-1" />
            <Network v-if="t === 'nic'" class="h-3.5 w-3.5 inline mr-1" />
            <Disc v-if="t === 'cdrom'" class="h-3.5 w-3.5 inline mr-1" />
            <Monitor v-if="t === 'vnc'" class="h-3.5 w-3.5 inline mr-1" />
            {{ { disk: '磁盘', nic: '网卡', cdrom: '光驱', vnc: 'VNC' }[t] }}
          </button>
        </div>

        <div class="p-5">
          <!-- 磁盘 -->
          <div v-if="tab === 'disk'" class="space-y-4">
            <div v-if="detail?.disks?.length" class="space-y-2">
              <p class="text-sm font-medium">当前磁盘</p>
              <div v-for="d in detail.disks" :key="d.device"
                class="flex items-center justify-between p-2 rounded border text-sm">
                <div>
                  <span class="font-mono">{{ d.device }}</span>
                  <span class="text-muted-foreground ml-2">{{ d.path }}</span>
                </div>
                <div class="flex items-center gap-1">
                  <Button variant="ghost" size="icon" :loading="saving === `disk-del-${d.device}`"
                    @click="detachDisk(d.device)" title="移除">
                    <Trash2 class="h-3.5 w-3.5 text-destructive" />
                  </Button>
                </div>
              </div>
            </div>
            <div class="border-t pt-4 space-y-3">
              <p class="text-sm font-medium flex items-center gap-1"><Plus class="h-3.5 w-3.5" /> 添加磁盘</p>
              <Input v-model="diskSource" placeholder="磁盘路径 /var/lib/libvirt/images/data.qcow2" />
              <div class="grid grid-cols-2 gap-2">
                <Input v-model="diskTarget" placeholder="目标设备 vdb" />
                <select v-model="diskDriver"
                  class="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm">
                  <option value="qcow2">qcow2</option>
                  <option value="raw">raw</option>
                </select>
              </div>
              <Button size="sm" :loading="saving === 'disk-add'" @click="attachDisk">
                <Plus class="h-3.5 w-3.5" /> 添加
              </Button>
            </div>
            <div class="border-t pt-4 space-y-3">
              <p class="text-sm font-medium">磁盘扩容 (需关机)</p>
              <Input v-model="resizePath" placeholder="磁盘路径" />
              <div class="flex gap-2">
                <Input v-model="resizeGB" type="number" placeholder="目标大小 GB" class="flex-1" />
                <Button size="sm" :loading="saving === 'resize'" @click="resizeDisk">扩容</Button>
              </div>
            </div>
          </div>

          <!-- 网卡 -->
          <div v-if="tab === 'nic'" class="space-y-4">
            <div v-if="detail?.nics?.length" class="space-y-2">
              <p class="text-sm font-medium">当前网卡</p>
              <div v-for="n in detail.nics" :key="n.mac"
                class="flex items-center justify-between p-2 rounded border text-sm">
                <div>
                  <span class="font-mono text-xs">{{ n.mac }}</span>
                  <span class="text-muted-foreground ml-2">{{ n.bridge || '-' }}</span>
                  <span class="text-muted-foreground ml-1">({{ n.model || '-' }})</span>
                </div>
                <Button variant="ghost" size="icon" :loading="saving === `nic-del-${n.mac}`"
                  @click="detachNIC(n.mac)" title="移除">
                  <Trash2 class="h-3.5 w-3.5 text-destructive" />
                </Button>
              </div>
            </div>
            <div class="border-t pt-4 space-y-3">
              <p class="text-sm font-medium flex items-center gap-1"><Plus class="h-3.5 w-3.5" /> 添加网卡</p>
              <div class="flex gap-2">
                <select v-model="nicType"
                  class="flex h-9 rounded-md border border-input bg-transparent px-3 py-1 text-sm w-28">
                  <option value="bridge">网桥</option>
                  <option value="network">虚拟网络</option>
                </select>
                <select v-model="nicSource"
                  class="flex h-9 flex-1 rounded-md border border-input bg-transparent px-3 py-1 text-sm">
                  <template v-if="nicType === 'bridge'">
                    <option v-for="b in bridges" :key="b" :value="b">{{ b }}</option>
                  </template>
                  <template v-else>
                    <option v-for="n in networks" :key="n.name" :value="n.name">{{ n.name }}</option>
                  </template>
                </select>
              </div>
              <Button size="sm" :loading="saving === 'nic-add'" @click="attachNIC">
                <Plus class="h-3.5 w-3.5" /> 添加
              </Button>
            </div>
          </div>

          <!-- 光驱 -->
          <div v-if="tab === 'cdrom'" class="space-y-4">
            <div class="space-y-3">
              <div>
                <label class="text-sm mb-1 block">光驱设备</label>
                <Input v-model="cdromTarget" placeholder="hda" />
              </div>
              <div>
                <label class="text-sm mb-1 block">ISO 镜像</label>
                <select v-model="cdromSource"
                  class="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm">
                  <option value="">选择 ISO...</option>
                  <option v-for="iso in isos" :key="iso.path" :value="iso.path">
                    {{ iso.name }} ({{ iso.size }})
                  </option>
                </select>
              </div>
              <div class="flex gap-2">
                <Button size="sm" :loading="saving === 'cdrom'" @click="mountISO">挂载</Button>
                <Button size="sm" variant="outline" :loading="saving === 'eject'" @click="ejectISO">弹出</Button>
              </div>
            </div>
          </div>

          <!-- VNC -->
          <div v-if="tab === 'vnc'" class="space-y-4">
            <div class="flex items-center justify-between p-4 rounded border">
              <div>
                <p class="text-sm font-medium">VNC 远程桌面</p>
                <p class="text-xs text-muted-foreground mt-1">
                  {{ vncEnabled ? '已启用 (端口: ' + (detail?.vncPort || 'auto') + ')' : '未启用' }}
                </p>
              </div>
              <Button size="sm" :variant="vncEnabled ? 'destructive' : 'default'"
                :loading="saving === 'vnc'" @click="toggleVNC">
                {{ vncEnabled ? '关闭' : '启用' }}
              </Button>
            </div>
            <p class="text-xs text-muted-foreground">修改后需重启虚拟机生效</p>
          </div>
        </div>

        <div class="flex justify-end p-5 border-t">
          <Button variant="outline" @click="close">关闭</Button>
        </div>
      </div>
    </div>
  </Teleport>
</template>
