<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'
import { NetworkList, BridgeList, NetworkStart, NetworkStop, NetworkAutostart, NATRuleList, NATRuleAdd, NATRuleDelete } from '../../wailsjs/go/main/App'
import Input from '@/components/ui/Input.vue'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import NetworkTopology from '@/components/NetworkTopology.vue'
import {
  Network, Globe, RotateCw, Loader2, Wifi,
  Play, Square, ToggleLeft, ToggleRight, GitBranch, List,
  ArrowRightLeft, Plus, Trash2,
} from 'lucide-vue-next'

const { t } = useI18n()
const props = defineProps<{ hostId: string; visible: boolean }>()
const toast = useToast()
const { request: confirmRequest } = useConfirm()
const netActionLoading = ref('')
const viewMode = ref<'list' | 'topology'>('list')

const networks = ref<any[]>([])
const bridges = ref<string[]>([])
const loadingNets = ref(false)
const loadingBridges = ref(false)

// NAT 端口转发
const natRules = ref<any[]>([])
const loadingNAT = ref(false)
const showNATForm = ref(false)
const natForm = ref({ proto: 'tcp', hostPort: '', vmIP: '', vmPort: '', comment: '' })
const natAdding = ref(false)

async function loadNetworks() {
  loadingNets.value = true
  try {
    const list = await NetworkList(props.hostId)
    networks.value = list || []
  } catch (e: any) {
    toast.error(t('networkPanel.loadNetFailed') + ': ' + e.toString())
  } finally {
    loadingNets.value = false
  }
}

async function loadBridges() {
  loadingBridges.value = true
  try {
    const list = await BridgeList(props.hostId)
    bridges.value = list || []
  } catch (e: any) {
    toast.error(t('networkPanel.loadBridgeFailed') + ': ' + e.toString())
  } finally {
    loadingBridges.value = false
  }
}

async function loadNATRules() {
  loadingNAT.value = true
  try {
    natRules.value = (await NATRuleList(props.hostId)) || []
  } catch {
    natRules.value = []
  } finally {
    loadingNAT.value = false
  }
}

async function addNATRule() {
  const f = natForm.value
  if (!f.hostPort || !f.vmIP || !f.vmPort) { toast.error(t('networkPanel.fillComplete')); return }
  natAdding.value = true
  try {
    await NATRuleAdd(props.hostId, f.proto, f.hostPort, f.vmIP, f.vmPort, f.comment)
    toast.success(t('networkPanel.ruleAdded'))
    natForm.value = { proto: 'tcp', hostPort: '', vmIP: '', vmPort: '', comment: '' }
    showNATForm.value = false
    await loadNATRules()
  } catch (e: any) {
    toast.error(t('networkPanel.addFailed') + ': ' + e.toString())
  } finally {
    natAdding.value = false
  }
}

async function deleteNATRule(rule: any) {
  const ok = await confirmRequest(t('networkPanel.deleteRule'), t('networkPanel.deleteRuleConfirm', { proto: rule.proto, hostPort: rule.hostPort, vmIP: rule.vmIP, vmPort: rule.vmPort }))
  if (!ok) return
  try {
    await NATRuleDelete(props.hostId, rule.proto, rule.hostPort, rule.vmIP, rule.vmPort)
    toast.success(t('networkPanel.ruleDeleted'))
    await loadNATRules()
  } catch (e: any) {
    toast.error(t('common.deleteFailed') + ': ' + e.toString())
  }
}

async function loadAll() {
  await Promise.all([loadNetworks(), loadBridges(), loadNATRules()])
}

async function startNet(name: string) {
  netActionLoading.value = `start-${name}`
  try {
    await NetworkStart(props.hostId, name)
    toast.success(t('networkPanel.netStarted', { name }))
    await loadNetworks()
  } catch (e: any) { toast.error(t('common.startFailed') + ': ' + e.toString()) }
  finally { netActionLoading.value = '' }
}

async function stopNet(name: string) {
  const ok = await confirmRequest(t('networkPanel.stopNetwork'), t('networkPanel.stopNetConfirm', { name }))
  if (!ok) return
  netActionLoading.value = `stop-${name}`
  try {
    await NetworkStop(props.hostId, name)
    toast.success(t('networkPanel.netStopped', { name }))
    await loadNetworks()
  } catch (e: any) { toast.error(t('common.stopFailed') + ': ' + e.toString()) }
  finally { netActionLoading.value = '' }
}

