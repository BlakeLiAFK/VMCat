<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { AuditListAll } from '../../wailsjs/go/main/App'
import { useAppStore } from '@/stores/app'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import { FileText, RotateCw, Download, Loader2 } from 'lucide-vue-next'

const { t } = useI18n()
const store = useAppStore()
const records = ref<any[]>([])
const loading = ref(false)
const filterHost = ref('')
const filterAction = ref('')

const filteredRecords = computed(() => {
  let list = records.value
  if (filterHost.value) {
    list = list.filter(r => r.hostId === filterHost.value)
  }
  if (filterAction.value) {
    list = list.filter(r => r.action.includes(filterAction.value))
  }
  return list
})

const hostOptions = computed(() => {
  return [{ label: t('audit.allHosts'), value: '' }, ...store.hosts.map(h => ({ label: h.name, value: h.id }))]
})

const actionOptions = computed(() => {
  const actions = new Set(records.value.map(r => r.action))
  return [{ label: t('audit.allActions'), value: '' }, ...Array.from(actions).sort().map(a => ({ label: t(`auditAction.${a}` as any), value: a }))]
})

function getHostName(hostId: string): string {
  return store.hosts.find(h => h.id === hostId)?.name || hostId.slice(0, 8)
}

function getActionLabel(action: string): string {
  return t(`auditAction.${action}` as any)
}

async function load() {
  loading.value = true
  try {
    records.value = (await AuditListAll(500)) || []
  } catch { records.value = [] }
  finally { loading.value = false }
}

function exportCSV() {
  const header = t('audit.csvHeader') + '\n'
  const rows = filteredRecords.value.map(r =>
    `${r.timestamp},${getHostName(r.hostId)},${r.vmName || '-'},${getActionLabel(r.action)},${r.detail || '-'}`
  ).join('\n')
  const blob = new Blob([header + rows], { type: 'text/csv' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `vmcat-audit-${new Date().toISOString().slice(0, 10)}.csv`
  a.click()
  URL.revokeObjectURL(url)
}

onMounted(load)
</script>

<template>
  <div class="p-6 max-w-5xl">
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold flex items-center gap-2">
        <FileText class="h-6 w-6" />
        {{ t('audit.title') }}
      </h1>
      <div class="flex items-center gap-2">
        <Button variant="outline" size="sm" @click="exportCSV" :disabled="filteredRecords.length === 0">
          <Download class="h-3.5 w-3.5" /> {{ t('audit.exportCSV') }}
        </Button>
        <Button variant="outline" size="sm" @click="load" :loading="loading">
          <RotateCw class="h-3.5 w-3.5" />
        </Button>
      </div>
    </div>

    <!-- 过滤 -->
    <div class="flex gap-2 mb-4">
      <select
        v-model="filterHost"
        class="h-8 text-sm rounded-md border border-input bg-transparent px-2 focus:outline-none focus:ring-1 focus:ring-ring"
      >
        <option v-for="opt in hostOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
      </select>
      <select
        v-model="filterAction"
        class="h-8 text-sm rounded-md border border-input bg-transparent px-2 focus:outline-none focus:ring-1 focus:ring-ring"
      >
        <option v-for="opt in actionOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
      </select>
      <span class="text-sm text-muted-foreground leading-8">{{ t('audit.records', { count: filteredRecords.length }) }}</span>
    </div>

    <!-- 加载中 -->
    <div v-if="loading && records.length === 0" class="text-center py-16">
      <Loader2 class="h-8 w-8 mx-auto animate-spin text-muted-foreground" />
    </div>

    <!-- 表格 -->
    <Card v-else-if="filteredRecords.length > 0">
      <table class="w-full">
        <thead>
          <tr class="border-b text-sm text-muted-foreground">
            <th class="text-left p-3 font-medium">{{ t('audit.time') }}</th>
            <th class="text-left p-3 font-medium">{{ t('audit.host') }}</th>
            <th class="text-left p-3 font-medium">{{ t('audit.vmName') }}</th>
            <th class="text-left p-3 font-medium">{{ t('audit.action') }}</th>
            <th class="text-left p-3 font-medium">{{ t('audit.detail') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="r in filteredRecords" :key="r.id" class="border-b last:border-0 text-sm">
            <td class="p-3 text-muted-foreground whitespace-nowrap">{{ r.timestamp }}</td>
            <td class="p-3">{{ getHostName(r.hostId) }}</td>
            <td class="p-3 font-mono text-xs">{{ r.vmName || '-' }}</td>
            <td class="p-3">
              <span class="px-1.5 py-0.5 rounded text-xs bg-muted">{{ getActionLabel(r.action) }}</span>
            </td>
            <td class="p-3 text-muted-foreground">{{ r.detail || '-' }}</td>
          </tr>
        </tbody>
      </table>
    </Card>

    <!-- 空状态 -->
    <div v-else class="text-center py-16 text-muted-foreground">
      <FileText class="h-12 w-12 mx-auto mb-3 opacity-50" />
      <p>{{ t('audit.noLogs') }}</p>
    </div>
  </div>
</template>
