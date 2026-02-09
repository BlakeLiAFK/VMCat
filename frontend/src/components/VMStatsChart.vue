<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { VMStatsHistory } from '@/api/backend'
import Chart from '@/components/ui/Chart.vue'
import type { ChartSeries } from '@/components/ui/Chart.vue'
import { Loader2 } from 'lucide-vue-next'

const { t } = useI18n()

const props = defineProps<{
  hostId: string
  vmName: string
  hours?: number
}>()

const loading = ref(false)
const labels = ref<string[]>([])
// 原始数据，series name 通过 computed 响应语言切换
const rawCpu = ref<number[]>([])
const rawMem = ref<number[]>([])
const rawRx = ref<number[]>([])
const rawTx = ref<number[]>([])

const cpuSeries = computed<ChartSeries[]>(() => {
  if (rawCpu.value.length === 0) return []
  return [
    { name: t('chart.cpu'), data: rawCpu.value, color: '#3b82f6' },
    { name: t('chart.memoryMB'), data: rawMem.value, color: '#a855f7' },
  ]
})

const netSeries = computed<ChartSeries[]>(() => {
  if (rawRx.value.length === 0) return []
  return [
    { name: 'RX (B/s)', data: rawRx.value, color: '#10b981' },
    { name: 'TX (B/s)', data: rawTx.value, color: '#f59e0b' },
  ]
})

const cpuMemTitle = computed(() => t('chart.cpuMemory'))
const netTitle = computed(() => t('chart.networkTraffic'))

function formatBytes(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  if (bytes < 1024 * 1024 * 1024) return (bytes / 1024 / 1024).toFixed(1) + ' MB'
  return (bytes / 1024 / 1024 / 1024).toFixed(1) + ' GB'
}

async function load() {
  loading.value = true
  try {
    const hours = props.hours || 24
    const records = await VMStatsHistory(props.hostId, props.vmName, hours)
    if (!records || records.length === 0) {
      labels.value = []
      rawCpu.value = []
      rawMem.value = []
      rawRx.value = []
      rawTx.value = []
      return
    }

    labels.value = records.map(r => {
      const d = new Date(r.timestamp)
      return `${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}`
    })

    rawCpu.value = records.map(r => Math.round(r.cpuPercent * 10) / 10)
    rawMem.value = records.map(r => Math.round(r.memUsed / 1024 / 1024))

    // 网络流量用差值计算速率
    const rxRates: number[] = []
    const txRates: number[] = []
    for (let i = 0; i < records.length; i++) {
      if (i === 0) {
        rxRates.push(0)
        txRates.push(0)
      } else {
        const dt = (new Date(records[i].timestamp).getTime() - new Date(records[i - 1].timestamp).getTime()) / 1000
        if (dt > 0) {
          rxRates.push(Math.max(0, (records[i].netRx - records[i - 1].netRx) / dt))
          txRates.push(Math.max(0, (records[i].netTx - records[i - 1].netTx) / dt))
        } else {
          rxRates.push(0)
          txRates.push(0)
        }
      }
    }

    rawRx.value = rxRates.map(v => Math.round(v))
    rawTx.value = txRates.map(v => Math.round(v))
  } catch {
    labels.value = []
    rawCpu.value = []
    rawMem.value = []
    rawRx.value = []
    rawTx.value = []
  } finally {
    loading.value = false
  }
}

onMounted(load)
watch(() => [props.hostId, props.vmName], load)
</script>

<template>
  <div>
    <div v-if="loading" class="flex items-center justify-center py-8">
      <Loader2 class="h-5 w-5 animate-spin text-muted-foreground" />
    </div>
    <div v-else-if="labels.length === 0" class="text-center py-8 text-sm text-muted-foreground">
      {{ t('chart.noHistory') }}
    </div>
    <template v-else>
      <Chart
        :title="cpuMemTitle"
        :labels="labels"
        :series="cpuSeries"
        height="300px"
      />
      <Chart
        :title="netTitle"
        :labels="labels"
        :series="netSeries"
        height="300px"
        class="mt-6"
      />
    </template>
  </div>
</template>
