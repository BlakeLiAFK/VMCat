<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@/composables/useToast'
import { VMSetVCPUs, VMSetMemory, VMSetAutostart, VMRename, NATRuleList, NATRuleAdd, NATRuleDelete } from '@/api/backend'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import { X, ArrowRightLeft, Plus, Trash2 } from 'lucide-vue-next'

const { t } = useI18n()
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

// NAT 端口转发
const natRules = ref<any[]>([])
const vmIPs = ref<string[]>([])
const showNATForm = ref(false)
const natForm = ref({ proto: 'tcp', hostPort: '', vmPort: '', comment: '' })
const natAdding = ref(false)

watch(() => props.open, (v) => {
  if (v && props.detail) {
    cpus.value = props.detail.cpus || 1
    memoryMB.value = props.detail.memoryMB || 1024
    autostart.value = props.detail.autostart || false
    newName.value = props.detail.name || ''
    // 提取 VM IP
    const ips: string[] = []
    for (const nic of props.detail.nics || []) {
      if (nic.ip) ips.push(nic.ip)
    }
    vmIPs.value = ips
    loadNATRules()
  }
})

async function loadNATRules() {
  try {
    const all = (await NATRuleList(props.hostId)) || []
    // 过滤出与本 VM IP 相关的规则
    natRules.value = vmIPs.value.length > 0
      ? all.filter((r: any) => vmIPs.value.includes(r.vmIP))
      : []
  } catch {
    natRules.value = []
  }
}

async function addNATRule() {
  if (vmIPs.value.length === 0) { toast.error(t('vmEdit.noVMIP')); return }
  const f = natForm.value
  if (!f.hostPort || !f.vmPort) { toast.error(t('vmEdit.fillPorts')); return }
  natAdding.value = true
  try {
    await NATRuleAdd(props.hostId, f.proto, f.hostPort, vmIPs.value[0], f.vmPort, f.comment || props.vmName)
    toast.success(t('vmEdit.ruleAdded'))
    natForm.value = { proto: 'tcp', hostPort: '', vmPort: '', comment: '' }
    showNATForm.value = false
    await loadNATRules()
  } catch (e: any) {
    toast.error(t('networkPanel.addFailed') + ': ' + e.toString())
  } finally { natAdding.value = false }
}

async function deleteNATRule(rule: any) {
  try {
    await NATRuleDelete(props.hostId, rule.proto, rule.hostPort, rule.vmIP, rule.vmPort)
    toast.success(t('networkPanel.ruleDeleted'))
    await loadNATRules()
  } catch (e: any) {
    toast.error(t('vmEdit.deleteFailed') + ': ' + e.toString())
  }
}

async function saveCPU() {
  saving.value = 'cpu'
  try {
    await VMSetVCPUs(props.hostId, props.vmName, Number(cpus.value))
    toast.success(t('vmEdit.cpuSet', { count: cpus.value }))
    emit('saved')
  } catch (e: any) {
    toast.error(t('vmEdit.cpuSetFailed') + ': ' + e.toString())
  } finally { saving.value = '' }
}

async function saveMemory() {
  saving.value = 'mem'
  try {
    await VMSetMemory(props.hostId, props.vmName, Number(memoryMB.value))
    toast.success(t('vmEdit.memorySet', { size: memoryMB.value }))
    emit('saved')
  } catch (e: any) {
    toast.error(t('vmEdit.memorySetFailed') + ': ' + e.toString())
  } finally { saving.value = '' }
}

async function saveAutostart() {
  saving.value = 'auto'
  try {
    await VMSetAutostart(props.hostId, props.vmName, autostart.value)
    toast.success(autostart.value ? t('vmEdit.autostartOn') : t('vmEdit.autostartOff'))
    emit('saved')
  } catch (e: any) {
    toast.error(t('common.setFailed') + ': ' + e.toString())
  } finally { saving.value = '' }
}

async function saveRename() {
  const name = newName.value.trim()
  if (!name || name === props.vmName) return
  saving.value = 'rename'
  try {
    await VMRename(props.hostId, props.vmName, name)
    toast.success(t('vmEdit.renamed', { name }))
    emit('saved', name)
  } catch (e: any) {
    toast.error(t('vmEdit.renameFailed') + ': ' + e.toString())
  } finally { saving.value = '' }
}

