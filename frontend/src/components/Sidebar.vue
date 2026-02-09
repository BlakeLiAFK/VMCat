<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter, useRoute } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { useTheme } from '@/composables/useTheme'
import { useSettings } from '@/composables/useSettings'
import { HostList, HostIsConnected, HostDisconnect, HostDelete, VMList, AppVersion, HostUpdate } from '@/api/backend'
import { useConnection } from '@/composables/useConnection'
import { Server, Plus, Settings, LayoutDashboard, Sun, Moon, ChevronDown, ChevronRight, GripVertical, FileText, Wifi, WifiOff } from 'lucide-vue-next'
import HostFormDialog from '@/components/HostFormDialog.vue'
import RemoteConnectDialog from '@/components/RemoteConnectDialog.vue'
import ContextMenu from '@/components/ui/ContextMenu.vue'
import type { MenuItem } from '@/components/ui/ContextMenu.vue'
import { useConfirm } from '@/composables/useConfirm'
import { useToast } from '@/composables/useToast'

const { t } = useI18n()
const router = useRouter()
const route = useRoute()
const store = useAppStore()
const { isDark, toggle: toggleTheme } = useTheme()
const { request: confirmRequest } = useConfirm()
const toast = useToast()
const { mode: connectionMode, isRemote, remoteVersion, disconnectRemote } = useConnection()
const showAddHost = ref(false)
const showEditHost = ref(false)
const showRemoteDialog = ref(false)
const editHostData = ref<any>(null)
const appVersion = ref('')
const { refreshIntervalMs } = useSettings()
const collapsedGroups = ref<Set<string>>(new Set())
const contextMenuRef = ref<InstanceType<typeof ContextMenu>>()
const dragHostId = ref<string | null>(null)

let refreshTimer: ReturnType<typeof setInterval> | null = null

// 标签分组
const hostGroups = computed(() => {
  const groups: Record<string, typeof store.hosts> = {}
  for (const h of store.hosts) {
    const tags = h.tags ? h.tags.split(',').map(s => s.trim()).filter(Boolean) : []
    if (tags.length === 0) {
      if (!groups['']) groups[''] = []
      groups[''].push(h)
    } else {
      for (const tag of tags) {
        if (!groups[tag]) groups[tag] = []
        groups[tag].push(h)
      }
    }
  }
  // 排序：有标签的组按字母排序，未分组放最后
  const sorted: { tag: string; hosts: typeof store.hosts }[] = []
  const tagNames = Object.keys(groups).filter(k => k !== '').sort()
  for (const tag of tagNames) {
    sorted.push({ tag, hosts: groups[tag] })
  }
  if (groups['']) {
    sorted.push({ tag: '', hosts: groups[''] })
  }
  return sorted
})

// 是否有任何标签存在
const hasTags = computed(() => store.hosts.some(h => h.tags && h.tags.trim()))

function toggleGroup(tag: string) {
  if (collapsedGroups.value.has(tag)) {
    collapsedGroups.value.delete(tag)
  } else {
    collapsedGroups.value.add(tag)
  }
}

async function loadHosts() {
  try {
    const list = await HostList()
    store.setHosts(list || [])
    for (const h of (list || [])) {
      const connected = await HostIsConnected(h.id)
      if (connected) {
        store.markConnected(h.id)
        try {
          const vms = await VMList(h.id)
          const arr = vms || []
          store.setHostVMCount(h.id, arr.length, arr.filter(v => v.state === 'running').length)
        } catch { /* 静默 */ }
      } else {
        store.markDisconnected(h.id)
      }
    }
  } catch (e) {
    console.error('load hosts:', e)
  }
}

function navigateTo(path: string) {
  router.push(path)
}

function isActive(path: string) {
  return route.path === path
}

function isHostActive(id: string) {
  return route.path.startsWith(`/host/${id}`)
}

// 右键菜单
function showHostContextMenu(e: MouseEvent, host: any) {
  const connected = store.isConnected(host.id)
  const items: MenuItem[] = connected
    ? [
        { label: t('host.openTerminal'), action: () => router.push(`/host/${host.id}/terminal`) },
        { label: t('sidebar.disconnectHost'), action: () => disconnectHost(host.id) },
        { label: '', action: () => {}, divider: true },
        { label: t('common.edit'), action: () => { editHostData.value = host; showEditHost.value = true } },
        { label: t('common.delete'), variant: 'destructive', action: () => deleteHost(host.id, host.name) },
      ]
    : [
        { label: t('common.edit'), action: () => { editHostData.value = host; showEditHost.value = true } },
        { label: t('common.delete'), variant: 'destructive', action: () => deleteHost(host.id, host.name) },
      ]
  contextMenuRef.value?.show(e)
  // 更新 items 需要动态绑定，这里通过 ref 传递
  contextMenuItems.value = items
}

const contextMenuItems = ref<MenuItem[]>([])

async function disconnectHost(id: string) {
  await HostDisconnect(id)
  store.markDisconnected(id)
  toast.info(t('common.disconnected'))
}

