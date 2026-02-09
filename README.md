# VMCat

Lightweight KVM virtual machine management tool via SSH.

Through SSH connection to KVM hosts, VMCat provides a unified graphical interface for virtual machine lifecycle management, hardware configuration, remote access, and resource monitoring.

## Features

### Host Management
- Multi-host management with SSH key/password authentication
- Connection pool with automatic reconnection
- Real-time resource monitoring (CPU, memory, disk, load)
- Host configuration import/export (JSON)

### VM Lifecycle
- Create VMs via `virt-install` with full parameter support
- Start, shutdown, reboot, suspend, resume, force stop
- Clone, rename, delete (with optional storage cleanup)
- Autostart configuration
- XML configuration editor

### Hardware Management
- Disk: attach, detach, resize (qemu-img)
- NIC: attach, detach (bridge/network mode)
- CD-ROM: mount/eject ISO
- VNC display: enable/disable

### Remote Access
- **SSH Terminal** - Full-featured terminal via xterm.js
- **VNC Remote Desktop** - In-app VNC viewer via noVNC
- **VM Serial Console** - `virsh console` integration
- All connections tunneled through SSH (no direct port exposure needed)

### Storage & Network
- Storage pool and volume management
- Virtual network and bridge listing
- ISO image discovery

### Snapshots
- Create, revert, delete snapshots

### Settings
- Dark/Light theme
- Host configuration backup/restore

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Framework | [Wails](https://wails.io/) v2 |
| Backend | Go 1.23 |
| Frontend | Vue 3 + TypeScript + Vite |
| UI | Tailwind CSS + Radix Vue + Lucide Icons |
| Terminal | xterm.js |
| VNC | noVNC 1.5 |
| Database | SQLite (WAL mode) |
| Transport | SSH + WebSocket |

## Architecture

```
+-------------------+     +--------------------+     +------------------+
|     Frontend      |     |      Backend       |     |    KVM Host      |
|                   |     |                    |     |                  |
|  Vue 3 + TS       |     |  Go (Wails IPC)    |     |  libvirt/virsh   |
|  xterm.js   ------+---->|  WebSocket Server  |---->|  VNC Server      |
|  noVNC      ------+---->|  SSH Pool/Tunnel   |---->|  Shell (PTY)     |
|  Pinia Store      |     |  SQLite Store      |     |                  |
+-------------------+     +--------------------+     +------------------+
        |                         |
        |    Wails IPC Binding    |
        +-------------------------+
```

**Remote Access Data Flow:**

```
Browser (xterm.js)  --> WebSocket --> Go Proxy --> SSH PTY  --> Host Shell
Browser (noVNC)     --> WebSocket --> Go Proxy --> SSH Tunnel --> VNC Server
```

## Project Structure

```
vmcat/
|-- main.go                     # Entry point, Wails app setup
|-- app.go                      # IPC binding layer (47 exported methods)
|-- wails.json                  # Wails configuration
|-- internal/
|   |-- ssh/
|   |   |-- client.go           # SSH client (connect, execute, PTY, tunnel)
|   |   +-- pool.go             # Connection pool management
|   |-- terminal/
|   |   +-- server.go           # WebSocket server (/ws/terminal, /ws/vnc)
|   |-- vm/
|   |   |-- manager.go          # VM lifecycle operations
|   |   |-- virsh.go            # virsh output parser
|   |   |-- model.go            # Data structures
|   |   |-- create.go           # virt-install wrapper
|   |   |-- hardware.go         # Disk, NIC, CD-ROM management
|   |   |-- storage.go          # Storage pool/volume
|   |   |-- network.go          # Virtual network/bridge
|   |   |-- snapshot.go         # Snapshot operations
|   |   +-- stats.go            # VM resource statistics
|   |-- store/
|   |   |-- store.go            # SQLite initialization
|   |   |-- host.go             # Host CRUD + import/export
|   |   +-- settings.go         # Key-value settings
|   +-- monitor/
|       +-- monitor.go          # Host resource collector
+-- frontend/
    |-- src/
    |   |-- App.vue
    |   |-- main.ts
    |   |-- router/index.ts     # 6 routes
    |   |-- stores/app.ts       # Pinia store
    |   |-- composables/        # useToast, useTheme
    |   |-- views/
    |   |   |-- Dashboard.vue       # Overview
    |   |   |-- HostDetail.vue      # Host info + VM list
    |   |   |-- VMDetail.vue        # VM info + snapshots
    |   |   |-- SSHTerminal.vue     # SSH terminal
    |   |   |-- VNCViewer.vue       # VNC remote desktop
    |   |   +-- Settings.vue        # App settings
    |   +-- components/
    |       |-- Sidebar.vue
    |       |-- HostFormDialog.vue
    |       |-- VMCreateDialog.vue
    |       |-- VMEditDialog.vue
    |       |-- VMHardwareDialog.vue
    |       |-- VMCloneDialog.vue
    |       |-- VMXMLDialog.vue
    |       +-- ui/                 # Base UI components
    +-- wailsjs/                    # Auto-generated Wails bindings
```

## Prerequisites

- Go 1.22+
- Node.js 20+
- Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

### Linux Build Dependencies

```bash
# Ubuntu/Debian
sudo apt-get install -y libgtk-3-dev libwebkit2gtk-4.1-dev

# Fedora/RHEL
sudo dnf install gtk3-devel webkit2gtk4.1-devel
```

### KVM Host Requirements

- libvirt + QEMU/KVM
- `virsh`, `virt-install`, `qemu-img` commands available
- SSH access (key or password)

## Development

```bash
# Install dependencies
go mod download
cd frontend && npm install && cd ..

# Run in development mode
wails dev
```

## Build

```bash
# macOS
wails build
xattr -cr build/bin/vmcat.app  # Remove quarantine attribute

# Linux
wails build -tags webkit2_41

# Windows
wails build -platform windows/amd64
```

## Data Storage

Application data is stored at `~/.vmcat/vmcat.db` (SQLite).

## License

Copyright (c) 2026 blake. All rights reserved.
