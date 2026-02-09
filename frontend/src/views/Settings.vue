<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useToast } from '@/composables/useToast'
import { useTheme } from '@/composables/useTheme'
import { HostExportJSON, HostImportJSON } from '../../wailsjs/go/main/App'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import {
  Sun, Moon, Download, Upload, Settings as SettingsIcon,
} from 'lucide-vue-next'

const toast = useToast()
const { isDark, toggle: toggleTheme } = useTheme()
const importing = ref(false)

async function exportHosts() {
  try {
    const json = await HostExportJSON()
    const blob = new Blob([json], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `vmcat-hosts-${new Date().toISOString().slice(0, 10)}.json`
    a.click()
    URL.revokeObjectURL(url)
    toast.success('导出成功')
  } catch (e: any) {
    toast.error('导出失败: ' + e.toString())
  }
}

async function importHosts() {
  const input = document.createElement('input')
  input.type = 'file'
  input.accept = '.json'
  input.onchange = async () => {
    const file = input.files?.[0]
    if (!file) return
    importing.value = true
    try {
      const text = await file.text()
      const count = await HostImportJSON(text)
      toast.success(`导入成功，新增 ${count} 台宿主机`)
    } catch (e: any) {
      toast.error('导入失败: ' + e.toString())
    } finally {
      importing.value = false
    }
  }
  input.click()
}
</script>

<template>
  <div class="p-6 max-w-2xl">
    <h1 class="text-2xl font-bold mb-6 flex items-center gap-2">
      <SettingsIcon class="h-6 w-6" />
      设置
    </h1>

    <!-- 外观 -->
    <Card class="mb-4">
      <div class="p-4 border-b">
        <h3 class="font-semibold">外观</h3>
      </div>
      <div class="p-4">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm font-medium">主题</p>
            <p class="text-xs text-muted-foreground">切换亮色/暗色模式</p>
          </div>
          <button
            class="flex items-center gap-2 px-3 py-1.5 rounded-md border hover:bg-accent transition-colors"
            @click="toggleTheme"
          >
            <Sun v-if="isDark" class="h-4 w-4" />
            <Moon v-else class="h-4 w-4" />
            <span class="text-sm">{{ isDark ? '暗色' : '亮色' }}</span>
          </button>
        </div>
      </div>
    </Card>

    <!-- 数据管理 -->
    <Card class="mb-4">
      <div class="p-4 border-b">
        <h3 class="font-semibold">数据管理</h3>
      </div>
      <div class="p-4 space-y-4">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm font-medium">导出宿主机</p>
            <p class="text-xs text-muted-foreground">导出所有宿主机配置 (不含密码)</p>
          </div>
          <Button variant="outline" size="sm" @click="exportHosts">
            <Download class="h-4 w-4" /> 导出
          </Button>
        </div>
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm font-medium">导入宿主机</p>
            <p class="text-xs text-muted-foreground">从 JSON 文件导入，已存在的会跳过</p>
          </div>
          <Button variant="outline" size="sm" :loading="importing" @click="importHosts">
            <Upload class="h-4 w-4" /> 导入
          </Button>
        </div>
      </div>
    </Card>

    <!-- 关于 -->
    <Card>
      <div class="p-4 border-b">
        <h3 class="font-semibold">关于</h3>
      </div>
      <div class="p-4 text-sm text-muted-foreground space-y-1">
        <p><span class="text-foreground font-medium">VMCat</span> v0.1.0</p>
        <p>轻量级 KVM 虚拟机管理工具</p>
        <p>Go + Wails + Vue 3 + TypeScript</p>
      </div>
    </Card>
  </div>
</template>
