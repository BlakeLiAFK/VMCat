import { defineStore } from 'pinia'
import { ref } from 'vue'

export interface Host {
  id: string
  name: string
  host: string
  port: number
  user: string
  authType: string
  keyPath: string
  password: string
  hostKey: string
  proxyAddr: string
  tags: string
  sortOrder: number
  createdAt: string
  updatedAt: string
}

export interface VM {
  id: number
  name: string
  state: string
  cpus: number
  memoryMB: number
  hostID: string
}

export const useAppStore = defineStore('app', () => {
  const hosts = ref<Host[]>([])
  const currentHostId = ref<string>('')
  const connectedHosts = ref<Set<string>>(new Set())
  const hostVMCounts = ref<Record<string, { total: number; running: number }>>({})

  function setHosts(list: Host[]) {
    hosts.value = list
  }

  function setCurrentHost(id: string) {
    currentHostId.value = id
  }

  function markConnected(id: string) {
    connectedHosts.value.add(id)
  }

  function markDisconnected(id: string) {
    connectedHosts.value.delete(id)
  }

  function isConnected(id: string): boolean {
    return connectedHosts.value.has(id)
  }

  function setHostVMCount(id: string, total: number, running: number) {
    hostVMCounts.value[id] = { total, running }
  }

  function getHostVMCount(id: string) {
    return hostVMCounts.value[id] || { total: 0, running: 0 }
  }

  return {
    hosts,
    currentHostId,
    connectedHosts,
    setHosts,
    setCurrentHost,
    markConnected,
    markDisconnected,
    isConnected,
    hostVMCounts,
    setHostVMCount,
    getHostVMCount,
  }
})
