export namespace monitor {
	
	export class HostStats {
	    cpuPercent: number;
	    memTotal: number;
	    memUsed: number;
	    memPercent: number;
	    diskTotal: number;
	    diskUsed: number;
	    diskPercent: number;
	    uptime: string;
	    loadAvg: string;
	
	    static createFrom(source: any = {}) {
	        return new HostStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cpuPercent = source["cpuPercent"];
	        this.memTotal = source["memTotal"];
	        this.memUsed = source["memUsed"];
	        this.memPercent = source["memPercent"];
	        this.diskTotal = source["diskTotal"];
	        this.diskUsed = source["diskUsed"];
	        this.diskPercent = source["diskPercent"];
	        this.uptime = source["uptime"];
	        this.loadAvg = source["loadAvg"];
	    }
	}

}

export namespace store {
	
	export class Flavor {
	    id: string;
	    name: string;
	    cpus: number;
	    memoryMB: number;
	    diskGB: number;
	    sortOrder: number;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Flavor(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.cpus = source["cpus"];
	        this.memoryMB = source["memoryMB"];
	        this.diskGB = source["diskGB"];
	        this.sortOrder = source["sortOrder"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class Host {
	    id: string;
	    name: string;
	    host: string;
	    port: number;
	    user: string;
	    authType: string;
	    keyPath: string;
	    password: string;
	    hostKey: string;
	    proxyAddr: string;
	    sortOrder: number;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Host(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.user = source["user"];
	        this.authType = source["authType"];
	        this.keyPath = source["keyPath"];
	        this.password = source["password"];
	        this.hostKey = source["hostKey"];
	        this.proxyAddr = source["proxyAddr"];
	        this.sortOrder = source["sortOrder"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class Image {
	    id: string;
	    hostId: string;
	    name: string;
	    basePath: string;
	    osVariant: string;
	    sortOrder: number;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Image(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.hostId = source["hostId"];
	        this.name = source["name"];
	        this.basePath = source["basePath"];
	        this.osVariant = source["osVariant"];
	        this.sortOrder = source["sortOrder"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class Instance {
	    id: number;
	    hostId: string;
	    vmName: string;
	    flavorId: string;
	    imageId: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Instance(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.hostId = source["hostId"];
	        this.vmName = source["vmName"];
	        this.flavorId = source["flavorId"];
	        this.imageId = source["imageId"];
	        this.createdAt = source["createdAt"];
	    }
	}

}

export namespace vm {
	
	export class Disk {
	    device: string;
	    path: string;
	    sizeGB: number;
	    format: string;
	
	    static createFrom(source: any = {}) {
	        return new Disk(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.device = source["device"];
	        this.path = source["path"];
	        this.sizeGB = source["sizeGB"];
	        this.format = source["format"];
	    }
	}
	export class DiskAttachParams {
	    source: string;
	    target: string;
	    driver: string;
	    cache: string;
	    devType: string;
	
	    static createFrom(source: any = {}) {
	        return new DiskAttachParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = source["source"];
	        this.target = source["target"];
	        this.driver = source["driver"];
	        this.cache = source["cache"];
	        this.devType = source["devType"];
	    }
	}
	export class ISOFile {
	    name: string;
	    path: string;
	    size: string;
	
	    static createFrom(source: any = {}) {
	        return new ISOFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.size = source["size"];
	    }
	}
	export class NIC {
	    mac: string;
	    bridge: string;
	    network: string;
	    ip: string;
	    model: string;
	
	    static createFrom(source: any = {}) {
	        return new NIC(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mac = source["mac"];
	        this.bridge = source["bridge"];
	        this.network = source["network"];
	        this.ip = source["ip"];
	        this.model = source["model"];
	    }
	}
	export class NICAttachParams {
	    type: string;
	    source: string;
	    model: string;
	
	    static createFrom(source: any = {}) {
	        return new NICAttachParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.source = source["source"];
	        this.model = source["model"];
	    }
	}
	export class Network {
	    name: string;
	    state: string;
	    autostart: string;
	    persistent: string;
	    bridge: string;
	
	    static createFrom(source: any = {}) {
	        return new Network(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.state = source["state"];
	        this.autostart = source["autostart"];
	        this.persistent = source["persistent"];
	        this.bridge = source["bridge"];
	    }
	}
	export class Snapshot {
	    name: string;
	    createdAt: string;
	    state: string;
	    parent: string;
	
	    static createFrom(source: any = {}) {
	        return new Snapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.createdAt = source["createdAt"];
	        this.state = source["state"];
	        this.parent = source["parent"];
	    }
	}
	export class StoragePool {
	    name: string;
	    state: string;
	    autostart: string;
	    persistent: string;
	    capacity: string;
	    allocation: string;
	    available: string;
	
	    static createFrom(source: any = {}) {
	        return new StoragePool(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.state = source["state"];
	        this.autostart = source["autostart"];
	        this.persistent = source["persistent"];
	        this.capacity = source["capacity"];
	        this.allocation = source["allocation"];
	        this.available = source["available"];
	    }
	}
	export class VM {
	    id: number;
	    name: string;
	    state: string;
	    cpus: number;
	    memoryMB: number;
	    hostID: string;
	
	    static createFrom(source: any = {}) {
	        return new VM(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.state = source["state"];
	        this.cpus = source["cpus"];
	        this.memoryMB = source["memoryMB"];
	        this.hostID = source["hostID"];
	    }
	}
	export class VMCreateParams {
	    name: string;
	    cpus: number;
	    memoryMB: number;
	    diskPath: string;
	    diskSizeGB: number;
	    cdrom: string;
	    network: string;
	    netType: string;
	    osVariant: string;
	    vnc: boolean;
	    bootDev: string;
	
	    static createFrom(source: any = {}) {
	        return new VMCreateParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.cpus = source["cpus"];
	        this.memoryMB = source["memoryMB"];
	        this.diskPath = source["diskPath"];
	        this.diskSizeGB = source["diskSizeGB"];
	        this.cdrom = source["cdrom"];
	        this.network = source["network"];
	        this.netType = source["netType"];
	        this.osVariant = source["osVariant"];
	        this.vnc = source["vnc"];
	        this.bootDev = source["bootDev"];
	    }
	}
	export class VMDetail {
	    id: number;
	    name: string;
	    state: string;
	    cpus: number;
	    memoryMB: number;
	    hostID: string;
	    autostart: boolean;
	    vncPort: number;
	    nics: NIC[];
	    disks: Disk[];
	
	    static createFrom(source: any = {}) {
	        return new VMDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.state = source["state"];
	        this.cpus = source["cpus"];
	        this.memoryMB = source["memoryMB"];
	        this.hostID = source["hostID"];
	        this.autostart = source["autostart"];
	        this.vncPort = source["vncPort"];
	        this.nics = this.convertValues(source["nics"], NIC);
	        this.disks = this.convertValues(source["disks"], Disk);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class VMResourceStats {
	    cpuTime: number;
	    cpuPercent: number;
	    vcpus: number;
	    memActual: number;
	    memRSS: number;
	    netRxBytes: number;
	    netTxBytes: number;
	    blockRdBytes: number;
	    blockWrBytes: number;
	
	    static createFrom(source: any = {}) {
	        return new VMResourceStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cpuTime = source["cpuTime"];
	        this.cpuPercent = source["cpuPercent"];
	        this.vcpus = source["vcpus"];
	        this.memActual = source["memActual"];
	        this.memRSS = source["memRSS"];
	        this.netRxBytes = source["netRxBytes"];
	        this.netTxBytes = source["netTxBytes"];
	        this.blockRdBytes = source["blockRdBytes"];
	        this.blockWrBytes = source["blockWrBytes"];
	    }
	}
	export class Volume {
	    name: string;
	    path: string;
	    type: string;
	    capacity: string;
	    allocation: string;
	
	    static createFrom(source: any = {}) {
	        return new Volume(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.type = source["type"];
	        this.capacity = source["capacity"];
	        this.allocation = source["allocation"];
	    }
	}

}

