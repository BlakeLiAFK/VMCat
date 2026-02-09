<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@/composables/useToast'
import {
  VMCreate, PoolList, BridgeList, NetworkList, ISOList, OSVariantList,
  VMGenerateCloudInit,
} from '../../wailsjs/go/main/App'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import { X, Loader2, ChevronDown, ChevronRight, Cloud } from 'lucide-vue-next'

const props = defineProps<{ open: boolean; hostId: string }>()
const emit = defineEmits<{ 'update:open': [v: boolean]; saved: [] }>()
const { t } = useI18n()
const toast = useToast()

// 表单数据
const form = ref({
  name: '',
  cpus: 2,
  memoryMB: 2048,
  diskSizeGB: 20,
  diskPath: '',
  cdrom: '',
  network: '',
  netType: 'bridge',
  osVariant: 'generic',
  vnc: true,
})

// Cloud-init 配置
const showCloudInit = ref(false)
const cloudInit = ref({
  enabled: false,
  hostname: '',
  user: '',
  password: '',
  sshKey: '',
  userData: '',
})

// 选项数据
const pools = ref<any[]>([])
const bridges = ref<string[]>([])
const networks = ref<any[]>([])
const isos = ref<any[]>([])
const osVariants = ref<string[]>([])
const loadingOptions = ref(false)
const creating = ref(false)

async function loadOptions() {
  loadingOptions.value = true
  try {
    const [p, b, n, i, o] = await Promise.all([
      PoolList(props.hostId).catch(() => []),
      BridgeList(props.hostId).catch(() => []),
      NetworkList(props.hostId).catch(() => []),
      ISOList(props.hostId).catch(() => []),
      OSVariantList(props.hostId).catch(() => []),
    ])
    pools.value = p || []
    bridges.value = b || []
    networks.value = n || []
    isos.value = i || []
    osVariants.value = o || []
    // 默认选第一个网桥
    if (bridges.value.length > 0 && !form.value.network) {
      form.value.network = bridges.value[0]
      form.value.netType = 'bridge'
    } else if (networks.value.length > 0 && !form.value.network) {
      form.value.network = networks.value[0].name
      form.value.netType = 'network'
    }
  } finally {
    loadingOptions.value = false
  }
}

watch(() => props.open, (v) => {
  if (v) loadOptions()
})

async function submit() {
  if (!form.value.name.trim()) {
    toast.warning(t('vmCreate.enterVMName'))
    return
  }
  creating.value = true
  try {
    // 如果启用了 cloud-init，先生成 ISO
    if (cloudInit.value.enabled) {
      const isoPath = `/var/lib/libvirt/images/${form.value.name}-cidata.iso`
      await VMGenerateCloudInit(props.hostId, isoPath, {
        hostname: cloudInit.value.hostname || form.value.name,
        user: cloudInit.value.user,
        password: cloudInit.value.password,
        sshKey: cloudInit.value.sshKey,
        userData: cloudInit.value.userData,
      })
    }
    await VMCreate(props.hostId, form.value as any)
    toast.success(t('vmCreate.createSuccess', { name: form.value.name }))
    emit('saved')
    close()
  } catch (e: any) {
    toast.error(t('vmCreate.createFailed') + ': ' + e.toString())
  } finally {
    creating.value = false
  }
}

function close() {
  emit('update:open', false)
  form.value = { name: '', cpus: 2, memoryMB: 2048, diskSizeGB: 20, diskPath: '', cdrom: '', network: '', netType: 'bridge', osVariant: 'generic', vnc: true }
  cloudInit.value = { enabled: false, hostname: '', user: '', password: '', sshKey: '', userData: '' }
  showCloudInit.value = false
}

