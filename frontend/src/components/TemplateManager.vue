<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'
import {
  FlavorList, FlavorAdd, FlavorUpdate, FlavorDelete,
  SettingGet, SettingSet,
} from '@/api/backend'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import {
  Cpu, Plus, Pencil, Trash2, Save, Loader2,
} from 'lucide-vue-next'

const { t } = useI18n()
const toast = useToast()
const { request: confirmRequest } = useConfirm()

// === Flavor ===
const flavors = ref<any[]>([])
const loadingFlavors = ref(false)
const showFlavorForm = ref(false)
const editingFlavor = ref<any>(null)
const flavorForm = ref({ name: '', cpus: 1, memoryMB: 1024, diskGB: 20 })

async function loadFlavors() {
  loadingFlavors.value = true
  try {
    flavors.value = (await FlavorList()) || []
  } catch (e: any) {
    toast.error(t('templateManager.loadFailed') + ': ' + e.toString())
  } finally {
    loadingFlavors.value = false
  }
}

function openFlavorForm(f?: any) {
  if (f) {
    editingFlavor.value = f
    flavorForm.value = { name: f.name, cpus: f.cpus, memoryMB: f.memoryMB, diskGB: f.diskGB }
  } else {
    editingFlavor.value = null
    flavorForm.value = { name: '', cpus: 1, memoryMB: 1024, diskGB: 20 }
  }
  showFlavorForm.value = true
}

async function saveFlavor() {
  const form = flavorForm.value
  if (!form.name.trim()) { toast.error(t('templateManager.enterName')); return }
  try {
    if (editingFlavor.value) {
      await FlavorUpdate({ ...editingFlavor.value, ...form })
      toast.success(t('templateManager.flavorUpdated'))
    } else {
      await FlavorAdd({ id: '', name: form.name, cpus: form.cpus, memoryMB: form.memoryMB, diskGB: form.diskGB, sortOrder: flavors.value.length, createdAt: '' })
      toast.success(t('templateManager.flavorAdded'))
    }
    showFlavorForm.value = false
    await loadFlavors()
  } catch (e: any) {
    toast.error(t('common.saveFailed') + ': ' + e.toString())
  }
}

async function deleteFlavor(id: string, name: string) {
  const ok = await confirmRequest(t('templateManager.deleteFlavor'), t('templateManager.deleteFlavorConfirm', { name }))
  if (!ok) return
  try {
    await FlavorDelete(id)
    toast.success(t('templateManager.flavorDeleted'))
    await loadFlavors()
  } catch (e: any) {
    toast.error(t('common.deleteFailed') + ': ' + e.toString())
  }
}

// === Instance Root ===
const instanceRoot = ref('/var/lib/libvirt/instances')
const savingRoot = ref(false)

async function loadInstanceRoot() {
  try {
    const val = await SettingGet('instance_root').catch(() => '')
    if (val) instanceRoot.value = val
  } catch { /* 静默 */ }
}

async function saveInstanceRoot() {
  savingRoot.value = true
  try {
    await SettingSet('instance_root', instanceRoot.value)
    toast.success(t('templateManager.instanceRootSaved'))
  } catch (e: any) {
    toast.error(t('common.saveFailed') + ': ' + e.toString())
  } finally {
    savingRoot.value = false
  }
}

function formatMem(mb: number) {
  return mb >= 1024 ? `${(mb / 1024).toFixed(mb % 1024 === 0 ? 0 : 1)} GB` : `${mb} MB`
}

onMounted(() => {
  loadFlavors()
  loadInstanceRoot()
})
</script>

