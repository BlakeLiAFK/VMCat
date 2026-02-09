<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useToast } from '@/composables/useToast'
import {
  FlavorList, ImageList, VMCreateFromTemplate, BridgeList, NetworkList,
} from '@/api/backend'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import { X, Loader2, Check, Server, Rocket } from 'lucide-vue-next'

const { t } = useI18n()
const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ 'update:open': [v: boolean]; deployed: [] }>()
const store = useAppStore()
const toast = useToast()

const prefix = ref('')
const flavors = ref<any[]>([])
const images = ref<any[]>([])
const bridges = ref<string[]>([])
const networks = ref<any[]>([])
const selectedFlavor = ref('')
const selectedImage = ref('')
const netType = ref('bridge')
const netName = ref('')
const refHostId = ref('')
const targetHostIds = ref<Set<string>>(new Set())
const loading = ref(false)
const deploying = ref(false)

interface DeployTask {
  hostId: string
  hostName: string
  status: 'pending' | 'running' | 'success' | 'error'
  error?: string
}
const tasks = ref<DeployTask[]>([])

const connectedHosts = computed(() => store.hosts.filter(h => store.isConnected(h.id)))

async function loadOptions() {
  loading.value = true
  try {
    flavors.value = (await FlavorList()) || []
    if (flavors.value.length > 0 && !selectedFlavor.value) {
      selectedFlavor.value = flavors.value[0].id
    }
  } catch { /* */ }
  finally { loading.value = false }
}

async function loadHostResources() {
  if (!refHostId.value) return
  try {
    const [img, br, net] = await Promise.all([
      ImageList(refHostId.value).catch(() => []),
      BridgeList(refHostId.value).catch(() => []),
      NetworkList(refHostId.value).catch(() => []),
    ])
    images.value = img || []
    bridges.value = br || []
    networks.value = net || []
    if (bridges.value.length > 0 && !netName.value) {
      netName.value = bridges.value[0]
      netType.value = 'bridge'
    }
    if (images.value.length > 0 && !selectedImage.value) {
      selectedImage.value = images.value[0].id
    }
  } catch { /* */ }
}

watch(() => props.open, (v) => {
  if (v) {
    loadOptions()
    tasks.value = []
    deploying.value = false
    prefix.value = ''
    selectedFlavor.value = ''
    selectedImage.value = ''
    targetHostIds.value.clear()
    if (connectedHosts.value.length > 0) {
      refHostId.value = connectedHosts.value[0].id
      loadHostResources()
    }
  }
})

watch(refHostId, () => loadHostResources())

function toggleHost(id: string) {
  if (targetHostIds.value.has(id)) {
    targetHostIds.value.delete(id)
  } else {
    targetHostIds.value.add(id)
  }
}

function close() { emit('update:open', false) }

