<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Sidebar from '@/components/Sidebar.vue'
import Breadcrumb from '@/components/Breadcrumb.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import CommandPalette from '@/components/CommandPalette.vue'
import HotkeyHelp from '@/components/HotkeyHelp.vue'
import HostFormDialog from '@/components/HostFormDialog.vue'
import { Toaster } from 'vue-sonner'
import { useTheme } from '@/composables/useTheme'
import { useHotkey } from '@/composables/useHotkey'
import { SettingGet } from '../wailsjs/go/main/App'

const { initTheme } = useTheme()
const { locale } = useI18n()
const router = useRouter()

const showPalette = ref(false)
const showHotkeyHelp = ref(false)
const showAddHostFromPalette = ref(false)
const sidebarRef = ref<InstanceType<typeof Sidebar>>()

useHotkey([
  { key: 'k', ctrl: true, meta: true, handler: () => { showPalette.value = true } },
  { key: 'n', ctrl: true, meta: true, handler: () => { showAddHostFromPalette.value = true } },
  { key: ',', ctrl: true, meta: true, handler: () => router.push('/settings') },
  { key: 'd', ctrl: true, meta: true, handler: () => router.push('/') },
  { key: '?', handler: () => { showHotkeyHelp.value = true } },
])

function onPaletteAddHost() {
  showPalette.value = false
  showAddHostFromPalette.value = true
}

function onPaletteHotkeyHelp() {
  showPalette.value = false
  showHotkeyHelp.value = true
}

function onHostSavedFromPalette() {
  showAddHostFromPalette.value = false
  sidebarRef.value?.loadHosts()
}

onMounted(() => {
  initTheme()
  SettingGet('language').then(lang => { if (lang) locale.value = lang }).catch(() => {})
})
</script>

<template>
  <div class="flex h-screen overflow-hidden">
    <Sidebar ref="sidebarRef" />
    <main class="flex-1 overflow-auto">
      <Breadcrumb />
      <router-view />
    </main>
  </div>
  <ConfirmDialog />
  <Toaster position="top-right" :duration="3000" rich-colors close-button />
  <CommandPalette
    :open="showPalette"
    @update:open="showPalette = $event"
    @show-add-host="onPaletteAddHost"
    @show-hotkey-help="onPaletteHotkeyHelp"
  />
  <HotkeyHelp
    :open="showHotkeyHelp"
    @update:open="showHotkeyHelp = $event"
  />
  <HostFormDialog
    :open="showAddHostFromPalette"
    @update:open="showAddHostFromPalette = $event"
    @saved="onHostSavedFromPalette"
  />
</template>
