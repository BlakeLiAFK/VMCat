<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@/composables/useToast'
import {
  ImageSourceList, ImageImport, ImageUpload,
} from '../../wailsjs/go/main/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import Dialog from '@/components/ui/Dialog.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import {
  Download, Upload, Globe, Loader2, CheckCircle, XCircle, ChevronRight, Copy,
} from 'lucide-vue-next'

const props = defineProps<{ open: boolean; hostId: string }>()
const emit = defineEmits<{
  'update:open': [v: boolean]
  imported: []
}>()

const toast = useToast()
const { t } = useI18n()

// 模式: source(预设源) / url(自定义URL) / upload(本地上传)
const mode = ref<'source' | 'url' | 'upload'>('source')
const sources = ref<any[]>([])
const loadingSources = ref(false)
const selectedSource = ref<any>(null)

// 表单
const customUrl = ref('')
const destDir = ref('/var/lib/libvirt/images')
const fileName = ref('')
const imageName = ref('')
const osVariant = ref('')
const localPath = ref('')

// 任务状态
const taskId = ref('')
const taskStatus = ref<'idle' | 'running' | 'done' | 'error'>('idle')
const taskPercent = ref(0)
const taskError = ref('')
const taskCurrent = ref(0)
const taskTotal = ref(0)

const destPath = computed(() => {
  const dir = destDir.value.replace(/\/+$/, '')
  return fileName.value ? `${dir}/${fileName.value}` : ''
})

const canStart = computed(() => {
  if (taskStatus.value === 'running') return false
  if (!destPath.value) return false
  if (!imageName.value.trim()) return false
  if (mode.value === 'source') return !!selectedSource.value
  if (mode.value === 'url') return !!customUrl.value.trim()
  if (mode.value === 'upload') return !!localPath.value.trim()
  return false
})

async function loadSources() {
  loadingSources.value = true
  try {
    sources.value = (await ImageSourceList()) || []
  } catch { sources.value = [] }
  finally { loadingSources.value = false }
}

function selectSource(src: any) {
  selectedSource.value = src
  imageName.value = src.name
  osVariant.value = src.osVariant || ''
  fileName.value = src.fileName || ''
}

function copyUrl(url: string, e: Event) {
  e.stopPropagation()
  navigator.clipboard.writeText(url)
  toast.success(t('imageImport.urlCopied'))
}

function onSelectUrl() {
  mode.value = 'url'
  selectedSource.value = null
  // 从 URL 自动提取文件名
}

function onUrlInput() {
  if (customUrl.value) {
    try {
      const u = new URL(customUrl.value)
      const parts = u.pathname.split('/')
      const fn = parts[parts.length - 1] || ''
      if (fn && !fileName.value) fileName.value = fn
      if (fn && !imageName.value) {
        imageName.value = fn.replace(/\.(qcow2|img|raw|vmdk)$/i, '').replace(/[-_]/g, ' ')
      }
    } catch { /* */ }
  }
}

async function selectLocalFile() {
  // Wails 没有直接文件选择 API，用 input[type=file] 模拟不行（桌面应用）
  // 用户需要手动输入路径
  mode.value = 'upload'
  selectedSource.value = null
}

async function startImport() {
  if (!canStart.value) return
  taskStatus.value = 'running'
  taskPercent.value = 0
  taskError.value = ''
  taskCurrent.value = 0
  taskTotal.value = 0

  try {
    let tid = ''
    if (mode.value === 'upload') {
      tid = await ImageUpload(props.hostId, localPath.value, destPath.value, imageName.value, osVariant.value)
    } else {
      const url = mode.value === 'source' ? selectedSource.value.url : customUrl.value
      tid = await ImageImport(props.hostId, url, destPath.value, imageName.value, osVariant.value)
    }
    taskId.value = tid
  } catch (e: any) {
    taskStatus.value = 'error'
    taskError.value = e.toString()
  }
}

function onProgress(data: any) {
  if (data.taskId !== taskId.value) return
  taskPercent.value = data.percent || 0
  taskCurrent.value = data.current || 0
  taskTotal.value = data.totalSize || 0
}

function onDone(data: any) {
  if (data.taskId !== taskId.value) return
  taskStatus.value = 'done'
  taskPercent.value = 100
  toast.success(t('imageImport.importSuccess', { name: imageName.value }))
  emit('imported')
}

