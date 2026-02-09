<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { ChevronRight, Home } from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const store = useAppStore()

interface BreadcrumbItem {
  label: string
  path?: string
}

const items = computed<BreadcrumbItem[]>(() => {
  const result: BreadcrumbItem[] = [{ label: '仪表盘', path: '/' }]
  const name = route.name as string

  if (name === 'dashboard') return result

  // 宿主机相关页面
  const hostId = route.params.id as string
  if (hostId) {
    const host = store.hosts.find(h => h.id === hostId)
    result.push({
      label: host?.name || hostId,
      path: `/host/${hostId}`,
    })
  }

  // VM 相关页面
  const vmName = route.params.name as string
  if (vmName) {
    result.push({
      label: vmName,
      path: `/host/${route.params.id}/vm/${vmName}`,
    })
  }

  // 特殊页面后缀
  if (name === 'terminal') {
    result.push({ label: 'SSH 终端' })
  } else if (name === 'vnc') {
    result.push({ label: 'VNC 远程桌面' })
  } else if (name === 'settings') {
    result.length = 1
    result.push({ label: '设置' })
  }

  return result
})
</script>

<template>
  <nav v-if="items.length > 1" class="flex items-center gap-1 px-6 pt-4 pb-0 text-sm">
    <template v-for="(item, idx) in items" :key="idx">
      <ChevronRight v-if="idx > 0" class="h-3.5 w-3.5 text-muted-foreground flex-shrink-0" />
      <button
        v-if="item.path && idx < items.length - 1"
        class="text-muted-foreground hover:text-foreground transition-colors truncate max-w-[160px]"
        @click="router.push(item.path)"
      >
        {{ item.label }}
      </button>
      <span v-else class="text-foreground font-medium truncate max-w-[200px]">
        {{ item.label }}
      </span>
    </template>
  </nav>
</template>
