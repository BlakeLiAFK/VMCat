import { reactive } from 'vue'

export interface ConfirmState {
  open: boolean
  title: string
  message: string
  variant: 'default' | 'destructive'
  confirmText: string
  cancelText: string
}

type ResolveFunc = (value: boolean) => void

const state = reactive<ConfirmState>({
  open: false,
  title: '',
  message: '',
  variant: 'default',
  confirmText: '',
  cancelText: '',
})

let resolvePromise: ResolveFunc | null = null

function request(
  title: string,
  message: string,
  opts?: { variant?: 'default' | 'destructive'; confirmText?: string; cancelText?: string },
): Promise<boolean> {
  state.title = title
  state.message = message
  state.variant = opts?.variant ?? 'default'
  state.confirmText = opts?.confirmText ?? ''
  state.cancelText = opts?.cancelText ?? ''
  state.open = true

  return new Promise<boolean>((resolve) => {
    resolvePromise = resolve
  })
}

function confirm() {
  state.open = false
  resolvePromise?.(true)
  resolvePromise = null
}

function cancel() {
  state.open = false
  resolvePromise?.(false)
  resolvePromise = null
}

export function useConfirm() {
  return { state, request, confirm, cancel }
}
