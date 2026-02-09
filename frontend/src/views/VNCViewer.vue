<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { useToast } from '@/composables/useToast'
import { TerminalPort, HostIsConnected, HostConnect, VMGet } from '../../wailsjs/go/main/App'
import { ArrowLeft, Loader2, Maximize, Minimize, ZoomIn, ZoomOut, Lock } from 'lucide-vue-next'

// noVNC 使用动态导入，在 initVNC 中按需加载

const route = useRoute()
const router = useRouter()
const store = useAppStore()
const toast = useToast()

const hostId = computed(() => route.params.id as string)
const vmName = computed(() => route.params.name as string)
const host = computed(() => store.hosts.find(h => h.id === hostId.value))

const vncRef = ref<HTMLDivElement>()
const status = ref<'connecting' | 'connected' | 'disconnected' | 'error'>('connecting')
const errorMsg = ref('')
const isFullscreen = ref(false)
const scaleMode = ref<'remote' | 'scale'>('scale')
const showPasswordDialog = ref(false)
const vncPassword = ref('')

let rfb: any = null

async function initVNC() {
  try {
    // 确保已连接 SSH
    const connected = await HostIsConnected(hostId.value)
    if (!connected) {
      await HostConnect(hostId.value)
      store.markConnected(hostId.value)
    }

    // 获取 VM 的 VNC 端口
    const detail = await VMGet(hostId.value, vmName.value)
    if (!detail.vncPort || detail.vncPort <= 0) {
      throw new Error(`VM "${detail.name}" 没有 VNC 端口`)
    }

    // 获取 WebSocket 代理端口
    const port = await TerminalPort()
    if (!port) {
      throw new Error('WebSocket 服务未启动')
    }

    // 动态导入 noVNC
    const { default: RFB } = await import('@novnc/novnc/lib/rfb.js')

    await nextTick()
    if (!vncRef.value) return

    // 创建 noVNC 连接
    const hostIP = host.value?.host || ''
    const url = `ws://127.0.0.1:${port}/ws/vnc?host=${hostId.value}&port=${detail.vncPort}&ip=${encodeURIComponent(hostIP)}`
    rfb = new RFB(vncRef.value, url)

    rfb.scaleViewport = true
    rfb.resizeSession = false

    rfb.addEventListener('connect', () => {
      status.value = 'connected'
      toast.success('VNC 已连接')
    })

    rfb.addEventListener('disconnect', (e: any) => {
      const detail = e.detail || {}
      if (detail.clean) {
        status.value = 'disconnected'
      } else {
        status.value = 'error'
        errorMsg.value = 'VNC 连接异常断开' + (detail.reason ? ': ' + detail.reason : '')
        toast.error('VNC 连接断开')
      }
    })

    rfb.addEventListener('credentialsrequired', (e: any) => {
      const types = e.detail?.types || []
      if (types.includes('password')) {
        showPasswordDialog.value = true
      }
    })

    rfb.addEventListener('securityfailure', (e: any) => {
      status.value = 'error'
      errorMsg.value = '认证失败: ' + (e.detail?.reason || '')
    })
  } catch (e: any) {
    status.value = 'error'
    errorMsg.value = e.message || e.toString()
    toast.error('VNC 连接失败: ' + (e.message || e.toString()))
  }
}

function toggleFullscreen() {
  const el = vncRef.value?.parentElement
  if (!el) return

  if (!document.fullscreenElement) {
    el.requestFullscreen()
    isFullscreen.value = true
  } else {
    document.exitFullscreen()
    isFullscreen.value = false
  }
}

function toggleScale() {
  if (!rfb) return
  if (scaleMode.value === 'scale') {
    rfb.scaleViewport = false
    rfb.resizeSession = true
    scaleMode.value = 'remote'
  } else {
    rfb.scaleViewport = true
    rfb.resizeSession = false
    scaleMode.value = 'scale'
  }
}

function submitPassword() {
  if (!rfb || !vncPassword.value) return
  rfb.sendCredentials({ password: vncPassword.value })
  showPasswordDialog.value = false
  vncPassword.value = ''
}

function sendCtrlAltDel() {
  if (rfb) {
    rfb.sendCtrlAltDel()
  }
}

onMounted(initVNC)

onUnmounted(() => {
  if (rfb) {
    rfb.disconnect()
    rfb = null
  }
  if (document.fullscreenElement) {
    document.exitFullscreen()
  }
})
</script>

