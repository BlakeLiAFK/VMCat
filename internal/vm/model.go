package vm

// VM 虚拟机数据模型
type VM struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	State    string `json:"state"` // running | shut off | paused | idle | crashed | ...
	CPUs     int    `json:"cpus"`
	MemoryMB int    `json:"memoryMB"`
	HostID   string `json:"hostID"`
}

// VMDetail 虚拟机详细信息
type VMDetail struct {
	VM
	Autostart bool   `json:"autostart"`
	VNCPort   int    `json:"vncPort"`
	NICs      []NIC  `json:"nics"`
	Disks     []Disk `json:"disks"`
}

// NIC 网络接口
type NIC struct {
	MAC     string `json:"mac"`
	Bridge  string `json:"bridge"`
	Network string `json:"network"`
	IP      string `json:"ip"`
	Model   string `json:"model"`
}

// Disk 磁盘信息
type Disk struct {
	Device string  `json:"device"`
	Path   string  `json:"path"`
	SizeGB float64 `json:"sizeGB"`
	Format string  `json:"format"`
}

// Snapshot 快照
type Snapshot struct {
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
	State     string `json:"state"`
	Parent    string `json:"parent"`
}

// StoragePool 存储池
type StoragePool struct {
	Name       string `json:"name"`
	State      string `json:"state"`
	Autostart  string `json:"autostart"`
	Persistent string `json:"persistent"`
	Capacity   string `json:"capacity"`
	Allocation string `json:"allocation"`
	Available  string `json:"available"`
}

// Volume 存储卷
type Volume struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Type       string `json:"type"`
	Capacity   string `json:"capacity"`
	Allocation string `json:"allocation"`
}

// Network 虚拟网络
type Network struct {
	Name       string `json:"name"`
	State      string `json:"state"`
	Autostart  string `json:"autostart"`
	Persistent string `json:"persistent"`
	Bridge     string `json:"bridge"`
}

// ISOFile ISO 镜像文件
type ISOFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size string `json:"size"`
}

// VMCreateParams 创建 VM 参数
type VMCreateParams struct {
	Name       string `json:"name"`
	CPUs       int    `json:"cpus"`
	MemoryMB   int    `json:"memoryMB"`
	DiskPath   string `json:"diskPath"`
	DiskSizeGB int    `json:"diskSizeGB"`
	CDROM      string `json:"cdrom"`
	Network    string `json:"network"`
	NetType    string `json:"netType"`
	OSVariant  string `json:"osVariant"`
	VNC        bool   `json:"vnc"`
	BootDev    string `json:"bootDev"`
}

// DiskAttachParams 添加磁盘参数
type DiskAttachParams struct {
	Source  string `json:"source"`
	Target  string `json:"target"`
	Driver  string `json:"driver"`
	Cache   string `json:"cache"`
	DevType string `json:"devType"`
}

// NICAttachParams 添加网卡参数
type NICAttachParams struct {
	Type   string `json:"type"`
	Source string `json:"source"`
	Model  string `json:"model"`
}

// NATRule NAT 端口转发规则
type NATRule struct {
	Proto    string `json:"proto"`    // tcp | udp
	HostPort string `json:"hostPort"` // 宿主机端口或端口范围 (如 "8080" 或 "8080:8090")
	VMIP     string `json:"vmIP"`
	VMPort   string `json:"vmPort"` // VM 端口或端口范围
	Comment  string `json:"comment"`
}

// VMResourceStats VM 实时资源统计
type VMResourceStats struct {
	CPUTime      uint64  `json:"cpuTime"`
	CPUPercent   float64 `json:"cpuPercent"`
	VCPUs        int     `json:"vcpus"`
	MemActual    uint64  `json:"memActual"`
	MemRSS       uint64  `json:"memRSS"`
	NetRxBytes   uint64  `json:"netRxBytes"`
	NetTxBytes   uint64  `json:"netTxBytes"`
	BlockRdBytes uint64  `json:"blockRdBytes"`
	BlockWrBytes uint64  `json:"blockWrBytes"`
}
