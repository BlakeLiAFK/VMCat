<script setup lang="ts">
import { useConfirm } from '@/composables/useConfirm'
import Dialog from '@/components/ui/Dialog.vue'
import Button from '@/components/ui/Button.vue'
import { AlertTriangle } from 'lucide-vue-next'

const { state, confirm, cancel } = useConfirm()
</script>

<template>
  <Dialog :open="state.open" @update:open="cancel">
    <div class="flex flex-col items-center text-center gap-4">
      <div
        class="h-12 w-12 rounded-full flex items-center justify-center"
        :class="state.variant === 'destructive' ? 'bg-destructive/10' : 'bg-amber-500/10'"
      >
        <AlertTriangle
          class="h-6 w-6"
          :class="state.variant === 'destructive' ? 'text-destructive' : 'text-amber-500'"
        />
      </div>
      <div>
        <h3 class="text-lg font-semibold">{{ state.title }}</h3>
        <p class="text-sm text-muted-foreground mt-1">{{ state.message }}</p>
      </div>
      <div class="flex gap-2 w-full justify-end pt-2">
        <Button variant="outline" @click="cancel">{{ state.cancelText }}</Button>
        <Button
          :variant="state.variant === 'destructive' ? 'destructive' : 'default'"
          @click="confirm"
        >
          {{ state.confirmText }}
        </Button>
      </div>
    </div>
  </Dialog>
</template>
