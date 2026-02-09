<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { Search, Server, Monitor, Settings, LayoutDashboard, Plus, Keyboard } from 'lucide-vue-next'

const { t } = useI18n()

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{
  'update:open': [value: boolean]
  'show-add-host': []
  'show-hotkey-help': []
}>()

const router = useRouter()
const store = useAppStore()
const query = ref('')
const selectedIndex = ref(0)
const inputRef = ref<HTMLInputElement>()

interface CommandItem {
  id: string
  label: string
  sublabel?: string
  icon: 'host' | 'vm' | 'action'
  action: () => void
}

const items = computed<CommandItem[]>(() => {
  const q = query.value.trim().toLowerCase()
  const result: CommandItem[] = []

  // 快捷操作
  const actions: CommandItem[] = [
    {
      id: 'action-dashboard',
      label: t('cmdPalette.dashboard'),
      sublabel: t('cmdPalette.dashboardSub'),
      icon: 'action',
      action: () => router.push('/'),
    },
    {
      id: 'action-add-host',
      label: t('cmdPalette.addHost'),
      sublabel: t('cmdPalette.addHostSub'),
      icon: 'action',
      action: () => emit('show-add-host'),
    },
    {
      id: 'action-settings',
      label: t('cmdPalette.settings'),
      sublabel: t('cmdPalette.settingsSub'),
      icon: 'action',
      action: () => router.push('/settings'),
    },
    {
      id: 'action-hotkeys',
      label: t('cmdPalette.hotkeys'),
      sublabel: t('cmdPalette.hotkeysSub'),
      icon: 'action',
      action: () => emit('show-hotkey-help'),
    },
  ]

  // 宿主机
  for (const host of store.hosts) {
    const connected = store.isConnected(host.id)
    result.push({
      id: `host-${host.id}`,
      label: host.name,
      sublabel: `${host.user}@${host.host}:${host.port}${connected ? ' (' + t('cmdPalette.hostConnected') + ')' : ''}`,
      icon: 'host',
      action: () => router.push(`/host/${host.id}`),
    })
  }

  // 合并操作
  result.push(...actions)

  // 搜索过滤
  if (q) {
    return result.filter(item =>
      item.label.toLowerCase().includes(q) ||
      (item.sublabel && item.sublabel.toLowerCase().includes(q))
    )
  }
  return result
})

function close() {
  emit('update:open', false)
  query.value = ''
  selectedIndex.value = 0
}

function execute(item: CommandItem) {
  close()
  item.action()
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    close()
  } else if (e.key === 'ArrowDown') {
    e.preventDefault()
    selectedIndex.value = Math.min(selectedIndex.value + 1, items.value.length - 1)
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    selectedIndex.value = Math.max(selectedIndex.value - 1, 0)
  } else if (e.key === 'Enter') {
    e.preventDefault()
    const item = items.value[selectedIndex.value]
    if (item) execute(item)
  }
}

watch(() => props.open, (val) => {
  if (val) {
    query.value = ''
    selectedIndex.value = 0
    nextTick(() => inputRef.value?.focus())
  }
})

watch(query, () => {
  selectedIndex.value = 0
})
</script>

<template>
  <Teleport to="body">
    <Transition name="palette">
      <div v-if="open" class="fixed inset-0 z-[100] flex items-start justify-center pt-[15vh]" @keydown="onKeydown">
        <!-- 遮罩 -->
        <div class="fixed inset-0 bg-black/50" @click="close" />
        <!-- 面板 -->
        <div class="relative z-10 w-full max-w-lg rounded-lg border bg-background shadow-2xl overflow-hidden">
          <!-- 搜索框 -->
          <div class="flex items-center gap-2 px-4 border-b">
            <Search class="h-4 w-4 text-muted-foreground flex-shrink-0" />
            <input
              ref="inputRef"
              v-model="query"
              :placeholder="t('cmdPalette.placeholder')"
              class="flex-1 h-12 bg-transparent text-sm focus:outline-none placeholder:text-muted-foreground"
            />
            <kbd class="text-xs text-muted-foreground bg-muted px-1.5 py-0.5 rounded">ESC</kbd>
          </div>
          <!-- 结果列表 -->
          <div class="max-h-72 overflow-auto py-1">
            <div v-if="items.length === 0" class="px-4 py-8 text-center text-sm text-muted-foreground">
              {{ t('cmdPalette.noResults') }}
            </div>
            <button
              v-for="(item, i) in items"
              :key="item.id"
              class="w-full flex items-center gap-3 px-4 py-2.5 text-sm text-left transition-colors"
              :class="i === selectedIndex ? 'bg-accent text-accent-foreground' : 'hover:bg-accent/50'"
              @click="execute(item)"
              @mouseenter="selectedIndex = i"
            >
              <Server v-if="item.icon === 'host'" class="h-4 w-4 text-muted-foreground flex-shrink-0" />
              <Monitor v-else-if="item.icon === 'vm'" class="h-4 w-4 text-muted-foreground flex-shrink-0" />
              <template v-else>
                <LayoutDashboard v-if="item.id === 'action-dashboard'" class="h-4 w-4 text-muted-foreground flex-shrink-0" />
                <Plus v-else-if="item.id === 'action-add-host'" class="h-4 w-4 text-muted-foreground flex-shrink-0" />
                <Settings v-else-if="item.id === 'action-settings'" class="h-4 w-4 text-muted-foreground flex-shrink-0" />
                <Keyboard v-else class="h-4 w-4 text-muted-foreground flex-shrink-0" />
              </template>
              <div class="flex-1 min-w-0">
                <p class="truncate">{{ item.label }}</p>
                <p v-if="item.sublabel" class="text-xs text-muted-foreground truncate">{{ item.sublabel }}</p>
              </div>
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.palette-enter-active,
.palette-leave-active {
  transition: opacity 0.15s ease;
}
.palette-enter-from,
.palette-leave-to {
  opacity: 0;
}
</style>
