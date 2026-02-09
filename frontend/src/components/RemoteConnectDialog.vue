<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useConnection } from '@/composables/useConnection'
import { Loader2, Wifi, WifiOff } from 'lucide-vue-next'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{
  (e: 'update:open', v: boolean): void
  (e: 'connected'): void
}>()

const { t } = useI18n()
const { connectRemote } = useConnection()

const serverUrl = ref('')
const apiKey = ref('')
const testing = ref(false)
const error = ref('')

// 重置状态
watch(() => props.open, (v) => {
  if (v) {
    error.value = ''
    testing.value = false
  }
})

async function handleConnect() {
  if (!serverUrl.value.trim()) return
  error.value = ''
  testing.value = true
  try {
    // 去掉末尾斜杠
    const baseURL = serverUrl.value.trim().replace(/\/+$/, '')
    await connectRemote({
      baseURL,
      token: apiKey.value.trim() || undefined,
    })
    emit('update:open', false)
    emit('connected')
  } catch (e: any) {
    error.value = e.message || String(e)
  } finally {
    testing.value = false
  }
}

function close() {
  emit('update:open', false)
}
</script>

<template>
  <Teleport to="body">
    <div
      v-if="open"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      @click.self="close"
    >
      <div class="bg-card border rounded-lg shadow-xl w-[420px] p-6">
        <!-- 标题 -->
        <div class="flex items-center gap-2 mb-5">
          <Wifi class="h-5 w-5 text-primary" />
          <h2 class="text-base font-semibold">{{ t('remote.connectTitle') }}</h2>
        </div>

        <!-- 表单 -->
        <div class="space-y-4">
          <div>
            <label class="text-sm font-medium mb-1.5 block">{{ t('remote.serverUrl') }}</label>
            <input
              v-model="serverUrl"
              type="text"
              :placeholder="t('remote.serverUrlPlaceholder')"
              class="w-full px-3 py-2 text-sm border rounded-md bg-background focus:outline-none focus:ring-2 focus:ring-ring"
              @keydown.enter="handleConnect"
            />
          </div>
          <div>
            <label class="text-sm font-medium mb-1.5 block">{{ t('remote.apiKey') }}</label>
            <input
              v-model="apiKey"
              type="password"
              :placeholder="t('remote.apiKeyPlaceholder')"
              class="w-full px-3 py-2 text-sm border rounded-md bg-background focus:outline-none focus:ring-2 focus:ring-ring"
              @keydown.enter="handleConnect"
            />
          </div>

          <!-- 错误提示 -->
          <div v-if="error" class="text-sm text-red-500 bg-red-500/10 rounded-md px-3 py-2">
            {{ error }}
          </div>
        </div>

        <!-- 按钮 -->
        <div class="flex justify-end gap-2 mt-6">
          <button
            class="px-4 py-2 text-sm rounded-md hover:bg-accent transition-colors"
            @click="close"
          >
            {{ t('common.cancel') }}
          </button>
          <button
            class="px-4 py-2 text-sm rounded-md bg-primary text-primary-foreground hover:bg-primary/90 transition-colors flex items-center gap-1.5"
            :disabled="testing || !serverUrl.trim()"
            @click="handleConnect"
          >
            <Loader2 v-if="testing" class="h-4 w-4 animate-spin" />
            {{ testing ? t('remote.testing') : t('remote.connect') }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>
