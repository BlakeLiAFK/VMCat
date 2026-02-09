<script setup lang="ts">
import { ref, watch } from 'vue'
import { useToast } from '@/composables/useToast'
import { VMGetXML, VMDefineXML } from '../../wailsjs/go/main/App'
import Button from '@/components/ui/Button.vue'
import { X, Loader2, Copy, Save } from 'lucide-vue-next'

const props = defineProps<{ open: boolean; hostId: string; vmName: string }>()
const emit = defineEmits<{ 'update:open': [v: boolean]; saved: [] }>()
const toast = useToast()

const xml = ref('')
const loading = ref(false)
const saving = ref(false)
const editing = ref(false)

watch(() => props.open, async (v) => {
  if (!v) return
  editing.value = false
  loading.value = true
  try {
    xml.value = await VMGetXML(props.hostId, props.vmName)
  } catch (e: any) {
    toast.error('获取 XML 失败: ' + e.toString())
  } finally {
    loading.value = false
  }
})

async function save() {
  saving.value = true
  try {
    await VMDefineXML(props.hostId, xml.value)
    toast.success('XML 配置已保存 (重启生效)')
    editing.value = false
    emit('saved')
  } catch (e: any) {
    toast.error('保存失败: ' + e.toString())
  } finally {
    saving.value = false
  }
}

function copyXML() {
  navigator.clipboard.writeText(xml.value)
  toast.success('已复制到剪贴板')
}

function close() { emit('update:open', false) }
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center">
      <div class="absolute inset-0 bg-black/50" @click="close" />
      <div class="relative bg-card border rounded-xl shadow-2xl w-[700px] max-h-[90vh] flex flex-col">
        <div class="flex items-center justify-between p-5 border-b flex-shrink-0">
          <h2 class="text-lg font-semibold">XML 配置</h2>
          <div class="flex items-center gap-2">
            <Button variant="ghost" size="icon" @click="copyXML" title="复制">
              <Copy class="h-4 w-4" />
            </Button>
            <button @click="close" class="p-1 rounded hover:bg-accent"><X class="h-4 w-4" /></button>
          </div>
        </div>

        <div class="flex-1 overflow-hidden">
          <div v-if="loading" class="p-12 text-center">
            <Loader2 class="h-6 w-6 animate-spin mx-auto text-muted-foreground" />
          </div>
          <textarea v-else v-model="xml" :readonly="!editing"
            class="w-full h-full p-4 font-mono text-xs bg-muted/30 border-0 resize-none focus:outline-none"
            style="min-height: 400px;"
            spellcheck="false"
          />
        </div>

        <div class="flex items-center justify-between p-5 border-t flex-shrink-0">
          <label class="flex items-center gap-2 text-sm cursor-pointer">
            <input type="checkbox" v-model="editing" class="rounded" />
            启用编辑
          </label>
          <div class="flex gap-2">
            <Button variant="outline" @click="close">关闭</Button>
            <Button v-if="editing" :loading="saving" @click="save">
              <Save class="h-4 w-4" /> 保存
            </Button>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>
