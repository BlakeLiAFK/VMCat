// 远程模式 HTTP 客户端

import type { RemoteConfig, RemoteResponse } from './types'

/**
 * 远程服务器 HTTP 客户端
 */
export class RemoteClient {
  private config: RemoteConfig

  constructor(config: RemoteConfig) {
    this.config = config
  }

  /**
   * 调用远程 API
   * @param action 动作名称 例: "vm.list"
   * @param data 请求数据
   */
  async call<T = any>(action: string, data?: any): Promise<T> {
    const url = `${this.config.baseURL}/v1/api.json`
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    }

    if (this.config.token) {
      headers['Authorization'] = `Bearer ${this.config.token}`
    }

    const response = await fetch(url, {
      method: 'POST',
      headers,
      body: JSON.stringify({
        action,
        data: data || {},
      }),
    })

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`)
    }

    const result: RemoteResponse<T> = await response.json()

    if (result.code !== 0) {
      throw new Error(result.msg || '远程调用失败')
    }

    return result.data
  }

  /**
   * 生成 WebSocket URL
   * @param path WebSocket 路径 例: "/ws/terminal"
   * @param params 查询参数
   */
  wsURL(path: string, params: Record<string, string>): string {
    const base = this.config.baseURL.replace(/^http/, 'ws')
    const qs = new URLSearchParams(params)
    if (this.config.token) {
      qs.set('token', this.config.token)
    }
    return `${base}${path}?${qs.toString()}`
  }
}
