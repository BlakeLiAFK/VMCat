<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { HostAdd, HostUpdate, HostTest } from '@/api/backend'
import type { Host } from '@/stores/app'
import Dialog from '@/components/ui/Dialog.vue'
import Input from '@/components/ui/Input.vue'
import Select from '@/components/ui/Select.vue'
import Button from '@/components/ui/Button.vue'

const { t } = useI18n()

const props = defineProps<{
  open: boolean
  host?: Host
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  saved: []
}>()

const form = ref({
  id: '',
  name: '',
  host: '',
  port: 22,
  user: 'root',
  authType: 'key',
  keyPath: '',
  password: '',
  proxyAddr: '',
  tags: '',
})

const testing = ref(false)
const saving = ref(false)
const testResult = ref<{ ok: boolean; msg: string } | null>(null)

const authOptions = [
  { label: 'SSH Key', value: 'key' },
  { label: 'Password', value: 'password' },
]

// 编辑模式时回填
watch(() => props.open, (val) => {
  if (val && props.host) {
    form.value = { ...props.host }
  } else if (val) {
    form.value = { id: '', name: '', host: '', port: 22, user: 'root', authType: 'key', keyPath: '', password: '', proxyAddr: '', tags: '' }
  }
  testResult.value = null
})

// 构建提交数据，确保类型正确
function buildPayload() {
  return { ...form.value, port: Number(form.value.port) || 22 }
}

async function onTest() {
  testing.value = true
  testResult.value = null
  try {
    const info = await HostTest(buildPayload() as any)
    testResult.value = { ok: true, msg: t('hostForm.testSuccess', { info }) }
  } catch (e: any) {
    testResult.value = { ok: false, msg: e.toString() }
  } finally {
    testing.value = false
  }
}

async function onSave() {
  saving.value = true
  try {
    const payload = buildPayload()
    if (payload.id) {
      await HostUpdate(payload as any)
    } else {
      await HostAdd(payload as any)
    }
    emit('saved')
    emit('update:open', false)
  } catch (e: any) {
    testResult.value = { ok: false, msg: e.toString() }
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <Dialog
    :open="open"
    :title="host ? t('hostForm.editHost') : t('hostForm.addHost')"
    @update:open="emit('update:open', $event)"
  >
    <form class="space-y-4" @submit.prevent="onSave">
      <!-- 名称 -->
      <div>
        <label class="text-sm font-medium mb-1 block">{{ t('hostForm.name') }}</label>
        <Input v-model="form.name" placeholder="homelab" />
      </div>

      <!-- 地址 + 端口 -->
      <div class="flex gap-2">
        <div class="flex-1">
          <label class="text-sm font-medium mb-1 block">{{ t('hostForm.address') }}</label>
          <Input v-model="form.host" placeholder="10.0.0.2" />
        </div>
        <div class="w-24">
          <label class="text-sm font-medium mb-1 block">{{ t('hostForm.port') }}</label>
          <Input v-model="form.port" type="number" placeholder="22" />
        </div>
      </div>

      <!-- 用户名 -->
      <div>
        <label class="text-sm font-medium mb-1 block">{{ t('hostForm.username') }}</label>
        <Input v-model="form.user" placeholder="root" />
      </div>

      <!-- 认证方式 -->
      <div>
        <label class="text-sm font-medium mb-1 block">{{ t('hostForm.authType') }}</label>
        <Select v-model="form.authType" :options="authOptions" />
      </div>

      <!-- Key 路径 -->
      <div v-if="form.authType === 'key'">
        <label class="text-sm font-medium mb-1 block">{{ t('hostForm.keyPath') }}</label>
        <Input v-model="form.keyPath" :placeholder="t('hostForm.keyPathPlaceholder')" />
      </div>

      <!-- 密码 -->
      <div v-if="form.authType === 'password'">
        <label class="text-sm font-medium mb-1 block">{{ t('hostForm.password') }}</label>
        <Input v-model="form.password" type="password" :placeholder="t('hostForm.passwordPlaceholder')" />
      </div>

      <!-- SOCKS5 代理 -->
      <div>
        <label class="text-sm font-medium mb-1 block">{{ t('hostForm.proxy') }}</label>
        <Input v-model="form.proxyAddr" :placeholder="t('hostForm.proxyPlaceholder')" />
      </div>

      <!-- 标签 -->
      <div>
        <label class="text-sm font-medium mb-1 block">{{ t('hostForm.tags') }}</label>
        <Input v-model="form.tags" :placeholder="t('hostForm.tagsPlaceholder')" />
        <div v-if="form.tags" class="flex flex-wrap gap-1 mt-1.5">
          <span
            v-for="tag in form.tags.split(',').map(s => s.trim()).filter(Boolean)"
            :key="tag"
            class="inline-flex items-center px-2 py-0.5 rounded-full text-xs bg-primary/10 text-primary"
          >{{ tag }}</span>
        </div>
      </div>

      <!-- 测试结果 -->
      <div v-if="testResult" class="text-sm p-2 rounded" :class="testResult.ok ? 'bg-green-500/10 text-green-600' : 'bg-red-500/10 text-red-600'">
        {{ testResult.msg }}
      </div>

      <!-- 操作按钮 -->
      <div class="flex gap-2 justify-end pt-2">
        <Button variant="outline" type="button" @click="emit('update:open', false)">{{ t('common.cancel') }}</Button>
        <Button variant="outline" type="button" :loading="testing" @click="onTest">{{ t('hostForm.testConnect') }}</Button>
        <Button type="submit" :loading="saving">{{ t('common.save') }}</Button>
      </div>
    </form>
  </Dialog>
</template>
