<script setup lang="ts">
import { onMounted } from 'vue'
import Sidebar from '@/components/Sidebar.vue'
import { Toaster } from 'vue-sonner'
import { useTheme } from '@/composables/useTheme'

const { initTheme } = useTheme()
onMounted(initTheme)
</script>

<template>
  <div class="flex h-screen overflow-hidden">
    <Sidebar />
    <main class="flex-1 overflow-auto">
      <router-view />
    </main>
  </div>
  <Toaster position="top-right" :duration="3000" rich-colors close-button />
</template>
