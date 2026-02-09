import { onMounted, onUnmounted } from 'vue'

interface HotkeyBinding {
  key: string
  ctrl?: boolean
  meta?: boolean
  shift?: boolean
  handler: () => void
  /** 在 input/textarea 中是否也触发 */
  allowInput?: boolean
}

const globalBindings: HotkeyBinding[] = []

function matchEvent(e: KeyboardEvent, b: HotkeyBinding): boolean {
  const needMod = b.ctrl || b.meta
  if (needMod) {
    // Cmd (macOS) 或 Ctrl (其他平台) 均可
    if (!e.metaKey && !e.ctrlKey) return false
  }
  if (b.shift && !e.shiftKey) return false
  return e.key.toLowerCase() === b.key.toLowerCase()
}

function isInputElement(el: EventTarget | null): boolean {
  if (!el || !(el instanceof HTMLElement)) return false
  const tag = el.tagName.toLowerCase()
  return tag === 'input' || tag === 'textarea' || el.isContentEditable
}

function handleKeydown(e: KeyboardEvent) {
  for (const b of globalBindings) {
    if (!b.allowInput && isInputElement(e.target)) continue
    if (matchEvent(e, b)) {
      e.preventDefault()
      e.stopPropagation()
      b.handler()
      return
    }
  }
}

let listenerInstalled = false

function ensureListener() {
  if (listenerInstalled) return
  document.addEventListener('keydown', handleKeydown, true)
  listenerInstalled = true
}

/** 注册全局快捷键，组件卸载时自动移除 */
export function useHotkey(bindings: HotkeyBinding[]) {
  ensureListener()
  onMounted(() => {
    globalBindings.push(...bindings)
  })
  onUnmounted(() => {
    for (const b of bindings) {
      const idx = globalBindings.indexOf(b)
      if (idx >= 0) globalBindings.splice(idx, 1)
    }
  })
}

/** 快捷键定义列表（用于帮助面板展示） */
export const hotkeyList = [
  { keys: 'Cmd/Ctrl+K', labelKey: 'hotkey.globalSearch' },
  { keys: 'Cmd/Ctrl+N', labelKey: 'hotkey.addHost' },
  { keys: 'Cmd/Ctrl+,', labelKey: 'hotkey.openSettings' },
  { keys: 'Cmd/Ctrl+D', labelKey: 'hotkey.openDashboard' },
  { keys: '?', labelKey: 'hotkey.showHelp' },
]
