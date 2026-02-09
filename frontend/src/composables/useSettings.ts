import { ref } from 'vue'
import { SettingGet } from '@/api/backend'

// 全局缓存，避免重复请求
const refreshInterval = ref(10)
const terminalFontSize = ref(14)
const isoSearchPaths = ref('/var/lib/libvirt/images,/home,/root,/tmp')
const loaded = ref(false)

async function loadSettings() {
  try {
    const ri = await SettingGet('refresh_interval').catch(() => '')
    if (ri) refreshInterval.value = Math.max(3, Number(ri) || 10)
    const fs = await SettingGet('terminal_font_size').catch(() => '')
    if (fs) terminalFontSize.value = Math.max(8, Math.min(32, Number(fs) || 14))
    const ip = await SettingGet('iso_search_paths').catch(() => '')
    if (ip) isoSearchPaths.value = ip
    loaded.value = true
  } catch { /* 静默 */ }
}

// 刷新间隔转毫秒
function refreshIntervalMs() {
  return refreshInterval.value * 1000
}

export function useSettings() {
  // 首次调用时自动加载
  if (!loaded.value) loadSettings()
  return {
    refreshInterval,
    refreshIntervalMs,
    terminalFontSize,
    isoSearchPaths,
    loaded,
    reload: loadSettings,
  }
}
