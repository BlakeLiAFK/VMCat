<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@/composables/useToast'
import { VMClone } from '../../wailsjs/go/main/App'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import { X } from 'lucide-vue-next'

const { t } = useI18n()
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
  if (!name) { toast.warning(t('vmClone.enterName')); return }
  if (name === props.vmName) { toast.warning(t('vmEdit.nameSameWarning')); return }
  cloning.value = true
  try {
    await VMClone(props.hostId, props.vmName, name)
    toast.success(t('vmClone.cloneSuccess', { name }))
    emit('saved')
    emit('update:open', false)
  } catch (e: any) {
    toast.error(t('vmClone.cloneFailed') + ': ' + e.toString())
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
          <h2 class="text-lg font-semibold">{{ t('vmClone.title') }}</h2>
          <button @click="close" class="p-1 rounded hover:bg-accent"><X class="h-4 w-4" /></button>
        </div>
        <div class="p-5 space-y-4">
          <div>
            <label class="text-sm mb-1 block">{{ t('vmEdit.originalVM') }}</label>
            <p class="text-sm font-mono bg-muted/50 p-2 rounded">{{ vmName }}</p>
          </div>
          <div>
            <label class="text-sm mb-1 block">{{ t('vmClone.newName') }} *</label>
            <Input v-model="newName" placeholder="my-vm-clone" @keyup.enter="submit" />
          </div>
          <p class="text-xs text-muted-foreground">{{ t('vmEdit.diskAutoCopy') }}</p>
        </div>
        <div class="flex justify-end gap-2 p-5 border-t">
          <Button variant="outline" @click="close">{{ t('common.cancel') }}</Button>
          <Button :loading="cloning" @click="submit">{{ t('vm.clone') }}</Button>
        </div>
      </div>
    </div>
  </Teleport>
</template>
