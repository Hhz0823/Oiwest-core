package config

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/Hhz0823/oiwest-core/app/common/net"
	"github.com/Hhz0823/oiwest-core/app/common/tls"
	"github.com/Hhz0823/oiwest-core/app/transport"
	"github.com/Hhz0823/oiwest-core/app/transport/bbr"
)

var (
	ErrInvalidConfig = errors.New("invalid configuration")
)

type Config struct {
	Log          *LogConfig          `json:"log"`
	Inbounds     []InboundConfig     `json:"inbounds"`
	Outbounds    []OutboundConfig    `json:"outbounds"`
	Routing      *RoutingConfig      `json:"routing"`
	DNS          *DNSConfig          `json:"dns"`
	Stats        *StatsConfig        `json:"stats"`
	Policy       *PolicyConfig       `json:"policy"`
	API          *APIConfig          `json:"api"`
	Transport    *TransportConfig    `json:"transport"`
	Observatory  *ObservatoryConfig  `json:"observatory"`
	Reverse      *ReverseConfig      `json:"reverse"`
	WorkerPool   *WorkerPoolConfig   `json:"workerPool"`
	BBR          *BBRGlobalConfig    `json:"bbr"`
	DualStack    *DualStackConfig    `json:"dualStack"`
	Certificate  *CertificateConfig  `json:"certificate"`
}

type LogConfig struct {
	Access   string `json:"access"`
	Error    string `json:"error"`
	LogLevel string `json:"loglevel"`
}

type InboundConfig struct {
	Tag            string                    `json:"tag"`
	Port           uint16                    `json:"port"`
	Listen         string                    `json:"listen"`
	Protocol       string                    `json:"protocol"`
	Settings       json.RawMessage           `json:"settings"`
	StreamSettings *transport.StreamSettings `json:"streamSettings"`
	Sniffing       *SniffingConfig           `json:"sniffing"`
	Allocate       *AllocateConfig           `json:"allocate"`
}

type OutboundConfig struct {
	Tag            string                    `json:"tag"`
	Protocol       string                    `json:"protocol"`
	Settings       json.RawMessage           `json:"settings"`
	StreamSettings *transport.StreamSettings `json:"streamSettings"`
	Proxy          *ProxyConfig              `json:"proxy"`
	Mux            *MuxConfig                `json:"mux"`
	SendThrough    string                    `json:"sendThrough"`
}

type RoutingConfig struct {
	DomainStrategy string          `json:"domainStrategy"`
	DomainMatcher  string          `json:"domainMatcher"`
	Rules          []RoutingRule   `json:"rules"`
	Balancers      []BalancerConfig `json:"balancers"`
}

type RoutingRule struct {
	Type        string   `json:"type"`
	Domain      []string `json:"domain"`
	IP          []string `json:"ip"`
	Port        string   `json:"port"`
	SourcePort  string   `json:"sourcePort"`
	Network     string   `json:"network"`
	Source      []string `json:"source"`
	User        []string `json:"user"`
	InboundTag  []string `json:"inboundTag"`
	Protocol    []string `json:"protocol"`
	Attributes  string   `json:"attrs"`
	OutboundTag string   `json:"outboundTag"`
	BalancerTag string   `json:"balancerTag"`
}

type BalancerConfig struct {
	Tag      string            `json:"tag"`
	Selector []string          `json:"selector"`
	Strategy *BalancerStrategy `json:"strategy"`
}

type BalancerStrategy struct {
	Type string `json:"type"`
}

type DNSConfig struct {
	Servers         []DNSServer       `json:"servers"`
	Hosts           map[string]string `json:"hosts"`
	ClientIp        string            `json:"clientIp"`
	Tag             string            `json:"tag"`
	QueryStrategy   string            `json:"queryStrategy"`
	DisableCache    bool              `json:"disableCache"`
	DisableFallback bool              `json:"disableFallback"`
}

type DNSServer struct {
	Address       string   `json:"address"`
	Port          uint16   `json:"port"`
	Domains       []string `json:"domains"`
	ExpectIPs     []string `json:"expectIPs"`
	SkipFallback  bool     `json:"skipFallback"`
	ClientIp      string   `json:"clientIp"`
	Tag           string   `json:"tag"`
}

type StatsConfig struct {
	Enabled bool `json:"enabled"`
}

type PolicyConfig struct {
	System *SystemPolicy           `json:"system"`
	Levels map[string]*LevelPolicy `json:"levels"`
}

type SystemPolicy struct {
	StatsInboundUplink    bool `json:"statsInboundUplink"`
	StatsInboundDownlink  bool `json:"statsInboundDownlink"`
	StatsOutboundUplink   bool `json:"statsOutboundUplink"`
	StatsOutboundDownlink bool `json:"statsOutboundDownlink"`
}

type LevelPolicy struct {
	Handshake         int  `json:"handshake"`
	ConnIdle          int  `json:"connIdle"`
	UplinkOnly        int  `json:"uplinkOnly"`
	DownlinkOnly      int  `json:"downlinkOnly"`
	StatsUserUplink   bool `json:"statsUserUplink"`
	StatsUserDownlink bool `json:"statsUserDownlink"`
	BufferSize        int  `json:"bufferSize"`
}

