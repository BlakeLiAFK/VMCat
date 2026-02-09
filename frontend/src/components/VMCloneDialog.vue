<script setup lang="ts">
import { ref, watch } from 'vue'
import { useToast } from '@/composables/useToast'
import { VMClone } from '../../wailsjs/go/main/App'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import { X } from 'lucide-vue-next'

const props = defineProps<{ open: boolean; hostId: string; vmName: string }>()
const emit = defineEmits<{ 'update:open': [v: boolean]; saved: [] }>()
const toast = useToast()

const newName = ref('')
const cloning = ref(false)

watch(() => props.open, (v) => {
  if (v) newName.value = props.vmName + '-clone'
})

async function submit() {
  const name = newName.value.trim()
  if (!name) { toast.warning('请输入新名称'); return }
  if (name === props.vmName) { toast.warning('名称不能与原 VM 相同'); return }
  cloning.value = true
  try {
    await VMClone(props.hostId, props.vmName, name)
    toast.success(`克隆成功: ${name}`)
    emit('saved')
    emit('update:open', false)
  } catch (e: any) {
    toast.error('克隆失败: ' + e.toString())
  } finally {
    cloning.value = false
  }
}

function close() { emit('update:open', false) }
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center">
      <div class="absolute inset-0 bg-black/50" @click="close" />
      <div class="relative bg-card border rounded-xl shadow-2xl w-[400px]">
        <div class="flex items-center justify-between p-5 border-b">
          <h2 class="text-lg font-semibold">克隆虚拟机</h2>
          <button @click="close" class="p-1 rounded hover:bg-accent"><X class="h-4 w-4" /></button>
        </div>
        <div class="p-5 space-y-4">
          <div>
            <label class="text-sm mb-1 block">原始 VM</label>
            <p class="text-sm font-mono bg-muted/50 p-2 rounded">{{ vmName }}</p>
          </div>
          <div>
            <label class="text-sm mb-1 block">新名称 *</label>
            <Input v-model="newName" placeholder="my-vm-clone" @keyup.enter="submit" />
          </div>
          <p class="text-xs text-muted-foreground">需要 VM 处于关机状态，磁盘将自动复制</p>
        </div>
        <div class="flex justify-end gap-2 p-5 border-t">
          <Button variant="outline" @click="close">取消</Button>
          <Button :loading="cloning" @click="submit">克隆</Button>
        </div>
      </div>
    </div>
  </Teleport>
</template>
