// 连接模式管理 - 本地/远程模式切换

import { ref, computed } from 'vue'
import { switchToRemote, switchToLocal, AppVersion } from '@/api/backend'
import type { RemoteConfig } from '@/api/types'

// 全局单例状态
const _mode = ref<'local' | 'remote'>('local')
const _remoteConfig = ref<RemoteConfig | null>(null)
const _remoteVersion = ref('')

export function useConnection() {
  const mode = computed(() => _mode.value)
  const isRemote = computed(() => _mode.value === 'remote')
  const remoteConfig = computed(() => _remoteConfig.value)
  const remoteVersion = computed(() => _remoteVersion.value)

  /** 连接远程服务器，失败时自动回退本地模式 */
  async function connectRemote(config: RemoteConfig): Promise<void> {
    switchToRemote(config)
    try {
      const ver = await AppVersion()
      _remoteVersion.value = ver
      _mode.value = 'remote'
      _remoteConfig.value = config
    } catch (e) {
      switchToLocal()
      throw e
    }
  }

  /** 断开远程，切回本地模式 */
  function disconnectRemote(): void {
    switchToLocal()
    _mode.value = 'local'
    _remoteConfig.value = null
    _remoteVersion.value = ''
  }

  return {
    mode,
    isRemote,
    remoteConfig,
    remoteVersion,
    connectRemote,
    disconnectRemote,
  }
}