async function toggleNetAutostart(name: string, current: string) {
  const enabled = current !== 'yes'
  netActionLoading.value = `auto-${name}`
  try {
    await NetworkAutostart(props.hostId, name, enabled)
    toast.success(t('networkPanel.netAutostart', { name, state: enabled ? t('storage.autostartOn') : t('storage.autostartOff') }))
    await loadNetworks()
  } catch (e: any) { toast.error(t('common.setFailed') + ': ' + e.toString()) }
  finally { netActionLoading.value = '' }
}

watch(() => props.visible, (v) => {
  if (v) loadAll()
})
</script>

<template>
  <div>
    <!-- 标题栏 -->
    <div class="flex items-center justify-between mb-4">
      <h2 class="text-lg font-semibold flex items-center gap-2">
        <Globe class="h-5 w-5" /> {{ t('networkPanel.title') }}
      </h2>
      <div class="flex items-center gap-2">
        <div class="flex border rounded-md overflow-hidden">
          <button
            class="px-2 py-1 text-xs transition-colors"
            :class="viewMode === 'list' ? 'bg-primary text-primary-foreground' : 'hover:bg-accent'"
            @click="viewMode = 'list'"
          ><List class="h-3.5 w-3.5" /></button>
          <button
            class="px-2 py-1 text-xs transition-colors"
            :class="viewMode === 'topology' ? 'bg-primary text-primary-foreground' : 'hover:bg-accent'"
            @click="viewMode = 'topology'"
          ><GitBranch class="h-3.5 w-3.5" /></button>
        </div>
        <Button variant="outline" size="sm" @click="loadAll" :loading="loadingNets || loadingBridges">
          <RotateCw class="h-3.5 w-3.5" /> {{ t('common.refresh') }}
        </Button>
      </div>
    </div>

    <!-- 拓扑视图 -->
    <NetworkTopology v-if="viewMode === 'topology'" :hostId="hostId" :visible="viewMode === 'topology'" />

    <!-- 加载中 -->
    <div v-else-if="loadingNets && networks.length === 0 && bridges.length === 0" class="text-center py-8">
      <Loader2 class="h-6 w-6 animate-spin mx-auto text-muted-foreground" />
    </div>

    <div v-else class="space-y-6">
      <!-- 虚拟网络 -->
      <section>
        <h3 class="text-sm font-medium text-muted-foreground mb-3 flex items-center gap-2">
          <Network class="h-4 w-4" /> {{ t('networkPanel.virtualNetwork') }}
        </h3>
        <div v-if="networks.length === 0" class="text-sm text-muted-foreground py-4 text-center">
          {{ t('networkPanel.noNetworks') }}
        </div>
        <div v-else class="space-y-2">
          <Card v-for="net in networks" :key="net.name" class="p-4">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-3">
                <Network class="h-4 w-4 text-muted-foreground" />
                <div>
                  <p class="text-sm font-medium">{{ net.name }}</p>
                  <p class="text-xs text-muted-foreground">
                    {{ t('networkPanel.bridgeLabel') }}: {{ net.bridge || '-' }}
                  </p>
                </div>
              </div>
              <div class="flex items-center gap-1">
                <Button
                  v-if="net.state !== 'active'"
                  variant="ghost" size="icon" :title="t('common.start')"
                  :loading="netActionLoading === `start-${net.name}`"
                  @click="startNet(net.name)"
                >
                  <Play class="h-3.5 w-3.5 text-green-500" />
                </Button>
                <Button
                  v-if="net.state === 'active'"
                  variant="ghost" size="icon" :title="t('common.stop')"
                  :loading="netActionLoading === `stop-${net.name}`"
                  @click="stopNet(net.name)"
                >
                  <Square class="h-3.5 w-3.5 text-muted-foreground" />
                </Button>
                <Button
                  variant="ghost" size="icon"
                  :title="net.autostart === 'yes' ? t('networkPanel.disableAutostart') : t('networkPanel.enableAutostart')"
                  :loading="netActionLoading === `auto-${net.name}`"
                  @click="toggleNetAutostart(net.name, net.autostart)"
                >
                  <ToggleRight v-if="net.autostart === 'yes'" class="h-3.5 w-3.5 text-blue-500" />
                  <ToggleLeft v-else class="h-3.5 w-3.5 text-muted-foreground" />
                </Button>
                <span
                  class="px-2 py-0.5 rounded text-xs ml-1"
                  :class="net.state === 'active' ? 'bg-green-500/10 text-green-600' : 'bg-muted text-muted-foreground'"
                >
                  {{ net.state === 'active' ? t('networkPanel.active') : net.state }}
                </span>
              </div>
            </div>
          </Card>
        </div>
      </section>

      <!-- 网桥 -->
      <section>
        <h3 class="text-sm font-medium text-muted-foreground mb-3 flex items-center gap-2">
          <Wifi class="h-4 w-4" /> {{ t('networkPanel.systemBridge') }}
        </h3>
        <div v-if="bridges.length === 0" class="text-sm text-muted-foreground py-4 text-center">
          {{ t('networkPanel.noBridges') }}
        </div>
        <div v-else class="flex flex-wrap gap-2">
          <Card v-for="br in bridges" :key="br" class="px-4 py-3 flex items-center gap-2">
            <Wifi class="h-3.5 w-3.5 text-muted-foreground" />
            <span class="text-sm font-mono">{{ br }}</span>
          </Card>
        </div>
      </section>
    </div>

    <!-- NAT 端口转发 -->
    <div v-if="viewMode === 'list'" class="mt-6">
      <section>
        <div class="flex items-center justify-between mb-3">
          <h3 class="text-sm font-medium text-muted-foreground flex items-center gap-2">
            <ArrowRightLeft class="h-4 w-4" /> {{ t('networkPanel.natPortForward') }}
          </h3>
          <div class="flex gap-2">
            <Button variant="outline" size="sm" @click="loadNATRules" :loading="loadingNAT">
              <RotateCw class="h-3.5 w-3.5" /> {{ t('common.refresh') }}
            </Button>
            <Button variant="outline" size="sm" @click="showNATForm = !showNATForm">
              <Plus class="h-3.5 w-3.5" /> {{ t('networkPanel.addRule') }}
            </Button>
          </div>
        </div>

        <!-- 添加表单 -->
        <Card v-if="showNATForm" class="mb-3 border-primary">
          <div class="p-4 space-y-3">
            <div class="grid grid-cols-5 gap-2">
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
                <label class="text-xs text-muted-foreground">{{ t('networkPanel.vmIP') }}</label>
                <Input v-model="natForm.vmIP" class="h-8 text-sm font-mono" placeholder="192.168.122.100" />
              </div>
              <div>
                <label class="text-xs text-muted-foreground">{{ t('networkPanel.vmPort') }}</label>
                <Input v-model="natForm.vmPort" class="h-8 text-sm" placeholder="22" />
              </div>
              <div>
                <label class="text-xs text-muted-foreground">{{ t('networkPanel.remark') }}</label>
                <Input v-model="natForm.comment" class="h-8 text-sm" placeholder="ssh" />
              </div>
            </div>
            <div class="flex justify-end gap-2">
              <Button variant="outline" size="sm" @click="showNATForm = false">{{ t('common.cancel') }}</Button>
              <Button size="sm" :loading="natAdding" @click="addNATRule">{{ t('common.add') }}</Button>
            </div>
          </div>
        </Card>

        <!-- 规则列表 -->
        <div v-if="loadingNAT && !natRules.length" class="text-center py-4">
          <Loader2 class="h-4 w-4 animate-spin mx-auto text-muted-foreground" />
        </div>
        <div v-else-if="!natRules.length" class="text-sm text-muted-foreground py-4 text-center">
          {{ t('networkPanel.noNATRules') }}
        </div>
        <div v-else class="space-y-2">
          <Card v-for="(rule, i) in natRules" :key="i" class="p-3">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-3 text-sm">
                <span class="px-2 py-0.5 rounded text-xs bg-blue-500/10 text-blue-600 font-mono uppercase">{{ rule.proto }}</span>
                <span class="font-mono">{{ t('networkPanel.hostPort') }}:{{ rule.hostPort }}</span>
                <ArrowRightLeft class="h-3.5 w-3.5 text-muted-foreground" />
                <span class="font-mono">{{ rule.vmIP }}:{{ rule.vmPort }}</span>
                <span v-if="rule.comment" class="text-xs text-muted-foreground">({{ rule.comment }})</span>
              </div>
              <Button variant="ghost" size="icon" @click="deleteNATRule(rule)" :title="t('networkPanel.deleteRuleBtn')">
                <Trash2 class="h-3.5 w-3.5 text-destructive" />
              </Button>
            </div>
          </Card>
        </div>
      </section>
    </div>
  </div>
</template>
