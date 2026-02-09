<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Loader2, Check, X } from 'lucide-vue-next'

const { t } = useI18n()

export interface BatchTask {
  name: string
  status: 'pending' | 'running' | 'success' | 'error'
  error?: string
}

const props = defineProps<{
  tasks: BatchTask[]
  visible: boolean
}>()

const emit = defineEmits<{ close: [] }>()
</script>

<template>
  <Teleport to="body">
    <Transition name="panel">
      <div v-if="visible && tasks.length > 0" class="fixed bottom-4 right-4 z-50 w-80 rounded-lg border bg-background shadow-lg">
        <div class="flex items-center justify-between p-3 border-b">
          <span class="text-sm font-medium">{{ t('batch.title') }}</span>
          <button class="text-xs text-muted-foreground hover:text-foreground" @click="emit('close')">{{ t('common.close') }}</button>
        </div>
        <div class="max-h-64 overflow-auto p-2 space-y-1">
          <div
            v-for="task in tasks"
            :key="task.name"
            class="flex items-center gap-2 px-2 py-1.5 rounded text-sm"
          >
            <Loader2 v-if="task.status === 'running'" class="h-3.5 w-3.5 animate-spin text-primary flex-shrink-0" />
            <Check v-else-if="task.status === 'success'" class="h-3.5 w-3.5 text-green-500 flex-shrink-0" />
            <X v-else-if="task.status === 'error'" class="h-3.5 w-3.5 text-red-500 flex-shrink-0" />
            <div v-else class="h-3.5 w-3.5 rounded-full border-2 border-muted-foreground/30 flex-shrink-0" />
            <span class="truncate flex-1" :class="task.status === 'error' ? 'text-red-500' : ''">{{ task.name }}</span>
            <span v-if="task.error" class="text-xs text-red-500 truncate max-w-[120px]" :title="task.error">{{ task.error }}</span>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.panel-enter-active,
.panel-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}
.panel-enter-from,
.panel-leave-to {
  opacity: 0;
  transform: translateY(10px);
}
</style>
