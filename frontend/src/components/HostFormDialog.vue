<script setup lang="ts">
import { ref, watch } from 'vue'
import { HostAdd, HostUpdate, HostTest } from '../../wailsjs/go/main/App'
import type { Host } from '@/stores/app'
import Dialog from '@/components/ui/Dialog.vue'
import Input from '@/components/ui/Input.vue'
import Select from '@/components/ui/Select.vue'
import Button from '@/components/ui/Button.vue'

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
    form.value = { id: '', name: '', host: '', port: 22, user: 'root', authType: 'key', keyPath: '', password: '', proxyAddr: '' }
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
    testResult.value = { ok: true, msg: `连接成功: ${info}` }
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
    :title="host ? '编辑宿主机' : '添加宿主机'"
    @update:open="emit('update:open', $event)"
  >
    <form class="space-y-4" @submit.prevent="onSave">
      <!-- 名称 -->
      <div>
        <label class="text-sm font-medium mb-1 block">名称</label>
        <Input v-model="form.name" placeholder="homelab" />
      </div>

      <!-- 地址 + 端口 -->
      <div class="flex gap-2">
        <div class="flex-1">
          <label class="text-sm font-medium mb-1 block">地址</label>
          <Input v-model="form.host" placeholder="10.0.0.2" />
        </div>
        <div class="w-24">
          <label class="text-sm font-medium mb-1 block">端口</label>
          <Input v-model="form.port" type="number" placeholder="22" />
        </div>
      </div>

      <!-- 用户名 -->
      <div>
        <label class="text-sm font-medium mb-1 block">用户名</label>
        <Input v-model="form.user" placeholder="root" />
      </div>

      <!-- 认证方式 -->
      <div>
        <label class="text-sm font-medium mb-1 block">认证方式</label>
        <Select v-model="form.authType" :options="authOptions" />
      </div>

      <!-- Key 路径 -->
      <div v-if="form.authType === 'key'">
        <label class="text-sm font-medium mb-1 block">Key 路径</label>
        <Input v-model="form.keyPath" placeholder="~/.ssh/id_rsa (留空使用默认)" />
      </div>

      <!-- 密码 -->
      <div v-if="form.authType === 'password'">
        <label class="text-sm font-medium mb-1 block">密码</label>
        <Input v-model="form.password" type="password" placeholder="SSH 密码" />
      </div>

      <!-- SOCKS5 代理 -->
      <div>
        <label class="text-sm font-medium mb-1 block">SOCKS5 代理</label>
        <Input v-model="form.proxyAddr" placeholder="留空则直连 (例: 127.0.0.1:1080)" />
      </div>

      <!-- 测试结果 -->
      <div v-if="testResult" class="text-sm p-2 rounded" :class="testResult.ok ? 'bg-green-500/10 text-green-600' : 'bg-red-500/10 text-red-600'">
        {{ testResult.msg }}
      </div>

      <!-- 操作按钮 -->
      <div class="flex gap-2 justify-end pt-2">
        <Button variant="outline" type="button" @click="emit('update:open', false)">取消</Button>
        <Button variant="outline" type="button" :loading="testing" @click="onTest">测试连接</Button>
        <Button type="submit" :loading="saving">保存</Button>
      </div>
    </form>
  </Dialog>
</template>
