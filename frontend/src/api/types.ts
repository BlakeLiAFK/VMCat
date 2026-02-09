// 前端 API 共享类型定义

/**
 * 远程服务器配置
 */
export interface RemoteConfig {
  /** 服务器地址(含端口) 例: https://192.168.1.100:8080 */
  baseURL: string
  /** 认证 Token (可选) */
  token?: string
}

/**
 * 远程 API 响应格式
 */
export interface RemoteResponse<T = any> {
  code: number
  msg: string
  data: T
}
