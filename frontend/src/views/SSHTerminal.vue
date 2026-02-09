<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { useToast } from '@/composables/useToast'
import { TerminalPort, HostIsConnected, HostConnect } from '../../wailsjs/go/main/App'
import { useSettings } from '@/composables/useSettings'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import '@xterm/xterm/css/xterm.css'
import { ArrowLeft, Loader2 } from 'lucide-vue-next'
import { useTheme } from '@/composables/useTheme'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const store = useAppStore()
const toast = useToast()
const { isDark } = useTheme()
const { terminalFontSize } = useSettings()

const hostId = computed(() => route.params.id as string)
const host = computed(() => store.hosts.find(h => h.id === hostId.value))

const termRef = ref<HTMLDivElement>()
const status = ref<'connecting' | 'connected' | 'error'>('connecting')
const errorMsg = ref('')

let terminal: Terminal | null = null
let fitAddon: FitAddon | null = null
let ws: WebSocket | null = null
let resizeObserver: ResizeObserver | null = null
let reconnectAttempts = 0
const maxReconnects = 3
let reconnectTimer: ReturnType<typeof setTimeout> | null = null

async function initTerminal() {
  try {
    // 确保已连接
    const connected = await HostIsConnected(hostId.value)
    if (!connected) {
      await HostConnect(hostId.value)
      store.markConnected(hostId.value)
    }

    // 获取 WebSocket 端口
    const port = await TerminalPort()
    if (!port) {
      throw new Error(t('sshTerminal.termServiceNotStarted'))
    }

    // 创建 xterm 实例
    terminal = new Terminal({
      cursorBlink: true,
      fontSize: terminalFontSize.value,
      fontFamily: 'Menlo, Monaco, "Courier New", monospace',
      theme: isDark.value ? {
        background: '#1a1a2e',
        foreground: '#e0e0e0',
        cursor: '#e0e0e0',
        selectionBackground: '#44475a',
      } : {
        background: '#ffffff',
        foreground: '#1d1d1f',
        cursor: '#1d1d1f',
        selectionBackground: '#b5d5ff',
      },
      allowProposedApi: true,
    })

    fitAddon = new FitAddon()
    terminal.loadAddon(fitAddon)
    terminal.loadAddon(new WebLinksAddon())

    await nextTick()
    if (!termRef.value) return

    terminal.open(termRef.value)
    fitAddon.fit()

    const rows = terminal.rows
    const cols = terminal.cols

    // 建立 WebSocket 连接（支持 cmd 参数用于 virsh console 等）
    const cmd = route.query.cmd as string || ''
    const cmdParam = cmd ? `&cmd=${encodeURIComponent(cmd)}` : ''
    ws = new WebSocket(`ws://127.0.0.1:${port}/ws/terminal?host=${hostId.value}&rows=${rows}&cols=${cols}${cmdParam}`)
    ws.binaryType = 'arraybuffer'

    ws.onopen = () => {
      status.value = 'connected'
      terminal?.focus()
    }

    ws.onmessage = (ev) => {
      const data = ev.data instanceof ArrayBuffer
        ? new TextDecoder().decode(ev.data)
        : ev.data
      terminal?.write(data)
    }

    ws.onclose = () => {
      if (status.value === 'connected' && reconnectAttempts < maxReconnects) {
        reconnectAttempts++
        const delay = reconnectAttempts * 2000
        terminal?.write(`\r\n\x1b[33m${t('sshTerminal.disconnecting', { delay: delay / 1000, current: reconnectAttempts, max: maxReconnects })}\x1b[0m\r\n`)
        status.value = 'connecting'
        reconnectTimer = setTimeout(() => reconnectWS(port, hostId.value, rows, cols, cmd), delay)
      } else if (status.value === 'connected') {
        terminal?.write(`\r\n\x1b[31m${t('sshTerminal.disconnectedFinal')}\x1b[0m\r\n`)
        status.value = 'error'
        errorMsg.value = t('sshTerminal.connectionLost')
      }
    }

    ws.onerror = () => {
      status.value = 'error'
      errorMsg.value = t('sshTerminal.connectFailedShort')
    }

    // 终端输入 -> WebSocket
    terminal.onData((data) => {
      if (ws?.readyState === WebSocket.OPEN) {
        ws.send(new TextEncoder().encode(data))
      }
    })

    // 终端尺寸变化 -> WebSocket resize 消息
    terminal.onResize(({ rows, cols }) => {
      if (ws?.readyState === WebSocket.OPEN) {
        const buf = new Uint8Array(5)
        buf[0] = 1 // resize 标记
        buf[1] = (rows >> 8) & 0xff
        buf[2] = rows & 0xff
        buf[3] = (cols >> 8) & 0xff
        buf[4] = cols & 0xff
        ws.send(buf.buffer)
      }
    })

    // 容器尺寸变化自动 fit
    resizeObserver = new ResizeObserver(() => {
      fitAddon?.fit()
    })
    resizeObserver.observe(termRef.value)
  } catch (e: any) {
    status.value = 'error'
    errorMsg.value = e.toString()
    toast.error(t('sshTerminal.connectFailed') + ': ' + e.toString())
  }
}

