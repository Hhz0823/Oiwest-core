export namespace services {
	
	export class DNSServerItem {
	    address: string;
	    port: number;
	    domains?: string[];
	    expectIPs?: string[];
	    skipFallback: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DNSServerItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.address = source["address"];
	        this.port = source["port"];
	        this.domains = source["domains"];
	        this.expectIPs = source["expectIPs"];
	        this.skipFallback = source["skipFallback"];
	    }
	}
	export class DNSConfig {
	    servers: DNSServerItem[];
	    hosts?: Record<string, string>;
	    clientIp?: string;
	    tag?: string;
	    queryStrategy: string;
	    disableCache: boolean;
	    disableFallback: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DNSConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.servers = this.convertValues(source["servers"], DNSServerItem);
	        this.hosts = source["hosts"];
	        this.clientIp = source["clientIp"];
	        this.tag = source["tag"];
	        this.queryStrategy = source["queryStrategy"];
	        this.disableCache = source["disableCache"];
	        this.disableFallback = source["disableFallback"];
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
	
	export class InboundSettings {
	    auth?: string;
	    udp: boolean;
	    user?: string;
	    pass?: string;
	    method?: string;
	    password?: string;
	
	    static createFrom(source: any = {}) {
	        return new InboundSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.auth = source["auth"];
	        this.udp = source["udp"];
	        this.user = source["user"];
	        this.pass = source["pass"];
	        this.method = source["method"];
	        this.password = source["password"];
	    }
	}
	export class InboundRule {
	    id: string;
	    tag: string;
	    port: number;
	    listen: string;
	    protocol: string;
	    enabled: boolean;
	    settings: InboundSettings;
	
	    static createFrom(source: any = {}) {
	        return new InboundRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.tag = source["tag"];
	        this.port = source["port"];
	        this.listen = source["listen"];
	        this.protocol = source["protocol"];
	        this.enabled = source["enabled"];
	        this.settings = this.convertValues(source["settings"], InboundSettings);
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
	
	export class LatencyResult {
	    nodeId: string;
	    latency: number;
	    success: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new LatencyResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nodeId = source["nodeId"];
	        this.latency = source["latency"];
	        this.success = source["success"];
	        this.error = source["error"];
	    }
	}
	export class ProxySettings {
	    mode: string;
	    proxyHost: string;
	    proxyPort: number;
	    bypassLocal: boolean;
	    pacUrl?: string;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ProxySettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.proxyHost = source["proxyHost"];
	        this.proxyPort = source["proxyPort"];
	        this.bypassLocal = source["bypassLocal"];
	        this.pacUrl = source["pacUrl"];
	        this.enabled = source["enabled"];
	    }
	}
	export class RoutingRule {
	    id: string;
	    name: string;
	    type: string;
	    domain: string[];
	    ip: string[];
	    port: string;
	    network: string;
	    protocol: string[];
	    inboundTag: string[];
	    outboundTag: string;
	    enabled: boolean;
	    sort: number;
	
	    static createFrom(source: any = {}) {
	        return new RoutingRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.domain = source["domain"];
	        this.ip = source["ip"];
	        this.port = source["port"];
	        this.network = source["network"];
	        this.protocol = source["protocol"];
	        this.inboundTag = source["inboundTag"];
	        this.outboundTag = source["outboundTag"];
	        this.enabled = source["enabled"];
	        this.sort = source["sort"];
	    }
	}
	export class ServerGroup {
	    name: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new ServerGroup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.count = source["count"];
	    }
	}
	export class ServerNode {
	    id: string;
	    name: string;
	    group: string;
	    protocol: string;
	    address: string;
	    port: number;
	    uuid?: string;
	    password?: string;
	    security?: string;
	    flow?: string;
	    network?: string;
	    path?: string;
	    host?: string;
	    tls: boolean;
	    sni?: string;
	    fingerprint?: string;
	    publicKey?: string;
	    shortId?: string;
	    spiderX?: string;
	    allowInsecure: boolean;
	    latency: number;
	    upload: number;
	    download: number;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new ServerNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.group = source["group"];
	        this.protocol = source["protocol"];
	        this.address = source["address"];
	        this.port = source["port"];
	        this.uuid = source["uuid"];
	        this.password = source["password"];
	        this.security = source["security"];
	        this.flow = source["flow"];
	        this.network = source["network"];
	        this.path = source["path"];
	        this.host = source["host"];
	        this.tls = source["tls"];
	        this.sni = source["sni"];
	        this.fingerprint = source["fingerprint"];
	        this.publicKey = source["publicKey"];
	        this.shortId = source["shortId"];
	        this.spiderX = source["spiderX"];
	        this.allowInsecure = source["allowInsecure"];
	        this.latency = source["latency"];
	        this.upload = source["upload"];
	        this.download = source["download"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class TrafficStats {
	    uploadSpeed: number;
	    downloadSpeed: number;
	    totalUpload: number;
	    totalDownload: number;
	    activeConns: number;
	    packetsSent: number;
	    packetsRecv: number;
	
	    static createFrom(source: any = {}) {
	        return new TrafficStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uploadSpeed = source["uploadSpeed"];
	        this.downloadSpeed = source["downloadSpeed"];
	        this.totalUpload = source["totalUpload"];
	        this.totalDownload = source["totalDownload"];
	        this.activeConns = source["activeConns"];
	        this.packetsSent = source["packetsSent"];
	        this.packetsRecv = source["packetsRecv"];
	    }
	}
	export class TransparentProxyConfig {
	    enabled: boolean;
	    mode: string;
	    redirectTcp: number;
	    redirectUdp: number;
	    bypassLan: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TransparentProxyConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.mode = source["mode"];
	        this.redirectTcp = source["redirectTcp"];
	        this.redirectUdp = source["redirectUdp"];
	        this.bypassLan = source["bypassLan"];
	    }
	}

}