async function deleteHost(id: string, name: string) {
  const ok = await confirmRequest(t('host.deleteHost'), t('host.deleteNameConfirm', { name }), { variant: 'destructive', confirmText: t('common.delete') })
  if (!ok) return
  try {
    await HostDelete(id)
    toast.success(t('common.deleted'))
    await loadHosts()
    if (route.path.startsWith(`/host/${id}`)) {
      router.push('/')
    }
  } catch (e: any) {
    toast.error(t('common.deleteFailed') + ': ' + e.toString())
  }
}

// 拖拽排序
function onDragStart(e: DragEvent, hostId: string) {
  dragHostId.value = hostId
  if (e.dataTransfer) {
    e.dataTransfer.effectAllowed = 'move'
  }
}

function onDragOver(e: DragEvent) {
  e.preventDefault()
  if (e.dataTransfer) {
    e.dataTransfer.dropEffect = 'move'
  }
}

async function onDrop(e: DragEvent, targetHostId: string) {
  e.preventDefault()
  if (!dragHostId.value || dragHostId.value === targetHostId) {
    dragHostId.value = null
    return
  }
  const hosts = [...store.hosts]
  const fromIdx = hosts.findIndex(h => h.id === dragHostId.value)
  const toIdx = hosts.findIndex(h => h.id === targetHostId)
  if (fromIdx < 0 || toIdx < 0) return
  const [moved] = hosts.splice(fromIdx, 1)
  hosts.splice(toIdx, 0, moved)
  // 更新 sortOrder
  for (let i = 0; i < hosts.length; i++) {
    hosts[i].sortOrder = i
  }
  store.setHosts(hosts)
  // 持久化
  for (const h of hosts) {
    try {
      await HostUpdate({ ...h } as any)
    } catch { /* 静默 */ }
  }
  dragHostId.value = null
}

function onDragEnd() {
  dragHostId.value = null
}

function onEditSaved() {
  showEditHost.value = false
  loadHosts()
}

onMounted(() => {
  loadHosts()
  AppVersion().then(v => { appVersion.value = v }).catch(() => {})
  refreshTimer = setInterval(loadHosts, refreshIntervalMs())
})

onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer)
})

defineExpose({ loadHosts })
</script>