type APIConfig struct {
	Tag      string   `json:"tag"`
	Services []string `json:"services"`
}

type TransportConfig struct {
	TCPSettings   *transport.TCPSettings    `json:"tcpSettings"`
	DCCPSettings  *transport.DCCPSettings   `json:"dccpSettings"`
	WSSettings    *transport.WSSettings     `json:"wsSettings"`
	QUICSettings  *transport.QUICSettings   `json:"quicSettings"`
	GRPCSettings  *transport.GRPCSettings   `json:"grpcSettings"`
	HTTP2Settings *transport.HTTP2Settings  `json:"http2Settings"`
	KCPSettings   *transport.KCPConfig      `json:"kcpSettings"`
	XHTTPSettings *transport.XHTTPConfig    `json:"xhttpSettings"`
	BBRSettings   *bbr.BBRConfig            `json:"bbrSettings"`
}

type SniffingConfig struct {
	Enabled      bool     `json:"enabled"`
	DestOverride []string `json:"destOverride"`
	MetadataOnly bool     `json:"metadataOnly"`
	RouteOnly    bool     `json:"routeOnly"`
}

type AllocateConfig struct {
	Strategy    string `json:"strategy"`
	Refresh     int    `json:"refresh"`
	Concurrency int    `json:"concurrency"`
}

type ProxyConfig struct {
	Tag                 string `json:"tag"`
	TransportLayerProxy bool   `json:"transportLayerProxy"`
}

type MuxConfig struct {
	Enabled     bool `json:"enabled"`
	Concurrency int  `json:"concurrency"`
}

type ObservatoryConfig struct {
	SubjectSelector   []string      `json:"subjectSelector"`
	ProbeURL          string        `json:"probeUrl"`
	ProbeInterval     time.Duration `json:"probeInterval"`
}

type ReverseConfig struct {
	Bridges []ReverseBridge `json:"bridges"`
	Portals []ReversePortal `json:"portals"`
}

type ReverseBridge struct {
	Tag    string `json:"tag"`
	Domain string `json:"domain"`
}

type ReversePortal struct {
	Tag    string `json:"tag"`
	Domain string `json:"domain"`
}

type WorkerPoolConfig struct {
	NumWorkers  int           `json:"numWorkers"`
	QueueSize   int           `json:"queueSize"`
	MaxRetries  int           `json:"maxRetries"`
	IdleTimeout time.Duration `json:"idleTimeout"`
}

type BBRGlobalConfig struct {
	Enabled   bool                `json:"enabled"`
	Algorithm bbr.BBRAlgorithm    `json:"algorithm"`
	Settings  *bbr.BBRConfig      `json:"settings"`
}

type DualStackConfig struct {
	Enabled     bool              `json:"enabled"`
	Preference  string            `json:"preference"`
	MultiLine   bool              `json:"multiLine"`
	Failover    bool              `json:"failover"`
	Strategy    string            `json:"strategy"`
	Config      *net.DualStackConfig `json:"config"`
}

type CertificateConfig struct {
	Enabled     bool                    `json:"enabled"`
	AutoGenerate bool                   `json:"autoGenerate"`
	Config      *tls.CertificateConfig  `json:"config"`
}

func ParseConfig(data []byte) (*Config, error) {
	cfg := &Config{
		Log: &LogConfig{
			LogLevel: "warning",
		},
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) ToJSON() ([]byte, error) {
	return json.MarshalIndent(c, "", "  ")
}

func DefaultConfig() *Config {
	return &Config{
		Log: &LogConfig{
			LogLevel: "warning",
		},
		WorkerPool: &WorkerPoolConfig{
			NumWorkers:  8,
			QueueSize:   1000,
			MaxRetries:  3,
			IdleTimeout: 30,
		},
		BBR: &BBRGlobalConfig{
			Enabled:   false,
			Algorithm: bbr.BBROriginal,
			Settings:  bbr.DefaultBBRConfig(bbr.BBROriginal),
		},
		DualStack: &DualStackConfig{
			Enabled:    true,
			Preference: "dual",
			MultiLine:  false,
			Failover:   true,
			Strategy:   "latency",
		},
		Certificate: &CertificateConfig{
			Enabled:      false,
			AutoGenerate: true,
			Config:       tls.DefaultCertificateConfig(),
		},
		Inbounds: []InboundConfig{
			{
				Tag:      "dccp-in",
				Port:     33445,
				Listen:   "0.0.0.0",
				Protocol: "dccp",
				StreamSettings: &transport.StreamSettings{
					Network:  transport.TransportDCCP,
					Security: "none",
					DCCPSettings: &transport.DCCPSettings{
						CCID:             4,
						ServiceCode:      "V2RY",
						MaxPacketSize:    1500,
						HandshakeTimeout: 15 * time.Second,
						MaxRetries:       3,
					},
				},
			},
		},
		Outbounds: []OutboundConfig{
			{
				Tag:      "direct",
				Protocol: "freedom",
			},
		},
	}
}
