<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@/composables/useToast'
import { useTheme } from '@/composables/useTheme'
import { HostExportJSON, HostImportJSON, SettingGet, SettingSet, AppVersion } from '@/api/backend'
import { useSettings } from '@/composables/useSettings'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import TemplateManager from '@/components/TemplateManager.vue'
import {
  Sun, Moon, Download, Upload, Settings as SettingsIcon, Save, Layers, Languages,
} from 'lucide-vue-next'

const { t, locale } = useI18n()

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
const alertCpuThreshold = ref('90')
const alertMemThreshold = ref('90')
const alertDiskThreshold = ref('85')

const currentLang = ref('zh')

async function switchLanguage(lang: string) {
  currentLang.value = lang
  locale.value = lang
  await SettingSet('language', lang)
}

async function loadSettings() {
  try {
    const ri = await SettingGet('refresh_interval').catch(() => '')
    if (ri) refreshInterval.value = ri
    const fs = await SettingGet('terminal_font_size').catch(() => '')
    if (fs) terminalFontSize.value = fs
    const ip = await SettingGet('iso_search_paths').catch(() => '')
    if (ip) isoSearchPaths.value = ip
    const ac = await SettingGet('alert_cpu_threshold').catch(() => '')
    if (ac) alertCpuThreshold.value = ac
    const am = await SettingGet('alert_mem_threshold').catch(() => '')
    if (am) alertMemThreshold.value = am
    const ad = await SettingGet('alert_disk_threshold').catch(() => '')
    if (ad) alertDiskThreshold.value = ad
    const lang = await SettingGet('language').catch(() => '')
    if (lang) { currentLang.value = lang; locale.value = lang }
  } catch { /* 静默 */ }
}