<template>
  <aside class="w-56 h-full border-r bg-muted/30 flex flex-col">
    <!-- Logo -->
    <div class="h-12 flex items-center px-4 border-b">
      <span class="font-bold text-lg tracking-tight">VMCat</span>
      <span class="ml-auto text-xs text-muted-foreground">v{{ appVersion }}</span>
    </div>

    <!-- 模式指示器 -->
    <div class="px-3 py-2 border-b">
      <button
        v-if="isRemote"
        class="w-full flex items-center gap-2 px-2 py-1.5 text-xs rounded-md bg-primary/10 text-primary hover:bg-primary/20 transition-colors"
        @click="disconnectRemote(); loadHosts()"
        :title="t('remote.switchToLocal')"
      >
        <Wifi class="h-3.5 w-3.5" />
        <span class="truncate flex-1 text-left">{{ t('remote.remoteMode') }}</span>
        <span class="text-primary/60">v{{ remoteVersion }}</span>
      </button>
      <button
        v-else
        class="w-full flex items-center gap-2 px-2 py-1.5 text-xs rounded-md hover:bg-accent transition-colors text-muted-foreground"
        @click="showRemoteDialog = true"
        :title="t('remote.connectRemote')"
      >
        <WifiOff class="h-3.5 w-3.5" />
        <span class="truncate flex-1 text-left">{{ t('remote.localMode') }}</span>
      </button>
    </div>

    <!-- 导航 -->
    <nav class="flex-1 overflow-auto py-2">
      <button
        class="w-full flex items-center gap-2 px-4 py-2 text-sm hover:bg-accent transition-colors"
        :class="{ 'bg-accent text-accent-foreground': isActive('/') }"
        @click="navigateTo('/')"
      >
        <LayoutDashboard class="h-4 w-4" />
        <span>{{ t('nav.dashboard') }}</span>
      </button>
      <button
        class="w-full flex items-center gap-2 px-4 py-2 text-sm hover:bg-accent transition-colors"
        :class="{ 'bg-accent text-accent-foreground': isActive('/audit') }"
        @click="navigateTo('/audit')"
      >
        <FileText class="h-4 w-4" />
        <span>{{ t('nav.auditLog') }}</span>
      </button>

      <!-- 有标签时分组显示 -->
      <template v-if="hasTags">
        <div v-for="group in hostGroups" :key="group.tag" class="mt-1">
          <button
            class="w-full flex items-center gap-1 px-4 py-1 text-xs font-medium text-muted-foreground uppercase tracking-wider hover:text-foreground transition-colors"
            @click="toggleGroup(group.tag)"
          >
            <ChevronDown v-if="!collapsedGroups.has(group.tag)" class="h-3 w-3" />
            <ChevronRight v-else class="h-3 w-3" />
            {{ group.tag || t('sidebar.ungrouped') }}
            <span class="ml-auto text-muted-foreground/60">{{ group.hosts.length }}</span>
          </button>
          <div v-show="!collapsedGroups.has(group.tag)">
            <button
              v-for="host in group.hosts"
              :key="host.id"
              class="w-full flex items-center gap-2 px-4 py-2 text-sm hover:bg-accent transition-colors group"
              :class="{ 'bg-accent text-accent-foreground': isHostActive(host.id) }"
              draggable="true"
              @click="navigateTo(`/host/${host.id}`)"
              @contextmenu="showHostContextMenu($event, host)"
              @dragstart="onDragStart($event, host.id)"
              @dragover="onDragOver"
              @drop="onDrop($event, host.id)"
              @dragend="onDragEnd"
            >
              <GripVertical class="h-3 w-3 text-muted-foreground/30 opacity-0 group-hover:opacity-100 transition-opacity cursor-grab flex-shrink-0" />
              <Server class="h-4 w-4 flex-shrink-0" />
              <span class="truncate flex-1 text-left">{{ host.name }}</span>
              <span
                v-if="store.isConnected(host.id) && store.getHostVMCount(host.id).total > 0"
                class="text-xs text-muted-foreground flex-shrink-0"
              >
                {{ store.getHostVMCount(host.id).running }}/{{ store.getHostVMCount(host.id).total }}
              </span>
              <span
                class="h-2 w-2 rounded-full flex-shrink-0"
                :class="store.isConnected(host.id) ? 'bg-green-500' : 'bg-muted-foreground/30'"
              />
            </button>
          </div>
        </div>
      </template>

      <!-- 无标签时平铺显示 -->
      <template v-else>
        <div class="mt-3 px-4">
          <span class="text-xs font-medium text-muted-foreground uppercase tracking-wider">{{ t('sidebar.hosts') }}</span>
        </div>
        <div class="mt-1">
          <button
            v-for="host in store.hosts"
            :key="host.id"
            class="w-full flex items-center gap-2 px-4 py-2 text-sm hover:bg-accent transition-colors group"
            :class="{ 'bg-accent text-accent-foreground': isHostActive(host.id) }"
            draggable="true"
            @click="navigateTo(`/host/${host.id}`)"
            @contextmenu="showHostContextMenu($event, host)"
            @dragstart="onDragStart($event, host.id)"
            @dragover="onDragOver"
            @drop="onDrop($event, host.id)"
            @dragend="onDragEnd"
          >
            <GripVertical class="h-3 w-3 text-muted-foreground/30 opacity-0 group-hover:opacity-100 transition-opacity cursor-grab flex-shrink-0" />
            <Server class="h-4 w-4 flex-shrink-0" />
            <span class="truncate flex-1 text-left">{{ host.name }}</span>
            <span
              v-if="store.isConnected(host.id) && store.getHostVMCount(host.id).total > 0"
              class="text-xs text-muted-foreground flex-shrink-0"
            >
              {{ store.getHostVMCount(host.id).running }}/{{ store.getHostVMCount(host.id).total }}
            </span>
            <span
              class="h-2 w-2 rounded-full flex-shrink-0"
              :class="store.isConnected(host.id) ? 'bg-green-500' : 'bg-muted-foreground/30'"
            />
          </button>
        </div>
      </template>
    </nav>

    <!-- 底部操作 -->
    <div class="border-t p-2 flex gap-1">
      <button
        class="flex-1 flex items-center justify-center gap-1 px-2 py-2 text-sm rounded hover:bg-accent transition-colors"
        @click="showAddHost = true"
      >
        <Plus class="h-4 w-4" />
        <span>{{ t('nav.addHost') }}</span>
      </button>
      <button
        class="flex items-center justify-center px-2 py-2 rounded hover:bg-accent transition-colors"
        @click="toggleTheme"
        :title="isDark ? t('sidebar.lightMode') : t('sidebar.darkMode')"
      >
        <Sun v-if="isDark" class="h-4 w-4" />
        <Moon v-else class="h-4 w-4" />
      </button>
      <button
        class="flex items-center justify-center px-2 py-2 rounded hover:bg-accent transition-colors"
        @click="navigateTo('/settings')"
      >
        <Settings class="h-4 w-4" />
      </button>
    </div>

    <HostFormDialog
      :open="showAddHost"
      @update:open="showAddHost = $event"
      @saved="loadHosts"
    />

    <HostFormDialog
      :open="showEditHost"
      :host="editHostData"
      @update:open="showEditHost = $event"
      @saved="onEditSaved"
    />

    <ContextMenu
      ref="contextMenuRef"
      :items="contextMenuItems"
    />

    <RemoteConnectDialog
      :open="showRemoteDialog"
      @update:open="showRemoteDialog = $event"
      @connected="loadHosts"
    />
  </aside>
</template>
