<script setup lang="ts">
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import {
  TitleComponent,
  TooltipComponent,
  LegendComponent,
  GridComponent,
} from 'echarts/components'
import VChart from 'vue-echarts'
import { computed } from 'vue'

use([
  CanvasRenderer,
  LineChart,
  TitleComponent,
  TooltipComponent,
  LegendComponent,
  GridComponent,
])

export interface ChartSeries {
  name: string
  data: number[]
  color?: string
}

const props = defineProps<{
  title?: string
  labels: string[]
  series: ChartSeries[]
  height?: string
  yAxisMax?: number
  yAxisFormatter?: (val: number) => string
}>()

const isDark = computed(() => document.documentElement.classList.contains('dark'))

const option = computed(() => {
  const textColor = isDark.value ? '#a1a1aa' : '#71717a'
  const gridColor = isDark.value ? '#27272a' : '#f4f4f5'

  return {
    title: props.title ? {
      text: props.title,
      left: 'left',
      textStyle: { fontSize: 13, fontWeight: 500, color: isDark.value ? '#e4e4e7' : '#18181b' },
    } : undefined,
    tooltip: {
      trigger: 'axis',
      backgroundColor: isDark.value ? '#27272a' : '#fff',
      borderColor: isDark.value ? '#3f3f46' : '#e4e4e7',
      textStyle: { color: isDark.value ? '#e4e4e7' : '#18181b', fontSize: 12 },
    },
    legend: {
      bottom: 0,
      textStyle: { color: textColor, fontSize: 11 },
    },
    grid: {
      left: 50,
      right: 20,
      top: props.title ? 40 : 15,
      bottom: 40,
    },
    xAxis: {
      type: 'category',
      data: props.labels,
      axisLabel: { color: textColor, fontSize: 10 },
      axisLine: { lineStyle: { color: gridColor } },
      splitLine: { show: false },
    },
    yAxis: {
      type: 'value',
      max: props.yAxisMax,
      axisLabel: {
        color: textColor,
        fontSize: 10,
        formatter: props.yAxisFormatter || undefined,
      },
      splitLine: { lineStyle: { color: gridColor } },
    },
    series: props.series.map(s => ({
      name: s.name,
      type: 'line',
      data: s.data,
      smooth: true,
      symbol: 'none',
      lineStyle: { width: 1.5 },
      areaStyle: { opacity: 0.05 },
      itemStyle: s.color ? { color: s.color } : undefined,
    })),
  }
})
</script>

<template>
  <VChart
    :option="option"
    :style="{ height: height || '250px', width: '100%' }"
    autoresize
  />
</template>
