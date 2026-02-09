<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'
import { ImageList, ImageAdd, ImageUpdate, ImageDelete, HostImageScan, HostImageDelete } from '@/api/backend'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import ImageImportDialog from '@/components/ImageImportDialog.vue'
import { Plus, Pencil, Trash2, Loader2, Image, Download, Search, FolderOpen, FileCheck } from 'lucide-vue-next'

const { t } = useI18n()
const props = defineProps<{ hostId: string; visible: boolean }>()
const toast = useToast()
const { request: confirmRequest } = useConfirm()

const images = ref<any[]>([])
const loading = ref(false)
const showForm = ref(false)
const showImport = ref(false)
const editing = ref<any>(null)
const form = ref({ name: '', basePath: '', osVariant: '' })

// 宿主机文件扫描
const hostFiles = ref<any[]>([])
const scanning = ref(false)

async function loadImages() {
  loading.value = true
  try {
    images.value = (await ImageList(props.hostId)) || []
  } catch (e: any) {
    toast.error(t('imageManager.loadFailed') + ': ' + e.toString())
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
  if (!f.name.trim() || !f.basePath.trim()) { toast.error(t('imageManager.fillNameAndPath')); return }
  try {
    if (editing.value) {
      await ImageUpdate({ ...editing.value, ...f })
      toast.success(t('imageManager.imageUpdated'))
    } else {
      await ImageAdd(props.hostId, { id: '', hostId: props.hostId, name: f.name, basePath: f.basePath, osVariant: f.osVariant, sortOrder: images.value.length, createdAt: '' })
      toast.success(t('imageManager.imageAdded'))
    }
    showForm.value = false
    await loadImages()
  } catch (e: any) {
    toast.error(t('common.saveFailed') + ': ' + e.toString())
  }
}

async function remove(id: string, name: string) {
  const ok = await confirmRequest(t('imageManager.deleteImage'), t('imageManager.deleteImageConfirm', { name }))
  if (!ok) return
  try {
    await ImageDelete(id)
    toast.success(t('imageManager.imageDeleted'))
    await loadImages()
  } catch (e: any) {
    toast.error(t('imageManager.imageDeleteFailed') + ': ' + e.toString())
  }
}

function onImported() {
  showImport.value = false
  loadImages()
}

async function scanHostFiles() {
  scanning.value = true
  try {
    hostFiles.value = (await HostImageScan(props.hostId)) || []
  } catch (e: any) {
    toast.error(t('imageManager.scanFailed') + ': ' + e.toString())
  } finally {
    scanning.value = false
  }
}

function isRegistered(path: string): boolean {
  return images.value.some((img: any) => img.basePath === path)
}

function registerFile(file: any) {
  const name = file.name.replace(/\.(qcow2|img|raw|vmdk)$/i, '').replace(/[-_]/g, ' ')
  form.value = { name, basePath: file.path, osVariant: '' }
  editing.value = null
  showForm.value = true
}

async function deleteHostFile(file: any) {
  const ok = await confirmRequest(t('imageManager.deleteHostFile'), t('imageManager.deleteHostFileConfirm', { path: file.path }), { variant: 'destructive' as const, confirmText: t('common.delete') })
  if (!ok) return
  try {
    await HostImageDelete(props.hostId, file.path)
    toast.success(t('imageManager.fileDeleted'))
    // 同时删除对应的数据库记录
    const matched = images.value.find((img: any) => img.basePath === file.path)
    if (matched) {
      await ImageDelete(matched.id).catch(() => {})
    }
    await Promise.all([scanHostFiles(), loadImages()])
  } catch (e: any) {
    toast.error(t('common.deleteFailed') + ': ' + e.toString())
  }
}

watch(() => props.visible, (v) => {
  if (v) {
    if (images.value.length === 0) loadImages()
    if (hostFiles.value.length === 0) scanHostFiles()
  }
})
</script>

<template>
  <div class="space-y-4">
    <Card>
      <div class="p-4 border-b flex items-center justify-between">
        <h3 class="font-semibold flex items-center gap-2">
          <Image class="h-4 w-4" /> {{ t('imageManager.baseImage') }}
        </h3>
        <div class="flex gap-2">
          <Button variant="outline" size="sm" @click="loadImages" :loading="loading">{{ t('common.refresh') }}</Button>
          <Button variant="outline" size="sm" @click="showImport = true">
            <Download class="h-3.5 w-3.5" /> {{ t('imageManager.importBtn') }}
          </Button>
          <Button variant="outline" size="sm" @click="openForm()">
            <Plus class="h-3.5 w-3.5" /> {{ t('common.add') }}
          </Button>
        </div>
      </div>

      <div v-if="loading && !images.length" class="p-6 text-center">
        <Loader2 class="h-5 w-5 animate-spin mx-auto text-muted-foreground" />
      </div>
      <div v-else-if="!images.length" class="p-8 text-center text-sm text-muted-foreground">
        {{ t('imageManager.noImagesTip2') }}
      </div>
      <table v-else class="w-full text-sm">
        <thead>
          <tr class="border-b text-muted-foreground">
            <th class="text-left p-3 font-medium">{{ t('imageManager.thName') }}</th>
            <th class="text-left p-3 font-medium">{{ t('imageManager.thBasePath') }}</th>
            <th class="text-left p-3 font-medium">{{ t('imageManager.thOSVariant') }}</th>
            <th class="text-right p-3 font-medium pr-4">{{ t('imageManager.thActions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="img in images" :key="img.id" class="border-b last:border-0 hover:bg-muted/30">
            <td class="p-3 font-medium">{{ img.name }}</td>
            <td class="p-3 font-mono text-xs text-muted-foreground">{{ img.basePath }}</td>
            <td class="p-3">{{ img.osVariant || '-' }}</td>
            <td class="p-3 text-right pr-4">
              <div class="flex items-center justify-end gap-1">
                <Button variant="ghost" size="icon" @click="openForm(img)" :title="t('common.edit')">
                  <Pencil class="h-3.5 w-3.5" />
                </Button>
                <Button variant="ghost" size="icon" @click="remove(img.id, img.name)" :title="t('common.delete')">
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
        <h3 class="font-semibold">{{ editing ? t('imageManager.editImage') : t('imageManager.addImage') }}</h3>
      </div>
      <div class="p-4 space-y-3">
        <div class="grid grid-cols-3 gap-3">
          <div>
            <label class="text-xs text-muted-foreground">{{ t('imageManager.displayName') }}</label>
            <Input v-model="form.name" class="h-8 text-sm" placeholder="Ubuntu 24.04" />
          </div>
          <div>
            <label class="text-xs text-muted-foreground">{{ t('imageManager.basePathLabel') }}</label>
            <Input v-model="form.basePath" class="h-8 text-sm" placeholder="/data/images/ubuntu-24.04.qcow2" />
          </div>
          <div>
            <label class="text-xs text-muted-foreground">{{ t('imageManager.osVariantOptional') }}</label>
            <Input v-model="form.osVariant" class="h-8 text-sm" placeholder="ubuntu24.04" />
          </div>
        </div>
        <div class="flex justify-end gap-2">
          <Button variant="outline" size="sm" @click="showForm = false">{{ t('common.cancel') }}</Button>
          <Button size="sm" @click="save">{{ t('common.save') }}</Button>
        </div>
      </div>
    </Card>
    <!-- 宿主机文件 -->
    <Card class="mt-4">
      <div class="p-4 border-b flex items-center justify-between">
        <h3 class="font-semibold flex items-center gap-2">
          <FolderOpen class="h-4 w-4" /> {{ t('imageManager.hostFiles') }}
        </h3>
        <Button variant="outline" size="sm" @click="scanHostFiles" :loading="scanning">
          <Search class="h-3.5 w-3.5" /> {{ t('imageManager.scan') }}
        </Button>
      </div>
      <div v-if="scanning && !hostFiles.length" class="p-6 text-center">
        <Loader2 class="h-5 w-5 animate-spin mx-auto text-muted-foreground" />
      </div>
      <div v-else-if="!hostFiles.length" class="p-8 text-center text-sm text-muted-foreground">
        {{ t('imageManager.scanTip') }}
      </div>
      <table v-else class="w-full text-sm">
        <thead>
          <tr class="border-b text-muted-foreground">
            <th class="text-left p-3 font-medium">{{ t('imageManager.thFileName') }}</th>
            <th class="text-left p-3 font-medium">{{ t('imageManager.thPath') }}</th>
            <th class="text-left p-3 font-medium">{{ t('imageManager.thSize') }}</th>
            <th class="text-left p-3 font-medium">{{ t('imageManager.thStatus') }}</th>
            <th class="text-right p-3 font-medium pr-4">{{ t('imageManager.thActions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="file in hostFiles" :key="file.path" class="border-b last:border-0 hover:bg-muted/30">
            <td class="p-3 font-medium">{{ file.name }}</td>
            <td class="p-3 font-mono text-xs text-muted-foreground">{{ file.path }}</td>
            <td class="p-3 text-xs">{{ file.size }}</td>
            <td class="p-3">
              <span v-if="isRegistered(file.path)" class="text-xs px-2 py-0.5 rounded bg-green-500/10 text-green-600 flex items-center gap-1 w-fit">
                <FileCheck class="h-3 w-3" /> {{ t('imageManager.registered') }}
              </span>
              <span v-else class="text-xs px-2 py-0.5 rounded bg-muted text-muted-foreground">{{ t('imageManager.unregistered') }}</span>
            </td>
            <td class="p-3 text-right pr-4">
              <div class="flex items-center justify-end gap-1">
                <Button v-if="!isRegistered(file.path)" variant="ghost" size="icon" @click="registerFile(file)" :title="t('imageManager.registerAsImage')">
                  <Plus class="h-3.5 w-3.5 text-primary" />
                </Button>
                <Button variant="ghost" size="icon" @click="deleteHostFile(file)" :title="t('imageManager.deleteFile')">
                  <Trash2 class="h-3.5 w-3.5 text-destructive" />
                </Button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </Card>

    <!-- 导入弹窗 -->
    <ImageImportDialog
      :open="showImport"
      :hostId="props.hostId"
      @update:open="showImport = $event"
      @imported="onImported"
    />
  </div>
</template>
