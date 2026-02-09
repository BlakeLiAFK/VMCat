<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useToast } from '@/composables/useToast'
import {
  HostDetectDistro, LibvirtSetupScriptList, HostRunScript, HostCheckTools,
} from '@/api/backend'
import Dialog from '@/components/ui/Dialog.vue'
import Button from '@/components/ui/Button.vue'
import {
  Terminal, Loader2, CheckCircle, XCircle, Copy, Play, ChevronDown, ChevronUp,
} from 'lucide-vue-next'

const props = defineProps<{ open: boolean; hostId: string }>()
const emit = defineEmits<{
  'update:open': [v: boolean]
  installed: []
}>()

const toast = useToast()

const distro = ref('')
const detecting = ref(false)
const scripts = ref<any[]>([])
const selectedScript = ref<any>(null)
const showScript = ref(false)
const running = ref(false)
const output = ref('')
const status = ref<'idle' | 'running' | 'done' | 'error'>('idle')

const recommendedScript = computed(() => {
  if (!distro.value || !scripts.value.length) return null
  return scripts.value.find((s: any) =>
    s.distros.split(',').some((d: string) => distro.value.includes(d))
  )
})

async function init() {
  detecting.value = true
  try {
    const [d, sl] = await Promise.all([
      HostDetectDistro(props.hostId).catch(() => 'unknown'),
      LibvirtSetupScriptList(),
    ])
    distro.value = d
    scripts.value = sl || []
    // 自动选择推荐脚本
    const rec = recommendedScript.value
    if (rec) selectedScript.value = rec
    else if (scripts.value.length) selectedScript.value = scripts.value[0]
  } finally {
    detecting.value = false
  }
}

function selectScript(s: any) {
  selectedScript.value = s
  showScript.value = false
}

function copyScript() {
  if (!selectedScript.value) return
  navigator.clipboard.writeText(selectedScript.value.script)
  toast.success('Script copied')
}

async function runScript() {
  if (!selectedScript.value) return
  running.value = true
  status.value = 'running'
  output.value = ''
  try {
    const result = await HostRunScript(props.hostId, selectedScript.value.script)
    output.value = result
    status.value = 'done'
    toast.success('Libvirt installed successfully')
    // 重新检测工具
    await HostCheckTools(props.hostId)
    emit('installed')
  } catch (e: any) {
    const msg = e.toString()
    // 提取 Output 部分
    const outMatch = msg.match(/Output:\s*([\s\S]*)/)
    output.value = outMatch ? outMatch[1] : msg
    status.value = 'error'
    toast.error('Script execution failed')
  } finally {
    running.value = false
  }
}

function close() {
  if (running.value) {
    toast.error('Please wait for script to finish')
    return
  }
  status.value = 'idle'
  output.value = ''
  showScript.value = false
  emit('update:open', false)
}

onMounted(init)
</script>

<template>
  <Dialog :open="open" size="lg" @update:open="close">
    <div class="space-y-4">
      <div class="flex items-center justify-between">
        <h2 class="text-lg font-semibold flex items-center gap-2">
          <Terminal class="h-5 w-5" /> Install Libvirt/KVM
        </h2>
        <button class="text-muted-foreground hover:text-foreground" @click="close">&times;</button>
      </div>

      <!-- 检测中 -->
      <div v-if="detecting" class="text-center py-6">
        <Loader2 class="h-5 w-5 animate-spin mx-auto text-muted-foreground" />
        <p class="text-sm text-muted-foreground mt-2">Detecting host distribution...</p>
      </div>

      <template v-else>
        <!-- 发行版信息 -->
        <div class="p-3 rounded-md bg-muted/50 border text-sm">
          <span class="text-muted-foreground">Detected distro: </span>
          <span class="font-medium">{{ distro || 'Unknown' }}</span>
          <span v-if="recommendedScript" class="text-xs text-primary ml-2">
            (Recommended: {{ recommendedScript.name }})
          </span>
        </div>

        <!-- 脚本选择 -->
        <div class="space-y-2">
          <label class="text-xs text-muted-foreground">Select install script</label>
          <div class="flex flex-wrap gap-2">
            <button
              v-for="s in scripts" :key="s.id"
              class="px-3 py-1.5 rounded-md border text-sm transition-all"
              :class="selectedScript?.id === s.id
                ? 'border-primary bg-primary/5 ring-1 ring-primary font-medium'
                : 'hover:border-primary/50'"
              @click="selectScript(s)"
            >
              {{ s.name }}
              <span v-if="recommendedScript?.id === s.id" class="text-xs text-primary ml-1">*</span>
            </button>
          </div>
        </div>

        <!-- 脚本预览 -->
        <div v-if="selectedScript">
          <button
            class="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
            @click="showScript = !showScript"
          >
            <component :is="showScript ? ChevronUp : ChevronDown" class="h-3 w-3" />
            {{ showScript ? 'Hide' : 'View' }} script
          </button>
          <div v-if="showScript" class="mt-2 relative">
            <pre class="text-xs font-mono bg-muted/50 border rounded-md p-3 max-h-48 overflow-auto whitespace-pre-wrap">{{ selectedScript.script }}</pre>
            <button
              class="absolute top-2 right-2 p-1 rounded hover:bg-accent text-muted-foreground hover:text-foreground"
              @click="copyScript"
            ><Copy class="h-3.5 w-3.5" /></button>
          </div>
        </div>

        <!-- 执行状态 -->
        <div v-if="status === 'running'" class="space-y-2">
          <div class="flex items-center gap-2 text-sm">
            <Loader2 class="h-4 w-4 animate-spin text-primary" />
            <span>Running install script...</span>
          </div>
        </div>

        <div v-if="status === 'done'" class="space-y-2">
          <div class="flex items-center gap-2 text-sm text-green-600">
            <CheckCircle class="h-4 w-4" />
            <span>Installation completed successfully</span>
          </div>
        </div>

        <div v-if="status === 'error'" class="space-y-2">
          <div class="flex items-center gap-2 text-sm text-destructive">
            <XCircle class="h-4 w-4" />
            <span>Installation failed</span>
          </div>
        </div>

        <!-- 输出日志 -->
        <div v-if="output" class="space-y-1">
          <label class="text-xs text-muted-foreground">Output</label>
          <pre class="text-xs font-mono bg-black/90 text-green-400 rounded-md p-3 max-h-48 overflow-auto whitespace-pre-wrap">{{ output }}</pre>
        </div>

        <!-- 操作按钮 -->
        <div class="flex justify-end gap-2 pt-2">
          <Button variant="outline" size="sm" @click="copyScript" :disabled="!selectedScript">
            <Copy class="h-3.5 w-3.5" /> Copy Script
          </Button>
          <Button
            size="sm"
            :disabled="!selectedScript || running"
            :loading="running"
            @click="runScript"
          >
            <Play class="h-3.5 w-3.5" /> Run on Host
          </Button>
        </div>
      </template>
    </div>
  </Dialog>
</template>
