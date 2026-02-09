<script setup lang="ts">
import { ref, watch } from 'vue'
import { useToast } from '@/composables/useToast'
import { VMSetVCPUs, VMSetMemory, VMSetAutostart, VMRename } from '../../wailsjs/go/main/App'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import { X } from 'lucide-vue-next'

const props = defineProps<{
  open: boolean
  hostId: string
  vmName: string
  detail: any
}>()
const emit = defineEmits<{ 'update:open': [v: boolean]; saved: [newName?: string] }>()
const toast = useToast()

const cpus = ref(1)
const memoryMB = ref(1024)
const autostart = ref(false)
const newName = ref('')
const saving = ref('')

watch(() => props.open, (v) => {
  if (v && props.detail) {
    cpus.value = props.detail.cpus || 1
    memoryMB.value = props.detail.memoryMB || 1024
    autostart.value = props.detail.autostart || false
    newName.value = props.detail.name || ''
  }
})

async function saveCPU() {
  saving.value = 'cpu'
  try {
    await VMSetVCPUs(props.hostId, props.vmName, Number(cpus.value))
    toast.success(`CPU 已设为 ${cpus.value} vCPU (重启生效)`)
    emit('saved')
  } catch (e: any) {
    toast.error('设置 CPU 失败: ' + e.toString())
  } finally { saving.value = '' }
}

async function saveMemory() {
  saving.value = 'mem'
  try {
    await VMSetMemory(props.hostId, props.vmName, Number(memoryMB.value))
    toast.success(`内存已设为 ${memoryMB.value} MB (重启生效)`)
    emit('saved')
  } catch (e: any) {
    toast.error('设置内存失败: ' + e.toString())
  } finally { saving.value = '' }
}

async function saveAutostart() {
  saving.value = 'auto'
  try {
    await VMSetAutostart(props.hostId, props.vmName, autostart.value)
    toast.success(autostart.value ? '已开启自动启动' : '已关闭自动启动')
    emit('saved')
  } catch (e: any) {
    toast.error('设置失败: ' + e.toString())
  } finally { saving.value = '' }
}

async function saveRename() {
  const name = newName.value.trim()
  if (!name || name === props.vmName) return
  saving.value = 'rename'
  try {
    await VMRename(props.hostId, props.vmName, name)
    toast.success(`已重命名为 ${name}`)
    emit('saved', name)
  } catch (e: any) {
    toast.error('重命名失败: ' + e.toString())
  } finally { saving.value = '' }
}

function close() { emit('update:open', false) }
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center">
      <div class="absolute inset-0 bg-black/50" @click="close" />
      <div class="relative bg-card border rounded-xl shadow-2xl w-[440px] max-h-[80vh] overflow-y-auto">
        <div class="flex items-center justify-between p-5 border-b">
          <h2 class="text-lg font-semibold">编辑虚拟机</h2>
          <button @click="close" class="p-1 rounded hover:bg-accent"><X class="h-4 w-4" /></button>
        </div>

        <div class="p-5 space-y-5">
          <!-- 重命名 -->
          <div>
            <label class="text-sm font-medium mb-2 block">名称 (需关机)</label>
            <div class="flex gap-2">
              <Input v-model="newName" class="flex-1" />
              <Button size="sm" :loading="saving === 'rename'" @click="saveRename"
                :disabled="!newName.trim() || newName.trim() === vmName">保存</Button>
            </div>
          </div>

          <!-- CPU -->
          <div>
            <label class="text-sm font-medium mb-2 block">CPU (vCPU) (重启生效)</label>
            <div class="flex gap-2">
              <Input v-model="cpus" type="number" class="flex-1" />
              <Button size="sm" :loading="saving === 'cpu'" @click="saveCPU">保存</Button>
            </div>
          </div>

          <!-- 内存 -->
          <div>
            <label class="text-sm font-medium mb-2 block">内存 MB (重启生效)</label>
            <div class="flex gap-2">
              <Input v-model="memoryMB" type="number" class="flex-1" />
              <Button size="sm" :loading="saving === 'mem'" @click="saveMemory">保存</Button>
            </div>
            <p class="text-xs text-muted-foreground mt-1">
              常用: 512, 1024, 2048, 4096, 8192, 16384
            </p>
          </div>

          <!-- Autostart -->
          <div>
            <label class="text-sm font-medium mb-2 block">自动启动</label>
            <div class="flex items-center gap-3">
              <label class="flex items-center gap-2 text-sm cursor-pointer">
                <input type="checkbox" v-model="autostart" class="rounded" />
                宿主机启动时自动启动此 VM
              </label>
              <Button size="sm" variant="outline" :loading="saving === 'auto'" @click="saveAutostart">保存</Button>
            </div>
          </div>
        </div>

        <div class="flex justify-end p-5 border-t">
          <Button variant="outline" @click="close">关闭</Button>
        </div>
      </div>
    </div>
  </Teleport>
</template>
