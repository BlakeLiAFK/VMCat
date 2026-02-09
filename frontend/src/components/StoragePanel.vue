<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@/composables/useToast'
import { PoolList, VolList, CreateVolume, DeleteVolume, PoolStart, PoolStop, PoolAutostart } from '@/api/backend'
import { useConfirm } from '@/composables/useConfirm'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import {
  Database, HardDrive, Plus, RotateCw, ChevronRight, ChevronDown, Loader2, Trash2,
  Play, Square, ToggleLeft, ToggleRight,
} from 'lucide-vue-next'

const { t } = useI18n()
const props = defineProps<{ hostId: string; visible: boolean }>()
const toast = useToast()
const { request: confirmRequest } = useConfirm()
const deletingVol = ref('')
const poolActionLoading = ref('')

const pools = ref<any[]>([])
const loadingPools = ref(false)
const expandedPool = ref('')
const volumes = ref<any[]>([])
const loadingVols = ref(false)

// 创建卷表单
const showCreateVol = ref(false)
const newVolPool = ref('')
const newVolName = ref('')
const newVolSize = ref(10)
const newVolFormat = ref('qcow2')
const creatingVol = ref(false)

async function loadPools() {
  loadingPools.value = true
  try {
    const list = await PoolList(props.hostId)
    pools.value = list || []
  } catch (e: any) {
    toast.error(t('storage.loadPoolFailed') + ': ' + e.toString())
  } finally {
    loadingPools.value = false
  }
}

async function togglePool(poolName: string) {
  if (expandedPool.value === poolName) {
    expandedPool.value = ''
    volumes.value = []
    return
  }
  expandedPool.value = poolName
  loadingVols.value = true
  try {
    const list = await VolList(props.hostId, poolName)
    volumes.value = list || []
  } catch (e: any) {
    toast.error(t('storage.loadVolFailed') + ': ' + e.toString())
    volumes.value = []
  } finally {
    loadingVols.value = false
  }
}

async function createVol() {
  if (!newVolPool.value || !newVolName.value.trim()) {
    toast.warning(t('storage.selectPoolAndName'))
    return
  }
  creatingVol.value = true
  try {
    const path = await CreateVolume(props.hostId, newVolPool.value, newVolName.value.trim(), Number(newVolSize.value), newVolFormat.value)
    toast.success(t('storage.volCreated', { path }))
    newVolName.value = ''
    showCreateVol.value = false
    // 刷新当前展开的存储池
    if (expandedPool.value === newVolPool.value) {
      await togglePool(newVolPool.value)
      expandedPool.value = newVolPool.value
    }
  } catch (e: any) {
    toast.error(t('storage.volCreateFailed') + ': ' + e.toString())
  } finally {
    creatingVol.value = false
  }
}

async function deleteVol(volName: string) {
  const ok = await confirmRequest(t('storage.deleteVolume'), t('storage.deleteVolumeConfirm', { name: volName }), { variant: 'destructive', confirmText: t('common.delete') })
  if (!ok) return
  deletingVol.value = volName
  try {
    await DeleteVolume(props.hostId, expandedPool.value, volName)
    toast.success(t('storage.volDeleted', { name: volName }))
    // 刷新当前展开的存储池
    if (expandedPool.value) {
      const poolName = expandedPool.value
      expandedPool.value = ''
      await togglePool(poolName)
    }
  } catch (e: any) {
    toast.error(t('storage.volDeleteFailed') + ': ' + e.toString())
  } finally {
    deletingVol.value = ''
  }
}

async function startPool(name: string) {
  poolActionLoading.value = `start-${name}`
  try {
    await PoolStart(props.hostId, name)
    toast.success(t('storage.poolStarted', { name }))
    await loadPools()
  } catch (e: any) { toast.error(t('common.startFailed') + ': ' + e.toString()) }
  finally { poolActionLoading.value = '' }
}

async function stopPool(name: string) {
  const ok = await confirmRequest(t('storage.stopPoolTitle'), t('storage.stopPoolConfirm', { name }))
  if (!ok) return
  poolActionLoading.value = `stop-${name}`
  try {
    await PoolStop(props.hostId, name)
    toast.success(t('storage.poolStopped', { name }))
    await loadPools()
  } catch (e: any) { toast.error(t('common.stopFailed') + ': ' + e.toString()) }
  finally { poolActionLoading.value = '' }
}

