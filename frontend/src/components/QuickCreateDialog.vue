<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useToast } from '@/composables/useToast'
import {
  FlavorList, ImageList, NetworkList, BridgeList,
  VMCreateFromTemplate,
} from '../../wailsjs/go/main/App'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import {
  X, Cpu, MemoryStick, HardDrive, ChevronRight, Loader2, Zap, Check,
} from 'lucide-vue-next'

const props = defineProps<{ hostId: string; show: boolean }>()
const emit = defineEmits<{ (e: 'close'): void; (e: 'created'): void }>()
const toast = useToast()

// 步骤
const step = ref(1)
const creating = ref(false)

// 数据
const flavors = ref<any[]>([])
const images = ref<any[]>([])
const networks = ref<any[]>([])
const bridges = ref<string[]>([])
const loadingData = ref(false)

// 选择
const selectedFlavor = ref<any>(null)
const selectedImage = ref<any>(null)
const vmName = ref('')
const netType = ref('network')
const netName = ref('')

const canCreate = computed(() => {
  return selectedFlavor.value && selectedImage.value && vmName.value.trim()
})

async function loadData() {
  loadingData.value = true
  try {
    const [fl, il, nl, bl] = await Promise.all([
      FlavorList().catch(() => []),
      ImageList(props.hostId).catch(() => []),
      NetworkList(props.hostId).catch(() => []),
      BridgeList(props.hostId).catch(() => []),
    ])
    flavors.value = fl || []
    images.value = il || []
    networks.value = nl || []
    bridges.value = bl || []
    // 默认选中第一个网络
    if (networks.value.length > 0) {
      netName.value = networks.value[0].name
      netType.value = 'network'
    } else if (bridges.value.length > 0) {
      netName.value = bridges.value[0]
      netType.value = 'bridge'
    }
  } finally {
    loadingData.value = false
  }
}

function formatMem(mb: number) {
  return mb >= 1024 ? `${(mb / 1024).toFixed(mb % 1024 === 0 ? 0 : 1)} GB` : `${mb} MB`
}

function reset() {
  step.value = 1
  selectedFlavor.value = null
  selectedImage.value = null
  vmName.value = ''
  creating.value = false
}

function close() {
  reset()
  emit('close')
}

function selectFlavor(f: any) {
  selectedFlavor.value = f
  step.value = 2
}

function selectImage(img: any) {
  selectedImage.value = img
  step.value = 3
}

async function create() {
  if (!canCreate.value) return
  creating.value = true
  try {
    await VMCreateFromTemplate(
      props.hostId,
      vmName.value.trim(),
      selectedFlavor.value.id,
      selectedImage.value.id,
      netType.value,
      netName.value,
    )
    toast.success(`VM "${vmName.value}" 创建成功`)
    emit('created')
    close()
  } catch (e: any) {
    toast.error('创建失败: ' + e.toString())
  } finally {
    creating.value = false
  }
}

watch(() => props.show, (v) => {
  if (v) {
    reset()
    loadData()
  }
})
</script>

