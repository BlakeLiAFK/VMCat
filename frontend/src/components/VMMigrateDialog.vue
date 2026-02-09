<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useToast } from '@/composables/useToast'
import { VMMigrate, VMMigrateOffline } from '@/api/backend'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import Dialog from '@/components/ui/Dialog.vue'
import Button from '@/components/ui/Button.vue'
import Select from '@/components/ui/Select.vue'
import { ArrowRight, Server, Loader2, CheckCircle, Wifi, WifiOff } from 'lucide-vue-next'

const { t } = useI18n()

const props = defineProps<{
  open: boolean
  hostId: string
  vmName: string
  vmState?: string
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  migrated: []
}>()

const store = useAppStore()
const toast = useToast()
const targetHostId = ref('')
const migrating = ref(false)
const mode = ref<'live' | 'offline'>('live')
const progressStep = ref('')
const progressDetail = ref('')

const availableHosts = computed(() => {
  return store.hosts
    .filter(h => h.id !== props.hostId && store.isConnected(h.id))
    .map(h => ({ label: h.name, value: h.id }))
})

let cleanupProgress: (() => void) | null = null

watch(() => props.open, (val) => {
  if (val) {
    targetHostId.value = ''
    migrating.value = false
    progressStep.value = ''
    progressDetail.value = ''
    // 根据 VM 状态自动选择模式
    mode.value = props.vmState === 'running' ? 'live' : 'offline'
    // 监听进度事件
    cleanupProgress = EventsOn('migrate:progress', (data: any) => {
      progressStep.value = data.step || ''
      progressDetail.value = data.detail || ''
    })
  } else {
    if (cleanupProgress) { cleanupProgress(); cleanupProgress = null }
  }
})

onUnmounted(() => {
  if (cleanupProgress) cleanupProgress()
})

async function doMigrate() {
  if (!targetHostId.value) {
    toast.warning(t('migrate.selectTarget'))
    return
  }
  migrating.value = true
  progressStep.value = ''
  progressDetail.value = ''
  try {
    if (mode.value === 'live') {
      await VMMigrate(props.hostId, props.vmName, targetHostId.value)
    } else {
      await VMMigrateOffline(props.hostId, props.vmName, targetHostId.value)
    }
    toast.success(t('migrate.migrateSuccessMsg', { name: props.vmName }))
    emit('migrated')
    emit('update:open', false)
  } catch (e: any) {
    toast.error(t('migrate.migrateFailed') + ': ' + e.toString())
  } finally {
    migrating.value = false
  }
}
</script>

<template>
  <Dialog :open="open" :title="t('migrate.migrateTitle')" size="lg" @update:open="emit('update:open', $event)">
    <div class="space-y-4">
      <div class="flex items-center gap-3 p-3 bg-muted rounded-lg text-sm">
        <Server class="h-4 w-4" />
        <span class="font-medium">{{ vmName }}</span>
        <ArrowRight class="h-4 w-4 text-muted-foreground" />
        <span class="text-muted-foreground">{{ t('migrate.targetHost') }}</span>
      </div>

      <!-- 迁移模式选择 -->
      <div>
        <label class="text-sm font-medium mb-2 block">{{ t('migrate.migrateMode') }}</label>
        <div class="grid grid-cols-2 gap-3">
          <button
            class="p-3 rounded-lg border text-left transition-all hover:border-primary/50"
            :class="mode === 'live' ? 'border-primary bg-primary/5 ring-1 ring-primary' : ''"
            @click="mode = 'live'"
            :disabled="migrating"
          >
            <div class="flex items-center gap-2 mb-1">
              <Wifi class="h-4 w-4 text-green-500" />
              <span class="text-sm font-medium">{{ t('migrate.liveMigrate') }}</span>
            </div>
            <p class="text-xs text-muted-foreground">{{ t('migrate.liveMigrateTip') }}</p>
          </button>
          <button
            class="p-3 rounded-lg border text-left transition-all hover:border-primary/50"
            :class="mode === 'offline' ? 'border-primary bg-primary/5 ring-1 ring-primary' : ''"
            @click="mode = 'offline'"
            :disabled="migrating"
          >
            <div class="flex items-center gap-2 mb-1">
              <WifiOff class="h-4 w-4 text-amber-500" />
              <span class="text-sm font-medium">{{ t('migrate.offlineMigrate') }}</span>
            </div>
            <p class="text-xs text-muted-foreground">{{ t('migrate.offlineMigrateTip') }}</p>
          </button>
        </div>
      </div>

      <div>
        <label class="text-sm font-medium mb-1 block">{{ t('migrate.targetHost') }}</label>
        <Select v-model="targetHostId" :options="availableHosts" :placeholder="t('migrate.selectHost')" />
        <p v-if="availableHosts.length === 0" class="text-xs text-muted-foreground mt-1">
          {{ t('migrate.needOtherHost') }}
        </p>
      </div>

      <!-- 模式提示 -->
      <div v-if="mode === 'live'" class="p-3 bg-amber-500/10 rounded text-xs text-amber-600 dark:text-amber-400">
        {{ t('migrate.liveMigrateNote') }}
      </div>
      <div v-else class="p-3 bg-blue-500/10 rounded text-xs text-blue-600 dark:text-blue-400 space-y-1">
        <p>{{ t('migrate.offlineMigrateNote') }}</p>
        <p>{{ t('migrate.offlineMigrateFlow') }}</p>
        <p v-if="vmState === 'running'" class="text-red-500 font-medium">{{ t('migrate.vmRunningWarning') }}</p>
      </div>

      <!-- 进度显示 -->
      <div v-if="migrating && mode === 'offline'" class="p-3 bg-muted rounded-lg space-y-2">
        <div class="flex items-center gap-2 text-sm">
          <Loader2 v-if="progressStep !== 'done'" class="h-4 w-4 animate-spin text-primary" />
          <CheckCircle v-else class="h-4 w-4 text-green-500" />
          <span class="font-medium">{{ progressStep }}</span>
        </div>
        <p class="text-xs text-muted-foreground font-mono">{{ progressDetail }}</p>
      </div>

      <div class="flex gap-2 justify-end pt-2">
        <Button variant="outline" @click="emit('update:open', false)" :disabled="migrating">{{ t('common.cancel') }}</Button>
        <Button :loading="migrating" :disabled="!targetHostId" @click="doMigrate">
          {{ mode === 'live' ? t('migrate.liveMigrateBtn') : t('migrate.offlineMigrateBtn') }}
        </Button>
      </div>
    </div>
  </Dialog>
</template>