function onError(data: any) {
  if (data.taskId !== taskId.value) return
  taskStatus.value = 'error'
  taskError.value = data.error || t('imageImport.unknownError')
  toast.error(t('imageImport.importFailed') + ': ' + taskError.value)
}

function formatSize(bytes: number): string {
  if (bytes <= 0) return '-'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(0) + ' KB'
  if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
  return (bytes / (1024 * 1024 * 1024)).toFixed(2) + ' GB'
}

function close() {
  if (taskStatus.value === 'running') {
    toast.error(t('imageImport.waitImport'))
    return
  }
  resetForm()
  emit('update:open', false)
}

function resetForm() {
  mode.value = 'source'
  selectedSource.value = null
  customUrl.value = ''
  fileName.value = ''
  imageName.value = ''
  osVariant.value = ''
  localPath.value = ''
  taskId.value = ''
  taskStatus.value = 'idle'
  taskPercent.value = 0
  taskError.value = ''
}

let cleanupProgress: (() => void) | null = null
let cleanupDone: (() => void) | null = null
let cleanupError: (() => void) | null = null

onMounted(() => {
  loadSources()
  cleanupProgress = EventsOn('image:import:progress', onProgress)
  cleanupDone = EventsOn('image:import:done', onDone)
  cleanupError = EventsOn('image:import:error', onError)
})

onUnmounted(() => {
  if (cleanupProgress) cleanupProgress()
  if (cleanupDone) cleanupDone()
  if (cleanupError) cleanupError()
})
</script>