function close() { emit('update:open', false) }
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center">
      <div class="absolute inset-0 bg-black/50" @click="close" />
      <div class="relative bg-card border rounded-xl shadow-2xl w-[560px] max-h-[85vh] overflow-y-auto">
        <div class="flex items-center justify-between p-5 border-b">
          <h2 class="text-lg font-semibold">{{ t('vmEdit.title') }}</h2>
          <button @click="close" class="p-1 rounded hover:bg-accent"><X class="h-4 w-4" /></button>
        </div>

        <div class="p-5 space-y-5">
          <!-- 重命名 -->
          <div>
            <label class="text-sm font-medium mb-2 block">{{ t('vmEdit.nameLabel') }}</label>
            <div class="flex gap-2">
              <Input v-model="newName" class="flex-1" />
              <Button size="sm" :loading="saving === 'rename'" @click="saveRename"
                :disabled="!newName.trim() || newName.trim() === vmName">{{ t('common.save') }}</Button>
            </div>
          </div>

          <!-- CPU -->
          <div>
            <label class="text-sm font-medium mb-2 block">{{ t('vmEdit.cpuLabel') }}</label>
            <div class="flex gap-2">
              <Input v-model="cpus" type="number" class="flex-1" />
              <Button size="sm" :loading="saving === 'cpu'" @click="saveCPU">{{ t('common.save') }}</Button>
            </div>
          </div>

          <!-- 内存 -->
          <div>
            <label class="text-sm font-medium mb-2 block">{{ t('vmEdit.memoryLabel') }}</label>
            <div class="flex gap-2">
              <Input v-model="memoryMB" type="number" class="flex-1" />
              <Button size="sm" :loading="saving === 'mem'" @click="saveMemory">{{ t('common.save') }}</Button>
            </div>
            <p class="text-xs text-muted-foreground mt-1">
              {{ t('vmEdit.memoryPresets') }}
            </p>
          </div>

          <!-- Autostart -->
          <div>
            <label class="text-sm font-medium mb-2 block">{{ t('vmEdit.autostartLabel') }}</label>
            <div class="flex items-center gap-3">
              <label class="flex items-center gap-2 text-sm cursor-pointer">
                <input type="checkbox" v-model="autostart" class="rounded" />
                {{ t('vmEdit.autostartTip') }}
              </label>
              <Button size="sm" variant="outline" :loading="saving === 'auto'" @click="saveAutostart">{{ t('common.save') }}</Button>
            </div>
          </div>

          <!-- NAT 端口转发 -->
          <div class="border-t pt-5">
            <div class="flex items-center justify-between mb-3">
              <label class="text-sm font-medium flex items-center gap-1.5">
                <ArrowRightLeft class="h-3.5 w-3.5" /> {{ t('vm.natPortForward') }}
              </label>
              <Button variant="outline" size="sm" @click="showNATForm = !showNATForm">
                <Plus class="h-3 w-3" /> {{ t('common.add') }}
              </Button>
            </div>
            <p v-if="vmIPs.length > 0" class="text-xs text-muted-foreground mb-2">
              {{ t('vmEdit.vmIPLabel') }}: <span class="font-mono">{{ vmIPs.join(', ') }}</span>
            </p>
            <p v-else class="text-xs text-muted-foreground mb-2">
              {{ t('vmEdit.vmNoIP') }}
            </p>

            <!-- 添加表单 -->
            <div v-if="showNATForm" class="p-3 rounded-lg border border-primary/50 mb-3 space-y-2">
              <div class="grid grid-cols-4 gap-2">
                <div>
                  <label class="text-xs text-muted-foreground">{{ t('networkPanel.protocol') }}</label>
                  <select v-model="natForm.proto" class="h-8 w-full rounded-md border bg-background px-2 text-sm">
                    <option value="tcp">TCP</option>
                    <option value="udp">UDP</option>
                  </select>
                </div>
                <div>
                  <label class="text-xs text-muted-foreground">{{ t('networkPanel.hostPort') }}</label>
                  <Input v-model="natForm.hostPort" class="h-8 text-sm" placeholder="8080" />
                </div>
                <div>
                  <label class="text-xs text-muted-foreground">{{ t('networkPanel.vmPort') }}</label>
                  <Input v-model="natForm.vmPort" class="h-8 text-sm" placeholder="80" />
                </div>
                <div>
                  <label class="text-xs text-muted-foreground">{{ t('networkPanel.remark') }}</label>
                  <Input v-model="natForm.comment" class="h-8 text-sm" :placeholder="vmName" />
                </div>
              </div>
              <div class="flex justify-end gap-2">
                <Button variant="outline" size="sm" @click="showNATForm = false">{{ t('common.cancel') }}</Button>
                <Button size="sm" :loading="natAdding" @click="addNATRule">{{ t('common.add') }}</Button>
              </div>
            </div>

            <!-- 规则列表 -->
            <div v-if="natRules.length" class="space-y-1.5">
              <div v-for="(rule, i) in natRules" :key="i"
                class="flex items-center justify-between p-2 rounded border text-sm">
                <div class="flex items-center gap-2">
                  <span class="px-1.5 py-0.5 rounded text-xs bg-blue-500/10 text-blue-600 font-mono uppercase">{{ rule.proto }}</span>
                  <span class="font-mono text-xs">{{ t('vmEdit.hostPortLabel', { port: rule.hostPort }) }}</span>
                  <ArrowRightLeft class="h-3 w-3 text-muted-foreground" />
                  <span class="font-mono text-xs">{{ rule.vmIP }}:{{ rule.vmPort }}</span>
                  <span v-if="rule.comment" class="text-xs text-muted-foreground">({{ rule.comment }})</span>
                </div>
                <button class="p-1 rounded hover:bg-accent" @click="deleteNATRule(rule)" :title="t('common.delete')">
                  <Trash2 class="h-3 w-3 text-destructive" />
                </button>
              </div>
            </div>
            <p v-else class="text-xs text-muted-foreground text-center py-2">{{ t('vmEdit.noRules') }}</p>
          </div>
        </div>

        <div class="flex justify-end p-5 border-t">
          <Button variant="outline" @click="close">{{ t('common.close') }}</Button>
        </div>
      </div>
    </div>
  </Teleport>
</template>
