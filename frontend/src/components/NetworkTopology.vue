<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { VMList, VMGet, NetworkList, BridgeList } from '../../wailsjs/go/main/App'
import { Loader2, Server, Network, Wifi, Monitor } from 'lucide-vue-next'

const { t } = useI18n()
const props = defineProps<{ hostId: string; visible: boolean }>()

const loading = ref(false)
const vms = ref<any[]>([])
const networks = ref<any[]>([])
const bridges = ref<string[]>([])

// 构建拓扑数据
const topology = computed(() => {
  // 网桥节点
  const bridgeNodes = bridges.value.map(br => ({
    type: 'bridge' as const,
    name: br,
    children: [] as { name: string; mac: string }[],
  }))
  // 虚拟网络节点
  const netNodes = networks.value.map(n => ({
    type: 'network' as const,
    name: n.name,
    state: n.state,
    bridge: n.bridge || '',
    children: [] as { name: string; mac: string }[],
  }))
  // 遍历 VM NIC，挂到对应节点
  for (const v of vms.value) {
    if (!v.nics) continue
    for (const nic of (v as any).nics || []) {
      const vmInfo = { name: v.name, mac: nic.mac || '' }
      if (nic.bridge) {
        const node = bridgeNodes.find(b => b.name === nic.bridge)
        if (node) node.children.push(vmInfo)
        else bridgeNodes.push({ type: 'bridge', name: nic.bridge, children: [vmInfo] })
      } else if (nic.network) {
        const node = netNodes.find(n => n.name === nic.network)
        if (node) node.children.push(vmInfo)
      }
    }
  }
  return { bridges: bridgeNodes, networks: netNodes }
})

async function load() {
  loading.value = true
  try {
    const [vmList, netList, brList] = await Promise.all([
      VMList(props.hostId).catch(() => []),
      NetworkList(props.hostId).catch(() => []),
      BridgeList(props.hostId).catch(() => []),
    ])
    // 获取每个 VM 的详情（含 NIC 信息）
    const details = await Promise.all(
      (vmList || []).map(v => VMGet(props.hostId, v.name).catch(() => null))
    )
    vms.value = details.filter(Boolean)
    networks.value = netList || []
    bridges.value = brList || []
  } finally { loading.value = false }
}

watch(() => props.visible, (v) => { if (v) load() }, { immediate: true })
</script>

<template>
  <div class="p-4">
    <div v-if="loading" class="text-center py-8">
      <Loader2 class="h-6 w-6 animate-spin mx-auto text-muted-foreground" />
    </div>

    <div v-else class="space-y-6">
      <!-- 宿主机节点 -->
      <div class="flex items-center gap-2 mb-4 p-3 bg-muted/50 rounded-lg">
        <Server class="h-5 w-5 text-primary" />
        <span class="font-semibold text-sm">{{ t('topology.host') }}</span>
        <span class="text-xs text-muted-foreground ml-2">
          {{ bridges.length }} {{ t('topology.bridge') }} / {{ networks.length }} {{ t('topology.network') }} / {{ vms.length }} {{ t('topology.vm') }}
        </span>
      </div>

      <!-- 网桥拓扑 -->
      <div v-for="br in topology.bridges" :key="'br-' + br.name" class="ml-4">
        <div class="flex items-start">
          <!-- 连接线 -->
          <div class="flex flex-col items-center mr-3">
            <div class="w-px h-4 bg-border" />
            <div class="h-8 w-8 rounded-lg bg-blue-500/10 flex items-center justify-center border border-blue-500/30">
              <Wifi class="h-4 w-4 text-blue-500" />
            </div>
            <div v-if="br.children.length > 0" class="w-px flex-1 bg-border min-h-[20px]" />
          </div>
          <div class="pt-4">
            <p class="text-sm font-medium">{{ br.name }}</p>
            <p class="text-xs text-muted-foreground">{{ t('topology.bridge') }}</p>
          </div>
        </div>
        <!-- VM 子节点 -->
        <div v-for="(vm, i) in br.children" :key="vm.name" class="ml-12 flex items-start">
          <div class="flex flex-col items-center mr-3">
            <div class="w-px h-3 bg-border" />
            <div class="h-6 w-6 rounded bg-green-500/10 flex items-center justify-center border border-green-500/30">
              <Monitor class="h-3 w-3 text-green-500" />
            </div>
            <div v-if="i < br.children.length - 1" class="w-px h-3 bg-border" />
          </div>
          <div class="pt-3">
            <p class="text-xs font-medium">{{ vm.name }}</p>
            <p v-if="vm.mac" class="text-[10px] text-muted-foreground font-mono">{{ vm.mac }}</p>
          </div>
        </div>
      </div>

      <!-- 虚拟网络拓扑 -->
      <div v-for="net in topology.networks" :key="'net-' + net.name" class="ml-4">
        <div class="flex items-start">
          <div class="flex flex-col items-center mr-3">
            <div class="w-px h-4 bg-border" />
            <div class="h-8 w-8 rounded-lg flex items-center justify-center border"
              :class="net.state === 'active' ? 'bg-purple-500/10 border-purple-500/30' : 'bg-muted border-muted-foreground/20'"
            >
              <Network class="h-4 w-4" :class="net.state === 'active' ? 'text-purple-500' : 'text-muted-foreground'" />
            </div>
            <div v-if="net.children.length > 0" class="w-px flex-1 bg-border min-h-[20px]" />
          </div>
          <div class="pt-4">
            <p class="text-sm font-medium">{{ net.name }}</p>
            <div class="flex items-center gap-2">
              <p class="text-xs text-muted-foreground">{{ t('topology.network') }}</p>
              <span
                class="text-[10px] px-1.5 py-0.5 rounded"
                :class="net.state === 'active' ? 'bg-green-500/10 text-green-600' : 'bg-muted text-muted-foreground'"
              >{{ net.state }}</span>
              <span v-if="net.bridge" class="text-[10px] text-muted-foreground">
                ({{ t('topology.bridgeTo') }}: {{ net.bridge }})
              </span>
            </div>
          </div>
        </div>
        <!-- VM 子节点 -->
        <div v-for="(vm, i) in net.children" :key="vm.name" class="ml-12 flex items-start">
          <div class="flex flex-col items-center mr-3">
            <div class="w-px h-3 bg-border" />
            <div class="h-6 w-6 rounded bg-green-500/10 flex items-center justify-center border border-green-500/30">
              <Monitor class="h-3 w-3 text-green-500" />
            </div>
            <div v-if="i < net.children.length - 1" class="w-px h-3 bg-border" />
          </div>
          <div class="pt-3">
            <p class="text-xs font-medium">{{ vm.name }}</p>
            <p v-if="vm.mac" class="text-[10px] text-muted-foreground font-mono">{{ vm.mac }}</p>
          </div>
        </div>
      </div>

      <!-- 空状态 -->
      <div v-if="topology.bridges.length === 0 && topology.networks.length === 0" class="text-center py-8 text-muted-foreground">
        <Network class="h-8 w-8 mx-auto mb-2 opacity-50" />
        <p class="text-sm">{{ t('topology.noNetwork') }}</p>
      </div>
    </div>
  </div>
</template>
