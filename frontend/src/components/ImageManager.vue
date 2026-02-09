<script setup lang="ts">
import { ref, watch } from 'vue'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'
import { ImageList, ImageAdd, ImageUpdate, ImageDelete } from '../../wailsjs/go/main/App'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import { Plus, Pencil, Trash2, Loader2, Image } from 'lucide-vue-next'

const props = defineProps<{ hostId: string; visible: boolean }>()
const toast = useToast()
const { request: confirmRequest } = useConfirm()

const images = ref<any[]>([])
const loading = ref(false)
const showForm = ref(false)
const editing = ref<any>(null)
const form = ref({ name: '', basePath: '', osVariant: '' })

async function loadImages() {
  loading.value = true
  try {
    images.value = (await ImageList(props.hostId)) || []
  } catch (e: any) {
    toast.error('加载镜像失败: ' + e.toString())
  } finally {
    loading.value = false
  }
}

function openForm(img?: any) {
  if (img) {
    editing.value = img
    form.value = { name: img.name, basePath: img.basePath, osVariant: img.osVariant }
  } else {
    editing.value = null
    form.value = { name: '', basePath: '', osVariant: '' }
  }
  showForm.value = true
}

async function save() {
  const f = form.value
  if (!f.name.trim() || !f.basePath.trim()) { toast.error('请填写名称和基础镜像路径'); return }
  try {
    if (editing.value) {
      await ImageUpdate({ ...editing.value, ...f })
      toast.success('镜像已更新')
    } else {
      await ImageAdd(props.hostId, { id: '', hostId: props.hostId, name: f.name, basePath: f.basePath, osVariant: f.osVariant, sortOrder: images.value.length, createdAt: '' })
      toast.success('镜像已添加')
    }
    showForm.value = false
    await loadImages()
  } catch (e: any) {
    toast.error('保存失败: ' + e.toString())
  }
}

async function remove(id: string, name: string) {
  const ok = await confirmRequest('删除镜像', `确认删除镜像 "${name}"?`)
  if (!ok) return
  try {
    await ImageDelete(id)
    toast.success('镜像已删除')
    await loadImages()
  } catch (e: any) {
    toast.error('删除失败: ' + e.toString())
  }
}

watch(() => props.visible, (v) => {
  if (v && images.value.length === 0) loadImages()
})
</script>

<template>
  <div class="space-y-4">
    <Card>
      <div class="p-4 border-b flex items-center justify-between">
        <h3 class="font-semibold flex items-center gap-2">
          <Image class="h-4 w-4" /> OS 基础镜像
        </h3>
        <div class="flex gap-2">
          <Button variant="outline" size="sm" @click="loadImages" :loading="loading">刷新</Button>
          <Button variant="outline" size="sm" @click="openForm()">
            <Plus class="h-3.5 w-3.5" /> 添加
          </Button>
        </div>
      </div>

      <div v-if="loading && !images.length" class="p-6 text-center">
        <Loader2 class="h-5 w-5 animate-spin mx-auto text-muted-foreground" />
      </div>
      <div v-else-if="!images.length" class="p-8 text-center text-sm text-muted-foreground">
        暂无基础镜像，请添加宿主机上的 qcow2 基础镜像用于快速创建 VM
      </div>
      <table v-else class="w-full text-sm">
        <thead>
          <tr class="border-b text-muted-foreground">
            <th class="text-left p-3 font-medium">名称</th>
            <th class="text-left p-3 font-medium">基础镜像路径</th>
            <th class="text-left p-3 font-medium">OS 变体</th>
            <th class="text-right p-3 font-medium pr-4">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="img in images" :key="img.id" class="border-b last:border-0 hover:bg-muted/30">
            <td class="p-3 font-medium">{{ img.name }}</td>
            <td class="p-3 font-mono text-xs text-muted-foreground">{{ img.basePath }}</td>
            <td class="p-3">{{ img.osVariant || '-' }}</td>
            <td class="p-3 text-right pr-4">
              <div class="flex items-center justify-end gap-1">
                <Button variant="ghost" size="icon" @click="openForm(img)" title="编辑">
                  <Pencil class="h-3.5 w-3.5" />
                </Button>
                <Button variant="ghost" size="icon" @click="remove(img.id, img.name)" title="删除">
                  <Trash2 class="h-3.5 w-3.5 text-destructive" />
                </Button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </Card>

    <!-- 编辑表单 -->
    <Card v-if="showForm" class="border-primary">
      <div class="p-4 border-b">
        <h3 class="font-semibold">{{ editing ? '编辑镜像' : '添加镜像' }}</h3>
      </div>
      <div class="p-4 space-y-3">
        <div class="grid grid-cols-3 gap-3">
          <div>
            <label class="text-xs text-muted-foreground">显示名称</label>
            <Input v-model="form.name" class="h-8 text-sm" placeholder="Ubuntu 24.04" />
          </div>
          <div>
            <label class="text-xs text-muted-foreground">基础镜像路径 (宿主机上)</label>
            <Input v-model="form.basePath" class="h-8 text-sm" placeholder="/data/images/ubuntu-24.04.qcow2" />
          </div>
          <div>
            <label class="text-xs text-muted-foreground">OS 变体 (可选)</label>
            <Input v-model="form.osVariant" class="h-8 text-sm" placeholder="ubuntu24.04" />
          </div>
        </div>
        <div class="flex justify-end gap-2">
          <Button variant="outline" size="sm" @click="showForm = false">取消</Button>
          <Button size="sm" @click="save">保存</Button>
        </div>
      </div>
    </Card>
  </div>
</template>