<template>
  <Teleport to="body">
    <div v-if="show" class="fixed inset-0 z-50 flex items-center justify-center">
      <div class="fixed inset-0 bg-black/50" @click="close" />
      <div class="relative bg-background rounded-xl shadow-2xl border w-[640px] max-h-[80vh] overflow-y-auto">
        <!-- 标题栏 -->
        <div class="flex items-center justify-between p-5 border-b">
          <div class="flex items-center gap-2">
            <Zap class="h-5 w-5 text-primary" />
            <h2 class="text-lg font-semibold">快速创建 VM</h2>
          </div>
          <button @click="close" class="text-muted-foreground hover:text-foreground">
            <X class="h-5 w-5" />
          </button>
        </div>

        <!-- 步骤指示 -->
        <div class="flex items-center gap-2 px-5 py-3 border-b text-sm">
          <span :class="step >= 1 ? 'text-primary font-medium' : 'text-muted-foreground'">1. 硬件规格</span>
          <ChevronRight class="h-3.5 w-3.5 text-muted-foreground" />
          <span :class="step >= 2 ? 'text-primary font-medium' : 'text-muted-foreground'">2. OS 模板</span>
          <ChevronRight class="h-3.5 w-3.5 text-muted-foreground" />
          <span :class="step >= 3 ? 'text-primary font-medium' : 'text-muted-foreground'">3. 配置</span>
        </div>

        <div class="p-5">
          <!-- 加载中 -->
          <div v-if="loadingData" class="text-center py-10">
            <Loader2 class="h-6 w-6 animate-spin mx-auto text-muted-foreground" />
          </div>

          <!-- Step 1: 选择 Flavor -->
          <div v-else-if="step === 1">
            <p class="text-sm text-muted-foreground mb-4">选择硬件规格</p>
            <div v-if="!flavors.length" class="text-center py-8 text-sm text-muted-foreground">
              暂无硬件规格，请先到设置 &gt; 模板页面添加
            </div>
            <div class="grid grid-cols-2 gap-3">
              <button
                v-for="f in flavors" :key="f.id"
                class="p-4 rounded-lg border text-left transition-all hover:border-primary/50 hover:bg-accent/50"
                :class="selectedFlavor?.id === f.id ? 'border-primary bg-primary/5 ring-1 ring-primary' : ''"
                @click="selectFlavor(f)"
              >
                <p class="font-medium mb-2">{{ f.name }}</p>
                <div class="flex items-center gap-4 text-xs text-muted-foreground">
                  <span class="flex items-center gap-1"><Cpu class="h-3 w-3" /> {{ f.cpus }} vCPU</span>
                  <span class="flex items-center gap-1"><MemoryStick class="h-3 w-3" /> {{ formatMem(f.memoryMB) }}</span>
                  <span class="flex items-center gap-1"><HardDrive class="h-3 w-3" /> {{ f.diskGB }} GB</span>
                </div>
              </button>
            </div>
          </div>

          <!-- Step 2: 选择 Image -->
          <div v-else-if="step === 2">
            <div class="flex items-center justify-between mb-4">
              <p class="text-sm text-muted-foreground">选择 OS 模板</p>
              <Button variant="ghost" size="sm" @click="step = 1">返回</Button>
            </div>
            <div v-if="!images.length" class="text-center py-8 text-sm text-muted-foreground">
              暂无 OS 模板，请先在当前宿主机「镜像」Tab 添加基础镜像
            </div>
            <div class="grid grid-cols-2 gap-3">
              <button
                v-for="img in images" :key="img.id"
                class="p-4 rounded-lg border text-left transition-all hover:border-primary/50 hover:bg-accent/50"
                :class="selectedImage?.id === img.id ? 'border-primary bg-primary/5 ring-1 ring-primary' : ''"
                @click="selectImage(img)"
              >
                <p class="font-medium mb-1">{{ img.name }}</p>
                <p class="text-xs text-muted-foreground truncate">{{ img.basePath }}</p>
                <p v-if="img.osVariant" class="text-xs text-muted-foreground mt-1">变体: {{ img.osVariant }}</p>
              </button>
            </div>
          </div>

          <!-- Step 3: 配置 -->
          <div v-else-if="step === 3">
            <div class="flex items-center justify-between mb-4">
              <p class="text-sm text-muted-foreground">确认配置</p>
              <Button variant="ghost" size="sm" @click="step = 2">返回</Button>
            </div>

            <!-- 已选摘要 -->
            <div class="grid grid-cols-2 gap-3 mb-4">
              <div class="p-3 rounded-lg bg-muted/50 border">
                <p class="text-xs text-muted-foreground mb-1">硬件规格</p>
                <p class="text-sm font-medium">{{ selectedFlavor?.name }}</p>
                <p class="text-xs text-muted-foreground">{{ selectedFlavor?.cpus }} vCPU / {{ formatMem(selectedFlavor?.memoryMB || 0) }} / {{ selectedFlavor?.diskGB }} GB</p>
              </div>
              <div class="p-3 rounded-lg bg-muted/50 border">
                <p class="text-xs text-muted-foreground mb-1">OS 模板</p>
                <p class="text-sm font-medium">{{ selectedImage?.name }}</p>
                <p class="text-xs text-muted-foreground truncate">{{ selectedImage?.basePath }}</p>
              </div>
            </div>

            <!-- VM 名称 -->
            <div class="mb-4">
              <label class="text-sm font-medium mb-1 block">VM 名称</label>
              <Input v-model="vmName" class="text-sm" placeholder="my-vm-01" />
            </div>

            <!-- 网络 -->
            <div class="mb-4">
              <label class="text-sm font-medium mb-1 block">网络</label>
              <div class="flex gap-2">
                <select v-model="netType" class="h-9 rounded-md border bg-background px-3 text-sm">
                  <option value="network">虚拟网络</option>
                  <option value="bridge">网桥</option>
                </select>
                <select v-model="netName" class="h-9 rounded-md border bg-background px-3 text-sm flex-1">
                  <template v-if="netType === 'network'">
                    <option v-for="n in networks" :key="n.name" :value="n.name">{{ n.name }}</option>
                  </template>
                  <template v-else>
                    <option v-for="b in bridges" :key="b" :value="b">{{ b }}</option>
                  </template>
                </select>
              </div>
            </div>

            <!-- 创建按钮 -->
            <div class="flex justify-end gap-2">
              <Button variant="outline" @click="close">取消</Button>
              <Button :disabled="!canCreate" :loading="creating" @click="create">
                <Zap class="h-4 w-4" /> 创建
              </Button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>