<template>
  <Dialog :open="open" size="xl" @update:open="close">
    <div class="space-y-4">
      <div class="flex items-center justify-between">
        <h2 class="text-lg font-semibold">{{ t('imageImport.title') }}</h2>
        <button class="text-muted-foreground hover:text-foreground" @click="close">&times;</button>
      </div>

      <!-- 正在进行中的任务 -->
      <div v-if="taskStatus === 'running'" class="space-y-3">
        <div class="flex items-center gap-2 text-sm">
          <Loader2 class="h-4 w-4 animate-spin text-primary" />
          <span>{{ mode === 'upload' ? t('imageImport.uploading') : t('imageImport.downloading') }}</span>
          <span class="ml-auto font-mono text-xs">{{ taskPercent }}%</span>
        </div>
        <div class="w-full h-2 bg-muted rounded-full overflow-hidden">
          <div
            class="h-full bg-primary rounded-full transition-all duration-300"
            :style="{ width: taskPercent + '%' }"
          />
        </div>
        <div class="text-xs text-muted-foreground">
          {{ formatSize(taskCurrent) }} / {{ formatSize(taskTotal) }}
        </div>
      </div>

      <!-- 完成 -->
      <div v-else-if="taskStatus === 'done'" class="text-center py-4 space-y-3">
        <CheckCircle class="h-10 w-10 text-green-500 mx-auto" />
        <p class="text-sm font-medium">{{ t('imageImport.importSuccessTitle', { name: imageName }) }}</p>
        <Button size="sm" @click="close">{{ t('common.close') }}</Button>
      </div>

      <!-- 错误 -->
      <div v-else-if="taskStatus === 'error'" class="text-center py-4 space-y-3">
        <XCircle class="h-10 w-10 text-destructive mx-auto" />
        <p class="text-sm text-destructive">{{ taskError }}</p>
        <div class="flex justify-center gap-2">
          <Button variant="outline" size="sm" @click="taskStatus = 'idle'">{{ t('common.retry') }}</Button>
          <Button variant="outline" size="sm" @click="close">{{ t('common.close') }}</Button>
        </div>
      </div>

      <!-- 表单 -->
      <template v-else>
        <!-- 模式切换 -->
        <div class="flex border rounded-md overflow-hidden">
          <button
            class="flex-1 px-3 py-2 text-sm flex items-center justify-center gap-1.5 transition-colors"
            :class="mode === 'source' ? 'bg-primary text-primary-foreground' : 'hover:bg-accent'"
            @click="mode = 'source'; selectedSource = null"
          ><Globe class="h-3.5 w-3.5" /> {{ t('imageImport.presetSource') }}</button>
          <button
            class="flex-1 px-3 py-2 text-sm flex items-center justify-center gap-1.5 transition-colors"
            :class="mode === 'url' ? 'bg-primary text-primary-foreground' : 'hover:bg-accent'"
            @click="onSelectUrl"
          ><Download class="h-3.5 w-3.5" /> {{ t('imageImport.customURL') }}</button>
          <button
            class="flex-1 px-3 py-2 text-sm flex items-center justify-center gap-1.5 transition-colors"
            :class="mode === 'upload' ? 'bg-primary text-primary-foreground' : 'hover:bg-accent'"
            @click="selectLocalFile"
          ><Upload class="h-3.5 w-3.5" /> {{ t('imageImport.localUpload') }}</button>
        </div>

        <!-- 预设源列表 -->
        <div v-if="mode === 'source'" class="max-h-48 overflow-auto space-y-1">
          <div v-if="loadingSources" class="text-center py-4">
            <Loader2 class="h-4 w-4 animate-spin mx-auto text-muted-foreground" />
          </div>
          <button
            v-else
            v-for="src in sources" :key="src.id"
            class="w-full p-3 rounded-md border text-left text-sm transition-all hover:border-primary/50"
            :class="selectedSource?.id === src.id ? 'border-primary bg-primary/5 ring-1 ring-primary' : ''"
            @click="selectSource(src)"
          >
            <div class="flex items-center justify-between">
              <span class="font-medium">{{ src.name }}</span>
              <div class="flex items-center gap-1">
                <button
                  class="p-1 rounded hover:bg-accent text-muted-foreground hover:text-foreground"
                  :title="t('imageImport.copyUrl')"
                  @click="copyUrl(src.url, $event)"
                ><Copy class="h-3 w-3" /></button>
                <ChevronRight class="h-3.5 w-3.5 text-muted-foreground" />
              </div>
            </div>
            <p v-if="src.description" class="text-xs text-muted-foreground mt-0.5">{{ src.description }}</p>
            <p class="text-xs text-muted-foreground/60 mt-0.5 truncate font-mono select-all">{{ src.url }}</p>
          </button>
        </div>

        <!-- 自定义 URL -->
        <div v-if="mode === 'url'" class="space-y-2">
          <label class="text-xs text-muted-foreground">{{ t('imageImport.downloadURL') }}</label>
          <Input
            v-model="customUrl"
            class="h-8 text-sm font-mono"
            placeholder="https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img"
            @input="onUrlInput"
          />
        </div>

        <!-- 本地上传 -->
        <div v-if="mode === 'upload'" class="space-y-2">
          <label class="text-xs text-muted-foreground">{{ t('imageImport.localPath') }}</label>
          <Input
            v-model="localPath"
            class="h-8 text-sm font-mono"
            placeholder="/Users/you/Downloads/ubuntu-24.04.qcow2"
          />
          <p class="text-xs text-muted-foreground">{{ t('imageImport.localPathTip') }}</p>
        </div>

        <!-- 通用配置 -->
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="text-xs text-muted-foreground">{{ t('imageImport.imageName') }}</label>
            <Input v-model="imageName" class="h-8 text-sm" placeholder="Ubuntu 24.04" />
          </div>
          <div>
            <label class="text-xs text-muted-foreground">{{ t('imageImport.osVariant') }}</label>
            <Input v-model="osVariant" class="h-8 text-sm" placeholder="ubuntu24.04" />
          </div>
        </div>
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="text-xs text-muted-foreground">{{ t('imageImport.destDir') }}</label>
            <Input v-model="destDir" class="h-8 text-sm font-mono" />
          </div>
          <div>
            <label class="text-xs text-muted-foreground">{{ t('imageImport.fileName') }}</label>
            <Input v-model="fileName" class="h-8 text-sm font-mono" placeholder="ubuntu-24.04.qcow2" />
          </div>
        </div>
        <p v-if="destPath" class="text-xs text-muted-foreground font-mono truncate">
          {{ t('imageImport.destLabel') }}: {{ destPath }}
        </p>

        <!-- 操作按钮 -->
        <div class="flex justify-end gap-2 pt-2">
          <Button variant="outline" size="sm" @click="close">{{ t('common.cancel') }}</Button>
          <Button size="sm" :disabled="!canStart" @click="startImport">
            <template v-if="mode === 'upload'">
              <Upload class="h-3.5 w-3.5" /> {{ t('imageImport.startUpload') }}
            </template>
            <template v-else>
              <Download class="h-3.5 w-3.5" /> {{ t('imageImport.startDownload') }}
            </template>
          </Button>
        </div>
      </template>
    </div>
  </Dialog>
</template>
