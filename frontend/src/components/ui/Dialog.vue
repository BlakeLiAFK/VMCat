<script setup lang="ts">
import { watch } from 'vue'

const props = defineProps<{
  open: boolean
  title?: string
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

// ESC 键关闭
function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') emit('update:open', false)
}

watch(() => props.open, (val) => {
  if (val) {
    document.addEventListener('keydown', onKeydown)
  } else {
    document.removeEventListener('keydown', onKeydown)
  }
})
</script>

<template>
  <Teleport to="body">
    <Transition name="dialog">
      <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center">
        <!-- 遮罩 -->
        <div
          class="fixed inset-0 bg-black/50"
          @click="emit('update:open', false)"
        />
        <!-- 内容 -->
        <div class="relative z-50 w-full max-w-lg rounded-lg border bg-background p-6 shadow-lg">
          <h2 v-if="title" class="text-lg font-semibold mb-4">{{ title }}</h2>
          <slot />
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.dialog-enter-active,
.dialog-leave-active {
  transition: opacity 0.15s ease;
}
.dialog-enter-from,
.dialog-leave-to {
  opacity: 0;
}
</style>