<template>
  <div class="h-full flex flex-col">
    <!-- 顶栏 -->
    <div class="flex items-center gap-3 px-4 py-2 border-b bg-muted/30 flex-shrink-0">
      <button
        class="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground transition-colors"
        @click="router.push(`/host/${hostId}/vm/${vmName}`)"
      >
        <ArrowLeft class="h-4 w-4" />
        返回
      </button>
      <span class="text-sm font-medium">{{ vmName }} - VNC 远程桌面</span>

      <!-- 工具栏 -->
      <div class="ml-auto flex items-center gap-1" v-if="status === 'connected'">
        <button
          class="px-2 py-1 text-xs rounded hover:bg-accent transition-colors"
          @click="sendCtrlAltDel"
          title="Ctrl+Alt+Del"
        >
          Ctrl+Alt+Del
        </button>
        <button
          class="p-1.5 rounded hover:bg-accent transition-colors"
          @click="toggleScale"
          :title="scaleMode === 'scale' ? '原始分辨率' : '缩放适配'"
        >
          <ZoomIn v-if="scaleMode === 'scale'" class="h-4 w-4" />
          <ZoomOut v-else class="h-4 w-4" />
        </button>
        <button
          class="p-1.5 rounded hover:bg-accent transition-colors"
          @click="toggleFullscreen"
          :title="isFullscreen ? '退出全屏' : '全屏'"
        >
          <Minimize v-if="isFullscreen" class="h-4 w-4" />
          <Maximize v-else class="h-4 w-4" />
        </button>
      </div>

      <!-- 状态指示 -->
      <span
        v-if="status !== 'connected'"
        class="text-xs px-2 py-0.5 rounded"
        :class="{
          'bg-yellow-500/10 text-yellow-600': status === 'connecting',
          'bg-red-500/10 text-red-600': status === 'error',
          'bg-muted text-muted-foreground': status === 'disconnected',
        }"
      >
        {{ status === 'connecting' ? '连接中...' : status === 'error' ? '连接失败' : '已断开' }}
      </span>
    </div>

    <!-- VNC 区域 -->
    <div class="flex-1 relative bg-black overflow-hidden">
      <!-- 加载中 -->
      <div
        v-if="status === 'connecting'"
        class="absolute inset-0 flex items-center justify-center z-10"
      >
        <div class="text-center">
          <Loader2 class="h-8 w-8 animate-spin text-white/60 mx-auto" />
          <p class="text-sm text-white/60 mt-2">正在连接 VNC...</p>
        </div>
      </div>

      <!-- 错误提示 -->
      <div
        v-if="status === 'error'"
        class="absolute inset-0 flex items-center justify-center z-10"
      >
        <div class="text-center">
          <p class="text-red-400 mb-2">{{ errorMsg }}</p>
          <button
            class="px-3 py-1.5 text-sm rounded bg-white/10 text-white hover:bg-white/20 transition-colors"
            @click="status = 'connecting'; initVNC()"
          >
            重试
          </button>
        </div>
      </div>

      <!-- VNC 密码弹窗 -->
      <div
        v-if="showPasswordDialog"
        class="absolute inset-0 flex items-center justify-center z-20 bg-black/60"
      >
        <div class="bg-card border rounded-lg p-6 w-80 shadow-xl">
          <div class="flex items-center gap-2 mb-4">
            <Lock class="h-5 w-5 text-muted-foreground" />
            <h3 class="text-sm font-medium">VNC 需要密码</h3>
          </div>
          <input
            v-model="vncPassword"
            type="password"
            placeholder="输入 VNC 密码"
            class="w-full px-3 py-2 text-sm border rounded-md bg-background focus:outline-none focus:ring-2 focus:ring-ring mb-4"
            @keydown.enter="submitPassword"
          />
          <div class="flex justify-end gap-2">
            <button
              class="px-3 py-1.5 text-xs rounded-md hover:bg-accent transition-colors"
              @click="showPasswordDialog = false; rfb?.disconnect()"
            >
              取消
            </button>
            <button
              class="px-3 py-1.5 text-xs rounded-md bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
              @click="submitPassword"
            >
              连接
            </button>
          </div>
        </div>
      </div>

      <!-- noVNC 渲染容器 -->
      <div ref="vncRef" class="h-full w-full" />
    </div>
  </div>
</template>

<style scoped>
/* noVNC canvas 自适应 */
:deep(canvas) {
  display: block;
}
</style>