async function deploy() {
  if (!prefix.value.trim()) { toast.warning(t('batch.enterPrefix')); return }
  if (!selectedFlavor.value) { toast.warning(t('batch.selectFlavorFirst')); return }
  if (!selectedImage.value) { toast.warning(t('batch.selectImageFirst')); return }
  if (targetHostIds.value.size === 0) { toast.warning(t('batch.selectTargetFirst')); return }

  deploying.value = true
  const hosts = connectedHosts.value.filter(h => targetHostIds.value.has(h.id))
  tasks.value = hosts.map(h => ({ hostId: h.id, hostName: h.name, status: 'pending' as const }))

  await Promise.allSettled(tasks.value.map(async (task, i) => {
    tasks.value[i].status = 'running'
    const vmName = `${prefix.value.trim()}-${task.hostName}`
    try {
      await VMCreateFromTemplate(task.hostId, vmName, selectedFlavor.value, selectedImage.value, netType.value, netName.value, '', '')
      tasks.value[i].status = 'success'
    } catch (e: any) {
      tasks.value[i].status = 'error'
      tasks.value[i].error = e.toString()
    }
  }))

  const ok = tasks.value.filter(t => t.status === 'success').length
  const fail = tasks.value.filter(t => t.status === 'error').length
  if (fail === 0) {
    toast.success(t('batch.allSuccess', { count: ok }))
  } else {
    toast.warning(t('batch.partialFail', { ok, fail }))
  }
  emit('deployed')
  deploying.value = false
}
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center">
      <div class="absolute inset-0 bg-black/50" @click="close" />
      <div class="relative bg-card border rounded-xl shadow-2xl w-[600px] max-h-[85vh] overflow-y-auto">
        <div class="flex items-center justify-between p-5 border-b sticky top-0 bg-card z-10">
          <h2 class="text-lg font-semibold flex items-center gap-2">
            <Rocket class="h-5 w-5" /> {{ t('batch.deploy') }}
          </h2>
          <button @click="close" class="p-1 rounded hover:bg-accent"><X class="h-4 w-4" /></button>
        </div>

        <div v-if="loading" class="p-12 text-center">
          <Loader2 class="h-6 w-6 animate-spin mx-auto text-muted-foreground" />
        </div>

        <div v-else class="p-5 space-y-5">
          <!-- VM 名称前缀 -->
          <div>
            <label class="text-sm font-medium mb-1 block">{{ t('batch.vmPrefix') }}</label>
            <Input v-model="prefix" placeholder="web-server" />
            <p class="text-xs text-muted-foreground mt-1">{{ t('batch.prefixHint', { prefix: prefix || 'prefix', hostName: '...' }) }}</p>
          </div>

          <!-- 参考宿主机 -->
          <div>
            <label class="text-sm font-medium mb-1 block">{{ t('batch.refHost') }}</label>
            <select v-model="refHostId" class="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring">
              <option v-for="h in connectedHosts" :key="h.id" :value="h.id">{{ h.name }}</option>
            </select>
          </div>

          <!-- 硬件规格 -->
          <div>
            <label class="text-sm font-medium mb-1 block">{{ t('batch.flavorSelect') }}</label>
            <div v-if="flavors.length === 0" class="text-xs text-muted-foreground">{{ t('batch.noFlavor') }}</div>
            <div v-else class="grid grid-cols-2 gap-2">
              <button
                v-for="f in flavors" :key="f.id"
                class="p-2 border rounded-md text-left text-sm transition-colors"
                :class="selectedFlavor === f.id ? 'border-primary bg-primary/5' : 'hover:bg-accent'"
                @click="selectedFlavor = f.id"
              >
                <p class="font-medium">{{ f.name }}</p>
                <p class="text-xs text-muted-foreground">{{ f.cpus }} vCPU / {{ f.memoryMB >= 1024 ? (f.memoryMB / 1024).toFixed(0) + 'G' : f.memoryMB + 'M' }} / {{ f.diskGB }}G</p>
              </button>
            </div>
          </div>

          <!-- 镜像 -->
          <div>
            <label class="text-sm font-medium mb-1 block">{{ t('batch.imageSelect') }}</label>
            <div v-if="images.length === 0" class="text-xs text-muted-foreground">{{ t('batch.noImage') }}</div>
            <div v-else class="grid grid-cols-2 gap-2">
              <button
                v-for="img in images" :key="img.id"
                class="p-2 border rounded-md text-left text-sm transition-colors"
                :class="selectedImage === img.id ? 'border-primary bg-primary/5' : 'hover:bg-accent'"
                @click="selectedImage = img.id"
              >
                <p class="font-medium">{{ img.name }}</p>
                <p class="text-xs text-muted-foreground">{{ img.osVariant }}</p>
              </button>
            </div>
          </div>

          <!-- 网络 -->
          <div>
            <label class="text-sm font-medium mb-1 block">{{ t('batch.network') }}</label>
            <div class="flex gap-2">
              <select v-model="netType" class="h-9 rounded-md border border-input bg-transparent px-2 text-sm">
                <option value="bridge">{{ t('batch.bridgeType') }}</option>
                <option value="network">{{ t('batch.virtualNetwork') }}</option>
              </select>
              <select v-model="netName" class="flex-1 h-9 rounded-md border border-input bg-transparent px-2 text-sm">
                <template v-if="netType === 'bridge'">
                  <option v-for="b in bridges" :key="b" :value="b">{{ b }}</option>
                </template>
                <template v-else>
                  <option v-for="n in networks" :key="n.name" :value="n.name">{{ n.name }}</option>
                </template>
              </select>
            </div>
          </div>

          <!-- 目标宿主机 -->
          <div>
            <label class="text-sm font-medium mb-1 block">{{ t('batch.targetHosts') }}</label>
            <div class="space-y-1 max-h-40 overflow-auto border rounded-md p-2">
              <label
                v-for="h in connectedHosts" :key="h.id"
                class="flex items-center gap-2 px-2 py-1.5 rounded text-sm cursor-pointer hover:bg-accent transition-colors"
              >
                <input type="checkbox" :checked="targetHostIds.has(h.id)" @change="toggleHost(h.id)" class="rounded" />
                <Server class="h-3.5 w-3.5 text-muted-foreground" />
                <span>{{ h.name }}</span>
                <span class="text-xs text-muted-foreground ml-auto">{{ h.host }}</span>
              </label>
            </div>
            <p class="text-xs text-muted-foreground mt-1">{{ t('batch.selectedCount', { count: targetHostIds.size }) }}</p>
          </div>

          <!-- 部署进度 -->
          <div v-if="tasks.length > 0" class="border rounded-md p-3 space-y-1">
            <p class="text-sm font-medium mb-2">{{ t('batch.deployProgress') }}</p>
            <div v-for="task in tasks" :key="task.hostId" class="flex items-center gap-2 text-sm">
              <Loader2 v-if="task.status === 'running'" class="h-3.5 w-3.5 animate-spin text-primary flex-shrink-0" />
              <Check v-else-if="task.status === 'success'" class="h-3.5 w-3.5 text-green-500 flex-shrink-0" />
              <X v-else-if="task.status === 'error'" class="h-3.5 w-3.5 text-red-500 flex-shrink-0" />
              <div v-else class="h-3.5 w-3.5 rounded-full border-2 border-muted-foreground/30 flex-shrink-0" />
              <span :class="task.status === 'error' ? 'text-red-500' : ''">{{ task.hostName }}</span>
              <span v-if="task.error" class="text-xs text-red-500 truncate ml-auto max-w-[200px]">{{ task.error }}</span>
            </div>
          </div>
        </div>

        <div class="flex justify-end gap-2 p-5 border-t sticky bottom-0 bg-card">
          <Button variant="outline" @click="close">{{ t('common.cancel') }}</Button>
          <Button :loading="deploying" @click="deploy" :disabled="deploying">
            <Rocket class="h-4 w-4" /> {{ t('batch.deployBtn') }}
          </Button>
        </div>
      </div>
    </div>
  </Teleport>
</template>