async function saveSetting(key: string, value: string, label: string) {
  savingKey.value = key
  try {
    await SettingSet(key, value)
    toast.success(t('settings.settingSaved', { label }))
    // 同步全局缓存
    useSettings().reload()
  } catch (e: any) {
    toast.error(t('common.saveFailed') + ': ' + e.toString())
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
    toast.success(t('settings.exportSuccess'))
  } catch (e: any) {
    toast.error(t('settings.exportFailed') + ': ' + e.toString())
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
      toast.success(t('settings.importSuccess', { count }))
    } catch (e: any) {
      toast.error(t('settings.importFailed') + ': ' + e.toString())
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
      {{ t('settings.title') }}
    </h1>

    <!-- Tab 切换 -->
    <div class="flex border-b mb-6">
      <button
        class="px-4 py-2.5 text-sm border-b-2 transition-colors -mb-px"
        :class="activeTab === 'general' ? 'border-primary text-foreground font-medium' : 'border-transparent text-muted-foreground hover:text-foreground'"
        @click="activeTab = 'general'"
      >
        <SettingsIcon class="h-3.5 w-3.5 inline mr-1" /> {{ t('settings.general') }}
      </button>
      <button
        class="px-4 py-2.5 text-sm border-b-2 transition-colors -mb-px"
        :class="activeTab === 'templates' ? 'border-primary text-foreground font-medium' : 'border-transparent text-muted-foreground hover:text-foreground'"
        @click="activeTab = 'templates'"
      >
        <Layers class="h-3.5 w-3.5 inline mr-1" /> {{ t('settings.templates') }}
      </button>
    </div>

    <!-- 模板管理 Tab -->
    <TemplateManager v-if="activeTab === 'templates'" />

    <!-- 通用设置 Tab -->
    <div v-if="activeTab === 'general'">

    <!-- 外观 -->
    <Card class="mb-4">
      <div class="p-4 border-b">
        <h3 class="font-semibold">{{ t('settings.appearance') }}</h3>
      </div>
      <div class="p-4">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm font-medium">{{ t('settings.theme') }}</p>
            <p class="text-xs text-muted-foreground">{{ t('settings.themeTip') }}</p>
          </div>
          <button
            class="flex items-center gap-2 px-3 py-1.5 rounded-md border hover:bg-accent transition-colors"
            @click="toggleTheme"
          >
            <Sun v-if="isDark" class="h-4 w-4" />
            <Moon v-else class="h-4 w-4" />
            <span class="text-sm">{{ isDark ? t('settings.dark') : t('settings.light') }}</span>
          </button>
        </div>
      </div>
    </Card>

    <!-- 语言 -->
    <Card class="mb-4">
      <div class="p-4 border-b">
        <h3 class="font-semibold">{{ t('settings.language') }}</h3>
      </div>
      <div class="p-4">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm font-medium">{{ t('settings.language') }}</p>
            <p class="text-xs text-muted-foreground">{{ t('settings.languageTip') }}</p>
          </div>
          <div class="flex items-center gap-1">
            <button
              class="px-3 py-1.5 text-sm rounded-md border transition-colors"
              :class="currentLang === 'zh' ? 'bg-primary text-primary-foreground border-primary' : 'hover:bg-accent'"
              @click="switchLanguage('zh')"
            >中文</button>
            <button
              class="px-3 py-1.5 text-sm rounded-md border transition-colors"
              :class="currentLang === 'en' ? 'bg-primary text-primary-foreground border-primary' : 'hover:bg-accent'"
              @click="switchLanguage('en')"
            >English</button>
          </div>
        </div>
      </div>
    </Card>

    <!-- 监控配置 -->
    <Card class="mb-4">
      <div class="p-4 border-b">
        <h3 class="font-semibold">{{ t('settings.monitorConfig') }}</h3>
      </div>
      <div class="p-4 space-y-4">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm font-medium">{{ t('settings.refreshInterval') }}</p>
            <p class="text-xs text-muted-foreground">{{ t('settings.refreshIntervalTip') }}</p>
          </div>
          <div class="flex items-center gap-2">
            <Input v-model="refreshInterval" type="number" class="w-20 h-8 text-sm" />
            <Button variant="outline" size="sm" :loading="savingKey === 'refresh_interval'" @click="saveSetting('refresh_interval', refreshInterval, t('settings.refreshIntervalLabel'))">
              <Save class="h-3.5 w-3.5" />
            </Button>
          </div>
        </div>
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm font-medium">{{ t('settings.terminalFontSize') }}</p>
            <p class="text-xs text-muted-foreground">{{ t('settings.terminalFontSizeTip') }}</p>
          </div>
          <div class="flex items-center gap-2">
            <Input v-model="terminalFontSize" type="number" class="w-20 h-8 text-sm" />
            <Button variant="outline" size="sm" :loading="savingKey === 'terminal_font_size'" @click="saveSetting('terminal_font_size', terminalFontSize, t('settings.terminalFontLabel'))">
              <Save class="h-3.5 w-3.5" />
            </Button>
          </div>
        </div>
        <!-- 告警阈值 -->
        <div class="pt-2 border-t">
          <p class="text-sm font-medium mb-2">{{ t('settings.alertThreshold') }}</p>
          <div class="grid grid-cols-3 gap-3">
            <div>
              <label class="text-xs text-muted-foreground">CPU (%)</label>
              <div class="flex items-center gap-1 mt-1">
                <Input v-model="alertCpuThreshold" type="number" class="h-8 text-sm" />
                <Button variant="outline" size="sm" :loading="savingKey === 'alert_cpu_threshold'" @click="saveSetting('alert_cpu_threshold', alertCpuThreshold, t('settings.cpuThreshold'))">
                  <Save class="h-3.5 w-3.5" />
                </Button>
              </div>
            </div>
            <div>
              <label class="text-xs text-muted-foreground">{{ t('settings.memThreshold') }} (%)</label>
              <div class="flex items-center gap-1 mt-1">
                <Input v-model="alertMemThreshold" type="number" class="h-8 text-sm" />
                <Button variant="outline" size="sm" :loading="savingKey === 'alert_mem_threshold'" @click="saveSetting('alert_mem_threshold', alertMemThreshold, t('settings.memThreshold'))">
                  <Save class="h-3.5 w-3.5" />
                </Button>
              </div>
            </div>
            <div>
              <label class="text-xs text-muted-foreground">{{ t('settings.diskThreshold') }} (%)</label>
              <div class="flex items-center gap-1 mt-1">
                <Input v-model="alertDiskThreshold" type="number" class="h-8 text-sm" />
                <Button variant="outline" size="sm" :loading="savingKey === 'alert_disk_threshold'" @click="saveSetting('alert_disk_threshold', alertDiskThreshold, t('settings.diskThreshold'))">
                  <Save class="h-3.5 w-3.5" />
                </Button>
              </div>
            </div>
          </div>
        </div>

        <div>
          <div class="flex items-center justify-between mb-2">
            <div>
              <p class="text-sm font-medium">{{ t('settings.isoSearchPaths') }}</p>
              <p class="text-xs text-muted-foreground">{{ t('settings.isoSearchPathsTip') }}</p>
            </div>
            <Button variant="outline" size="sm" :loading="savingKey === 'iso_search_paths'" @click="saveSetting('iso_search_paths', isoSearchPaths, t('settings.isoSearchPaths'))">
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
        <h3 class="font-semibold">{{ t('settings.dataManagement') }}</h3>
      </div>
      <div class="p-4 space-y-4">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm font-medium">{{ t('settings.exportHosts') }}</p>
            <p class="text-xs text-muted-foreground">{{ t('settings.exportHostsTip') }}</p>
          </div>
          <Button variant="outline" size="sm" @click="exportHosts">
            <Download class="h-4 w-4" /> {{ t('common.export') }}
          </Button>
        </div>
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm font-medium">{{ t('settings.importHosts') }}</p>
            <p class="text-xs text-muted-foreground">{{ t('settings.importHostsTip') }}</p>
          </div>
          <Button variant="outline" size="sm" :loading="importing" @click="importHosts">
            <Upload class="h-4 w-4" /> {{ t('common.import') }}
          </Button>
        </div>
      </div>
    </Card>

    <!-- 关于 -->
    <Card>
      <div class="p-4 border-b">
        <h3 class="font-semibold">{{ t('settings.about') }}</h3>
      </div>
      <div class="p-4 text-sm text-muted-foreground space-y-1">
        <p><span class="text-foreground font-medium">VMCat</span> v{{ appVersion }}</p>
        <p>{{ t('settings.aboutDesc') }}</p>
        <p>Go + Wails + Vue 3 + TypeScript</p>
      </div>
    </Card>

    </div><!-- /通用设置 Tab -->
  </div>
</template>
