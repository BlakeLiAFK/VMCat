<script setup lang="ts">
import { ref, watch } from 'vue'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'
import { NetworkList, BridgeList, NetworkStart, NetworkStop, NetworkAutostart } from '../../wailsjs/go/main/App'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import {
  Network, Globe, RotateCw, Loader2, Wifi,
  Play, Square, ToggleLeft, ToggleRight,
} from 'lucide-vue-next'

const props = defineProps<{ hostId: string; visible: boolean }>()
const toast = useToast()
const { request: confirmRequest } = useConfirm()
const netActionLoading = ref('')

const networks = ref<any[]>([])
const bridges = ref<string[]>([])
const loadingNets = ref(false)
const loadingBridges = ref(false)

async function loadNetworks() {
  loadingNets.value = true
  try {
    const list = await NetworkList(props.hostId)
    networks.value = list || []
  } catch (e: any) {
    toast.error('加载虚拟网络失败: ' + e.toString())
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
    toast.error('加载网桥失败: ' + e.toString())
  } finally {
    loadingBridges.value = false
  }
}

async function loadAll() {
  await Promise.all([loadNetworks(), loadBridges()])
}

async function startNet(name: string) {
  netActionLoading.value = `start-${name}`
  try {
    await NetworkStart(props.hostId, name)
    toast.success(`网络 ${name} 已启动`)
    await loadNetworks()
  } catch (e: any) { toast.error('启动失败: ' + e.toString()) }
  finally { netActionLoading.value = '' }
}

async function stopNet(name: string) {
  const ok = await confirmRequest('停止网络', `确认停止网络 "${name}"? 使用此网络的 VM 将失去网络连接。`)
  if (!ok) return
  netActionLoading.value = `stop-${name}`
  try {
    await NetworkStop(props.hostId, name)
    toast.success(`网络 ${name} 已停止`)
    await loadNetworks()
  } catch (e: any) { toast.error('停止失败: ' + e.toString()) }
  finally { netActionLoading.value = '' }
}

async function toggleNetAutostart(name: string, current: string) {
  const enabled = current !== 'yes'
  netActionLoading.value = `auto-${name}`
  try {
    await NetworkAutostart(props.hostId, name, enabled)
    toast.success(`网络 ${name} 自启动已${enabled ? '开启' : '关闭'}`)
    await loadNetworks()
  } catch (e: any) { toast.error('设置失败: ' + e.toString()) }
  finally { netActionLoading.value = '' }
}

watch(() => props.visible, (v) => {
  if (v && networks.value.length === 0 && bridges.value.length === 0) loadAll()
})
</script>

<template>
  <div>
    <!-- 标题栏 -->
    <div class="flex items-center justify-between mb-4">
      <h2 class="text-lg font-semibold flex items-center gap-2">
        <Globe class="h-5 w-5" /> 网络管理
      </h2>
      <Button variant="outline" size="sm" @click="loadAll" :loading="loadingNets || loadingBridges">
        <RotateCw class="h-3.5 w-3.5" /> 刷新
      </Button>
    </div>

    <!-- 加载中 -->
    <div v-if="loadingNets && networks.length === 0 && bridges.length === 0" class="text-center py-8">
      <Loader2 class="h-6 w-6 animate-spin mx-auto text-muted-foreground" />
    </div>

    <div v-else class="space-y-6">
      <!-- 虚拟网络 -->
      <section>
        <h3 class="text-sm font-medium text-muted-foreground mb-3 flex items-center gap-2">
          <Network class="h-4 w-4" /> 虚拟网络 (libvirt)
        </h3>
        <div v-if="networks.length === 0" class="text-sm text-muted-foreground py-4 text-center">
          未发现虚拟网络
        </div>
        <div v-else class="space-y-2">
          <Card v-for="net in networks" :key="net.name" class="p-4">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-3">
                <Network class="h-4 w-4 text-muted-foreground" />
                <div>
                  <p class="text-sm font-medium">{{ net.name }}</p>
                  <p class="text-xs text-muted-foreground">
                    网桥: {{ net.bridge || '-' }}
                  </p>
                </div>
              </div>
              <div class="flex items-center gap-1">
                <Button
                  v-if="net.state !== 'active'"
                  variant="ghost" size="icon" title="启动"
                  :loading="netActionLoading === `start-${net.name}`"
                  @click="startNet(net.name)"
                >
                  <Play class="h-3.5 w-3.5 text-green-500" />
                </Button>
                <Button
                  v-if="net.state === 'active'"
                  variant="ghost" size="icon" title="停止"
                  :loading="netActionLoading === `stop-${net.name}`"
                  @click="stopNet(net.name)"
                >
                  <Square class="h-3.5 w-3.5 text-muted-foreground" />
                </Button>
                <Button
                  variant="ghost" size="icon"
                  :title="net.autostart === 'yes' ? '关闭自启动' : '开启自启动'"
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
                  {{ net.state === 'active' ? '活跃' : net.state }}
                </span>
              </div>
            </div>
          </Card>
        </div>
      </section>

      <!-- 网桥 -->
      <section>
        <h3 class="text-sm font-medium text-muted-foreground mb-3 flex items-center gap-2">
          <Wifi class="h-4 w-4" /> 系统网桥
        </h3>
        <div v-if="bridges.length === 0" class="text-sm text-muted-foreground py-4 text-center">
          未发现系统网桥
        </div>
        <div v-else class="flex flex-wrap gap-2">
          <Card v-for="br in bridges" :key="br" class="px-4 py-3 flex items-center gap-2">
            <Wifi class="h-3.5 w-3.5 text-muted-foreground" />
            <span class="text-sm font-mono">{{ br }}</span>
          </Card>
        </div>
      </section>
    </div>
  </div>
</template>
