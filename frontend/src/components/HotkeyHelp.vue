<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { hotkeyList } from '@/composables/useHotkey'
import { Keyboard } from 'lucide-vue-next'

const { t } = useI18n()
defineProps<{ open: boolean }>()
const emit = defineEmits<{ 'update:open': [value: boolean] }>()
</script>

<template>
  <Teleport to="body">
    <Transition name="help">
      <div v-if="open" class="fixed inset-0 z-[100] flex items-center justify-center">
        <div class="fixed inset-0 bg-black/50" @click="emit('update:open', false)" />
        <div class="relative z-10 w-full max-w-sm rounded-lg border bg-background p-6 shadow-2xl">
          <h2 class="text-lg font-semibold mb-4 flex items-center gap-2">
            <Keyboard class="h-5 w-5" />
            {{ t('hotkey.title') }}
          </h2>
          <div class="space-y-2">
            <div
              v-for="item in hotkeyList"
              :key="item.keys"
              class="flex items-center justify-between py-1.5"
            >
              <span class="text-sm">{{ t(item.labelKey) }}</span>
              <kbd class="text-xs bg-muted px-2 py-1 rounded font-mono">{{ item.keys }}</kbd>
            </div>
          </div>
          <div class="mt-4 pt-4 border-t flex justify-end">
            <button
              class="text-sm text-muted-foreground hover:text-foreground transition-colors"
              @click="emit('update:open', false)"
            >
              {{ t('common.close') }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.help-enter-active,
.help-leave-active {
  transition: opacity 0.15s ease;
}
.help-enter-from,
.help-leave-to {
  opacity: 0;
}
</style>