function setNetworkOption(value: string, type: string) {
  form.value.network = value
  form.value.netType = type
}
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center">
      <div class="absolute inset-0 bg-black/50" @click="close" />
      <div class="relative bg-card border rounded-xl shadow-2xl w-[560px] max-h-[85vh] overflow-y-auto">
        <!-- 标题 -->
        <div class="flex items-center justify-between p-5 border-b sticky top-0 bg-card z-10">
          <h2 class="text-lg font-semibold">{{ t('vmCreate.title') }}</h2>
          <button @click="close" class="p-1 rounded hover:bg-accent"><X class="h-4 w-4" /></button>
        </div>

        <!-- 加载中 -->
        <div v-if="loadingOptions" class="p-12 text-center">
          <Loader2 class="h-6 w-6 animate-spin mx-auto text-muted-foreground" />
          <p class="text-sm text-muted-foreground mt-2">{{ t('vmCreate.loadingResources') }}</p>
        </div>

        <!-- 表单 -->
        <div v-else class="p-5 space-y-5">
          <!-- 基本信息 -->
          <section>
            <h3 class="text-sm font-medium mb-3 text-muted-foreground">{{ t('vmCreate.basicInfo') }}</h3>
            <div class="space-y-3">
              <div>
                <label class="text-sm mb-1 block">{{ t('vmCreate.name') }} *</label>
                <Input v-model="form.name" placeholder="my-vm" />
              </div>
              <div class="grid grid-cols-2 gap-3">
                <div>
                  <label class="text-sm mb-1 block">{{ t('vmCreate.cpu') }}</label>
                  <Input v-model="form.cpus" type="number" placeholder="2" />
                </div>
                <div>
                  <label class="text-sm mb-1 block">{{ t('vmCreate.memoryMB') }}</label>
                  <Input v-model="form.memoryMB" type="number" placeholder="2048" />
                </div>
              </div>
              <div>
                <label class="text-sm mb-1 block">{{ t('vmCreate.osVariant') }}</label>
                <select
                  v-model="form.osVariant"
                  class="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                >
                  <option v-for="v in osVariants" :key="v" :value="v">{{ v }}</option>
                </select>
              </div>
            </div>
          </section>

          <!-- 存储 -->
          <section>
            <h3 class="text-sm font-medium mb-3 text-muted-foreground">{{ t('vmCreate.storage') }}</h3>
            <div class="space-y-3">
              <div>
                <label class="text-sm mb-1 block">{{ t('vmCreate.diskSizeGB') }}</label>
                <Input v-model="form.diskSizeGB" type="number" placeholder="20" />
              </div>
              <div>
                <label class="text-sm mb-1 block">{{ t('vmCreate.diskPath') }}</label>
                <Input v-model="form.diskPath" placeholder="/var/lib/libvirt/images/my-vm.qcow2" />
              </div>
            </div>
          </section>

          <!-- 安装介质 -->
          <section>
            <h3 class="text-sm font-medium mb-3 text-muted-foreground">{{ t('vmCreate.installMedia') }}</h3>
            <div>
              <label class="text-sm mb-1 block">{{ t('vmCreate.isoImage') }}</label>
              <select
                v-model="form.cdrom"
                class="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              >
                <option value="">{{ t('vmCreate.noISO') }}</option>
                <option v-for="iso in isos" :key="iso.path" :value="iso.path">
                  {{ iso.name }} ({{ iso.size }})
                </option>
              </select>
              <p class="text-xs text-muted-foreground mt-1">
                {{ t('vmCreate.isoSearchPaths') }}: /var/lib/libvirt/images, /home, /root, /tmp
              </p>
            </div>
          </section>

          <!-- 网络 -->
          <section>
            <h3 class="text-sm font-medium mb-3 text-muted-foreground">{{ t('vmCreate.network') }}</h3>
            <div class="space-y-2">
              <div v-if="bridges.length > 0">
                <label class="text-xs text-muted-foreground block mb-1">{{ t('vmCreate.bridge') }}</label>
                <div class="flex flex-wrap gap-2">
                  <button
                    v-for="br in bridges" :key="br"
                    class="px-3 py-1 text-xs rounded-md border transition-colors"
                    :class="form.network === br && form.netType === 'bridge' ? 'bg-primary text-primary-foreground border-primary' : 'hover:bg-accent'"
                    @click="setNetworkOption(br, 'bridge')"
                  >{{ br }}</button>
                </div>
              </div>
              <div v-if="networks.length > 0">
                <label class="text-xs text-muted-foreground block mb-1">{{ t('vmCreate.virtualNetwork') }}</label>
                <div class="flex flex-wrap gap-2">
                  <button
                    v-for="n in networks" :key="n.name"
                    class="px-3 py-1 text-xs rounded-md border transition-colors"
                    :class="form.network === n.name && form.netType === 'network' ? 'bg-primary text-primary-foreground border-primary' : 'hover:bg-accent'"
                    @click="setNetworkOption(n.name, 'network')"
                  >{{ n.name }} ({{ n.state }})</button>
                </div>
              </div>
            </div>
          </section>

          <!-- 显示 -->
          <section>
            <h3 class="text-sm font-medium mb-3 text-muted-foreground">{{ t('vmCreate.display') }}</h3>
            <label class="flex items-center gap-2 text-sm cursor-pointer">
              <input type="checkbox" v-model="form.vnc" class="rounded" />
              {{ t('vmCreate.enableVNC') }}
            </label>
          </section>

          <!-- Cloud-Init -->
          <section>
            <button
              class="flex items-center gap-2 text-sm font-medium text-muted-foreground hover:text-foreground transition-colors w-full"
              @click="showCloudInit = !showCloudInit"
            >
              <Cloud class="h-4 w-4" />
              <span>{{ t('vmCreate.cloudInit') }}</span>
              <ChevronDown v-if="showCloudInit" class="h-3.5 w-3.5 ml-auto" />
              <ChevronRight v-else class="h-3.5 w-3.5 ml-auto" />
            </button>
            <div v-if="showCloudInit" class="mt-3 space-y-3">
              <label class="flex items-center gap-2 text-sm cursor-pointer">
                <input type="checkbox" v-model="cloudInit.enabled" class="rounded" />
                {{ t('vmCreate.enableCloudInit') }}
              </label>
              <template v-if="cloudInit.enabled">
                <div>
                  <label class="text-sm mb-1 block">{{ t('vmCreate.hostname') }}</label>
                  <Input v-model="cloudInit.hostname" :placeholder="form.name || 'my-vm'" />
                </div>
                <div class="grid grid-cols-2 gap-3">
                  <div>
                    <label class="text-sm mb-1 block">{{ t('vmCreate.username') }}</label>
                    <Input v-model="cloudInit.user" placeholder="user" />
                  </div>
                  <div>
                    <label class="text-sm mb-1 block">{{ t('vmCreate.password') }}</label>
                    <Input v-model="cloudInit.password" type="password" placeholder="(可选)" />
                  </div>
                </div>
                <div>
                  <label class="text-sm mb-1 block">{{ t('vmCreate.sshKey') }}</label>
                  <textarea
                    v-model="cloudInit.sshKey"
                    class="flex w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                    rows="2"
                    placeholder="ssh-rsa AAAA... user@host"
                  />
                </div>
                <div>
                  <label class="text-sm mb-1 block">{{ t('vmCreate.customUserData') }}</label>
                  <textarea
                    v-model="cloudInit.userData"
                    class="flex w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm font-mono shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                    rows="4"
                    placeholder="#cloud-config&#10;packages:&#10;  - nginx"
                  />
                  <p class="text-xs text-muted-foreground mt-1">{{ t('vmCreate.userDataOverride') }}</p>
                </div>
              </template>
            </div>
          </section>
        </div>

        <!-- 操作按钮 -->
        <div class="flex justify-end gap-2 p-5 border-t sticky bottom-0 bg-card">
          <Button variant="outline" @click="close">{{ t('common.cancel') }}</Button>
          <Button :loading="creating" @click="submit">{{ t('common.create') }}</Button>
        </div>
      </div>
    </div>
  </Teleport>
</template>