<template>
  <div class="space-y-6">
    <!-- Instance Root 配置 -->
    <Card>
      <div class="p-4 border-b">
        <h3 class="font-semibold">{{ t('templateManager.instanceStorage') }}</h3>
      </div>
      <div class="p-4">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm font-medium">{{ t('templateManager.instanceRoot') }}</p>
            <p class="text-xs text-muted-foreground">{{ t('templateManager.instanceRootTip') }}</p>
          </div>
          <div class="flex items-center gap-2">
            <Input v-model="instanceRoot" class="w-64 h-8 text-sm" placeholder="/var/lib/libvirt/instances" />
            <Button variant="outline" size="sm" :loading="savingRoot" @click="saveInstanceRoot">
              <Save class="h-3.5 w-3.5" />
            </Button>
          </div>
        </div>
      </div>
    </Card>

    <!-- 硬件规格 -->
    <Card>
      <div class="p-4 border-b flex items-center justify-between">
        <h3 class="font-semibold flex items-center gap-2"><Cpu class="h-4 w-4" /> {{ t('templateManager.flavorTitle') }}</h3>
        <Button variant="outline" size="sm" @click="openFlavorForm()">
          <Plus class="h-3.5 w-3.5" /> {{ t('common.add') }}
        </Button>
      </div>
      <div v-if="loadingFlavors && !flavors.length" class="p-6 text-center">
        <Loader2 class="h-5 w-5 animate-spin mx-auto text-muted-foreground" />
      </div>
      <div v-else-if="!flavors.length" class="p-6 text-center text-sm text-muted-foreground">
        {{ t('templateManager.noFlavorsTip2') }}
      </div>
      <table v-else class="w-full text-sm">
        <thead>
          <tr class="border-b text-muted-foreground">
            <th class="text-left p-3 font-medium">{{ t('templateManager.thName') }}</th>
            <th class="text-center p-3 font-medium">{{ t('templateManager.thVCPU') }}</th>
            <th class="text-center p-3 font-medium">{{ t('templateManager.thMemory') }}</th>
            <th class="text-center p-3 font-medium">{{ t('templateManager.thDisk') }}</th>
            <th class="text-right p-3 font-medium pr-4">{{ t('templateManager.thActions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="f in flavors" :key="f.id" class="border-b last:border-0 hover:bg-muted/30">
            <td class="p-3 font-medium">{{ f.name }}</td>
            <td class="p-3 text-center">{{ f.cpus }} {{ t('templateManager.cores') }}</td>
            <td class="p-3 text-center">{{ formatMem(f.memoryMB) }}</td>
            <td class="p-3 text-center">{{ f.diskGB }} GB</td>
            <td class="p-3 text-right pr-4">
              <div class="flex items-center justify-end gap-1">
                <Button variant="ghost" size="icon" @click="openFlavorForm(f)" :title="t('common.edit')">
                  <Pencil class="h-3.5 w-3.5" />
                </Button>
                <Button variant="ghost" size="icon" @click="deleteFlavor(f.id, f.name)" :title="t('common.delete')">
                  <Trash2 class="h-3.5 w-3.5 text-destructive" />
                </Button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </Card>

    <!-- Flavor 编辑表单 -->
    <Card v-if="showFlavorForm" class="border-primary">
      <div class="p-4 border-b">
        <h3 class="font-semibold">{{ editingFlavor ? t('templateManager.editFlavor') : t('templateManager.addFlavor') }}</h3>
      </div>
      <div class="p-4 space-y-3">
        <div class="grid grid-cols-4 gap-3">
          <div>
            <label class="text-xs text-muted-foreground">{{ t('templateManager.name') }}</label>
            <Input v-model="flavorForm.name" class="h-8 text-sm" placeholder="small" />
          </div>
          <div>
            <label class="text-xs text-muted-foreground">{{ t('templateManager.thVCPU') }}</label>
            <Input v-model.number="flavorForm.cpus" type="number" class="h-8 text-sm" />
          </div>
          <div>
            <label class="text-xs text-muted-foreground">{{ t('templateManager.memory') }}</label>
            <Input v-model.number="flavorForm.memoryMB" type="number" class="h-8 text-sm" />
          </div>
          <div>
            <label class="text-xs text-muted-foreground">{{ t('templateManager.diskGB') }}</label>
            <Input v-model.number="flavorForm.diskGB" type="number" class="h-8 text-sm" />
          </div>
        </div>
        <div class="flex justify-end gap-2">
          <Button variant="outline" size="sm" @click="showFlavorForm = false">{{ t('common.cancel') }}</Button>
          <Button size="sm" @click="saveFlavor">{{ t('common.save') }}</Button>
        </div>
      </div>
    </Card>

  </div>
</template>
