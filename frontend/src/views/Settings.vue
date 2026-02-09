<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useToast } from '@/composables/useToast'
import { useTheme } from '@/composables/useTheme'
import { HostExportJSON, HostImportJSON, SettingGet, SettingSet, AppVersion } from '../../wailsjs/go/main/App'
import { useSettings } from '@/composables/useSettings'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import TemplateManager from '@/components/TemplateManager.vue'
import {
  Sun, Moon, Download, Upload, Settings as SettingsIcon, Save, Layers,
} from 'lucide-vue-next'

const activeTab = ref<'general' | 'templates'>('general')

const toast = useToast()
const { isDark, toggle: toggleTheme } = useTheme()
const importing = ref(false)
const appVersion = ref('')
const savingKey = ref('')

// 配置项
const refreshInterval = ref('10')
const terminalFontSize = ref('14')
const isoSearchPaths = ref('/var/lib/libvirt/images,/home,/root,/tmp')

async function loadSettings() {
  try {
    const ri = await SettingGet('refresh_interval').catch(() => '')
    if (ri) refreshInterval.value = ri
    const fs = await SettingGet('terminal_font_size').catch(() => '')
    if (fs) terminalFontSize.value = fs
    const ip = await SettingGet('iso_search_paths').catch(() => '')
    if (ip) isoSearchPaths.value = ip
  } catch { /* 静默 */ }
}

async function saveSetting(key: string, value: string, label: string) {
  savingKey.value = key
  try {
    await SettingSet(key, value)
    toast.success(`${label}已保存`)
    // 同步全局缓存
    useSettings().reload()
  } catch (e: any) {
    toast.error('保存失败: ' + e.toString())
  } finally {
    savingKey.value = ''
  }
}

onMounted(() => {
  loadSettings()
  AppVersion().then(v => { appVersion.value = v }).catch(() => {})
})

// 数据管理
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
  <div class="p-6 max-w-3xl">
    <h1 class="text-2xl font-bold mb-4 flex items-center gap-2">
      <SettingsIcon class="h-6 w-6" />
      设置
    </h1>

    <!-- Tab 切换 -->
    <div class="flex border-b mb-6">
      <button
        class="px-4 py-2.5 text-sm border-b-2 transition-colors -mb-px"
        :class="activeTab === 'general' ? 'border-primary text-foreground font-medium' : 'border-transparent text-muted-foreground hover:text-foreground'"
        @click="activeTab = 'general'"
      >
        <SettingsIcon class="h-3.5 w-3.5 inline mr-1" /> 通用
      </button>
      <button
        class="px-4 py-2.5 text-sm border-b-2 transition-colors -mb-px"
        :class="activeTab === 'templates' ? 'border-primary text-foreground font-medium' : 'border-transparent text-muted-foreground hover:text-foreground'"
        @click="activeTab = 'templates'"
      >
        <Layers class="h-3.5 w-3.5 inline mr-1" /> 模板
      </button>
    </div>

    <!-- 模板管理 Tab -->
    <TemplateManager v-if="activeTab === 'templates'" />

    <!-- 通用设置 Tab -->
    <div v-if="activeTab === 'general'">

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

    <!-- 监控配置 -->
    <Card class="mb-4">
      <div class="p-4 border-b">
        <h3 class="font-semibold">监控配置</h3>
      </div>
      <div class="p-4 space-y-4">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm font-medium">刷新间隔 (秒)</p>
            <p class="text-xs text-muted-foreground">宿主机和 VM 列表自动刷新间隔</p>
          </div>
          <div class="flex items-center gap-2">
            <Input v-model="refreshInterval" type="number" class="w-20 h-8 text-sm" />
            <Button variant="outline" size="sm" :loading="savingKey === 'refresh_interval'" @click="saveSetting('refresh_interval', refreshInterval, '刷新间隔')">
              <Save class="h-3.5 w-3.5" />
            </Button>
          </div>
        </div>
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm font-medium">终端字体大小 (px)</p>
            <p class="text-xs text-muted-foreground">SSH 终端字体大小</p>
          </div>
          <div class="flex items-center gap-2">
            <Input v-model="terminalFontSize" type="number" class="w-20 h-8 text-sm" />
            <Button variant="outline" size="sm" :loading="savingKey === 'terminal_font_size'" @click="saveSetting('terminal_font_size', terminalFontSize, '终端字体')">
              <Save class="h-3.5 w-3.5" />
            </Button>
          </div>
        </div>
        <div>
          <div class="flex items-center justify-between mb-2">
            <div>
              <p class="text-sm font-medium">ISO 搜索路径</p>
              <p class="text-xs text-muted-foreground">多个路径用英文逗号分隔</p>
            </div>
            <Button variant="outline" size="sm" :loading="savingKey === 'iso_search_paths'" @click="saveSetting('iso_search_paths', isoSearchPaths, 'ISO 搜索路径')">
              <Save class="h-3.5 w-3.5" />
            </Button>
          </div>
          <Input v-model="isoSearchPaths" class="text-sm" placeholder="/var/lib/libvirt/images,/home" />
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
        <p><span class="text-foreground font-medium">VMCat</span> v{{ appVersion }}</p>
        <p>轻量级 KVM 虚拟机管理工具</p>
        <p>Go + Wails + Vue 3 + TypeScript</p>
      </div>
    </Card>

    </div><!-- /通用设置 Tab -->
  </div>
</template>
