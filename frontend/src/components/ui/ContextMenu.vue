<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'

export interface MenuItem {
  label: string
  icon?: any
  action: () => void
  variant?: 'default' | 'destructive'
  divider?: boolean
}

const props = defineProps<{
  items: MenuItem[]
}>()

const emit = defineEmits<{ close: [] }>()

const visible = ref(false)
const x = ref(0)
const y = ref(0)
const menuRef = ref<HTMLDivElement>()

function show(event: MouseEvent) {
  event.preventDefault()
  event.stopPropagation()
  x.value = event.clientX
  y.value = event.clientY
  visible.value = true
  // 防止超出视窗
  nextTick(() => {
    if (!menuRef.value) return
    const rect = menuRef.value.getBoundingClientRect()
    if (rect.right > window.innerWidth) {
      x.value = window.innerWidth - rect.width - 8
    }
    if (rect.bottom > window.innerHeight) {
      y.value = window.innerHeight - rect.height - 8
    }
  })
}

function hide() {
  visible.value = false
  emit('close')
}

function onItemClick(item: MenuItem) {
  hide()
  item.action()
}

function onOutsideClick(e: MouseEvent) {
  if (menuRef.value && !menuRef.value.contains(e.target as Node)) {
    hide()
  }
}

onMounted(() => {
  document.addEventListener('click', onOutsideClick, true)
  document.addEventListener('contextmenu', onOutsideClick, true)
})

onUnmounted(() => {
  document.removeEventListener('click', onOutsideClick, true)
  document.removeEventListener('contextmenu', onOutsideClick, true)
})

defineExpose({ show, hide })
</script>

<template>
  <Teleport to="body">
    <Transition name="ctx">
      <div
        v-if="visible"
        ref="menuRef"
        class="fixed z-[110] min-w-[160px] rounded-md border bg-popover p-1 shadow-lg"
        :style="{ left: x + 'px', top: y + 'px' }"
      >
        <template v-for="(item, i) in items" :key="i">
          <div v-if="item.divider" class="my-1 h-px bg-border" />
          <button
            v-else
            class="w-full flex items-center gap-2 px-2.5 py-1.5 text-sm rounded-sm transition-colors"
            :class="item.variant === 'destructive'
              ? 'text-destructive hover:bg-destructive/10'
              : 'hover:bg-accent'"
            @click="onItemClick(item)"
          >
            <component v-if="item.icon" :is="item.icon" class="h-3.5 w-3.5" />
            {{ item.label }}
          </button>
        </template>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.ctx-enter-active,
.ctx-leave-active {
  transition: opacity 0.1s ease, transform 0.1s ease;
}
.ctx-enter-from,
.ctx-leave-to {
  opacity: 0;
  transform: scale(0.95);
}
</style>
