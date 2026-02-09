<script setup lang="ts">
import { cn } from '@/lib/utils'
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  variant?: 'default' | 'secondary' | 'destructive' | 'outline' | 'success' | 'warning'
}>(), {
  variant: 'default',
})

const variantClasses: Record<string, string> = {
  default: 'border-transparent bg-primary text-primary-foreground',
  secondary: 'border-transparent bg-secondary text-secondary-foreground',
  destructive: 'border-transparent bg-destructive text-destructive-foreground',
  outline: 'text-foreground',
  success: 'border-transparent bg-green-500/15 text-green-600 dark:text-green-400',
  warning: 'border-transparent bg-yellow-500/15 text-yellow-600 dark:text-yellow-400',
}

const classes = computed(() =>
  cn(
    'inline-flex items-center rounded-md border px-2.5 py-0.5 text-xs font-semibold transition-colors',
    variantClasses[props.variant],
  )
)
</script>

<template>
  <span :class="classes"><slot /></span>
</template>
