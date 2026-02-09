# 后端模块扩展

> 创建: 2026-02-08
> 状态: 进行中

## 目标

创建 monitor、terminal、snapshot、settings 等后端模块，扩展 SSH client 和 store 能力。

## 步骤

- [ ] 1. 创建 internal/monitor/monitor.go - 宿主机资源监控采集器
- [ ] 2. 创建 internal/vm/snapshot.go - 快照管理模块
- [ ] 3. 创建 internal/store/settings.go - 设置项 CRUD
- [ ] 4. 创建 internal/terminal/server.go - WebSocket 终端服务
- [ ] 5. 修改 internal/ssh/client.go - 添加 ShellSession 和 OpenShell
- [ ] 6. 修改 internal/store/host.go - 添加导入导出和排序方法
- [ ] 7. 编译验证 go build ./...

## 完成标准

- [ ] 所有文件创建/修改完成
- [ ] go build ./... 编译通过
- [ ] 代码风格与现有代码一致
- [ ] 注释使用中文

## 备注

涉及 6 个文件（4 个新建，2 个修改），需要注意 import 兼容性。