// WebSocket 重连
function reconnectWS(port: number, host: string, rows: number, cols: number, cmd: string) {
  if (!terminal) return
  const cmdParam = cmd ? `&cmd=${encodeURIComponent(cmd)}` : ''
  ws = new WebSocket(`ws://127.0.0.1:${port}/ws/terminal?host=${host}&rows=${rows}&cols=${cols}${cmdParam}`)
  ws.binaryType = 'arraybuffer'

  ws.onopen = () => {
    status.value = 'connected'
    reconnectAttempts = 0
    terminal?.write(`\r\n\x1b[32m${t('sshTerminal.reconnected')}\x1b[0m\r\n`)
    terminal?.focus()
  }

  ws.onmessage = (ev) => {
    const data = ev.data instanceof ArrayBuffer
      ? new TextDecoder().decode(ev.data)
      : ev.data
    terminal?.write(data)
  }

  ws.onclose = () => {
    if (status.value === 'connected' && reconnectAttempts < maxReconnects) {
      reconnectAttempts++
      const delay = reconnectAttempts * 2000
      terminal?.write(`\r\n\x1b[33m${t('sshTerminal.disconnecting', { delay: delay / 1000, current: reconnectAttempts, max: maxReconnects })}\x1b[0m\r\n`)
      status.value = 'connecting'
      reconnectTimer = setTimeout(() => reconnectWS(port, host, rows, cols, cmd), delay)
    } else {
      terminal?.write(`\r\n\x1b[31m${t('sshTerminal.disconnectedSimple')}\x1b[0m\r\n`)
      status.value = 'error'
      errorMsg.value = t('sshTerminal.connectionLost')
    }
  }

  ws.onerror = () => {
    // onclose 会处理重连
  }
}

onMounted(initTerminal)

onUnmounted(() => {
  if (reconnectTimer) clearTimeout(reconnectTimer)
  reconnectAttempts = maxReconnects // 阻止卸载后重连
  if (resizeObserver) resizeObserver.disconnect()
  if (ws) ws.close()
  if (terminal) terminal.dispose()
})
</script>

<template>
  <div class="h-full flex flex-col">
    <!-- 顶栏 -->
    <div class="flex items-center gap-3 px-4 py-2 border-b bg-muted/30">
      <button
        class="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground transition-colors"
        @click="router.push(`/host/${hostId}`)"
      >
        <ArrowLeft class="h-4 w-4" />
        {{ t('common.back') }}
      </button>
      <span class="text-sm font-medium">{{ host?.name || '' }} - {{ route.query.cmd ? t('sshTerminal.vmConsole') : t('sshTerminal.title') }}</span>
      <span
        class="ml-auto text-xs px-2 py-0.5 rounded"
        :class="{
          'bg-green-500/10 text-green-600': status === 'connected',
          'bg-yellow-500/10 text-yellow-600': status === 'connecting',
          'bg-red-500/10 text-red-600': status === 'error',
        }"
      >
        {{ status === 'connected' ? t('sshTerminal.connected') : status === 'connecting' ? t('sshTerminal.connecting') : t('sshTerminal.connectFailedShort') }}
      </span>
    </div>

    <!-- 终端区域 -->
    <div class="flex-1 relative">
      <div
        v-if="status === 'connecting'"
        class="absolute inset-0 flex items-center justify-center bg-background"
      >
        <Loader2 class="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
      <div
        v-if="status === 'error'"
        class="absolute inset-0 flex items-center justify-center bg-background text-muted-foreground"
      >
        <p>{{ errorMsg }}</p>
      </div>
      <div ref="termRef" class="h-full w-full" />
    </div>
  </div>
</template>

<style scoped>
.xterm {
  height: 100%;
  padding: 4px;
}
</style>