async function togglePoolAutostart(name: string, current: string) {
  const enabled = current !== 'yes'
  poolActionLoading.value = `auto-${name}`
  try {
    await PoolAutostart(props.hostId, name, enabled)
    toast.success(t('storage.poolAutostart', { name, state: enabled ? t('storage.autostartOn') : t('storage.autostartOff') }))
    await loadPools()
  } catch (e: any) { toast.error(t('common.setFailed') + ': ' + e.toString()) }
  finally { poolActionLoading.value = '' }
}

watch(() => props.visible, (v) => {
  if (v && pools.value.length === 0) loadPools()
})
</script>

<template>
  <div>
    <!-- 标题栏 -->
    <div class="flex items-center justify-between mb-4">
      <h2 class="text-lg font-semibold flex items-center gap-2">
        <Database class="h-5 w-5" /> {{ t('storage.title') }}
      </h2>
      <div class="flex items-center gap-2">
        <Button variant="outline" size="sm" @click="loadPools" :loading="loadingPools">
          <RotateCw class="h-3.5 w-3.5" /> {{ t('common.refresh') }}
        </Button>
        <Button size="sm" @click="showCreateVol = !showCreateVol">
          <Plus class="h-3.5 w-3.5" /> {{ t('storage.createVolume') }}
        </Button>
      </div>
    </div>

    <!-- 创建卷表单 -->
    <Card v-if="showCreateVol" class="mb-4 p-4">
      <h3 class="text-sm font-medium mb-3">{{ t('storage.createVolumeTitle') }}</h3>
      <div class="grid grid-cols-4 gap-3">
        <div>
          <label class="text-xs text-muted-foreground mb-1 block">{{ t('storage.pool') }}</label>
          <select
            v-model="newVolPool"
            class="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm focus:outline-none focus:ring-1 focus:ring-ring"
          >
            <option value="">{{ t('storage.selectPool') }}</option>
            <option v-for="p in pools" :key="p.name" :value="p.name">{{ p.name }}</option>
          </select>
        </div>
        <div>
          <label class="text-xs text-muted-foreground mb-1 block">{{ t('storage.volName') }}</label>
          <Input v-model="newVolName" placeholder="data.qcow2" />
        </div>
        <div>
          <label class="text-xs text-muted-foreground mb-1 block">{{ t('storage.sizeGB') }}</label>
          <Input v-model="newVolSize" type="number" placeholder="10" />
        </div>
        <div>
          <label class="text-xs text-muted-foreground mb-1 block">{{ t('storage.format') }}</label>
          <div class="flex items-center gap-2">
            <select
              v-model="newVolFormat"
              class="flex h-9 flex-1 rounded-md border border-input bg-transparent px-3 py-1 text-sm focus:outline-none focus:ring-1 focus:ring-ring"
            >
              <option value="qcow2">qcow2</option>
              <option value="raw">raw</option>
            </select>
            <Button size="sm" :loading="creatingVol" @click="createVol">{{ t('common.create') }}</Button>
          </div>
        </div>
      </div>
    </Card>

    <!-- 加载中 -->
    <div v-if="loadingPools && pools.length === 0" class="text-center py-8">
      <Loader2 class="h-6 w-6 animate-spin mx-auto text-muted-foreground" />
    </div>

    <!-- 存储池列表 -->
    <div v-else-if="pools.length > 0" class="space-y-2">
      <Card v-for="pool in pools" :key="pool.name">
        <!-- 存储池行 -->
        <button
          class="w-full flex items-center gap-3 p-4 text-left hover:bg-muted/50 transition-colors"
          @click="togglePool(pool.name)"
        >
          <ChevronDown v-if="expandedPool === pool.name" class="h-4 w-4 flex-shrink-0" />
          <ChevronRight v-else class="h-4 w-4 flex-shrink-0" />
          <Database class="h-4 w-4 flex-shrink-0 text-muted-foreground" />
          <div class="flex-1">
            <p class="text-sm font-medium">{{ pool.name }}</p>
            <p class="text-xs text-muted-foreground">
              {{ pool.capacity }} | {{ t('storage.used') }} {{ pool.allocation }} | {{ t('storage.available') }} {{ pool.available }}
            </p>
          </div>
          <div class="flex items-center gap-1" @click.stop>
            <Button
              v-if="pool.state !== 'running'"
              variant="ghost" size="icon" :title="t('storage.startPool')"
              :loading="poolActionLoading === `start-${pool.name}`"
              @click="startPool(pool.name)"
            >
              <Play class="h-3.5 w-3.5 text-green-500" />
            </Button>
            <Button
              v-if="pool.state === 'running'"
              variant="ghost" size="icon" :title="t('storage.stopPool')"
              :loading="poolActionLoading === `stop-${pool.name}`"
              @click="stopPool(pool.name)"
            >
              <Square class="h-3.5 w-3.5 text-muted-foreground" />
            </Button>
            <Button
              variant="ghost" size="icon"
              :title="pool.autostart === 'yes' ? t('storage.disableAutostart') : t('storage.enableAutostart')"
              :loading="poolActionLoading === `auto-${pool.name}`"
              @click="togglePoolAutostart(pool.name, pool.autostart)"
            >
              <ToggleRight v-if="pool.autostart === 'yes'" class="h-3.5 w-3.5 text-blue-500" />
              <ToggleLeft v-else class="h-3.5 w-3.5 text-muted-foreground" />
            </Button>
          </div>
          <span
            class="px-2 py-0.5 rounded text-xs"
            :class="pool.state === 'running' ? 'bg-green-500/10 text-green-600' : 'bg-muted text-muted-foreground'"
          >
            {{ pool.state === 'running' ? t('storage.active') : pool.state }}
          </span>
        </button>

        <!-- 卷列表 -->
        <div v-if="expandedPool === pool.name" class="border-t">
          <div v-if="loadingVols" class="p-4 text-center">
            <Loader2 class="h-4 w-4 animate-spin mx-auto text-muted-foreground" />
          </div>
          <div v-else-if="volumes.length === 0" class="p-4 text-center text-sm text-muted-foreground">
            {{ t('storage.noPoolVolumes') }}
          </div>
          <table v-else class="w-full text-sm">
            <thead>
              <tr class="border-b text-muted-foreground">
                <th class="text-left p-3 font-medium pl-12">{{ t('storage.thName') }}</th>
                <th class="text-left p-3 font-medium">{{ t('storage.thPath') }}</th>
                <th class="text-left p-3 font-medium">{{ t('storage.thType') }}</th>
                <th class="text-left p-3 font-medium">{{ t('storage.thCapacity') }}</th>
                <th class="text-left p-3 font-medium">{{ t('storage.thAllocated') }}</th>
                <th class="text-right p-3 font-medium pr-4">{{ t('storage.thActions') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="vol in volumes" :key="vol.name" class="border-b last:border-0 hover:bg-muted/30">
                <td class="p-3 pl-12 font-medium">
                  <div class="flex items-center gap-2">
                    <HardDrive class="h-3.5 w-3.5 text-muted-foreground" />
                    {{ vol.name }}
                  </div>
                </td>
                <td class="p-3 font-mono text-xs text-muted-foreground">{{ vol.path }}</td>
                <td class="p-3">{{ vol.type || '-' }}</td>
                <td class="p-3">{{ vol.capacity || '-' }}</td>
                <td class="p-3">{{ vol.allocation || '-' }}</td>
                <td class="p-3 text-right pr-4">
                  <Button variant="ghost" size="icon"
                    :loading="deletingVol === vol.name"
                    @click="deleteVol(vol.name)" :title="t('storage.deleteVol')">
                    <Trash2 class="h-3.5 w-3.5 text-destructive" />
                  </Button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </Card>
    </div>

    <!-- 空状态 -->
    <div v-else class="text-center py-8 text-muted-foreground">
      <Database class="h-8 w-8 mx-auto mb-2 opacity-50" />
      <p class="text-sm">{{ t('storage.noPools') }}</p>
    </div>
  </div>
</template>
