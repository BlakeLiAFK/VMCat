<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { HostStatsHistory } from '@/api/backend'
import Chart from '@/components/ui/Chart.vue'
import type { ChartSeries } from '@/components/ui/Chart.vue'
import { Loader2 } from 'lucide-vue-next'

const { t } = useI18n()

const props = defineProps<{
  hostId: string
  hours?: number
}>()

const loading = ref(false)
const labels = ref<string[]>([])
// 原始数据，series name 通过 computed 响应语言切换
const rawData = ref<{ cpu: number[]; mem: number[]; disk: number[] }>({ cpu: [], mem: [], disk: [] })

const series = computed<ChartSeries[]>(() => {
  if (rawData.value.cpu.length === 0) return []
  return [
    { name: t('chart.cpu'), data: rawData.value.cpu, color: '#3b82f6' },
    { name: t('chart.memory'), data: rawData.value.mem, color: '#a855f7' },
    { name: t('chart.disk'), data: rawData.value.disk, color: '#f97316' },
  ]
})

const chartTitle = computed(() => t('chart.resourceTrend'))

async function load() {
  loading.value = true
  try {
    const hours = props.hours || 24
    const records = await HostStatsHistory(props.hostId, hours)
    if (!records || records.length === 0) {
      labels.value = []
      rawData.value = { cpu: [], mem: [], disk: [] }
      return
    }

    labels.value = records.map(r => {
      const d = new Date(r.timestamp)
      return `${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}`
    })

    rawData.value = {
      cpu: records.map(r => Math.round(r.cpuPercent * 10) / 10),
      mem: records.map(r => Math.round(r.memPercent * 10) / 10),
      disk: records.map(r => Math.round(r.diskPercent * 10) / 10),
    }
  } catch {
    labels.value = []
    rawData.value = { cpu: [], mem: [], disk: [] }
  } finally {
    loading.value = false
  }
}

onMounted(load)
watch(() => props.hostId, load)
</script>

<template>
  <div>
    <div v-if="loading" class="flex items-center justify-center py-8">
      <Loader2 class="h-5 w-5 animate-spin text-muted-foreground" />
    </div>
    <div v-else-if="labels.length === 0" class="text-center py-8 text-sm text-muted-foreground">
      {{ t('chart.noHistory') }}
    </div>
    <Chart
      v-else
      :title="chartTitle"
      :labels="labels"
      :series="series"
      :yAxisMax="100"
      :yAxisFormatter="(v: number) => v + '%'"
      height="280px"
    />
  </div>
</template>
