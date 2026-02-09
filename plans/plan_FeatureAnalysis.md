# VMCat 功能分析与开发规划

> 创建: 2026-02-09
> 状态: 讨论中

## 一、现有功能盘点

### 后端 (Go)

| 模块        | 功能                                               | 状态               |
| ----------- | -------------------------------------------------- | ------------------ |
| 宿主机管理  | 增删改查、连接/断开、测试、导入导出                | OK                 |
| VM 生命周期 | 列表、详情、启动、关机、强制关闭、重启、暂停、恢复 | OK                 |
| VM 设置     | SetAutostart                                       | 后端有，前端未暴露 |
| 快照管理    | 列表、创建、删除、恢复                             | OK                 |
| 监控        | 宿主机资源统计 (CPU/MEM/Disk)                      | OK                 |
| 终端        | SSH WebSocket 终端                                 | OK                 |
| VNC         | noVNC + SSH 隧道代理                               | OK (待实测)        |
| 设置        | 键值对存储                                         | OK                 |

### 前端 (Vue3)

| 页面        | 功能                          | 状态         |
| ----------- | ----------------------------- | ------------ |
| Dashboard   | 宿主机概览                    | OK           |
| HostDetail  | VM 列表 + 资源监控 + 批量操作 | OK           |
| VMDetail    | 查看配置 + 快照管理           | 只读，无编辑 |
| SSHTerminal | SSH 终端                      | OK           |
| VNCViewer   | VNC 远程桌面                  | OK (待实测)  |
| Settings    | 设置                          | OK           |

---

## 二、缺失功能分析

### P0 - 核心缺失 (创建/删除/编辑 VM)

#### 1. 创建虚拟机

- **后端**: 无任何创建 VM 的代码
- **需要**: `virt-install` 命令封装
- **前端**: 创建 VM 向导表单
- **依赖**: 存储池管理、网络列表、ISO 镜像列表
- **virsh 命令**: `virt-install --name xxx --vcpus 2 --memory 2048 --disk path=xxx --cdrom xxx --network bridge=br0 --graphics vnc`

#### 2. 删除虚拟机

- **后端**: 无 `virsh undefine` 功能
- **需要**: VMDelete 方法 (undefine + 可选删除磁盘)
- **前端**: VMDetail 页面添加删除按钮

#### 3. 编辑 VM 配置

目前 VMDetail 页面是**纯只读**的，以下配置都无法修改：

| 配置项    | virsh 命令                   | 复杂度 |
| --------- | ---------------------------- | ------ |
| CPU 数量  | `virsh setvcpus`             | 低     |
| 内存大小  | `virsh setmem` / edit XML    | 低     |
| Autostart | `virsh autostart` (后端已有) | 低     |
| 添加磁盘  | `virsh attach-disk`          | 中     |
| 删除磁盘  | `virsh detach-disk`          | 中     |
| 添加网卡  | `virsh attach-interface`     | 中     |
| 删除网卡  | `virsh detach-interface`     | 中     |
| VNC 开关  | 编辑 XML (graphics 设备)     | 中     |
| Boot 顺序 | 编辑 XML                     | 中     |
| 光驱挂载  | `virsh change-media`         | 低     |

### P1 - 重要功能

#### 4. 存储池/卷管理

- `virsh pool-list` / `virsh vol-list <pool>`
- 创建 VM 时需要选存储位置
- 创建磁盘镜像 `qemu-img create`

#### 5. 网络管理

- `virsh net-list` / `virsh net-info`
- 创建 VM / 添加网卡时需要选网络

#### 6. ISO 镜像管理

- 列出宿主机上的 ISO 文件
- 创建 VM 时需要选 ISO 安装源

#### 7. VM 克隆

- `virt-clone --original xxx --name xxx --auto-clone`
- 快速复制已有 VM

#### 8. VM 资源实时统计

- `virsh domstats <name>` (CPU/MEM/IO 实时数据)
- 设计文档已提到但未实现

### P2 - 锦上添花

#### 9. VM XML 配置查看/编辑

- 查看原始 XML (`virsh dumpxml`)
- 高级用户直接编辑 XML (`virsh define`)

#### 10. VM 迁移

- `virsh migrate` (在线迁移到其他宿主机)

#### 11. 其他小功能

- VM 重命名 (`virsh domrename`)
- 控制台日志查看
- VM 配置导出/导入
- 磁盘扩容 (`qemu-img resize`)

---

## 三、建议开发优先级

### 第一批: VM 基础 CRUD (让功能闭环)

1. **删除 VM** - 最简单，一个命令
2. **Autostart 切换** - 后端已有，加前端按钮即可
3. **编辑 CPU/内存** - 常用操作
4. **光驱挂载/弹出** - 安装系统必备

### 第二批: 创建 VM (核心能力)

5. **列出存储池/网络** - 创建 VM 的前置
6. **列出 ISO 镜像** - 创建 VM 的前置
7. **创建 VM 向导** - 完整的创建流程

### 第三批: 硬件热插拔

8. **添加/删除磁盘**
9. **添加/删除网卡**
10. **VNC 开关配置**

### 第四批: 高级功能

11. **VM 克隆**
12. **VM 资源实时统计**
13. **XML 配置查看/编辑**
14. **磁盘扩容**

---

## 四、执行计划

### 后端 Go

| #   | 文件                      | 内容                                                               | 状态 |
| --- | ------------------------- | ------------------------------------------------------------------ | ---- |
| 1   | `internal/vm/model.go`    | 新增 StoragePool, Volume, Network, VMCreateParams 等结构           | 完成 |
| 2   | `internal/vm/manager.go`  | 新增 Delete, Rename, SetVCPUs, SetMemory, GetXML, DefineXML, Clone | 完成 |
| 3   | `internal/vm/hardware.go` | 新建: 磁盘/网卡增删, CD-ROM, VNC 开关, 磁盘扩容                    | 完成 |
| 4   | `internal/vm/storage.go`  | 新建: PoolList, VolList, CreateVolume                              | 完成 |
| 5   | `internal/vm/network.go`  | 新建: NetworkList, BridgeList                                      | 完成 |
| 6   | `internal/vm/create.go`   | 新建: Create(virt-install), ISOList, OSVariantList                 | 完成 |
| 7   | `internal/vm/stats.go`    | 新建: VMStats (domstats)                                           | 完成 |
| 8   | `app.go`                  | 添加所有新绑定方法 (~30 个)                                        | 完成 |

### 前端 Vue

| #   | 文件                              | 内容                          | 状态 |
| --- | --------------------------------- | ----------------------------- | ---- |
| 9   | `components/VMCreateDialog.vue`   | 创建 VM 向导                  | 完成 |
| 10  | `views/VMDetail.vue`              | 重构: 编辑/删除/克隆/硬件管理 | 完成 |
| 11  | `components/VMEditDialog.vue`     | 编辑 CPU/内存/Autostart       | 完成 |
| 12  | `components/VMHardwareDialog.vue` | 磁盘/网卡添加对话框           | 完成 |
| 13  | `components/VMXMLDialog.vue`      | XML 查看/编辑                 | 完成 |
| 14  | `components/VMCloneDialog.vue`    | 克隆对话框                    | 完成 |
| 15  | `views/HostDetail.vue`            | 添加创建 VM 按钮              | 完成 |

### 验证

| #   | 内容                             | 状态         |
| --- | -------------------------------- | ------------ |
| 16  | `wails generate module` 生成绑定 | 完成         |
| 17  | `wails build` 编译验证           | 完成 (1m11s) |
